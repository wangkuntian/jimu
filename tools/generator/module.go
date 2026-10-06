package generator

import "jimu/tools/generator/module"

// GenerateModule 生成完整的模块骨架（Clean Architecture 分层）。
func GenerateModule(name string) error {
	return module.GenerateModule(name)
}

// GenerateModuleAt 在指定框架仓根目录生成完整的模块骨架。
func GenerateModuleAt(root, name string) error {
	return module.GenerateModuleAt(root, name)
}
