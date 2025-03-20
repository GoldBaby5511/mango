package goaes

import (
	"encoding/base64"
	"testing"
)

func TestEncrypt(t *testing.T) {
	raw := []byte("password123")
	key := []byte("asdf1234qwer7894")
	str, err := Encrypt(raw, key, nil, CBC, false)
	if err == nil {
		t.Log("suc", str)
	} else {
		t.Fatal("fail", err)
	}
}

func TestDncrypt(t *testing.T) {
	raw := []byte("pqjPM0GJUjlgryzMaslqBAzIknumcdgey1MN+ylWHqY=")
	key := []byte("asdf1234qwer7894")
	str, err := Decrypt(raw, key, nil, CBC, false)
	if err == nil {
		t.Log("suc", str)
	} else {
		t.Fatal("fail", err)
	}
}

func TestEcbEncrypt(t *testing.T) {
	raw := []byte("password1234")
	key := []byte("asdf1234qwer7894")

	str := EcbEncrypt(raw, key)
	res := base64.StdEncoding.EncodeToString(str)
	if res == "eIfgwnbjlf1OymWjfJUFZw==" {
		t.Log("suc ", res)
	} else {
		t.Fatal("fail ", res)
	}
}

func TestEcbDecrypt(t *testing.T) {
	raw := "eIfgwnbjlf1OymWjfJUFZw=="
	key := []byte("asdf1234qwer7894")

	res, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatal(err)
	}
	str := EcbDecrypt(res, key)
	if err == nil && string(str) == "password1234" {
		t.Log("suc ", string(str))
	} else {
		t.Fatal("fail ", err, ", ", string(str))
	}
}
