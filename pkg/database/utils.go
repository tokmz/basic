package database

import (
	"encoding/json"
)

// deepCopyJSON 使用JSON序列化/反序列化执行深拷贝
// 这是线程安全的，能正确处理复杂的嵌套结构
func deepCopyJSON(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

// safeDeepCopy 安全地执行深拷贝并提供回退机制
func safeDeepCopy(src, dst interface{}) bool {
	return deepCopyJSON(src, dst) == nil
}
