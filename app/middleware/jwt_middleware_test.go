package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iskandar221201/goigniter/app/middleware"
	"github.com/iskandar221201/goigniter/config"
	"github.com/iskandar221201/goigniter/system"
)

const testSecret = "test-secret"

func newToken(t *testing.T, claims system.JWTClaims) string {
	t.Helper()

	cfg := &config.Config{JWT: config.JWTConfig{Secret: testSecret, ExpireHours: 1}}
	token, err := system.GenerateToken(claims, cfg)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	return token
}

func TestJWTProtected(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value("user").(*system.JWTClaims)
		if !ok {
			t.Error("claims not found in request context")
			return
		}
		w.Header().Set("X-User-ID", "1")
		if claims.UserID != 99 {
			t.Errorf("UserID = %d, want 99", claims.UserID)
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.JWTProtected(testSecret)(next)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{name: "missing header", authHeader: "", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", authHeader: "Token abc", wantStatus: http.StatusUnauthorized},
		{name: "empty bearer", authHeader: "Bearer ", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", authHeader: "Bearer not-a-token", wantStatus: http.StatusUnauthorized},
		{name: "invalid signature", authHeader: "Bearer " + newToken(t, system.JWTClaims{UserID: 1}) + "x", wantStatus: http.StatusUnauthorized},
		{name: "valid token", authHeader: "Bearer " + newToken(t, system.JWTClaims{UserID: 99}), wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRoleGuard(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withClaims := func(role string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ctx := context.WithValue(req.Context(), "user", &system.JWTClaims{Role: role})
		return req.WithContext(ctx)
	}

	tests := []struct {
		name       string
		roles      []string
		req        *http.Request
		wantStatus int
	}{
		{name: "allowed role", roles: []string{"admin"}, req: withClaims("admin"), wantStatus: http.StatusOK},
		{name: "allowed among many", roles: []string{"admin", "superadmin"}, req: withClaims("superadmin"), wantStatus: http.StatusOK},
		{name: "forbidden role", roles: []string{"admin"}, req: withClaims("user"), wantStatus: http.StatusForbidden},
		{name: "no claims", roles: []string{"admin"}, req: httptest.NewRequest(http.MethodGet, "/", nil), wantStatus: http.StatusUnauthorized},
		{name: "wrong claims type", roles: []string{"admin"}, req: func() *http.Request {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			return req.WithContext(context.WithValue(req.Context(), "user", "admin"))
		}(), wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			middleware.RoleGuard(tt.roles...)(next).ServeHTTP(rec, tt.req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
