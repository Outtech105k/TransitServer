package models

import (
	"fmt"
	"reflect"
)

// CheckExclusiveParams は、指定された2つのJSONタグに基づいて、どちらか一方のフィールドがnilであるかを確認する。
// エラーがあればエラーメッセージを返す。
func CheckExclusiveParams(v any, tag1, tag2 string) (bool, string) {
	val := reflect.ValueOf(v)

	var field1, field2 reflect.Value
	var tag1Found, tag2Found bool

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		tag := field.Tag.Get("json")

		if tag == tag1 {
			field1 = val.Field(i)
			tag1Found = true
		} else if tag == tag2 {
			field2 = val.Field(i)
			tag2Found = true
		}
	}

	// フィールドが見つからなかった場合、タグ名を返す
	if !tag1Found {
		return false, fmt.Sprintf("`%s` or must be set.", tag1)
	}
	if !tag2Found {
		return false, fmt.Sprintf("`%s` or must be set.", tag2)
	}

	// 両方が設定されている場合エラー (両方nil、もしくは両方非nilであればエラー)
	if (!field1.IsNil() && !field2.IsNil()) || (field1.IsNil() && field2.IsNil()) {
		return false, fmt.Sprintf("Either `%s` or `%s` must be set, but not both.", tag1, tag2)
	}

	// 問題がなければtrueを返す
	return true, ""
}
