package system

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBaseControllerRespondHelpers(t *testing.T) {
	bc := &BaseController{}

	t.Run("Respond", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := bc.Respond(rec, http.StatusOK, "OK", nil); err != nil {
			t.Fatalf("Respond() error = %v", err)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("RespondError", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := bc.RespondError(rec, http.StatusForbidden, "Forbidden"); err != nil {
			t.Fatalf("RespondError() error = %v", err)
		}
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("RespondValidation", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if err := bc.RespondValidation(rec, map[string]string{"email": "invalid"}); err != nil {
			t.Fatalf("RespondValidation() error = %v", err)
		}
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want 422", rec.Code)
		}
	})
}

type bodyInput struct {
	Email string `validate:"required,email"`
}

func TestBodyParse(t *testing.T) {
	bc := &BaseController{}

	t.Run("invalid JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{not json"))

		if err := bc.BodyParse(rec, req, &bodyInput{}); err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"nope"}`))

		if err := bc.BodyParse(rec, req, &bodyInput{}); err == nil {
			t.Fatal("expected validation error")
		}
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want 422", rec.Code)
		}
	})

	t.Run("valid body", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"user@example.com"}`))
		out := &bodyInput{}

		if err := bc.BodyParse(rec, req, out); err != nil {
			t.Fatalf("BodyParse() error = %v", err)
		}
		if out.Email != "user@example.com" {
			t.Errorf("Email = %q, want user@example.com", out.Email)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("expected no response body, got %q", rec.Body.String())
		}
	})
}

func TestCurrentUser(t *testing.T) {
	bc := &BaseController{}
	claims := &JWTClaims{UserID: 7, Email: "user@example.com"}

	t.Run("with claims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user", claims))

		got := bc.CurrentUser(req)
		if got == nil {
			t.Fatal("CurrentUser() = nil, want claims")
		}
		if got.UserID != claims.UserID {
			t.Errorf("UserID = %d, want %d", got.UserID, claims.UserID)
		}
	})

	t.Run("without claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		if got := bc.CurrentUser(req); got != nil {
			t.Errorf("CurrentUser() = %v, want nil", got)
		}
	})

	t.Run("wrong type in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(context.WithValue(req.Context(), "user", "not-claims"))

		if got := bc.CurrentUser(req); got != nil {
			t.Errorf("CurrentUser() = %v, want nil", got)
		}
	})
}
