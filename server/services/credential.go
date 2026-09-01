package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"MSMP/server/config"
	"golang.org/x/crypto/ssh"
)

var ErrCredentialKeyMissing = errors.New("credential key not configured: set security.credentialkey (base64 32-byte key)")

type CredentialService struct {
	key []byte
}

func NewCredentialService(cfg *config.Config) (*CredentialService, error) {
	raw, err := base64.StdEncoding.DecodeString(cfg.Security.CredentialKey)
	if err != nil {
		return nil, fmt.Errorf("invalid credential key (must be base64): %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("credential key must be 32 bytes, got %d", len(raw))
	}
	return &CredentialService{key: raw}, nil
}

func (s *CredentialService) Encrypt(plaintext string) (string, error) {
	if s == nil || len(s.key) == 0 {
		return "", ErrCredentialKeyMissing
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (s *CredentialService) Decrypt(ciphertext string) (string, error) {
	if s == nil || len(s.key) == 0 {
		return "", ErrCredentialKeyMissing
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext base64: %w", err)
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func GenerateCredentialKeyB64() (string, error) {
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(k), nil
}

// GenerateSSHKeypair returns an OpenSSH-format public key and PEM private key.
func GenerateSSHKeypair() (pub string, priv string, err error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	privDER := x509.MarshalPKCS1PrivateKey(k)
	privBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER}
	priv = string(pem.EncodeToMemory(privBlock))
	pubSSH, err := ssh.NewPublicKey(&k.PublicKey)
	if err != nil {
		return "", "", err
	}
	pub = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pubSSH)))
	return pub, priv, nil
}
