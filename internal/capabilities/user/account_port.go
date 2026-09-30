package user

import (
	"context"
	"errors"

	"jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	dbctx "jimu/internal/kernel/db"

	"gorm.io/gorm"
)

const AccountPortName = "user.account"

type accountAdapter struct {
	repo domain.UserRepository
}

func NewAccountRepository(repo domain.UserRepository) contract.AccountRepository {
	return &accountAdapter{repo: repo}
}

func (a *accountAdapter) FindByID(ctx context.Context, id uint64) (*contract.Account, error) {
	u, err := a.repo.FindByID(ctx, id)
	return accountView(u), mapNotFound(err)
}
func (a *accountAdapter) FindByUsername(ctx context.Context, username string) (*contract.Account, error) {
	u, err := a.repo.FindByUsername(ctx, username)
	return accountView(u), mapNotFound(err)
}
func (a *accountAdapter) FindByEmailHash(ctx context.Context, hash string) (*contract.Account, error) {
	u, err := a.repo.FindByEmailHash(ctx, hash)
	return accountView(u), mapNotFound(err)
}
func (a *accountAdapter) Create(ctx context.Context, account *contract.Account) error {
	u := &domain.User{ID: account.ID, Username: account.Username, Password: account.Password, Email: account.Email, Phone: account.Phone, Status: account.Status, TenantID: account.TenantID}
	var err error
	if tx, ok := dbctx.TransactionFromContext(ctx); ok {
		err = tx.WithContext(ctx).Create(u).Error
	} else {
		err = a.repo.Create(ctx, u)
	}
	if err != nil {
		return err
	}
	account.ID, account.CreatedAt, account.UpdatedAt = u.ID, u.CreatedAt, u.UpdatedAt
	return nil
}
func (a *accountAdapter) UpdatePassword(ctx context.Context, id uint64, hash string) error {
	return a.repo.UpdatePassword(ctx, id, hash)
}
func accountView(u *domain.User) *contract.Account {
	if u == nil {
		return nil
	}
	return &contract.Account{ID: u.ID, Username: u.Username, Password: u.Password, Email: u.Email, Phone: u.Phone, Status: u.Status, TenantID: u.TenantID, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}
}
func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return contract.ErrNotFound
	}
	return err
}

var _ contract.AccountRepository = (*accountAdapter)(nil)
