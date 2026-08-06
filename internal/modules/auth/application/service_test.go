package application

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	authdomain "jimu/internal/modules/auth/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/auth"
	apperrors "jimu/internal/shared/errors"

	"golang.org/x/crypto/bcrypt"
)

func TestLoginHidesCredentialFailures(t *testing.T) {
	ctx := context.Background()
	jwtUtil := auth.New("01234567890123456789012345678901", "jimu", 30, 7)
	store := newFakeSessionStore()
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	service := NewAuthService(repo, jwtUtil, store, nil, 30)

	_, missingErr := service.Login(ctx, "missing", "correct")
	_, wrongPasswordErr := service.Login(ctx, "alice", "wrong")
	if appCode(missingErr) != apperrors.CodeInvalidCredentials {
		t.Fatalf("missing user code = %d", appCode(missingErr))
	}
	if appCode(wrongPasswordErr) != apperrors.CodeInvalidCredentials {
		t.Fatalf("wrong password code = %d", appCode(wrongPasswordErr))
	}
	if missingErr.Error() != wrongPasswordErr.Error() {
		t.Fatalf("credential errors differ: %q vs %q", missingErr.Error(), wrongPasswordErr.Error())
	}
}

func TestLoginRejectsDisabledUser(t *testing.T) {
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 0),
	}, newFakeSessionStore())
	_, err := service.Login(context.Background(), "alice", "correct")
	if appCode(err) != apperrors.CodeInvalidCredentials {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeInvalidCredentials)
	}
}

func TestLoginCreatesRefreshSession(t *testing.T) {
	store := newFakeSessionStore()
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}, store)
	pair, err := service.Login(context.Background(), " Alice ", "correct")
	if err != nil {
		t.Fatal(err)
	}
	if len(store.created) != 1 {
		t.Fatalf("created sessions = %d, want 1", len(store.created))
	}
	accessClaims, err := service.jwtUtil.Parse(pair.AccessToken, auth.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	refreshClaims, err := service.jwtUtil.Parse(pair.RefreshToken, auth.TokenTypeRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if accessClaims.SessionID != refreshClaims.SessionID || accessClaims.SessionID == "" {
		t.Fatalf("session mismatch: access=%q refresh=%q", accessClaims.SessionID, refreshClaims.SessionID)
	}
	if store.created[0].userID != 42 || store.created[0].sessionID != refreshClaims.SessionID || store.created[0].tokenID != refreshClaims.ID {
		t.Fatalf("created session = %#v refresh = %#v", store.created[0], refreshClaims)
	}
	if got := service.userRepo.(*fakeUserRepo).lookups[0]; got != "alice" {
		t.Fatalf("lookup username = %q, want alice", got)
	}
}

func TestRegisterRejectsDuplicateUsername(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}}
	service := NewAuthService(repo, auth.New("01234567890123456789012345678901", "jimu", 30, 7), newFakeSessionStore(), nil, 30)

	_, err := service.Register(context.Background(), " Alice ", "secret123")
	if appCode(err) != apperrors.CodeUserExists {
		t.Fatalf("code = %d, want %d", appCode(err), apperrors.CodeUserExists)
	}
}

func TestRefreshRotatesOnlyRefreshTokens(t *testing.T) {
	store := newFakeSessionStore()
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}, store)
	pair, err := service.Login(context.Background(), "alice", "correct")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Refresh(context.Background(), pair.AccessToken); appCode(err) != apperrors.CodeUnauthorized {
		t.Fatalf("access token refresh code = %d", appCode(err))
	}
	next, err := service.Refresh(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if next.RefreshToken == pair.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if len(store.rotated) != 1 {
		t.Fatalf("rotations = %d, want 1", len(store.rotated))
	}
	if _, err := service.Refresh(context.Background(), pair.RefreshToken); appCode(err) != apperrors.CodeUnauthorized {
		t.Fatalf("reuse code = %d, want %d", appCode(err), apperrors.CodeUnauthorized)
	}
}

func TestLogoutRevokesSessions(t *testing.T) {
	store := newFakeSessionStore()
	service := newTestService(t, map[string]*userdomain.User{
		"alice": userWithPassword(t, 42, "alice", "correct", 1),
	}, store)
	pair, err := service.Login(context.Background(), "alice", "correct")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.jwtUtil.Parse(pair.AccessToken, auth.TokenTypeAccess)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Logout(context.Background(), claims.UserID, claims.SessionID); err != nil {
		t.Fatal(err)
	}
	if len(store.revoked) != 1 || store.revoked[0] != claims.SessionID {
		t.Fatalf("revoked = %#v", store.revoked)
	}
	if err := service.LogoutAll(context.Background(), claims.UserID); err != nil {
		t.Fatal(err)
	}
	if len(store.revokedAll) != 1 || store.revokedAll[0] != claims.UserID {
		t.Fatalf("revokedAll = %#v", store.revokedAll)
	}
}

type fakeUserRepo struct {
	users   map[string]*userdomain.User
	lookups []string
	created []string
}

func (r *fakeUserRepo) FindByID(context.Context, uint64) (*userdomain.User, error) {
	return nil, stderrors.New("not implemented")
}

func (r *fakeUserRepo) FindByUsername(_ context.Context, username string) (*userdomain.User, error) {
	r.lookups = append(r.lookups, username)
	user, ok := r.users[username]
	if !ok {
		return nil, stderrors.New("not found")
	}
	return user, nil
}

func (r *fakeUserRepo) List(context.Context, int, int, string, string) ([]userdomain.User, int64, error) {
	return nil, 0, stderrors.New("not implemented")
}

func (r *fakeUserRepo) Create(_ context.Context, user *userdomain.User) error {
	username := normalizeUsername(user.Username)
	if _, ok := r.users[username]; ok {
		return stderrors.New("duplicate username")
	}
	user.Username = username
	r.users[username] = user
	r.created = append(r.created, username)
	return nil
}

func (r *fakeUserRepo) Update(context.Context, *userdomain.User) error {
	return nil
}

func (r *fakeUserRepo) Delete(context.Context, uint64) error {
	return nil
}

type sessionRecord struct {
	userID    uint64
	sessionID string
	tokenID   string
}

type fakeSessionStore struct {
	sessions   map[string]sessionRecord
	created    []sessionRecord
	rotated    []sessionRecord
	revoked    []string
	revokedAll []uint64
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: make(map[string]sessionRecord)}
}

func (s *fakeSessionStore) Create(_ context.Context, userID uint64, sessionID, tokenID string, _ time.Duration) error {
	record := sessionRecord{userID: userID, sessionID: sessionID, tokenID: tokenID}
	s.sessions[sessionID] = record
	s.created = append(s.created, record)
	return nil
}

func (s *fakeSessionStore) Rotate(_ context.Context, userID uint64, sessionID, oldTokenID, newTokenID string, _ time.Duration) error {
	record, ok := s.sessions[sessionID]
	if !ok || record.userID != userID {
		return auth.ErrSessionNotFound
	}
	if record.tokenID != oldTokenID {
		return auth.ErrTokenReuse
	}
	record.tokenID = newTokenID
	s.sessions[sessionID] = record
	s.rotated = append(s.rotated, record)
	return nil
}

func (s *fakeSessionStore) Revoke(_ context.Context, _ uint64, sessionID string) error {
	delete(s.sessions, sessionID)
	s.revoked = append(s.revoked, sessionID)
	return nil
}

func (s *fakeSessionStore) RevokeAll(_ context.Context, userID uint64) error {
	for sessionID, record := range s.sessions {
		if record.userID == userID {
			delete(s.sessions, sessionID)
		}
	}
	s.revokedAll = append(s.revokedAll, userID)
	return nil
}

func newTestService(t *testing.T, users map[string]*userdomain.User, store auth.SessionStore) *AuthService {
	t.Helper()
	return NewAuthService(&fakeUserRepo{users: users}, auth.New("01234567890123456789012345678901", "jimu", 30, 7), store, nil, 30)
}

func userWithPassword(t *testing.T, id uint64, username, password string, status int8) *userdomain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	return &userdomain.User{ID: id, Username: username, Password: string(hash), Status: status}
}

func appCode(err error) int {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code
	}
	return 0
}

var _ authdomain.AuthServiceInterface = (*AuthService)(nil)
