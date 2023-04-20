package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

// setting aes key
var AES_KEY = []byte("#HvL%$ZJCAAiUZnk.@C2qbqCeQB1iXe0")

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

// AES CBC
func encrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS7Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

// AES Decrypt
func decrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS7UnPadding(origData)
	return origData, nil
}

// aes encrypt code
func AESEncrypt(expireDay string) (string, error) {
	expireDayString, errEncrypt := encrypt([]byte(expireDay), AES_KEY)
	if errEncrypt != nil {
		return "", errors.New("code encrypt error")
	}
	return base64.StdEncoding.EncodeToString(expireDayString), nil
}

// aes decrypt code
func AESDecrypt(encryptCode string) (string, error) {
	dataByte, errBase64 := base64.StdEncoding.DecodeString(encryptCode)
	if errBase64 != nil {
		return "", errors.New("code decrypt base error")
	}
	decryptStr, errDecrypt := decrypt([]byte(dataByte), AES_KEY)
	if errDecrypt != nil {
		return "", errors.New("code decrypt error")
	}
	return string(decryptStr), nil
}
