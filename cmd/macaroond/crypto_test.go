package main
import (
	qt "github.com/frankban/quicktest"
	"testing"
	"encoding/base64"
)

func TestEncryptDecrypt(t *testing.T) {
	c := qt.New(t)
	key := "hello"
	data := []byte("some data")
	boxed := encrypt(data, key)
	unboxed, err := decrypt(boxed, key)
	c.Check(err, qt.Equals, nil)
	c.Check(string(unboxed), qt.Equals, string(data))
}

func TestDecryptKnown(t *testing.T) {
	c := qt.New(t)
	b64data := `AJeHvmYj+RPZ9g1mqANspIdwC0Pr8HPUAVikvOz0xv7JTMK7H3erM0B74VY8TL2lkz2Fc1UN9TXec7mPmQVZ5Q`
	data, err := base64.RawStdEncoding.DecodeString(b64data)
	t.Logf("err: %v", err)
	c.Assert(err, qt.Equals, nil)

	unboxed, err := decrypt(data, "")
	c.Check(err, qt.Equals, nil)
	c.Assert(len(unboxed), qt.Equals, 24)
}