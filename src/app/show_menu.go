package app

import (
	"autossh/src/utils"
	"fmt"
	"strings"
)

type Operation struct {
	Key     string
	Label   string
	End     bool
	Process func(cfg *Config, args []string) error
}

var menuMap [][]Operation

var operations = make(map[string]Operation)

func init() {
	menuMap = [][]Operation{
		{
			{Key: "add", Label: "添加", Process: handleAdd},
			{Key: "edit", Label: "编辑", Process: handleEdit},
			{Key: "remove", Label: "删除", Process: handleRemove},
		},
		{
			{Key: "exit", Label: "退出", End: true},
		},
	}

	// 初始化operations映射
	for i := 0; i < len(menuMap); i++ {
		for j := 0; j < len(menuMap[i]); j++ {
			operation := menuMap[i][j]
			operations[operation.Key] = operation
		}
	}
}

func showMenu() {
	// 美化菜单显示 - 固定宽度版本
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│  📝 add     - 添加新服务器    │  ✏️  edit   - 编辑服务器    │")
	fmt.Println("│  🗑️ remove  - 删除服务器      │  🚪  exit   - 退出程序      │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
}

// 新增：支持动态宽度的菜单显示
func showMenuWithWidth(width int) {
	// 菜单内容
	line1 := "  📝 add     - 添加新服务器    │  ✏️  edit   - 编辑服务器    "
	line2 := "  🗑️ remove  - 删除服务器      │  🚪  exit   - 退出程序      "

	// 计算每行的实际宽度
	line1Width := utils.ZhLen(line1)
	line2Width := utils.ZhLen(line2)

	// 使用较长的行作为基准
	contentWidth := line1Width
	if line2Width > contentWidth {
		contentWidth = line2Width
	}

	// 确保菜单宽度不超过指定宽度
	if contentWidth+4 > width {
		width = contentWidth + 4
	}

	// 顶部边框
	fmt.Println("┌" + strings.Repeat("─", width-2) + "┐")

	// 第一行
	fmt.Print("│")
	fmt.Print(line1)
	padding1 := width - line1Width - 2
	if padding1 < 0 {
		padding1 = 0
	}
	fmt.Print(strings.Repeat(" ", padding1))
	fmt.Println("│")

	// 第二行
	fmt.Print("│")
	fmt.Print(line2)
	padding2 := width - line2Width - 2
	if padding2 < 0 {
		padding2 = 0
	}
	fmt.Print(strings.Repeat(" ", padding2))
	fmt.Println("│")

	// 底部边框
	fmt.Println("└" + strings.Repeat("─", width-2) + "┘")
}

func operationFormat(operation Operation) string {
	return "[" + operation.Key + "] " + operation.Label
}

func stringPadding(str string, paddingLen int) string {
	if len(str) < paddingLen {
		return stringPadding(str+" ", paddingLen)
	} else {
		return str
	}
}
