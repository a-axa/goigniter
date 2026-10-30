package system

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
)

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) APIResponse {
	t.Helper()

	var body APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return body
}

func TestSuccess(t *testing.T) {
	rec := httptest.NewRecorder()

	err := Success(rec, http.StatusCreated, "Created", map[string]string{"id": "1"})
	if err != nil {
		t.Fatalf("Success() error = %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	body := decodeBody(t, rec)
	if !body.Status {
		t.Error("Status = false, want true")
	}
	if body.Message != "Created" {
		t.Errorf("Message = %q, want %q", body.Message, "Created")
	}
	if body.Data == nil {
		t.Error("Data must not be omitted")
	}
}

func TestError(t *testing.T) {
	rec := httptest.NewRecorder()

	if err := Error(rec, http.StatusNotFound, "User not found"); err != nil {
		t.Fatalf("Error() error = %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	body := decodeBody(t, rec)
	if body.Status {
		t.Error("Status = true, want false")
	}
	if body.Message != "User not found" {
		t.Errorf("Message = %q, want %q", body.Message, "User not found")
	}
}

func TestValidationError(t *testing.T) {
	rec := httptest.NewRecorder()

	errs := map[string]string{"email": "email is required"}
	if err := ValidationError(rec, errs); err != nil {
		t.Fatalf("ValidationError() error = %v", err)
	}

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}

	body := decodeBody(t, rec)
	if body.Status {
		t.Error("Status = true, want false")
	}
	if body.Message != "Validation failed" {
		t.Errorf("Message = %q, want %q", body.Message, "Validation failed")
	}
	if body.Errors == nil {
		t.Error("Errors must be set")
	}
}

type parseInput struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=8"`
	Name     string `validate:"required,max=5"`
}

func TestParseValidationErrors(t *testing.T) {
	err := validator.New().Struct(parseInput{Email: "bad", Password: "short", Name: "too-long"})
	if err == nil {
		t.Fatal("expected validation error")
	}

	got := ParseValidationErrors(err)
	if got == nil {
		t.Fatal("ParseValidationErrors() returned nil")
	}

	want := map[string]string{
		"email":    "must be a valid email address",
		"password": "must be at least 8 characters",
		"name":     "must be at most 5 characters",
	}

	if len(got) != len(want) {
		t.Fatalf("got %d errors, want %d: %v", len(got), len(want), got)
	}
	for field, msg := range want {
		if got[field] != msg {
			t.Errorf("errors[%q] = %q, want %q", field, got[field], msg)
		}
	}
}

func TestParseValidationErrorsRequiredAndNonValidation(t *testing.T) {
	err := validator.New().Struct(parseInput{})
	errs := ParseValidationErrors(err)

	if errs["email"] != "Email is required" {
		t.Errorf("errors[email] = %q, want %q", errs["email"], "Email is required")
	}

	if got := ParseValidationErrors(errors.New("plain error")); got != nil {
		t.Errorf("ParseValidationErrors(non-validation) = %v, want nil", got)
	}
}

func TestWrap(t *testing.T) {
	t.Run("handler error becomes 500", func(t *testing.T) {
		handler := Wrap(func(w http.ResponseWriter, r *http.Request) error {
			return errors.New("boom")
		})

		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if body := decodeBody(t, rec); body.Message != "boom" {
			t.Errorf("Message = %q, want %q", body.Message, "boom")
		}
	})

	t.Run("does not overwrite an already written response", func(t *testing.T) {
		handler := Wrap(func(w http.ResponseWriter, r *http.Request) error {
			_ = ValidationError(w, map[string]string{"email": "invalid"})
			return errors.New("validation failed")
		})

		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(http.MethodPost, "/", nil))

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
		}
		if body := decodeBody(t, rec); body.Message != "Validation failed" {
			t.Errorf("Message = %q, want %q", body.Message, "Validation failed")
		}
	})

	t.Run("nil error keeps handler response", func(t *testing.T) {
		handler := Wrap(func(w http.ResponseWriter, r *http.Request) error {
			return Success(w, http.StatusOK, "OK", nil)
		})

		rec := httptest.NewRecorder()
		handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}
