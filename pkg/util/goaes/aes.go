package goaes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"

	"mango/pkg/log"
)

type AESMODE int

const (
	ECB AESMODE = iota
	CBC
	CFB
	OFB
	CTR
	GCM
)

// 使用PKCS7进行填充
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}
func PKCS7UnPadding(ciphertext []byte) []byte {
	length := len(ciphertext)
	//去掉最后一次的padding
	unpadding := int(ciphertext[length-1])
	return ciphertext[:(length - unpadding)]
}

func ZeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

func ZeroUnPadding(origData []byte) []byte {
	return bytes.TrimRightFunc(origData, func(r rune) bool {
		return r == rune(0)
	})
}

func AesCBCEncrypt(plaintext, key, iv []byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Debug("", "错误 - %v", err.Error())
		return nil, err
	}
	//填充内容，如果不足16位字符
	blockSize := block.BlockSize()
	originData := PKCS7Padding(plaintext, blockSize)
	//加密方式
	blockMode := cipher.NewCBCEncrypter(block, iv)
	//加密，输出到[]byte数组
	crypted := make([]byte, len(originData))
	blockMode.CryptBlocks(crypted, originData)
	return crypted, nil
}

func AesCBCDecrypt(ciphertext, key, iv []byte) (plaintext []byte, err error) {
	//生成密码数据块cipher.Block
	block, _ := aes.NewCipher(key)
	//解密模式
	blockMode := cipher.NewCBCDecrypter(block, iv)
	//输出到[]byte数组
	originData := make([]byte, len(ciphertext))
	blockMode.CryptBlocks(originData, ciphertext)
	//去除填充,并返回
	return PKCS7UnPadding(originData), nil
}

func AesCFBEncrypt(plaintext, key, iv []byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext = make([]byte, aes.BlockSize+len(plaintext))
	if iv == nil {
		iv = ciphertext[:aes.BlockSize]
	}
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

func AesCFBDecrypt(ciphertext, key, iv []byte) (plaintext []byte, err error) {
	block, _ := aes.NewCipher(key)
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext too short")
	}
	if iv == nil {
		iv = ciphertext[:aes.BlockSize]
	}
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}

func Encrypt(rawData, key, iv []byte, mode AESMODE, base64Code bool) ([]byte, error) {
	var encrypted []byte
	switch mode {
	case CBC:
		encrypted, _ = AesCBCEncrypt(rawData, key, iv)
	case CFB:
		encrypted, _ = AesCFBEncrypt(rawData, key, iv)
	default:
		return nil, nil
	}
	if base64Code {
		encrypted = []byte(base64.StdEncoding.EncodeToString(encrypted))
	}
	return encrypted, nil
}

func Decrypt(encrypted, key, iv []byte, mode AESMODE, base64Code bool) ([]byte, error) {
	if base64Code {
		encrypted, _ = base64.StdEncoding.DecodeString(string(encrypted))
	}
	switch mode {
	case CBC:
		return AesCBCDecrypt(encrypted, key, iv)
	case CFB:
		return AesCFBDecrypt(encrypted, key, iv)
	default:
		return nil, nil
	}
}

func EcbDecrypt(data, key []byte) []byte {
	block, _ := aes.NewCipher(key)
	decrypted := make([]byte, len(data))
	size := block.BlockSize()

	for bs, be := 0, size; bs < len(data); bs, be = bs+size, be+size {
		block.Decrypt(decrypted[bs:be], data[bs:be])
	}

	return PKCS7UnPadding(decrypted)
}

func EcbEncrypt(data, key []byte) []byte {
	block, _ := aes.NewCipher(key)
	data = PKCS7Padding(data, block.BlockSize())
	decrypted := make([]byte, len(data))
	size := block.BlockSize()

	for bs, be := 0, size; bs < len(data); bs, be = bs+size, be+size {
		block.Encrypt(decrypted[bs:be], data[bs:be])
	}

	return decrypted
}
