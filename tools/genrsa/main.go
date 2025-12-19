package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mjwhitta/cli"
)

var reWrap *regexp.Regexp = regexp.MustCompile(`.{1,58}`)

func init() {
	cli.Align = true
	cli.Banner = "" +
		filepath.Base(os.Args[0]) + " [OPTIONS] >priv.go 2>pub.go"

	cli.Info("Super simple RSA key generator.")
	cli.Parse()
}

func main() {
	var b []byte
	var bits int64 = 4096
	var e error
	var privKey *rsa.PrivateKey

	if cli.NArg() > 0 {
		if bits, e = strconv.ParseInt(cli.Arg(0), 10, 64); e != nil {
			panic(e)
		}
	}

	privKey, e = rsa.GenerateKey(rand.Reader, int(bits))
	if e != nil {
		panic(e)
	}

	b = x509.MarshalPKCS1PrivateKey(privKey)
	printKey("priv", b, os.Stdout)

	b = x509.MarshalPKCS1PublicKey(&privKey.PublicKey)
	printKey("pub", b, os.Stderr)
}

func printKey(name string, key []byte, w io.Writer) {
	var b64 string = base64.StdEncoding.EncodeToString(key)
	var m []string = reWrap.FindAllString(b64, -1)
	var sb strings.Builder

	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"crypto/rsa\"\n")
	sb.WriteString("\t\"crypto/x509\"\n")
	sb.WriteString("\t\"encoding/base64\"\n")
	sb.WriteString(")\n\n")

	switch name {
	case "priv":
		sb.WriteString("var priv *rsa.PrivateKey\n\n")
	case "pub":
		sb.WriteString("var pub *rsa.PublicKey\n\n")
	}

	sb.WriteString("func init() {\n")
	sb.WriteString("\tvar b []byte\n\n")
	sb.WriteString(
		"\tb, _ = base64.StdEncoding.DecodeString(\"\"",
	)

	for _, line := range m {
		sb.WriteString(" +\n\t\t\"" + line + "\"")
	}

	sb.WriteString(",\n\t)\n\n")

	switch name {
	case "priv":
		sb.WriteString("\tpriv, _ = x509.ParsePKCS1PrivateKey(b)\n")
	case "pub":
		sb.WriteString("\tpub, _ = x509.ParsePKCS1PublicKey(b)\n")
	}

	sb.WriteString("}\n")

	_, _ = io.WriteString(w, sb.String())
}
