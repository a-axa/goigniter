package system

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/iskandar221201/goigniter/config"
)

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:      "test-secret",
			ExpireHours: 1,
		},
	}
}

func TestGenerateAndParseToken(t *testing.T) {
	cfg := testConfig()
	want := JWTClaims{UserID: 42, Email: "user@example.com", Role: "admin"}

	token, err := GenerateToken(want, cfg)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ParseToken(token, cfg.JWT.Secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.UserID != want.UserID {
		t.Errorf("UserID = %d, want %d", claims.UserID, want.UserID)
	}
	if claims.Email != want.Email {
		t.Errorf("Email = %q, want %q", claims.Email, want.Email)
	}
	if claims.Role != want.Role {
		t.Errorf("Role = %q, want %q", claims.Role, want.Role)
	}
}

func TestGenerateTokenSetsExpiry(t *testing.T) {
	cfg := testConfig()

	before := time.Now()
	token, err := GenerateToken(JWTClaims{UserID: 1}, cfg)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ParseToken(token, cfg.JWT.Secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		t.Fatal("IssuedAt and ExpiresAt must be set")
	}

	wantExpiry := before.Add(time.Duration(cfg.JWT.ExpireHours) * time.Hour)
	delta := claims.ExpiresAt.Time.Sub(wantExpiry)
	if delta < -2*time.Second || delta > 2*time.Second {
		t.Errorf("ExpiresAt = %v, want about %v", claims.ExpiresAt.Time, wantExpiry)
	}
}

func TestParseTokenRejectsInvalidInput(t *testing.T) {
	cfg := testConfig()

	valid, err := GenerateToken(JWTClaims{UserID: 1}, cfg)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	t.Run("wrong secret", func(t *testing.T) {
		if _, err := ParseToken(valid, "another-secret"); err == nil {
			t.Fatal("expected error for wrong secret")
		}
	})

	t.Run("tampered token", func(t *testing.T) {
		tampered := valid[:len(valid)-1] + "x"
		if _, err := ParseToken(tampered, cfg.JWT.Secret); err == nil {
			t.Fatal("expected error for tampered token")
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		if _, err := ParseToken("not-a-token", cfg.JWT.Secret); err == nil {
			t.Fatal("expected error for malformed token")
		}
	})

	t.Run("unexpected signing method", func(t *testing.T) {
		none := jwt.NewWithClaims(jwt.SigningMethodNone, JWTClaims{UserID: 1})
		signed, err := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("sign none token: %v", err)
		}

		if _, err := ParseToken(signed, cfg.JWT.Secret); err == nil {
			t.Fatal("expected error for non-HMAC signing method")
		}
	})
}

func TestParseTokenRejectsExpiredToken(t *testing.T) {
	cfg := testConfig()
	cfg.JWT.ExpireHours = -1

	token, err := GenerateToken(JWTClaims{UserID: 1}, cfg)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if _, err := ParseToken(token, cfg.JWT.Secret); err == nil {
		t.Fatal("expected error for expired token")
	}
}
