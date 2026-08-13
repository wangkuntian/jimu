package application

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	admindomain "jimu/internal/modules/admin/domain"
	userdomain "jimu/internal/modules/user/domain"
	"jimu/internal/platform/importer"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

const validCSV = "username,password,email\nalice,secret123,a@b.com\n"
const badCSV = "username,password,email\n,secret123,a@b.com\n"

func newImportService(db *gorm.DB) *ImportService {
	return NewImportService(&fakeImportJobRepo{}, &fakeUserRepository{}, db)
}

func TestImportServicePreview(t *testing.T) {
	ctx := context.Background()
	svc := newImportService(newSqliteDB(t, &userdomain.User{}))

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
	db := newSqliteDB(t, &userdomain.User{})
	svc := newImportService(db)
	result, job, err := svc.Import(ctx, importer.FormatCSV, strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, admindomain.ImportJobCompleted, job.Status)
	assert.Equal(t, 1, result.SuccessRows)
	assert.Equal(t, 1, job.TotalRows)
	assert.Equal(t, uint64(42), job.ID)
	// 用户确实落库
	var count int64
	assert.NoError(t, db.Model(&userdomain.User{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)

	// 校验失败：不创建任务
	svc = newImportService(newSqliteDB(t, &userdomain.User{}))
	result, job, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(badCSV), "users", 1, "users.csv")
	assert.NoError(t, err)
	assert.Nil(t, job)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.ErrorRows)

	// 不支持的类型
	svc = newImportService(newSqliteDB(t, &userdomain.User{}))
	_, _, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(validCSV), "bogus", 1, "users.csv")
	assert.Error(t, err)

	// 不支持的格式
	_, _, err = svc.Import(ctx, importer.Format("xml"), strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.Error(t, err)

	// 解析失败
	_, _, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(""), "users", 1, "users.csv")
	assert.Error(t, err)

	// 创建任务失败
	svc = NewImportService(&fakeImportJobRepo{create: func(ctx context.Context, job *admindomain.ImportJob) error {
		return errors.New("db down")
	}}, &fakeUserRepository{}, newSqliteDB(t, &userdomain.User{}))
	_, _, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.Error(t, err)

	// 更新任务失败（返回 result + job + err）
	svc = NewImportService(&fakeImportJobRepo{update: func(ctx context.Context, job *admindomain.ImportJob) error {
		return errors.New("db down")
	}}, &fakeUserRepository{}, newSqliteDB(t, &userdomain.User{}))
	result, job, err = svc.Import(ctx, importer.FormatCSV, strings.NewReader(validCSV), "users", 1, "users.csv")
	assert.Error(t, err)
	assert.NotNil(t, job)
	assert.Nil(t, result)
}

func TestImportServiceGetImportJob(t *testing.T) {
	ctx := context.Background()
	svc := newImportService(nil)

	job, err := svc.GetImportJob(ctx, 42)
	assert.NoError(t, err)
	assert.Equal(t, uint64(42), job.ID)

	// 未找到
	svc = NewImportService(&fakeImportJobRepo{findByID: func(ctx context.Context, id uint64) (*admindomain.ImportJob, error) {
		return nil, gorm.ErrRecordNotFound
	}}, &fakeUserRepository{}, nil)
	_, err = svc.GetImportJob(ctx, 42)
	assert.Error(t, err)

	// 其他错误
	svc = NewImportService(&fakeImportJobRepo{findByID: func(ctx context.Context, id uint64) (*admindomain.ImportJob, error) {
		return nil, errors.New("boom")
	}}, &fakeUserRepository{}, nil)
	_, err = svc.GetImportJob(ctx, 42)
	assert.Error(t, err)
}

func TestImportServiceInsertUser(t *testing.T) {
	ctx := context.Background()
	db := newSqliteDB(t, &userdomain.User{})
	svc := newImportService(db)
	err := svc.insertUser(ctx, map[string]string{"username": "carol", "password": "secret123"})
	assert.NoError(t, err)

	var user userdomain.User
	assert.NoError(t, db.First(&user, "username = ?", "carol").Error)
	assert.Equal(t, int8(1), user.Status)
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
	svc := newImportService(newSqliteDB(t, &userdomain.User{}))
	buf := bytes.NewBufferString(validCSV)
	result, err := svc.Preview(ctx, importer.FormatCSV, buf, "users")
	assert.NoError(t, err)
	assert.Equal(t, 1, result.TotalRows)
}
