package db

import (
	"fmt"
	"testing"

	"jimu/internal/platform/encryption"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const encTestKey = "0123456789abcdef0123456789abcdef"

type contact struct {
	ID        uint64 `gorm:"primaryKey"`
	Email     string `gorm:"type:text" encryption:"true"`
	EmailHash string `gorm:"size:64;uniqueIndex" blind:"email"`
	Phone     string `gorm:"type:text" encryption:"true"`
	PhoneHash string `gorm:"size:64;uniqueIndex" blind:"phone"`
}

func (contact) TableName() string { return "contacts" }

// contactPtr 与 contact 等价，但盲索引为指针字段（空源→NULL），
// 用于验证可选 email/phone 的多用户不触发唯一冲突。
type contactPtr struct {
	ID        uint64  `gorm:"primaryKey"`
	Email     string  `gorm:"type:text" encryption:"true"`
	EmailHash *string `gorm:"size:64;uniqueIndex" blind:"email"`
}

func (contactPtr) TableName() string { return "contact_ptrs" }

func newEncryptionTestDB(t *testing.T, key string) *gorm.DB {
	t.Helper()
	// 每个测试独立内存库，避免共享 cache 串数据
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	RegisterEncryptionHooks(db, encryption.New(key))
	require.NoError(t, db.AutoMigrate(&contact{}))
	return db
}

func TestEncryptionHookEncryptsOnWriteDecryptsOnRead(t *testing.T) {
	cipher := encryption.New(encTestKey)
	db := newEncryptionTestDB(t, encTestKey)

	require.NoError(t, db.Create(&contact{Email: "user@example.com", Phone: "13800138000"}).Error)

	// 落库为密文（非明文）+ 盲索引已写入
	var stored string
	require.NoError(t, db.Raw("SELECT email FROM contacts LIMIT 1").Scan(&stored).Error)
	assert.NotEqual(t, "user@example.com", stored)
	assert.Contains(t, stored, "enc:v1:")

	var got contact
	require.NoError(t, db.First(&got).Error)
	assert.Equal(t, "user@example.com", got.Email)
	assert.Equal(t, "13800138000", got.Phone)
	assert.Equal(t, cipher.BlindIndex("user@example.com"), got.EmailHash)
	assert.Equal(t, cipher.BlindIndex("13800138000"), got.PhoneHash)
}

func TestEncryptionHookEmptySourceStoresNullAndCoexists(t *testing.T) {
	db := newEncryptionTestDB(t, encTestKey)
	require.NoError(t, db.AutoMigrate(&contactPtr{}))
	// 两个无 email 用户：hash 均为 NULL，唯一索引不得冲突
	require.NoError(t, db.Create(&contactPtr{}).Error)
	require.NoError(t, db.Create(&contactPtr{Email: ""}).Error)
	require.NoError(t, db.Create(&contactPtr{Email: "b@example.com"}).Error)

	var rows []contactPtr
	require.NoError(t, db.Find(&rows).Error)
	assert.Len(t, rows, 3)
	assert.Nil(t, rows[0].EmailHash, "空源字段盲索引必须为 NULL")
	assert.NotNil(t, rows[2].EmailHash, "非空源字段盲索引必须计算")
}

func TestEncryptionHookUniqueHashEnforced(t *testing.T) {
	db := newEncryptionTestDB(t, encTestKey)
	require.NoError(t, db.Create(&contact{Email: "dup@example.com"}).Error)
	err := db.Create(&contact{Email: "dup@example.com"}).Error
	assert.Error(t, err, "同一 email 盲索引唯一约束必须拦截")
}

func TestEncryptionHookNoKeyStoresPlaintextButHashes(t *testing.T) {
	db := newEncryptionTestDB(t, "")

	require.NoError(t, db.Create(&contact{Email: "plain@example.com", Phone: "13900139000"}).Error)

	var stored string
	require.NoError(t, db.Raw("SELECT email FROM contacts LIMIT 1").Scan(&stored).Error)
	assert.Equal(t, "plain@example.com", stored, "明文模式不加密")

	var got contact
	require.NoError(t, db.First(&got).Error)
	assert.Equal(t, "plain@example.com", got.Email)
	assert.NotEmpty(t, got.EmailHash, "明文模式仍计算盲索引")
}

func TestEncryptionHookBatchCreate(t *testing.T) {
	db := newEncryptionTestDB(t, encTestKey)
	rows := []contact{
		{Email: "a@example.com", Phone: "13000000001"},
		{Email: "b@example.com", Phone: "13000000002"},
	}
	require.NoError(t, db.Create(&rows).Error)

	var got []contact
	require.NoError(t, db.Find(&got).Error)
	assert.Len(t, got, 2)
	assert.Equal(t, "a@example.com", got[0].Email)
	assert.Equal(t, "13000000002", got[1].Phone)
}
