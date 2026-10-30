package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/iskandar221201/goigniter/app/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:model_%d?mode=memory&cache=shared", time.Now().UnixNano())
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

	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func seed(t *testing.T, db *gorm.DB, users ...models.User) {
	t.Helper()

	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
}

func TestUserFindByEmail(t *testing.T) {
	db := newTestDB(t)
	seed(t, db, models.User{Name: "Budi", Email: "budi@example.com", Password: "hash"})

	t.Run("found", func(t *testing.T) {
		user, err := new(models.User).FindByEmail(db, "budi@example.com")
		if err != nil {
			t.Fatalf("FindByEmail() error = %v", err)
		}
		if user.Name != "Budi" {
			t.Errorf("Name = %q, want %q", user.Name, "Budi")
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := new(models.User).FindByEmail(db, "ghost@example.com"); err == nil {
			t.Fatal("expected an error for a missing email")
		}
	})
}

func TestUserFindByID(t *testing.T) {
	db := newTestDB(t)
	user := models.User{Name: "Budi", Email: "budi@example.com", Password: "hash"}
	seed(t, db, user)

	t.Run("found", func(t *testing.T) {
		got, err := new(models.User).FindByID(db, 1)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if got.Email != user.Email {
			t.Errorf("Email = %q, want %q", got.Email, user.Email)
		}
	})

	t.Run("not found", func(t *testing.T) {
		if _, err := new(models.User).FindByID(db, 9999); err == nil {
			t.Fatal("expected an error for a missing ID")
		}
	})
}

func TestUserFindAll(t *testing.T) {
	db := newTestDB(t)

	users, err := new(models.User).FindAll(db)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(users) != 0 {
		t.Errorf("len(users) = %d, want 0", len(users))
	}

	seed(t, db,
		models.User{Name: "Budi", Email: "budi@example.com", Password: "hash"},
		models.User{Name: "Siti", Email: "siti@example.com", Password: "hash"},
	)

	users, err = new(models.User).FindAll(db)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("len(users) = %d, want 2", len(users))
	}
}

func TestUserFindAllExcludesSoftDeleted(t *testing.T) {
	db := newTestDB(t)
	seed(t, db,
		models.User{Name: "Budi", Email: "budi@example.com", Password: "hash"},
		models.User{Name: "Siti", Email: "siti@example.com", Password: "hash"},
	)

	if err := db.Delete(&models.User{}, 1).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	users, err := new(models.User).FindAll(db)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(users) != 1 {
		t.Errorf("len(users) = %d, want 1", len(users))
	}
	if users[0].Email == "budi@example.com" {
		t.Error("soft-deleted user must not be returned")
	}
}

func TestUserEmailIsUnique(t *testing.T) {
	db := newTestDB(t)
	seed(t, db, models.User{Name: "Budi", Email: "budi@example.com", Password: "hash"})

	err := db.Create(&models.User{Name: "Other", Email: "budi@example.com", Password: "hash"}).Error
	if err == nil {
		t.Fatal("expected unique constraint violation")
	}
}
