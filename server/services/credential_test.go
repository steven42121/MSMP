package services

import (
	"crypto/rand"
	"testing"
)

func TestCredentialEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	svc := &CredentialService{key: key}

	plaintexts := []string{
		"password",
		"admin123!@#",
		"root:secret",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIE...\n-----END RSA PRIVATE KEY-----",
	}

	for _, pt := range plaintexts {
		enc, err := svc.Encrypt(pt)
		if err != nil {
			t.Fatalf("Encrypt(%q) error: %v", pt, err)
		}
		if enc == pt {
			t.Errorf("Encrypt(%q) returned plaintext unchanged", pt)
		}

		dec, err := svc.Decrypt(enc)
		if err != nil {
			t.Fatalf("Decrypt error: %v", err)
		}
		if dec != pt {
			t.Errorf("Decrypt(Encrypt(%q)) = %q, want %q", pt, dec, pt)
		}
	}
}

func TestCredentialDecryptWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	svc1 := &CredentialService{key: key1}
	svc2 := &CredentialService{key: key2}

	enc, err := svc1.Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc2.Decrypt(enc); err == nil {
		t.Error("Decrypt with wrong key should fail")
	}
}

func TestCredentialMissingKey(t *testing.T) {
	svc := &CredentialService{} // empty key
	if _, err := svc.Encrypt("x"); err == nil {
		t.Error("Encrypt with empty key should fail")
	}
	if _, err := svc.Decrypt("x"); err == nil {
		t.Error("Decrypt with empty key should fail")
	}
}