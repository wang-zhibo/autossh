package app

import (
	"fmt"
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
	// 美化菜单显示
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│  📝 add     - 添加新服务器    │  ✏️  edit   - 编辑服务器    │")
	fmt.Println("│  🗑️  remove  - 删除服务器    │  🚪 exit    - 退出程序      │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
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
