package utils

import (
	"unicode"
)

// 计算字符宽度（中文和emoji）- 终端兼容版本
func ZhLen(str string) int {
	length := 0
	runes := []rune(str)
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		
		if r == '\t' {
			// 制表符按4个空格计算
			length += 4
		} else if r < 32 {
			// 控制字符不计算宽度
			continue
		} else if r < 127 {
			// ASCII字符宽度为1
			length += 1
		} else if unicode.Is(unicode.Scripts["Han"], r) ||
			unicode.Is(unicode.Scripts["Hiragana"], r) ||
			unicode.Is(unicode.Scripts["Katakana"], r) ||
			unicode.Is(unicode.Scripts["Hangul"], r) {
			// 中日韩字符宽度为2
			length += 2
		} else if isWideEmoji(r) {
			// 宽emoji字符宽度为2，但对于某些终端兼容性问题，使用1.5倍宽度
			length += 2
			// 检查是否有变体选择器跟随
			if i+1 < len(runes) && isVariationSelector(runes[i+1]) {
				i++ // 跳过变体选择器
			}
		} else if unicode.In(r, unicode.Mn, unicode.Me, unicode.Mc) {
			// 组合字符不增加宽度
			continue
		} else if isVariationSelector(r) {
			// 变体选择器不增加宽度
			continue
		} else {
			// 其他字符宽度为1
			length += 1
		}
	}

	return length
}

// 判断是否为宽emoji字符 - 更精确的检测
// 判断是否为宽emoji字符 - 更精确的检测
func isWideEmoji(r rune) bool {
	// 基本emoji范围
	if (r >= 0x1F600 && r <= 0x1F64F) || // 表情符号
		(r >= 0x1F300 && r <= 0x1F5FF) || // 杂项符号和象形文字（包含👋 0x1F44B）
		(r >= 0x1F680 && r <= 0x1F6FF) || // 交通和地图符号
		(r >= 0x1F1E6 && r <= 0x1F1FF) || // 区域指示符号
		(r >= 0x2600 && r <= 0x26FF) ||   // 杂项符号
		(r >= 0x2700 && r <= 0x27BF) {    // 装饰符号
		return true
	}
	
	// 特定的emoji字符
	switch r {
	case 0x1F4C1, 0x1F4C2: // 📁 📂
		return true
	case 0x1F3F7: // 🏷️ (标签)
		return true
	case 0x23F3: // ⏳ (沙漏)
		return true
	case 0x2705: // ✅ (白色重勾号)
		return true
	case 0x1F44B: // 👋 (挥手) - 明确添加确保识别
		return true
	default:
		return false
	}
}

// 判断是否为变体选择器
func isVariationSelector(r rune) bool {
	return (r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF)
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
