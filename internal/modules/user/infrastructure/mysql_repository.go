package infrastructure

import (
	"context"

	"jimu/internal/modules/user/domain"

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
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) FindByEmailHash(ctx context.Context, hash string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("email_hash = ?", hash).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) FindByPhoneHash(ctx context.Context, hash string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("phone_hash = ?", hash).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *mysqlRepository) List(ctx context.Context, offset, limit int, sort, order string) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.User{})
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
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *mysqlRepository) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *mysqlRepository) UpdatePassword(ctx context.Context, id uint64, hashedPassword string) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("password", hashedPassword).Error
}

func (r *mysqlRepository) UpdateTOTP(ctx context.Context, id uint64, secret string, enabled bool) error {
	// encryption hook 通过对 User 实体的 Create/Update 回调处理 encryption:"true" 字段，
	// 因此这里加载完整实体再 Save，确保 totp_secret 走加密/盲索引通路。
	user := domain.User{ID: id}
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return err
	}
	user.TOTPSecret = secret
	user.TOTPEnabled = enabled
	return r.db.WithContext(ctx).Save(&user).Error
}

func (r *mysqlRepository) Delete(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, id).Error
}
