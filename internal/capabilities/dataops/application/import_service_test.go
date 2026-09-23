package application

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	// 注册 CSV 驱动：应用层经 importer.Get 查进程级注册表，测试二进制需显式编入驱动包。
	_ "jimu/internal/capabilities/dataops/csv"
	importdomain "jimu/internal/capabilities/dataops/domain"
	"jimu/internal/capabilities/dataops/importer"
	"jimu/internal/kernel/tenant"
	apperrors "jimu/internal/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const validCSV = "username,password,email\nalice,secret123,a@b.com\n"
const badCSV = "username,password,email\n,secret123,a@b.com\n"

func newImportService(db *gorm.DB) *ImportService {
	return NewImportService(&fakeImportJobRepo{}, db)
}

func TestImportServicePreview(t *testing.T) {
	ctx := context.Background()
	svc := newImportService(newSqliteDB(t, &importUser{}))

	// 成功
	result, err := svc.Preview(ctx, importer.FormatCSV, strings.NewReader(validCSV), "users")
	assert.NoError(t, err)
	assert.Equal(t, 1, result.TotalRows)
	assert.Equal(t, 1, result.SuccessRows)

	// 校验错误行
	result, err = svc.Preview(ctx, importer.FormatCSV, strings.NewReader(badCSV), "users")
	assert.NoError(t, err)
	assert.Equal(t, 1, result.ErrorRows)

	// 不支持的导入类型
	_, err = svc.Preview(ctx, importer.FormatCSV, strings.NewReader(validCSV), "bogus")
	assert.Error(t, err)

	// 不支持的格式
	_, err = svc.Preview(ctx, importer.Format("xml"), strings.NewReader(validCSV), "users")
	assert.Error(t, err)

	// 解析失败（空文件无表头）
	_, err = svc.Preview(ctx, importer.FormatCSV, strings.NewReader(""), "users")
	assert.Error(t, err)
}

func TestImportServiceImport(t *testing.T) {
	ctx := context.Background()

	// 成功导入
	db := newSqliteDB(t, &importUser{})
	svc := newImportService(db)
	result, job, err := svc.Import(ctx, importer.FormatCSV, strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, importdomain.ImportJobCompleted, job.Status)
	assert.Equal(t, 1, result.SuccessRows)
	assert.Equal(t, 1, job.TotalRows)
	assert.Equal(t, uint64(42), job.ID)
	// 用户确实落库
	var count int64
	assert.NoError(t, db.Model(&importUser{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)

	// 校验失败：不创建任务
	svc = newImportService(newSqliteDB(t, &importUser{}))
	result, job, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(badCSV), "users", 1, "users.csv")
	assert.NoError(t, err)
	assert.Nil(t, job)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.ErrorRows)

	// 不支持的类型
	svc = newImportService(newSqliteDB(t, &importUser{}))
	_, _, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(validCSV), "bogus", 1, "users.csv")
	assert.Error(t, err)

	// 不支持的格式
	_, _, err = svc.Import(ctx, importer.Format("xml"), strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.Error(t, err)

	// 解析失败
	_, _, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(""), "users", 1, "users.csv")
	assert.Error(t, err)

	// 创建任务失败
	svc = NewImportService(&fakeImportJobRepo{create: func(ctx context.Context, job *importdomain.ImportJob) error {
		return errors.New("db down")
	}}, newSqliteDB(t, &importUser{}))
	_, _, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.Error(t, err)

	// 更新任务失败（返回 result + job + err）
	svc = NewImportService(&fakeImportJobRepo{update: func(ctx context.Context, job *importdomain.ImportJob) error {
		return errors.New("db down")
	}}, newSqliteDB(t, &importUser{}))
	result, job, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.Error(t, err)
	assert.NotNil(t, job)
	assert.Nil(t, result)
}

// TestImportServiceUncompiledFormat 未编译的格式（本测试二进制只 blank import csv 驱动）
// 必须按参数错误返回：驱动缺失是 fail-closed 的调用方问题（400），不是内部错误（500）。
// Preview 与 Import 两条路径各自包了 apperrors.Wrap，两条都钉住。
func TestImportServiceUncompiledFormat(t *testing.T) {
	require.NotContains(t, importer.RegisteredFormats(), importer.FormatExcel,
		"excel 驱动不该编入本测试二进制，否则本用例失去区分力")

	ctx := context.Background()
	svc := newImportService(newSqliteDB(t, &importUser{}))

	_, err := svc.Preview(ctx, importer.FormatExcel, strings.NewReader(validCSV), "users")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInvalidParam), "Preview 应返回 CodeInvalidParam，实际：%v", err)

	_, _, err = svc.Import(ctx, importer.FormatExcel, strings.NewReader(validCSV), "users", 1, "users.xlsx")
	require.Error(t, err)
	assert.True(t, apperrors.IsCode(err, apperrors.CodeInvalidParam), "Import 应返回 CodeInvalidParam，实际：%v", err)
}

func TestImportServiceGetImportJob(t *testing.T) {
	ctx := context.Background()
	svc := newImportService(nil)

	job, err := svc.GetImportJob(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, uint64(42), job.ID)

	// 未找到
	svc = NewImportService(&fakeImportJobRepo{findByID: func(ctx context.Context, id uint64) (*importdomain.ImportJob, error) {
		return nil, gorm.ErrRecordNotFound
	}}, nil)
	_, err = svc.GetImportJob(ctx, 42)
	assert.Error(t, err)

	// 其他错误
	svc = NewImportService(&fakeImportJobRepo{findByID: func(ctx context.Context, id uint64) (*importdomain.ImportJob, error) {
		return nil, errors.New("boom")
	}}, nil)
	_, err = svc.GetImportJob(ctx, 42)
	assert.Error(t, err)
}

func TestImportServiceInsertUser(t *testing.T) {
	db := newSqliteDB(t, &importUser{})
	svc := newImportService(db)

	// 上下文无租户：归默认租户
	err := svc.insertUser(context.Background(), map[string]string{"username": "carol", "password": "secret123"})
	assert.NoError(t, err)

	var user importUser
	assert.NoError(t, db.First(&user, "username = ?", "carol").Error)
	assert.Equal(t, int8(1), user.Status)
	assert.Equal(t, tenant.DefaultTenantID, user.TenantID)

	// 上下文有租户：归属该租户（不得悬空为 0）
	err = svc.insertUser(tenant.WithTenant(context.Background(), 7), map[string]string{"username": "dave", "password": "secret123"})
	assert.NoError(t, err)

	var imported importUser
	assert.NoError(t, db.First(&imported, "username = ?", "dave").Error)
	assert.Equal(t, uint64(7), imported.TenantID)
}

func TestRulesFor(t *testing.T) {
	rules, err := rulesFor("users")
	assert.NoError(t, err)
	assert.Len(t, rules.Fields, 3)

	_, err = rulesFor("nope")
	assert.Error(t, err)
}

// 确保 bytes 导入仅用于显式构造 io.Reader 的场景（如已存在的 buffer 输入）
func TestImportServicePreviewFromBuffer(t *testing.T) {
	ctx := context.Background()
	svc := newImportService(newSqliteDB(t, &importUser{}))
	buf := bytes.NewBufferString(validCSV)
	result, err := svc.Preview(ctx, importer.FormatCSV, buf, "users")
	assert.NoError(t, err)
	assert.Equal(t, 1, result.TotalRows)
}

func TestImportServiceJobTenant(t *testing.T) {
	// 导入任务归属上下文租户
	var created *importdomain.ImportJob
	svc := NewImportService(&fakeImportJobRepo{create: func(ctx context.Context, job *importdomain.ImportJob) error {
		created = job
		return nil
	}}, newSqliteDB(t, &importUser{}))
	_, job, err := svc.Import(tenant.WithTenant(context.Background(), 7), importer.FormatCSV, strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, uint64(7), created.TenantID)

	// 跨租户查询导入任务不可见，同租户可见
	svc = NewImportService(&fakeImportJobRepo{findByID: func(ctx context.Context, id uint64) (*importdomain.ImportJob, error) {
		return &importdomain.ImportJob{ID: id, TenantID: 2}, nil
	}}, nil)
	_, err = svc.GetImportJob(tenant.WithTenant(context.Background(), 7), 1)
	assert.Error(t, err)
	_, err = svc.GetImportJob(tenant.WithTenant(context.Background(), 2), 1)
	assert.NoError(t, err)
}
