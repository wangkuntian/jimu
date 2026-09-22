package enterprise

// 驱动级可插拔（设计 §3.7）：显式 import 本形态选中的驱动包，注册发生在驱动包 init()。
// 这份清单必须与 assembly.go 里对应能力的 Drivers 声明逐值一致
// （make check-capabilities 静态校验；full 形态同一写法）。
import (
	_ "jimu/internal/capabilities/dataops/csv"
	_ "jimu/internal/capabilities/dataops/excel"
	_ "jimu/internal/capabilities/storage/local"
	_ "jimu/internal/capabilities/storage/s3"
)
