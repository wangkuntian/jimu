package application

import (
	"context"
	stderrors "errors"
	"io"
	"time"

	"jimu/internal/modules/admin/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/importer"
	apperrors "jimu/internal/shared/errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// userImportFields 用户导入字段校验规则
var userImportFields = []importer.FieldRule{
	{Field: "username", Type: importer.TypeString, Required: true, Unique: true},
	{Field: "password", Type: importer.TypeString, Required: true},
	{Field: "email", Type: importer.TypeEmail},
}

// importRegistry 预注册的导入器 registry
var importRegistry = func() *importer.Registry {
	r := importer.NewRegistry()
	r.Register(importer.FormatCSV, importer.NewCSVImporter())
	r.Register(importer.FormatExcel, importer.NewExcelImporter())
	return r
}()

// ImportService 数据导入服务
type ImportService struct {
	importJobs domain.ImportJobRepository
	userRepo   userdomain.UserRepository
	userDB     *gorm.DB
}

// NewImportService 创建导入服务
func NewImportService(importJobs domain.ImportJobRepository, userRepo userdomain.UserRepository, userDB *gorm.DB) *ImportService {
	return &ImportService{importJobs: importJobs, userRepo: userRepo, userDB: userDB}
}

// Preview 解析并校验文件，不落库
func (s *ImportService) Preview(ctx context.Context, format importer.Format, file io.Reader, importType string) (*importer.ImportResult, error) {
	rules, err := rulesFor(importType)
	if err != nil {
		return nil, err
	}
	imp, err := importRegistry.Get(format)
	if err != nil {
		return nil, err
	}
	rows, err := imp.Parse(ctx, file)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInvalidParam, "parse file failed", err)
	}
	return imp.Validate(ctx, rows, rules)
}

// Import 解析、校验并逐行导入用户，返回结果并记录导入任务
func (s *ImportService) Import(ctx context.Context, format importer.Format, file io.Reader, importType string, createdBy uint64, filename string) (*importer.ImportResult, *domain.ImportJob, error) {
	rules, err := rulesFor(importType)
	if err != nil {
		return nil, nil, err
	}
	imp, err := importRegistry.Get(format)
	if err != nil {
		return nil, nil, err
	}
	rows, err := imp.Parse(ctx, file)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.CodeInvalidParam, "parse file failed", err)
	}
	// 先校验整批，存在校验错误的行不落库
	validation, err := imp.Validate(ctx, rows, rules)
	if err != nil {
		return nil, nil, err
	}
	if validation.ErrorRows > 0 {
		return validation, nil, nil
	}

	job := &domain.ImportJob{
		Type:      importType,
		Filename:  filename,
		Status:    domain.ImportJobProcessing,
		TotalRows: len(rows),
		CreatedBy: createdBy,
	}
	if err := s.importJobs.Create(ctx, job); err != nil {
		return nil, nil, apperrors.Wrap(apperrors.CodeInternalError, "create import job failed", err)
	}

	// 逐行落库，失败计入 error_rows
	result := importer.NewImportResult(len(rows))
	success := 0
	for i, row := range rows {
		if err := s.insertUser(ctx, row); err != nil {
			result.AddError(i+1, "row", err.Error(), row["username"])
			continue
		}
		success++
	}
	result.SuccessRows = success
	result.ErrorRows = len(rows) - success
	result.Finalize(time.Now())

	job.Status = domain.ImportJobCompleted
	job.SuccessRows = success
	job.ErrorRows = result.ErrorRows
	job.CompletedAt = time.Now()
	if result.ErrorRows > 0 {
		job.Status = domain.ImportJobFailed
	}
	if err := s.importJobs.Update(ctx, job); err != nil {
		return nil, job, apperrors.Wrap(apperrors.CodeInternalError, "update import job failed", err)
	}
	return result, job, nil
}

// GetImportJob 查询导入任务状态
func (s *ImportService) GetImportJob(ctx context.Context, id uint64) (*domain.ImportJob, error) {
	job, err := s.importJobs.FindByID(ctx, id)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.Wrap(apperrors.CodeNotFound, "import job not found", err)
		}
		return nil, apperrors.Wrap(apperrors.CodeInternalError, "get import job failed", err)
	}
	return job, nil
}

func (s *ImportService) insertUser(ctx context.Context, row map[string]string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(row["password"]), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := &userdomain.User{
		Username: row["username"],
		Password: string(hash),
		Status:   1,
	}
	return s.userDB.WithContext(ctx).Create(user).Error
}

func rulesFor(importType string) (importer.ValidationRules, error) {
	switch importType {
	case "users":
		return importer.ValidationRules{Fields: userImportFields}, nil
	default:
		return importer.ValidationRules{}, apperrors.New(apperrors.CodeInvalidParam, "unsupported import type: "+importType)
	}
}
