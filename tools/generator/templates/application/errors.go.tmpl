package application

import (
	stderrors "errors"

	mysqlerr "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func isDuplicateKey(err error) bool {
	if stderrors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysqlerr.MySQLError
	return stderrors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
