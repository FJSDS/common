package utils

import (
	"bytes"
	"crypto/des"
	"errors"
)

//ECB PKCS5Padding
func PKCS5Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

//ECB PKCS5UNPadding
func PKCS5UNPadding(origData []byte) []byte {
	length := len(origData)
	unPadding := int(origData[length-1])
	return origData[:(length - unPadding)]
}

func DesEncrypt(origData, key []byte) ([]byte, error) {
	if len(origData) < 1 || len(key) < 1 {
		return nil, errors.New("wrong data or key")
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	if len(origData)%bs != 0 {
		return nil, errors.New("wrong padding")
	}
	out := make([]byte, len(origData))
	dst := out
	for len(origData) > 0 {
		block.Encrypt(dst, origData[:bs])
		origData = origData[bs:]
		dst = dst[bs:]
	}
	return out, nil
}

func DesDecrypt(crypt, key []byte) ([]byte, error) {
	if len(crypt) < 1 || len(key) < 1 {
		return nil, errors.New("wrong data or key")
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(crypt))
	dst := out
	bs := block.BlockSize()
	if len(crypt)%bs != 0 {
		return nil, errors.New("wrong crypt size")
	}

	for len(crypt) > 0 {
		block.Decrypt(dst, crypt[:bs])
		crypt = crypt[bs:]
		dst = dst[bs:]
	}

	return out, nil
}

//[golang ECB 3DES Encrypt]
func TripleEcbDesEncrypt(origData, key []byte) ([]byte, error) {
	tKey := make([]byte, 24, 24)
	copy(tKey, key)
	k1 := tKey[:8]
	k2 := tKey[8:16]
	k3 := tKey[16:]

	block, err := des.NewCipher(k1)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	origData = PKCS5Padding(origData, bs)

	buf1, err := DesEncrypt(origData, k1)
	if err != nil {
		return nil, err
	}
	buf2, err := DesDecrypt(buf1, k2)
	if err != nil {
		return nil, err
	}
	out, err := DesEncrypt(buf2, k3)
	if err != nil {
		return nil, err
	}
	return out, nil
}

//[golang ECB 3DES Decrypt]
func TripleEcbDesDecrypt(crypt, key []byte) ([]byte, error) {
	tKey := make([]byte, 24, 24)
	copy(tKey, key)
	k1 := tKey[:8]
	k2 := tKey[8:16]
	k3 := tKey[16:]
	buf1, err := DesDecrypt(crypt, k3)
	if err != nil {
		return nil, err
	}
	buf2, err := DesEncrypt(buf1, k2)
	if err != nil {
		return nil, err
	}
	out, err := DesDecrypt(buf2, k1)
	if err != nil {
		return nil, err
	}
	out = PKCS5UNPadding(out)
	return out, nil
}
