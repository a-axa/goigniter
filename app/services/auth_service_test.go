package services

import (
	"testing"

	"github.com/iskandar221201/goigniter/app/helpers"
	"github.com/iskandar221201/goigniter/app/models"
	"github.com/iskandar221201/goigniter/app/validations"
	"github.com/iskandar221201/goigniter/system"
)

func TestAuthServiceRegister(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, newTestConfig())

	input := validations.RegisterInput{
		Name:     "Budi",
		Email:    "budi@example.com",
		Password: "rahasia123",
	}

	user, err := svc.Register(input)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.ID == 0 {
		t.Error("expected user ID to be populated")
	}
	if user.Name != input.Name || user.Email != input.Email {
		t.Errorf("user = %+v, want name %q and email %q", user, input.Name, input.Email)
	}
	if user.Password == input.Password {
		t.Error("password must be stored hashed")
	}
	if !helpers.CheckPassword(input.Password, user.Password) {
		t.Error("stored hash does not match the plain password")
	}
}

func TestAuthServiceRegisterDuplicateEmail(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, newTestConfig())

	input := validations.RegisterInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"}
	if _, err := svc.Register(input); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	if _, err := svc.Register(input); err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestAuthServiceLogin(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig()
	svc := NewAuthService(db, cfg)

	input := validations.RegisterInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"}
	registered, err := svc.Register(input)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	token, err := svc.Login(validations.LoginInput{Email: input.Email, Password: input.Password})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}

	claims, err := system.ParseToken(token, cfg.JWT.Secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != registered.ID {
		t.Errorf("UserID = %d, want %d", claims.UserID, registered.ID)
	}
	if claims.Email != input.Email {
		t.Errorf("Email = %q, want %q", claims.Email, input.Email)
	}
	if claims.Role != "user" {
		t.Errorf("Role = %q, want default %q", claims.Role, "user")
	}
}

func TestAuthServiceLoginInvalidCredentials(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, newTestConfig())

	input := validations.RegisterInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"}
	if _, err := svc.Register(input); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	tests := []struct {
		name  string
		input validations.LoginInput
	}{
		{name: "wrong password", input: validations.LoginInput{Email: input.Email, Password: "salah-sekali"}},
		{name: "unknown email", input: validations.LoginInput{Email: "ghost@example.com", Password: input.Password}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := svc.Login(tt.input)
			if err == nil {
				t.Fatal("expected an error")
			}
			if err.Error() != "invalid credentials" {
				t.Errorf("error = %q, want %q", err.Error(), "invalid credentials")
			}
			if token != "" {
				t.Errorf("token = %q, want empty", token)
			}
		})
	}
}

func TestAuthServiceLoginSoftDeletedUser(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db, newTestConfig())

	input := validations.RegisterInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"}
	registered, err := svc.Register(input)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if err := db.Delete(&models.User{}, registered.ID).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	if _, err := svc.Login(validations.LoginInput{Email: input.Email, Password: input.Password}); err == nil {
		t.Fatal("soft-deleted user must not be able to log in")
	}
}
