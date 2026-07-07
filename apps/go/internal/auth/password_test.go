package auth

import "testing"

func TestHashPasswordRoundTrip(t *testing.T) {
	const password = "correct horse battery staple"
	encoded, err := HashPassword(password, DefaultArgon2Params())
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	ok, err := VerifyPassword(password, encoded)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if !ok {
		t.Fatal("VerifyPassword = false for the correct password, want true")
	}
}

func TestHashPasswordProducesUniqueSalt(t *testing.T) {
	p := DefaultArgon2Params()
	a, _ := HashPassword("same", p)
	b, _ := HashPassword("same", p)
	if a == b {
		t.Fatal("two hashes of the same password are identical; salt is not random")
	}
}

func TestVerifyPasswordWrong(t *testing.T) {
	encoded, _ := HashPassword("right", DefaultArgon2Params())
	ok, err := VerifyPassword("wrong", encoded)
	if err != nil {
		t.Fatalf("VerifyPassword: %v", err)
	}
	if ok {
		t.Fatal("VerifyPassword = true for the wrong password, want false")
	}
}

func TestVerifyPasswordMalformed(t *testing.T) {
	if _, err := VerifyPassword("x", "not-a-phc-string"); err == nil {
		t.Fatal("expected an error for a malformed hash, got nil")
	}
}
