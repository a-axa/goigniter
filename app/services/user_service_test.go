package services

import (
	"testing"

	"github.com/iskandar221201/goigniter/app/helpers"
	"github.com/iskandar221201/goigniter/app/models"
	"github.com/iskandar221201/goigniter/app/validations"
)

func TestUserServiceCreate(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	t.Run("defaults role to user", func(t *testing.T) {
		user, err := svc.Create(validations.CreateUserInput{
			Name:     "Budi",
			Email:    "budi@example.com",
			Password: "rahasia123",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		stored, err := svc.GetByID(user.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if stored.Role != "user" {
			t.Errorf("Role = %q, want %q", stored.Role, "user")
		}
		if !helpers.CheckPassword("rahasia123", stored.Password) {
			t.Error("password must be stored hashed")
		}
	})

	t.Run("honours explicit role", func(t *testing.T) {
		user, err := svc.Create(validations.CreateUserInput{
			Name:     "Admin",
			Email:    "admin@example.com",
			Password: "rahasia123",
			Role:     "admin",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if user.Role != "admin" {
			t.Errorf("Role = %q, want %q", user.Role, "admin")
		}
	})
}

func TestUserServiceGetAll(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	users, err := svc.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(users) != 0 {
		t.Errorf("len(users) = %d, want 0", len(users))
	}

	for _, input := range []validations.CreateUserInput{
		{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"},
		{Name: "Siti", Email: "siti@example.com", Password: "rahasia123"},
	} {
		if _, err := svc.Create(input); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	users, err = svc.GetAll()
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("len(users) = %d, want 2", len(users))
	}
}

func TestUserServiceGetByID(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	created, err := svc.Create(validations.CreateUserInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("found", func(t *testing.T) {
		got, err := svc.GetByID(created.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if got.Email != created.Email {
			t.Errorf("Email = %q, want %q", got.Email, created.Email)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := svc.GetByID(9999); err == nil {
			t.Fatal("expected an error for a missing user")
		}
	})
}

func TestUserServiceUpdate(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	created, err := svc.Create(validations.CreateUserInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	t.Run("updates name and role", func(t *testing.T) {
		updated, err := svc.Update(created.ID, validations.UpdateUserInput{Name: "Budi Santoso", Role: "admin"})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.Name != "Budi Santoso" || updated.Role != "admin" {
			t.Errorf("updated = %+v, want name %q and role %q", updated, "Budi Santoso", "admin")
		}

		stored, err := svc.GetByID(created.ID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if stored.Name != "Budi Santoso" || stored.Role != "admin" {
			t.Errorf("stored = %+v, want persisted changes", stored)
		}
	})

	t.Run("empty input keeps existing values", func(t *testing.T) {
		updated, err := svc.Update(created.ID, validations.UpdateUserInput{})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.Name != "Budi Santoso" || updated.Role != "admin" {
			t.Errorf("updated = %+v, want values unchanged", updated)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := svc.Update(9999, validations.UpdateUserInput{Name: "Ghost"}); err == nil {
			t.Fatal("expected an error for a missing user")
		}
	})
}

func TestUserServiceDelete(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	created, err := svc.Create(validations.CreateUserInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := svc.GetByID(created.ID); err == nil {
		t.Fatal("expected deleted user to be unavailable")
	}

	var count int64
	if err := db.Unscoped().Model(&models.User{}).Where("id = ?", created.ID).Count(&count).Error; err != nil {
		t.Fatalf("count unscoped: %v", err)
	}
	if count != 1 {
		t.Errorf("row count = %d, want 1 (soft delete)", count)
	}

	if err := svc.Delete(9999); err == nil {
		t.Fatal("expected an error when deleting a missing user")
	}
}

func TestUserServiceDeleteTwice(t *testing.T) {
	db := newTestDB(t)
	svc := NewUserService(db)

	created, err := svc.Create(validations.CreateUserInput{Name: "Budi", Email: "budi@example.com", Password: "rahasia123"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("first Delete() error = %v", err)
	}
	if err := svc.Delete(created.ID); err == nil {
		t.Fatal("expected an error when deleting an already deleted user")
	}
}
