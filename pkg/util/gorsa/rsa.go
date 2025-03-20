package gorsa

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

type PriKeyType uint

const (
	PKCS1 PriKeyType = iota
	PKCS8
)

//私钥签名
func Sign(data, privateKey []byte, keyType PriKeyType) ([]byte, error) {
	h := sha256.New()
	h.Write(data)
	hashed := h.Sum(nil)
	priv, err := getPriKey(privateKey, keyType)
	if err != nil {
		return nil, err
	}
	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed)
}

//公钥验证
func SignVer(data, signature, publicKey []byte) error {
	hashed := sha256.Sum256(data)
	//获取公钥
	pub, err := getPubKey(publicKey)
	if err != nil {
		return err
	}
	//验证签名
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], signature)
}

// 公钥加密
func Encrypt(data, publicKey []byte) ([]byte, error) {
	//获取公钥
	pub, err := getPubKey(publicKey)
	if err != nil {
		return nil, err
	}
	//加密
	return rsa.EncryptPKCS1v15(rand.Reader, pub, data)
}

// 私钥解密,privateKey为pem文件里的字符
func Decrypt(encData, privateKey []byte, keyType PriKeyType) ([]byte, error) {
	//解析PKCS1a或者PKCS8格式的私钥
	priv, err := getPriKey(privateKey, keyType)
	if err != nil {
		return nil, err
	}
	// 解密
	return rsa.DecryptPKCS1v15(rand.Reader, priv, encData)
}

func getPubKey(publicKey []byte) (*rsa.PublicKey, error) {
	//解密pem格式的公钥
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("public key error")
	}
	// 解析公钥
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	// 类型断言
	if pub, ok := pubInterface.(*rsa.PublicKey); ok {
		return pub, nil
	} else {
		return nil, errors.New("public key error")
	}
}

func getPriKey(privateKey []byte, keyType PriKeyType) (*rsa.PrivateKey, error) {
	//获取私钥
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("private key error!")
	}
	var priKey *rsa.PrivateKey
	var err error
	switch keyType {
	case PKCS1:
		{
			priKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
		}
	case PKCS8:
		{
			prkI, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			priKey = prkI.(*rsa.PrivateKey)
		}
	default:
		{
			return nil, errors.New("unsupport private key type")
		}
	}
	return priKey, nil
}
