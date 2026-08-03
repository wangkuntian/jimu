package interfaces

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/modules/user/application"
	"jimu/internal/modules/user/domain"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestUserListRejectsInvalidPaginationContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users", NewUserHandler(nil).List)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users?sort=password", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserListUsesDefaultPaginationBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users", func(c *gin.Context) {
		c.Set("request_id", "rid-list")
		NewUserHandler(nil).List(c)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want nil service to fail after pagination", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["request_id"] != "rid-list" {
		t.Fatalf("body = %#v", body)
	}
}

func TestUserCreateReturnsCreatedDTO(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/users", NewUserHandler(application.NewUserService(&fakeUserRepository{})).Create)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"username":"alice","password":"secret123"}`)))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if strings.Contains(w.Body.String(), "password") {
		t.Fatalf("sensitive body: %s", w.Body.String())
	}
}

func TestUserDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/users/:id", NewUserHandler(application.NewUserService(&fakeUserRepository{})).Delete)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/users/7", nil))

	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}

type fakeUserRepository struct{}

func (r *fakeUserRepository) FindByID(context.Context, uint64) (*domain.User, error) {
	return &domain.User{ID: 7, Username: "alice", Status: 1}, nil
}

func (r *fakeUserRepository) FindByUsername(context.Context, string) (*domain.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepository) List(context.Context, int, int, string, string) ([]domain.User, int64, error) {
	return nil, 0, nil
}

func (r *fakeUserRepository) Create(_ context.Context, user *domain.User) error {
	user.ID = 7
	return nil
}

func (r *fakeUserRepository) Update(context.Context, *domain.User) error { return nil }
func (r *fakeUserRepository) Delete(context.Context, uint64) error       { return nil }
