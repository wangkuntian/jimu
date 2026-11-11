package report

import (
	"slices"
	"sort"
	"strings"
)

// FormatDrivers 把驱动选择格式化为报告用的稳定文本：<capability>=<driver1>,<driver2>。
func FormatDrivers(drivers map[string][]string) []string {
	values := make([]string, 0, len(drivers))
	for name, list := range drivers {
		if len(list) == 0 {
			continue
		}
		drivers := slices.Clone(list)
		slices.Sort(drivers)
		values = append(values, name+"="+strings.Join(drivers, ","))
	}
	sort.Strings(values)
	return values
}
