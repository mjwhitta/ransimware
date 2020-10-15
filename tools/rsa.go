package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"io"
	"os"
	"regexp"
	"strings"
)

var r *regexp.Regexp

func init() {
	r = regexp.MustCompile(`.{1,59}`)
}

func main() {
	var b []byte
	var e error
	var privkey *rsa.PrivateKey

	if privkey, e = rsa.GenerateKey(rand.Reader, 4096); e != nil {
		panic(e)
	}

	b = x509.MarshalPKCS1PrivateKey(privkey)
	printKey("priv", b, os.Stdout)

	b = x509.MarshalPKCS1PublicKey(&privkey.PublicKey)
	printKey("pub", b, os.Stderr)
}

func printKey(name string, key []byte, w io.Writer) {
	var b64 string = base64.StdEncoding.EncodeToString(key)
	var out = []string{
		"package main\n",
		"import (",
		"\t\"crypto/rsa\"",
		"\t\"crypto/x509\"",
		"\t\"encoding/base64\"",
		"\t\"strings\"",
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
	out = append(out, "\tvar b64 = []string{")

	for _, line := range r.FindAllString(b64, -1) {
		out = append(out, "\t\t\""+line+"\",")
	}

	out = append(out, "\t}")
	out = append(out, "\tvar e error\n")

	out = append(
		out,
		strings.Join(
			[]string{
				"\tb, e = base64.StdEncoding.DecodeString(",
				"strings.Join(b64, \"\")",
				")",
			},
			"",
		),
	)
	out = append(out, "\tif e != nil {")
	out = append(out, "\t\tpanic(e)")
	out = append(out, "\t}\n")

	switch name {
	case "priv":
		out = append(
			out,
			"\tif priv, e = x509.ParsePKCS1PrivateKey(b); e != nil {",
		)
	case "pub":
		out = append(
			out,
			"\tif pub, e = x509.ParsePKCS1PublicKey(b); e != nil {",
		)
	}

	out = append(out, "\t\tpanic(e)")
	out = append(out, "\t}")
	out = append(out, "}\n")

	io.WriteString(w, strings.Join(out, "\n"))
}
