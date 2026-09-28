package interfaces

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"jimu/internal/capabilities/product/application"
	"jimu/internal/capabilities/product/domain"
	"jimu/internal/kernel/http/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestProductCreateReturnsCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewProductHandler(application.NewProductService(&fakeProductRepository{}))
	r.POST("/products", middleware.ValidateJSON(&application.CreateProductRequest{}), handler.Create)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(`{"name":"one","description":"desc"}`)))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestProductDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewProductHandler(application.NewProductService(&fakeProductRepository{}))
	r.DELETE("/products/:id", handler.Delete)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/products/7", nil))

	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("status = %d body = %q", w.Code, w.Body.String())
	}
}

type fakeProductRepository struct{}

func (r *fakeProductRepository) FindByID(context.Context, uint64) (*domain.Product, error) {
	return &domain.Product{ID: 7, Name: "one"}, nil
}
func (r *fakeProductRepository) List(context.Context, int, int, string, string) ([]domain.Product, int64, error) {
	return nil, 0, nil
}
func (r *fakeProductRepository) Create(_ context.Context, entity *domain.Product) error {
	entity.ID = 7
	return nil
}
func (r *fakeProductRepository) Update(context.Context, *domain.Product) error {
	return nil
}
func (r *fakeProductRepository) Delete(context.Context, uint64) error {
	return nil
}

var _ = gorm.ErrRecordNotFound
