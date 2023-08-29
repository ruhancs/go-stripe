package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type Encryption struct {
	Key []byte//secret key para encriptografar
}

func (e *Encryption) Encrypt(text string) (string, error) {
	plainText := []byte(text)

	block,err := aes.NewCipher(e.Key)
	if err != nil {
		return "", err
	}

	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _,err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)
	//retorna o dado encryptografado em formato de string
	return base64.URLEncoding.EncodeToString(cipherText), nil
}

func (e *Encryption) Decrypt(cryptoText string) (string, error) {
	// transformar de string para slice de byte
	cipherText,_ := base64.URLEncoding.DecodeString(cryptoText)
	
	block,err := aes.NewCipher(e.Key)
	if err != nil {
		return "", err
	}

	if len(cipherText) < aes.BlockSize {
		return "", err
	}

	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherText, cipherText)

	return fmt.Sprintf("%s", cipherText), nil
}