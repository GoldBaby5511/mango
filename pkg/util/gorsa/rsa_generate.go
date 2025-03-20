package gorsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

func GenerateRSAKey(bits int, keyType PriKeyType) (private, public []byte, err error) {
	// 生成私钥文件
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		panic(err)
	}
	var derStream []byte
	if keyType == PKCS8 {
		derStream, err = x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			panic(err)
		}
	} else {
		derStream = x509.MarshalPKCS1PrivateKey(privateKey)
	}
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	private = pem.EncodeToMemory(block)
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	public = pem.EncodeToMemory(block)
	return
}
