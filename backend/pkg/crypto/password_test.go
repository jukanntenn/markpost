package crypto

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword"

	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	if hashed == "" {
		t.Fatal("hashed password is empty")
	}

	if hashed == password {
		t.Fatal("hashed password should not equal plain password")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword"

	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	ok, err := CheckPassword(password, hashed)
	if err != nil {
		t.Fatalf("CheckPassword error: %v", err)
	}

	if !ok {
		t.Fatal("expected password to match")
	}

	ok, err = CheckPassword("wrongpassword", hashed)
	if err != nil {
		t.Fatalf("CheckPassword error: %v", err)
	}

	if ok {
		t.Fatal("expected password not to match")
	}
}
