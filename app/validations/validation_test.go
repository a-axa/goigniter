package validations_test

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/iskandar221201/goigniter/app/validations"
)

func TestRegisterInput(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name    string
		input   validations.RegisterInput
		wantErr bool
	}{
		{
			name:  "valid",
			input: validations.RegisterInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"},
		},
		{
			name:    "invalid email",
			input:   validations.RegisterInput{Name: "Budi", Email: "nope", Password: "rahasia123"},
			wantErr: true,
		},
		{
			name:    "short password",
			input:   validations.RegisterInput{Name: "Budi", Email: "budi@example.com", Password: "1234567"},
			wantErr: true,
		},
		{
			name:  "password exactly 8 characters is allowed",
			input: validations.RegisterInput{Name: "Budi", Email: "budi@example.com", Password: "12345678"},
		},
		{
			name:    "one character name",
			input:   validations.RegisterInput{Name: "B", Email: "budi@example.com", Password: "rahasia123"},
			wantErr: true,
		},
		{
			name:    "empty payload",
			input:   validations.RegisterInput{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoginInput(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name    string
		input   validations.LoginInput
		wantErr bool
	}{
		{name: "valid", input: validations.LoginInput{Email: "budi@example.com", Password: "x"}},
		{name: "invalid email", input: validations.LoginInput{Email: "nope", Password: "x"}, wantErr: true},
		{name: "missing password", input: validations.LoginInput{Email: "budi@example.com"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateUserInput(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name    string
		input   validations.CreateUserInput
		wantErr bool
	}{
		{
			name:  "valid without role",
			input: validations.CreateUserInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"},
		},
		{
			name:  "valid admin role",
			input: validations.CreateUserInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123", Role: "admin"},
		},
		{
			name:    "invalid role",
			input:   validations.CreateUserInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123", Role: "owner"},
			wantErr: true,
		},
		{
			name:    "missing name",
			input:   validations.CreateUserInput{Email: "budi@example.com", Password: "rahasia123"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateUserInput(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name    string
		input   validations.UpdateUserInput
		wantErr bool
	}{
		{name: "empty payload is allowed", input: validations.UpdateUserInput{}},
		{name: "name only", input: validations.UpdateUserInput{Name: "Budi Santoso"}},
		{name: "role only", input: validations.UpdateUserInput{Role: "admin"}},
		{name: "invalid role", input: validations.UpdateUserInput{Role: "owner"}, wantErr: true},
		{name: "name too short", input: validations.UpdateUserInput{Name: "B"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
