package helpers_test

import (
	"strings"
	"testing"

	"github.com/iskandar221201/goigniter/app/helpers"
)

func TestHashPasswordProducesBcryptHash(t *testing.T) {
	const plain = "secret123"

	hash, err := helpers.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == plain {
		t.Fatal("hash must not equal the plain password")
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("expected a bcrypt hash, got %q", hash)
	}
}

func TestHashPasswordUsesUniqueSalt(t *testing.T) {
	const plain = "secret123"

	first, err := helpers.HashPassword(plain)
	if err != nil {
		t.Fatalf("first HashPassword() error = %v", err)
	}

	second, err := helpers.HashPassword(plain)
	if err != nil {
		t.Fatalf("second HashPassword() error = %v", err)
	}

	if first == second {
		t.Fatal("hashing the same password twice must produce different hashes")
	}
}

func TestCheckPassword(t *testing.T) {
	const plain = "secret123"

	hash, err := helpers.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name  string
		plain string
		hash  string
		want  bool
	}{
		{name: "matching password", plain: plain, hash: hash, want: true},
		{name: "wrong password", plain: "wrong", hash: hash, want: false},
		{name: "empty password", plain: "", hash: hash, want: false},
		{name: "malformed hash", plain: plain, hash: "not-a-hash", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := helpers.CheckPassword(tt.plain, tt.hash); got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}
