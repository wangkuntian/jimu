package queue

import (
	"context"
	"time"

	"jimu/internal/modules/admin/domain"
)

// MySQLStore MySQL 持久化存储
type MySQLStore struct {
	jobRepo     domain.JobRepository
	historyRepo domain.JobHistoryRepository
	deadRepo    domain.DeadLetterRepository
}

// NewMySQLStore 创建 MySQL 存储
func NewMySQLStore(jobRepo domain.JobRepository, historyRepo domain.JobHistoryRepository, deadRepo domain.DeadLetterRepository) *MySQLStore {
	return &MySQLStore{
		jobRepo:     jobRepo,
		historyRepo: historyRepo,
		deadRepo:    deadRepo,
	}
}

// CreateJob 创建任务记录
func (s *MySQLStore) CreateJob(ctx context.Context, jobType, payload string, maxAttempts int) (*domain.Job, error) {
	job := &domain.Job{
		Type:        jobType,
		Payload:     payload,
		Status:      domain.JobStatusPending,
		MaxAttempts: maxAttempts,
		NextRunAt:   time.Now(),
	}
	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// MarkRunning 标记任务执行中
func (s *MySQLStore) MarkRunning(ctx context.Context, jobID uint64) error {
	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}
	job.Status = domain.JobStatusRunning
	job.Attempts++
	return s.jobRepo.Update(ctx, job)
}

// MarkSuccess 标记任务成功
func (s *MySQLStore) MarkSuccess(ctx context.Context, jobID uint64, durationMs int64) error {
	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}
	job.Status = domain.JobStatusSuccess
	job.Error = ""
	if err := s.jobRepo.Update(ctx, job); err != nil {
		return err
	}
	history := &domain.JobHistory{
		JobID:     jobID,
		Status:    "success",
		Duration:  durationMs,
		StartedAt: time.Now().Add(-time.Duration(durationMs) * time.Millisecond),
		EndedAt:   time.Now(),
	}
	return s.historyRepo.Create(ctx, history)
}

// MarkFailed 标记任务失败
func (s *MySQLStore) MarkFailed(ctx context.Context, jobID uint64, jobType, payload string, execErr error, durationMs int64) error {
	job, err := s.jobRepo.FindByID(ctx, jobID)
	if err != nil {
		return err
	}
	job.Error = execErr.Error()
	history := &domain.JobHistory{
		JobID:     jobID,
		Status:    "failed",
		Error:     execErr.Error(),
		Duration:  durationMs,
		StartedAt: time.Now().Add(-time.Duration(durationMs) * time.Millisecond),
		EndedAt:   time.Now(),
	}
	if hErr := s.historyRepo.Create(ctx, history); hErr != nil {
		return hErr
	}
	if job.Attempts >= job.MaxAttempts {
		job.Status = domain.JobStatusDead
		if err := s.jobRepo.Update(ctx, job); err != nil {
			return err
		}
		dead := &domain.DeadLetter{
			JobID:      jobID,
			Type:       jobType,
			Payload:    payload,
			FailReason: execErr.Error(),
		}
		return s.deadRepo.Create(ctx, dead)
	}
	job.Status = domain.JobStatusPending
	return s.jobRepo.Update(ctx, job)
}
