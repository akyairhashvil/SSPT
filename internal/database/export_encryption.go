package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
)

type encryptedExport struct {
	Encrypted bool   `json:"encrypted"`
	Nonce     string `json:"nonce"`
	Data      string `json:"data"`
}

func encryptData(payload []byte, passphrase string) ([]byte, error) {
	hash := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(hash[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, payload, nil)
	wrapped := encryptedExport{
		Encrypted: true,
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
		Data:      base64.StdEncoding.EncodeToString(ciphertext),
	}
	return json.Marshal(wrapped)
}
