package grpc

import (
	"context"

	"jimu/internal/contract"
)

type testUserinfoSource struct {
	users []contract.Userinfo
}

func (s testUserinfoSource) GetByID(_ context.Context, id uint64) (*contract.Userinfo, error) {
	for i := range s.users {
		if s.users[i].ID == id {
			return &s.users[i], nil
		}
	}
	return nil, contract.ErrNotFound
}

func (s testUserinfoSource) FindByUsername(_ context.Context, username string) (*contract.Userinfo, error) {
	for i := range s.users {
		if s.users[i].Username == username {
			return &s.users[i], nil
		}
	}
	return nil, contract.ErrNotFound
}

func (s testUserinfoSource) List(_ context.Context, page, pageSize int) ([]contract.Userinfo, int64, error) {
	total := int64(len(s.users))
	start := (page - 1) * pageSize
	if start >= len(s.users) {
		return nil, total, nil
	}
	end := start + pageSize
	if end > len(s.users) {
		end = len(s.users)
	}
	return s.users[start:end], total, nil
}

var _ contract.UserinfoSource = testUserinfoSource{}
