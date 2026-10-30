package routes_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/iskandar221201/goigniter/app/models"
	"github.com/iskandar221201/goigniter/config"
	"github.com/iskandar221201/goigniter/routes"
	"github.com/iskandar221201/goigniter/system"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupRouter(t *testing.T) (http.Handler, *config.Config) {
	t.Helper()

	dsn := fmt.Sprintf("file:routes_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	previousDB := system.DB
	system.DB = db
	t.Cleanup(func() {
		system.DB = previousDB
		_ = sqlDB.Close()
	})

	cfg := &config.Config{
		App: config.AppConfig{Name: "Test", Env: "test", Port: 0},
		JWT: config.JWTConfig{Secret: "test-secret", ExpireHours: 1},
	}

	router := chi.NewRouter()
	routes.Register(router, cfg)

	return router, cfg
}

func request(t *testing.T, router http.Handler, method, path, body, token string) (*httptest.ResponseRecorder, system.APIResponse) {
	t.Helper()

	var reader = strings.NewReader(body)

	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var parsed system.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&parsed); err != nil {
		t.Fatalf("%s %s: decode response body: %v (raw: %s)", method, path, err, rec.Body.String())
	}

	return rec, parsed
}

func login(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()

	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	rec, parsed := request(t, router, http.MethodPost, "/api/v1/auth/login", body, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (message: %s)", rec.Code, parsed.Message)
	}

	data, ok := parsed.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("login data = %T, want object", parsed.Data)
	}

	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Fatal("login response does not contain a token")
	}

	return token
}

func TestAuthEndpoints(t *testing.T) {
	router, _ := setupRouter(t)

	t.Run("register validation failure", func(t *testing.T) {
		rec, parsed := request(t, router, http.MethodPost, "/api/v1/auth/register",
			`{"name":"B","email":"not-an-email","password":"short"}`, "")

		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
		if parsed.Status {
			t.Error("Status = true, want false")
		}

		errs, ok := parsed.Errors.(map[string]interface{})
		if !ok {
			t.Fatalf("Errors = %T, want object", parsed.Errors)
		}
		for _, field := range []string{"name", "email", "password"} {
			if _, exists := errs[field]; !exists {
				t.Errorf("expected validation error for %q, got %v", field, errs)
			}
		}
	})

	t.Run("register invalid JSON", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodPost, "/api/v1/auth/register", `{oops`, "")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("register success", func(t *testing.T) {
		rec, parsed := request(t, router, http.MethodPost, "/api/v1/auth/register",
			`{"name":"Budi","email":"budi@example.com","password":"rahasia123"}`, "")

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (message: %s)", rec.Code, parsed.Message)
		}
		if !parsed.Status {
			t.Error("Status = false, want true")
		}
		if strings.Contains(rec.Body.String(), "password") {
			t.Errorf("response must not leak password: %s", rec.Body.String())
		}
	})

	t.Run("register duplicate email", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodPost, "/api/v1/auth/register",
			`{"name":"Budi","email":"budi@example.com","password":"rahasia123"}`, "")

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want 422", rec.Code)
		}
	})

	t.Run("login wrong password", func(t *testing.T) {
		rec, parsed := request(t, router, http.MethodPost, "/api/v1/auth/login",
			`{"email":"budi@example.com","password":"salah-sekali"}`, "")

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		if parsed.Message != "invalid credentials" {
			t.Errorf("Message = %q, want %q", parsed.Message, "invalid credentials")
		}
	})

	t.Run("login success", func(t *testing.T) {
		token := login(t, router, "budi@example.com", "rahasia123")
		if token == "" {
			t.Fatal("empty token")
		}
	})
}

func TestUserEndpoints(t *testing.T) {
	router, _ := setupRouter(t)

	if rec, _ := request(t, router, http.MethodPost, "/api/v1/auth/register",
		`{"name":"Admin","email":"admin@example.com","password":"rahasia123"}`, ""); rec.Code != http.StatusCreated {
		t.Fatalf("seed register status = %d, want 201", rec.Code)
	}

	token := login(t, router, "admin@example.com", "rahasia123")

	t.Run("unauthenticated request", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodGet, "/api/v1/users", "", "")

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("list users", func(t *testing.T) {
		rec, parsed := request(t, router, http.MethodGet, "/api/v1/users", "", token)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		users, ok := parsed.Data.([]interface{})
		if !ok {
			t.Fatalf("Data = %T, want array", parsed.Data)
		}
		if len(users) != 1 {
			t.Errorf("len(users) = %d, want 1", len(users))
		}
	})

	t.Run("create user with invalid role", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodPost, "/api/v1/users",
			`{"name":"Siti","email":"siti@example.com","password":"rahasia123","role":"owner"}`, token)

		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("status = %d, want 422", rec.Code)
		}
	})

	t.Run("create user", func(t *testing.T) {
		rec, parsed := request(t, router, http.MethodPost, "/api/v1/users",
			`{"name":"Siti","email":"siti@example.com","password":"rahasia123","role":"user"}`, token)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (message: %s)", rec.Code, parsed.Message)
		}
	})

	t.Run("show user with invalid id", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodGet, "/api/v1/users/abc", "", token)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("show missing user", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodGet, "/api/v1/users/9999", "", token)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("show user", func(t *testing.T) {
		rec, parsed := request(t, router, http.MethodGet, "/api/v1/users/1", "", token)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		user, ok := parsed.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Data = %T, want object", parsed.Data)
		}
		if user["email"] != "admin@example.com" {
			t.Errorf("email = %v, want admin@example.com", user["email"])
		}
	})

	t.Run("update user", func(t *testing.T) {
		rec, parsed := request(t, router, http.MethodPut, "/api/v1/users/2",
			`{"name":"Siti Aminah","role":"admin"}`, token)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (message: %s)", rec.Code, parsed.Message)
		}
		user, ok := parsed.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Data = %T, want object", parsed.Data)
		}
		if user["name"] != "Siti Aminah" || user["role"] != "admin" {
			t.Errorf("updated user = %v, want name Siti Aminah and role admin", user)
		}
	})

	t.Run("update missing user", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodPut, "/api/v1/users/9999", `{"name":"Ghost"}`, token)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("delete user then show", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodDelete, "/api/v1/users/2", "", token)
		if rec.Code != http.StatusOK {
			t.Fatalf("delete status = %d, want 200", rec.Code)
		}

		rec, _ = request(t, router, http.MethodGet, "/api/v1/users/2", "", token)
		if rec.Code != http.StatusNotFound {
			t.Errorf("show after delete status = %d, want 404", rec.Code)
		}
	})

	t.Run("delete missing user", func(t *testing.T) {
		rec, _ := request(t, router, http.MethodDelete, "/api/v1/users/9999", "", token)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestRouteRegistration(t *testing.T) {
	router, _ := setupRouter(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown route status = %d, want 404", rec.Code)
	}
}
