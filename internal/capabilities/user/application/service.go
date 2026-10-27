package application

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"jimu/internal/capabilities/user/domain"
	"jimu/internal/contract"
	dbutil "jimu/internal/kernel/db"
	"jimu/internal/kernel/tenant"
	"jimu/internal/shared/errors"
	"jimu/internal/shared/pagination"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repo     domain.UserRepository
	outbox   contract.EventWriter
	cipher   contract.BlindIndexer
	db       *gorm.DB
	roles    contract.UserRoleAssigner
	tenants  contract.TenantInfoSource
	quota    TenantQuota
	sessions SessionRevoker
}

// TenantQuota 租户配额校验（由 tenant 能力实现；nil = 未启用配额）。
type TenantQuota interface {
	CheckUserQuota(ctx context.Context, tenantID uint64) error
}

// SessionRevoker 吊销用户全部登录会话（由 kernel/auth 的 SessionStore 实现）。
type SessionRevoker interface {
	RevokeAll(ctx context.Context, userID uint64) error
}

func NewUserService(repo domain.UserRepository, deps ...interface{}) *UserService {
	s := &UserService{repo: repo}
	for _, dep := range deps {
		switch d := dep.(type) {
		case contract.EventWriter:
			s.outbox = d
		case contract.BlindIndexer:
			s.cipher = d
		case contract.UserRoleAssigner:
			s.roles = d
		case contract.TenantInfoSource:
			s.tenants = d
		case TenantQuota:
			s.quota = d
		case SessionRevoker:
			s.sessions = d
		}
	}
	return s
}

func (s *UserService) WithDB(db *gorm.DB) *UserService {
	s.db = db
	return s
}

func (s *UserService) WithRoles(roles contract.UserRoleAssigner) *UserService {
	s.roles = roles
	return s
}

func (s *UserService) WithQuota(quota TenantQuota) *UserService {
	s.quota = quota
	return s
}

func (s *UserService) WithSessionRevoker(revoker SessionRevoker) *UserService {
	s.sessions = revoker
	return s
}

// tenantAllowed 判断目标资源租户是否允许当前上下文访问。
// ctxTenant=0 表示平台级视角（无租户上下文），放行所有资源。
func tenantAllowed(resourceTenant, ctxTenant uint64) bool {
	return ctxTenant == 0 || resourceTenant == 0 || resourceTenant == ctxTenant
}

func (s *UserService) Create(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	targetTenantID, err := s.createTenantID(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	roleIDs, err := parseStringIDs(req.RoleIDs)
	if err != nil {
		return nil, errors.New(errors.CodeInvalidParam, "invalid role id")
	}
	if len(roleIDs) > 0 && s.roles == nil {
		return nil, errors.New(errors.CodeInternalError, "role assignment is not configured")
	}
	if s.quota != nil {
		if err := s.quota.CheckUserQuota(ctx, targetTenantID); err != nil {
			return nil, err
		}
	}

	existing, err := s.repo.FindByUsername(ctx, req.Username)
	if err == nil && existing != nil {
		return nil, errors.New(errors.CodeConflict, "username already exists")
	}
	if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to find user", err)
	}

	if req.Email != "" && s.cipher != nil {
		existing, err := s.repo.FindByEmailHash(ctx, s.cipher.BlindIndex(req.Email))
		if err == nil && existing != nil {
			return nil, errors.New(errors.CodeConflict, "email already exists")
		}
		if err != nil && !stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.Wrap(errors.CodeInternalError, "failed to find user by email", err)
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInternalError, "failed to hash password", err)
	}

	user := &domain.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   1,
		TenantID: targetTenantID,
	}
	create := func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, user); err != nil {
			return err
		}
		if len(roleIDs) > 0 {
			return s.roles.AssignRoleIDs(txCtx, user.ID, roleIDs)
		}
		return nil
	}
	if s.db != nil {
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return create(dbutil.WithTransaction(ctx, tx))
		})
	} else {
		err = create(ctx)
	}
	if err != nil {
		var appErr *errors.AppError
		if stderrors.As(err, &appErr) {
			return nil, err
		}
		if isDuplicateUserError(err) {
			return nil, errors.Wrap(errors.CodeConflict, "username or contact already exists", err)
		}
		return nil, errors.Wrap(errors.CodeInternalError, "failed to create user", err)
	}

	s.publishUserCreated(ctx, user)
	return s.hydrateOne(ctx, *user)
}

func (s *UserService) createTenantID(ctx context.Context, rawTenantID string) (uint64, error) {
	ctxTenantID := tenant.FromContext(ctx)
	if tenant.IsPlatformScope(ctx) {
		ctxTenantID = 0
	}
	requestedTenantID := uint64(0)
	if rawTenantID != "" {
		id, err := strconv.ParseUint(rawTenantID, 10, 64)
		if err != nil {
			return 0, errors.New(errors.CodeInvalidParam, "invalid tenant_id")
		}
		requestedTenantID = id
	}
	if ctxTenantID != 0 {
		if ctxTenantID == tenant.DefaultTenantID {
			return 0, errors.New(errors.CodeForbidden, "default tenant is platform-managed")
		}
		if requestedTenantID != 0 && requestedTenantID != ctxTenantID {
			return 0, errors.New(errors.CodeForbidden, "tenant scope mismatch")
		}
		return ctxTenantID, nil
	}
	if requestedTenantID == 0 {
		return 0, errors.New(errors.CodeInvalidParam, "tenant_id is required for platform users")
	}
	if s.tenants == nil {
		return 0, errors.New(errors.CodeInternalError, "tenant lookup is not configured")
	}
	tenants, err := s.tenants.GetByIDs(ctx, []uint64{requestedTenantID})
	if err != nil {
		return 0, errors.Wrap(errors.CodeInternalError, "failed to load tenant", err)
	}
	if _, ok := tenants[requestedTenantID]; !ok {
		return 0, errors.New(errors.CodeNotFound, "tenant not found")
	}
	return requestedTenantID, nil
}

func (s *UserService) Get(ctx context.Context, id uint64) (*UserResponse, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		code := errors.CodeInternalError
		message := "failed to get user"
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			code = errors.CodeNotFound
			message = "user not found"
		}
		return nil, errors.Wrap(code, message, err)
	}
	if !canAccessTenant(ctx, user.TenantID) {
		return nil, errors.New(errors.CodeNotFound, "user not found")
	}
	return s.hydrateOne(ctx, *user)
}

func (s *UserService) List(ctx context.Context, p pagination.Pagination, filter domain.UserListFilter) ([]UserResponse, int64, error) {
	ctxTenantID := tenant.FromContext(ctx)
	if tenant.IsPlatformScope(ctx) {
		ctxTenantID = 0
	}
	if ctxTenantID != 0 {
		if ctxTenantID == tenant.DefaultTenantID {
			return nil, 0, errors.New(errors.CodeForbidden, "default tenant is platform-managed")
		}
		if filter.TenantID != 0 && filter.TenantID != ctxTenantID {
			return nil, 0, errors.New(errors.CodeForbidden, "tenant scope mismatch")
		}
		filter.TenantID = ctxTenantID
	}
	users, total, err := s.repo.List(ctx, filter, p.GetOffset(), p.GetLimit(), p.Sort, p.Order)
	if err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternalError, "failed to list users", err)
	}
	out, err := s.hydrateMany(ctx, users)
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *UserService) Update(ctx context.Context, id uint64, req UpdateUserRequest) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "user not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get user", err)
	}
	if !canAccessTenant(ctx, user.TenantID) {
		return errors.New(errors.CodeNotFound, "user not found")
	}
	wasActive := user.Status == 1
	changes := make([]string, 0, 3)
	if req.Status != nil && user.Status != *req.Status {
		user.Status = *req.Status
		changes = append(changes, "status")
	}
	if req.Email != nil {
		user.Email = *req.Email
		changes = append(changes, "email")
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
		changes = append(changes, "phone")
	}
	persist := func(txCtx context.Context) error {
		if wasActive && req.Status != nil && *req.Status == 0 && s.roles != nil {
			if err := s.roles.EnsureCanDeactivateUser(txCtx, user.ID); err != nil {
				return err
			}
		}
		return s.repo.Update(txCtx, user)
	}
	if s.db != nil && wasActive && req.Status != nil && *req.Status == 0 {
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return persist(dbutil.WithTransaction(ctx, tx))
		})
	} else {
		err = persist(ctx)
	}
	if err != nil {
		var appErr *errors.AppError
		if stderrors.As(err, &appErr) {
			return err
		}
		if isDuplicateUserError(err) {
			return errors.Wrap(errors.CodeConflict, "contact already exists", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to update user", err)
	}
	if req.Status != nil && *req.Status == 0 {
		if err := s.revokeSessions(ctx, id); err != nil {
			return err
		}
	}
	s.publishUserUpdated(ctx, id, changes)
	return nil
}

func (s *UserService) Delete(ctx context.Context, id uint64) error {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Wrap(errors.CodeNotFound, "user not found", err)
		}
		return errors.Wrap(errors.CodeInternalError, "failed to get user", err)
	}
	if !canAccessTenant(ctx, user.TenantID) {
		return errors.New(errors.CodeNotFound, "user not found")
	}
	remove := func(txCtx context.Context) error {
		if s.roles != nil {
			if err := s.roles.EnsureCanDeactivateUser(txCtx, user.ID); err != nil {
				return err
			}
		}
		return s.repo.Delete(txCtx, user.ID)
	}
	if s.db != nil {
		err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return remove(dbutil.WithTransaction(ctx, tx))
		})
	} else {
		err = remove(ctx)
	}
	if err != nil {
		var appErr *errors.AppError
		if stderrors.As(err, &appErr) {
			return err
		}
		return errors.Wrap(errors.CodeInternalError, "failed to delete user", err)
	}
	if err := s.revokeSessions(ctx, id); err != nil {
		return err
	}
	s.publishUserDeleted(ctx, id)
	return nil
}

func (s *UserService) revokeSessions(ctx context.Context, userID uint64) error {
	if s.sessions == nil {
		return nil
	}
	if err := s.sessions.RevokeAll(ctx, userID); err != nil {
		return errors.Wrap(errors.CodeInternalError, "failed to revoke user sessions", err)
	}
	return nil
}

func (s *UserService) hydrateOne(ctx context.Context, user domain.User) (*UserResponse, error) {
	rows, err := s.hydrateMany(ctx, []domain.User{user})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func canAccessTenant(ctx context.Context, resourceTenant uint64) bool {
	if tenant.IsPlatformScope(ctx) {
		return true
	}
	if resourceTenant == tenant.DefaultTenantID {
		return false
	}
	return tenantAllowed(resourceTenant, tenant.FromContext(ctx))
}

func (s *UserService) hydrateMany(ctx context.Context, users []domain.User) ([]UserResponse, error) {
	out := ToUserResponses(users)
	if len(users) == 0 {
		return out, nil
	}

	userIDs := make([]uint64, 0, len(users))
	tenantIDs := make([]uint64, 0, len(users))
	seenTenants := make(map[uint64]bool, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
		if user.TenantID != 0 && !seenTenants[user.TenantID] {
			seenTenants[user.TenantID] = true
			tenantIDs = append(tenantIDs, user.TenantID)
		}
	}

	roleMap := make(map[uint64][]contract.UserRole)
	if s.roles != nil {
		var err error
		roleMap, err = s.roles.RolesForUsers(ctx, userIDs)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "failed to load user roles", err)
		}
	}
	tenantMap := make(map[uint64]contract.TenantInfo)
	if s.tenants != nil && len(tenantIDs) > 0 {
		var err error
		tenantMap, err = s.tenants.GetByIDs(ctx, tenantIDs)
		if err != nil {
			return nil, errors.Wrap(errors.CodeInternalError, "failed to load tenants", err)
		}
	}

	for i, user := range users {
		roles := roleMap[user.ID]
		out[i].Roles = make([]RoleRef, 0, len(roles))
		for _, role := range roles {
			out[i].Roles = append(out[i].Roles, RoleRef{
				ID:   strconv.FormatUint(role.ID, 10),
				Name: role.Name,
			})
		}
		if info, ok := tenantMap[user.TenantID]; ok {
			out[i].Tenant.Name = info.Name
		}
	}
	return out, nil
}

func parseStringIDs(values []string) ([]uint64, error) {
	out := make([]uint64, 0, len(values))
	seen := make(map[uint64]struct{}, len(values))
	for _, value := range values {
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func isDuplicateUserError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate") || strings.Contains(message, "unique constraint")
}

// publishUserCreated 写入用户创建事件（统一可靠投递路径）。
// 业务事务已提交，outbox 写失败不回滚业务，但必须记录，避免静默丢事件。
func (s *UserService) publishUserCreated(ctx context.Context, user *domain.User) {
	if s.outbox == nil {
		return
	}
	payload, err := json.Marshal(contract.UserCreatedEvent{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
	})
	if err != nil {
		log.Printf("user: marshal created event: %v", err)
		return
	}
	if err := s.outbox.WriteEvent(ctx, nil, contract.OutboxEvent{
		AggregateID: fmt.Sprintf("user:%d", user.ID),
		EventType:   contract.EventUserCreated,
		Payload:     payload,
	}); err != nil {
		log.Printf("user: write outbox event %s: %v", contract.EventUserCreated, err)
	}
}

// publishUserUpdated 写入用户更新事件。
func (s *UserService) publishUserUpdated(ctx context.Context, id uint64, changes []string) {
	if s.outbox == nil || len(changes) == 0 {
		return
	}
	payload, err := json.Marshal(contract.UserUpdatedEvent{
		UserID:  id,
		Changes: changes,
	})
	if err != nil {
		log.Printf("user: marshal updated event: %v", err)
		return
	}
	if err := s.outbox.WriteEvent(ctx, nil, contract.OutboxEvent{
		AggregateID: fmt.Sprintf("user:%d", id),
		EventType:   contract.EventUserUpdated,
		Payload:     payload,
	}); err != nil {
		log.Printf("user: write outbox event %s: %v", contract.EventUserUpdated, err)
	}
}

// publishUserDeleted 写入用户删除事件。
func (s *UserService) publishUserDeleted(ctx context.Context, id uint64) {
	if s.outbox == nil {
		return
	}
	payload, err := json.Marshal(contract.UserDeletedEvent{UserID: id})
	if err != nil {
		log.Printf("user: marshal deleted event: %v", err)
		return
	}
	if err := s.outbox.WriteEvent(ctx, nil, contract.OutboxEvent{
		AggregateID: fmt.Sprintf("user:%d", id),
		EventType:   contract.EventUserDeleted,
		Payload:     payload,
	}); err != nil {
		log.Printf("user: write outbox event %s: %v", contract.EventUserDeleted, err)
	}
}

// BatchDelete 批量软删除用户（逐个执行，返回成功/失败计数）。
func (s *UserService) BatchDelete(ctx context.Context, ids []uint64) (BatchResult, error) {
	result := BatchResult{}
	var firstErr error
	for _, id := range ids {
		if err := s.Delete(ctx, id); err != nil {
			result.Failed++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		result.Success++
	}
	if result.Success == 0 && result.Failed > 0 {
		return result, firstErr
	}
	return result, nil
}
