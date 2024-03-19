package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/mjwhitta/cli"
	hl "github.com/mjwhitta/hilighter"
)

var r *regexp.Regexp

func init() {
	cli.Align = true
	cli.Banner = hl.Sprintf(
		"%s [OPTIONS] >priv.go 2>pub.go",
		os.Args[0],
	)
	cli.Info("Super simple RSA key generator.")
	cli.Parse()

	r = regexp.MustCompile(`.{1,58}`)
}

func main() {
	var b []byte
	var bits int64 = 4096
	var e error
	var privkey *rsa.PrivateKey

	if cli.NArg() > 0 {
		if bits, e = strconv.ParseInt(cli.Arg(0), 10, 64); e != nil {
			panic(e)
		}
	}

	privkey, e = rsa.GenerateKey(rand.Reader, int(bits))
	if e != nil {
		panic(e)
	}

	b = x509.MarshalPKCS1PrivateKey(privkey)
	printKey("priv", b, os.Stdout)

	b = x509.MarshalPKCS1PublicKey(&privkey.PublicKey)
	printKey("pub", b, os.Stderr)
}

func printKey(name string, key []byte, w io.Writer) {
	var b64 string = base64.StdEncoding.EncodeToString(key)
	var m []string = r.FindAllString(b64, -1)
	var out []string = []string{
		"package main\n",
		"import (",
		"\t\"crypto/rsa\"",
		"\t\"crypto/x509\"",
		"\t\"encoding/base64\"",
		")\n",
	}

	switch name {
	case "priv":
		out = append(out, "var priv *rsa.PrivateKey\n")
	case "pub":
		out = append(out, "var pub *rsa.PublicKey\n")
	}

	out = append(out, "func init() {")
	out = append(out, "\tvar b []byte")
	out = append(out, "")

	out = append(
		out,
		"\tb, _ = base64.StdEncoding.DecodeString(\"\" +",
	)
	for i, line := range m {
		if i == len(m)-1 {
			out = append(out, "\t\t\""+line+"\",")
		} else {
			out = append(out, "\t\t\""+line+"\" +")
		}
	}
	out = append(out, "\t)")

	switch name {
	case "priv":
		out = append(out, "\tpriv, _ = x509.ParsePKCS1PrivateKey(b)")
	case "pub":
		out = append(out, "\tpub, _ = x509.ParsePKCS1PublicKey(b)")
	}

	out = append(out, "}\n")

	io.WriteString(w, strings.Join(out, "\n"))
}
