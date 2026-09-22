package search

import (
	"jimu/internal/assembly"
	"jimu/internal/contract"
)

// Wire 装配搜索能力：search 仅声明 Descriptor（拥有 search_documents 表与迁移），
// 无 Module 实例与端口；Searcher 由检索调用方按数据库方言自行构造。
func Wire(*assembly.Context) (contract.Module, error) { return nil, nil }
