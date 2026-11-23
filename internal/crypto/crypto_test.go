package crypto

import (
	"bytes"
	"testing"
)

func TestArgon2DeriveAndVerify(t *testing.T) {
	password := "SecretMasterPassword123!"
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}
	if len(salt) != SaltLen {
		t.Fatalf("expected salt len %d, got %d", SaltLen, len(salt))
	}

	key := DeriveKey(password, salt)
	if len(key) != ArgonKeyLen {
		t.Fatalf("expected key len %d, got %d", ArgonKeyLen, len(key))
	}

	verifier := ComputeVerifier(key, salt)
	if !VerifyKey(key, salt, verifier) {
		t.Fatalf("VerifyKey should succeed for correct key")
	}

	wrongKey := DeriveKey("WrongPassword", salt)
	if VerifyKey(wrongKey, salt, verifier) {
		t.Fatalf("VerifyKey should fail for incorrect key")
	}
}

func TestEncryptDecryptAESGCM(t *testing.T) {
	key := DeriveKey("StrongMasterKey!", []byte("fixed_test_salt_12345678"))
	original := []byte("vps-root-password-ssh-credentials-12345")

	encrypted, err := EncryptAESGCM(original, key)
	if err != nil {
		t.Fatalf("EncryptAESGCM failed: %v", err)
	}

	if encrypted == string(original) {
		t.Fatalf("ciphertext matches plaintext")
	}

	decrypted, err := DecryptAESGCM(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptAESGCM failed: %v", err)
	}

	if !bytes.Equal(decrypted, original) {
		t.Fatalf("expected %s, got %s", string(original), string(decrypted))
	}

	// Test with wrong key
	wrongKey := DeriveKey("OtherKey", []byte("fixed_test_salt_12345678"))
	_, errWrong := DecryptAESGCM(encrypted, wrongKey)
	if errWrong == nil {
		t.Fatalf("DecryptAESGCM should fail with wrong key")
	}
}

func TestMemclr(t *testing.T) {
	b := []byte{1, 2, 3, 4, 5}
	Memclr(b)
	for i, v := range b {
		if v != 0 {
			t.Fatalf("byte at %d is not zero: %d", i, v)
		}
	}
}
