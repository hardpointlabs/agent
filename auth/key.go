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
	"sync"
)

var (
	cachedKeyPair *KeyPair
	cacheOnce     sync.Once
	cachePath     string
	cacheErr      error
)

type KeyPair struct {
	private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

func LoadOrCreateKeyPair(path string) (*KeyPair, error) {
	if path == "" {
		// in container environments we may not have a writable
		// file system, so just generate an ephemeral keypair
		// in memory
		cacheOnce.Do(func() {
			cachedKeyPair, cacheErr = generateKeyPair()
		})
		return cachedKeyPair, cacheErr
	}

	if path == cachePath && cachedKeyPair != nil {
		return cachedKeyPair, nil
	}

	keyFile := filepath.Join(path, "/key")
	log.Printf("Looking for private key %s\n", keyFile)
	if b, err := os.ReadFile(keyFile); err == nil {
		priv := ed25519.PrivateKey(b)
		pub := priv.Public().(ed25519.PublicKey)
		kp := &KeyPair{private: priv, Public: pub}
		cachedKeyPair = kp
		cachePath = path
		return kp, nil
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

	cachedKeyPair = kp
	cachePath = path

	log.Printf("No key pair found, created a new one in %s\n", keyFile)

	return kp, nil
}

func generateKeyPair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &KeyPair{private: priv, Public: pub}, nil
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

func ReadFingerprintFromFile(keyDir string) {
	fingerprint, err := os.ReadFile(filepath.Join(keyDir, "/fingerprint"))
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
	os.Stdout.Write(fingerprint)
	os.Stdout.WriteString("\n")
}
