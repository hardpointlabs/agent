package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"
)

// don't want to expose the private half outside the auth package to prevent
// people accidentally sending it down the wire :)
type KeyPair struct {
	private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

func LoadOrCreateKeyPair(path string) (*KeyPair, error) {
	keyFile := filepath.Join(path, "/key")
	log.Printf("Looking for private key %s\n", keyFile)
	if b, err := os.ReadFile(keyFile); err == nil {
		priv := ed25519.PrivateKey(b)
		pub := priv.Public().(ed25519.PublicKey)
		return &KeyPair{private: priv, Public: pub}, nil
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyFile, priv, 0600); err != nil {
		return nil, err
	}

	log.Printf("No key pair found, created a new one in %s\n", keyFile)

	return &KeyPair{private: priv, Public: pub}, nil
}

func (k *KeyPair) Sign(input []byte) ([]byte, error) {
	sig, err := k.private.Sign(rand.Reader, input, nil)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

func (k *KeyPair) Fingerprint() []byte {
	h := sha256.Sum256(k.Public)
	return []byte(hex.EncodeToString(h[:]))
}
