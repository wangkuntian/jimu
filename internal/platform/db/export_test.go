package db

// CapabilitiesInRunOrder 导出内部迭代序选择函数，供 db_test 白盒测试钉死
// down/redo 反转清单切片的行为（测试文件在 db_test 包，无法直接访问未导出符号）。
var CapabilitiesInRunOrder = capabilitiesInRunOrder
