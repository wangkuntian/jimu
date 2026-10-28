package infrastructure

import (
	"context"

	"jimu/internal/capabilities/user/domain"
	dbutil "jimu/internal/kernel/db"
	"jimu/internal/shared/sqlpattern"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type mysqlRepository struct {
	db *gorm.DB
}

func NewMysqlRepository(db *gorm.DB) domain.UserRepository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	var user domain.User
	err := r.dbFor(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := r.dbFor(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) FindByEmailHash(ctx context.Context, hash string) (*domain.User, error) {
	var user domain.User
	err := r.dbFor(ctx).Where("email_hash = ?", hash).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) FindByPhoneHash(ctx context.Context, hash string) (*domain.User, error) {
	var user domain.User
	err := r.dbFor(ctx).Where("phone_hash = ?", hash).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// List 分页查询用户。filter.TenantID 非 0 时仅返回该租户下的用户（0=平台级视角，不过滤）。
func (r *mysqlRepository) List(ctx context.Context, filter domain.UserListFilter, offset, limit int, sort, order string) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64
	db := r.dbFor(ctx).Model(&domain.User{})
	if filter.TenantID != 0 {
		db = db.Where("tenant_id = ?", filter.TenantID)
	}
	if filter.Username != "" {
		db = db.Where("username LIKE ? ESCAPE '!'", sqlpattern.LikeContains(filter.Username))
	}
	if filter.Status != nil {
		db = db.Where("status = ?", *filter.Status)
	}
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: sort},
		Desc:   order == "desc",
	}).Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

func (r *mysqlRepository) Create(ctx context.Context, user *domain.User) error {
	return r.dbFor(ctx).Create(user).Error
}

func (r *mysqlRepository) Update(ctx context.Context, user *domain.User) error {
	return r.dbFor(ctx).Save(user).Error
}

func (r *mysqlRepository) UpdatePassword(ctx context.Context, id uint64, hashedPassword string) error {
	return r.dbFor(ctx).Model(&domain.User{}).Where("id = ?", id).Update("password", hashedPassword).Error
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint64) error {
	return r.dbFor(ctx).Delete(&domain.User{}, id).Error
}

func (r *mysqlRepository) dbFor(ctx context.Context) *gorm.DB {
	if tx, ok := dbutil.TransactionFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}
