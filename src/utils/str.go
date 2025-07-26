package utils

import (
	"unicode"
)

// 计算字符宽度（中文和emoji）
func ZhLen(str string) int {
	length := 0
	for _, c := range str {
		if unicode.Is(unicode.Scripts["Han"], c) {
			// 中文字符宽度为2
			length += 2
		} else if (c >= 0x1f300 && c <= 0x1f9ff) || // 各种符号和象形文字
			(c >= 0x2600 && c <= 0x26ff) || // 杂项符号
			(c >= 0x2700 && c <= 0x27bf) || // 装饰符号
			(c >= 0x1f000 && c <= 0x1f2ff) || // 麻将牌等
			(c >= 0x1f600 && c <= 0x1f64f) || // 表情符号
			(c >= 0x1f680 && c <= 0x1f6ff) || // 交通和地图符号
			(c >= 0x1f700 && c <= 0x1f77f) || // 炼金术符号
			(c >= 0x1f780 && c <= 0x1f7ff) || // 几何形状扩展
			(c >= 0x1f800 && c <= 0x1f8ff) || // 补充箭头-C
			(c >= 0x1f900 && c <= 0x1f9ff) { // 补充符号和象形文字
			// emoji和特殊符号宽度为2
			length += 2
		} else {
			// ASCII和其他字符宽度为1
			length += 1
		}
	}

	return length
}

// 左右填充
// title 主体内容
// c 填充符号
// maxlength 总长度
// 如： title = 测试 c=* maxlength = 10 返回 ** 返回 **
func FormatSeparator(title string, c string, maxlength int) string {
	charslen := (maxlength - ZhLen(title)) / 2
	chars := ""
	for i := 0; i < charslen; i++ {
		chars += c
	}

	return chars + title + chars
}

// 右填充
//func AppendRight(body string, char string, maxlength int) string {
//	length := ZhLen(body)
//	if length >= maxlength {
//		return body
//	}
//
//	for i := 0; i < maxlength-length; i++ {
//		body = body + char
//	}
//
//	return body
//}
