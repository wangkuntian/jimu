package db

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrConcurrentUpdate 乐观锁冲突：目标行已被并发修改（version 不匹配）。
// 调用方应转换为 409（shared/errors.CodeConflict）或自行重试。
var ErrConcurrentUpdate = errors.New("db: concurrent update detected")

// LockRow 在事务内以 SELECT ... FOR UPDATE 锁定单行，串行化「读-改-写」，
// 避免并发请求互相覆盖（如替换全部关联记录）。必须在事务中调用；
// 等待锁的时间受 ctx 控制，超时由数据库驱动返回错误。
//
// SQLite 不支持行锁语法且其写事务本身串行，故降级为普通查询。
// 返回 *gorm.DB 以便调用方链式判断 .Error（与 GORM 原生风格一致）。
func LockRow(tx *gorm.DB, dest interface{}, id uint64) *gorm.DB {
	if tx.Name() == "sqlite" {
		return tx.First(dest, id)
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(dest, id)
}

// SaveOptimistic 乐观锁更新：以读取时的 version 作为 WHERE 条件，并把 version 自增。
// 未命中任何行说明该行已被并发修改，返回 ErrConcurrentUpdate。
// updates 只需包含业务字段，version 由本函数维护。
func SaveOptimistic(tx *gorm.DB, model interface{}, id uint64, oldVersion int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("db: SaveOptimistic requires at least one field")
	}
	updates["version"] = oldVersion + 1

	res := tx.Model(model).
		Where("id = ? AND version = ?", id, oldVersion).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConcurrentUpdate
	}
	return nil
}
