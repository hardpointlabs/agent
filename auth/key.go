package auth

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

	kp := &KeyPair{private: priv, Public: pub}

	fingerprintFile := filepath.Join(path, "/fingerprint")
	if err := os.WriteFile(fingerprintFile, kp.Fingerprint(), 0644); err != nil {
		return nil, err
	}

	log.Printf("No key pair found, created a new one in %s\n", keyFile)

	return kp, nil
}

func (k *KeyPair) Sign(input []byte) ([]byte, error) {
	sig, err := k.private.Sign(nil, input, &ed25519.Options{Hash: crypto.Hash(0)})
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(sig)
	return []byte(encoded), nil
}

func (k *KeyPair) Fingerprint() []byte {
	h := sha256.Sum256(k.Public)
	return []byte(hex.EncodeToString(h[:]))[:32]
}
