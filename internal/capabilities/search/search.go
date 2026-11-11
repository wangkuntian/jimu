// Package search 提供基于业务数据库的关键词检索，具体方言由驱动包提供。
package search

import (
	"errors"
	"sort"
	"strings"

	"jimu/internal/capabilities/search/searchcore"

	"gorm.io/gorm"
)

type (
	// Document 待索引文档。
	Document = searchcore.Document
	// Result 检索结果。
	Result = searchcore.Result
	// Searcher 检索器接口。
	Searcher = searchcore.Searcher
)

// Factory 构造指定方言的检索器。
type Factory func(*gorm.DB) (Searcher, error)

var drivers = map[string]Factory{}

// Register 注册检索驱动，仅供驱动包在 init() 中调用。
func Register(name string, f Factory) {
	if _, dup := drivers[name]; dup {
		panic("search: driver already registered: " + name)
	}
	drivers[name] = f
}

// RegisteredDrivers 返回本构建已注册的驱动名（升序）。
func RegisteredDrivers() []string {
	out := make([]string, 0, len(drivers))
	for name := range drivers {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// New 按数据库方言选择已编译的驱动。
func New(gdb *gorm.DB) (Searcher, error) {
	if gdb == nil {
		return nil, errors.New("search: nil database")
	}
	name := gdb.Name()
	driver, ok := drivers[name]
	if !ok {
		return nil, searchcore.NewUnsupportedDialectError(name, driverNames(RegisteredDrivers()))
	}
	return driver(gdb)
}

func driverNames(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}
