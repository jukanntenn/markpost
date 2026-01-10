package utils

import "testing"

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"normal string", "password"},
		{"complex string", "ComplexP@ssw0rd!@#"},
		{"very long string", "aVeryLongPasswordThatIsMuchLongerThanTypicalPasswords123456789"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := HashPassword(test.input)
			if err != nil {
				t.Errorf("HashPassword(%s) error = %v", test.input, err)
			}
			if got == "" {
				t.Errorf("HashPassword(%s) = empty string", test.input)
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	hashed, err := HashPassword("password")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		hash    string
		wantOK  bool
		wantErr bool
	}{
		{"matching password", "password", hashed, true, false},
		{"wrong password", "wrongpassword", hashed, false, false},
		{"invalid hash format", "password", "not-a-bcrypt-hash", false, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok, err := CheckPassword(test.input, test.hash)
			if ok != test.wantOK {
				t.Errorf("CheckPassword(%s) ok = %v, want %v", test.input, ok, test.wantOK)
			}
			if (err != nil) != test.wantErr {
				t.Errorf("CheckPassword(%s) err presence = %v, want %v (err=%v)", test.input, err != nil, test.wantErr, err)
			}
		})
	}
}
