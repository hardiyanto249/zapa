package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"log"
	"os"
)

var aesKey []byte

// InitCrypto loads the AES key from env
func InitCrypto() {
	keyHex := os.Getenv("AES_KEY")
	if keyHex == "" {
		log.Println("Warning: AES_KEY not set in environment. Encryption will fail or be insecure.")
		// We could fatal here, but let's allow start for now, maybe with error.
		return 
	}
	
	var err error
	aesKey, err = hex.DecodeString(keyHex)
	if err != nil {
		log.Printf("Error decoding AES_KEY: %v", err)
	}
}

func getKey() []byte {
	if len(aesKey) == 0 {
		// Attempt lazy load
		InitCrypto()
		if len(aesKey) == 0 {
			// Panic or return nil? 
			// If we return nil, NewCipher will panic.
			// Let's panic with clear message.
			log.Panic("AES_KEY is not initialized. Please set AES_KEY in environment variables.")
		}
	}
	return aesKey
}

func encrypt(text string) (string, error) {
	block, err := aes.NewCipher(getKey())
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, []byte(text), nil)
	return "ENC:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(cryptoText string) (string, error) {
	if cryptoText == "" {
		return "", nil
	}
	
	// Check for ENC: prefix
	if len(cryptoText) < 4 || cryptoText[:4] != "ENC:" {
		return cryptoText, nil // Treat as plaintext
	}
	
	// Remove prefix
	actualCrypto := cryptoText[4:]
	
	data, err := base64.StdEncoding.DecodeString(actualCrypto)
	if err != nil {
		return cryptoText, nil // Should not happen if prefix matches, but safe fallback
	}
	
	block, err := aes.NewCipher(getKey())
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return cryptoText, nil // Too short, assume plaintext
	}
	
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return cryptoText, nil // Decryption failed, assume plaintext (or wrong key)
	}
	
	return string(plaintext), nil
}
