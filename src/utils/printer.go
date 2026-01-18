package utils

import "fmt"

// 打印一行信息
// 字体颜色为默色
func Logln(a ...interface{}) {
	fmt.Println(a...)
}

func Logf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}

// 打印（不换行）
// 字体颜色为默色
func Log(a ...interface{}) {
	fmt.Print(a...)
}

// 打印一行错误
// 字体颜色为红色
func Errorln(a ...interface{}) {
	fmt.Print("\033[31m")
	Logln(a...)
	fmt.Print("\033[0m")
}

func Errorf(format string, args ...interface{}) {
	Errorln(fmt.Sprintf(format, args...))
}

func Warnf(format string, args ...interface{}) {
	Logln(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...interface{}) {
	Logln(fmt.Sprintf(format, args...))
}

func Debugf(format string, args ...interface{}) {
	Logln(fmt.Sprintf(format, args...))
}

// 二维数组对齐
//func Align(arr [][]string) [][]string {
//	for column := 0; column < 2; column++ {
//		columnWidth := getColumnWidth(arr, column)
//
//		for index := range arr {
//			arr[index][column] = AppendRight(arr[index][column], " ", columnWidth)
//		}
//	}
//
//	return arr
//}
//
//func getColumnWidth(arr [][]string, column int) int {
//	maxWidth := 0
//	for _, row := range arr {
//		width := int(ZhLen(row[column]))
//		if maxWidth < width {
//			maxWidth = width
//		}
//	}
//
//	return maxWidth
//}
