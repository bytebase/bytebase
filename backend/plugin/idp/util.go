// Package idp is the plugin for Identity Provider.
package idp

import "strings"

// GetValueWithKey returns the value of the key in the data.
func GetValueWithKey(data map[string]interface{}, key string) interface{} {
	keys := strings.Split(key, ".")
	value := data[keys[0]]

	if len(keys) > 1 {
		if subData, ok := value.(map[string]interface{}); ok {
			return GetValueWithKey(subData, strings.Join(keys[1:], "."))
		}
		return nil
	}

	return value
}
