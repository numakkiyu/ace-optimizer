package main

import (
	"fmt"
	"strings"
)

const (
	MenuMain = iota
	MenuSettings
	MenuHelp
)

const (
	AppName   = "ACE反作弊优化工具"
	Version   = "v2.1.0"
	Author    = "作者：北海的佰川"
	Website   = "GitHub：https://github.com/numakkiyu/ace-optimizer"
	License   = "开源协议：MIT License"
	Copyright = "反馈：https://me.tianbeigm.cn"
)

type MenuItem struct {
	Key         string
	Description string
	Help        string
	Details     []string
}

var mainMenu = []MenuItem{
	{
		Key:         "1",
		Description: "正常启动",
		Help:        "普通模式启动游戏，不做任何修改",
		Details: []string{
			"• 直接启动游戏，不做任何修改",
			"• 适合正常玩家使用",
		},
	},
	{
		Key:         "2",
		Description: "亲和启动",
		Help:        "将ACE反作弊限制在特定CPU核心上运行",
		Details: []string{
			"• 将ACE反作弊限制在单个CPU核心上运行",
			"• 自动选择最优核心，避免影响游戏性能",
			"• 仅在游戏运行后可用",
		},
	},
	{
		Key:         "3",
		Description: "挂起启动",
		Help:        "暂时挂起ACE反作弊进程（风险操作）",
		Details: []string{
			"• 启动游戏时自动关闭ACE服务",
			"• 等待游戏和ACE完全启动后挂起进程",
			"• 仅在游戏运行后可用",
			"• 当游戏运行的时候，并且面板提示反作弊未运行的时候，说明挂起成功",
			"• ⚠️ 警告：部分电脑在运行。程序的时候可能会导致游戏闪退。严重导致极小可能封号，请谨慎使用",
		},
	},
	{
		Key:         "4",
		Description: "设置",
		Help:        "配置游戏路径和程序选项",
	},
	{
		Key:         "5",
		Description: "关于/帮助",
		Help:        "查看使用说明和关于信息",
	},
	{
		Key:         "0",
		Description: "退出",
		Help:        "退出程序",
	},
}

var noticeHelp = []string{
	"1. 必须以管理员身份运行此程序",
	"2. 首次使用请先设置游戏路径",
	"3. 亲和启动和挂起启动仅在游戏运行后可用",
	"4. 挂起模式有被检测的风险，请谨慎使用",
}

func drawTitle() {
	width := 60
	title := fmt.Sprintf("%s %s", AppName, Version)
	padding := (width - len(title)) / 2

	fmt.Println(colorize(strings.Repeat("═", width), FgCyan))
	fmt.Println(colorize(fmt.Sprintf("║%s%s%s║",
		strings.Repeat(" ", padding),
		title,
		strings.Repeat(" ", width-padding-len(title)-2)),
		FgCyan))
	fmt.Println(colorize(strings.Repeat("═", width), FgCyan))
	fmt.Println()
}

func drawMenu(items []MenuItem) {
	maxKeyLen := 0
	maxDescLen := 0

	for _, item := range items {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
		if len(item.Description) > maxDescLen {
			maxDescLen = len(item.Description)
		}
	}

	for _, item := range items {
		keyPart := colorize(fmt.Sprintf("[%s]", item.Key), FgYellow)
		descPart := colorize(item.Description, FgWhite)
		fmt.Printf("%s %-*s", keyPart, maxDescLen+4, descPart)
		if item.Help != "" {
			fmt.Printf(" - %s", colorize(item.Help, FgCyan))
		}
		fmt.Println()
	}
	fmt.Println()
}

func drawSettingsMenu(config *Config) {
	items := []MenuItem{
		{
			Key:         "1",
			Description: fmt.Sprintf("游戏路径: %s", config.GamePath),
			Help:        "设置游戏启动器路径",
			Details:     []string{},
		},
		{
			Key:         "2",
			Description: fmt.Sprintf("运行后关闭: %v", config.AutoClose),
			Help:        "设置是否在操作完成后自动退出程序",
			Details:     []string{},
		},
		{
			Key:         "0",
			Description: "返回主菜单",
			Help:        "",
			Details:     []string{},
		},
	}
	drawMenu(items)
}

func drawAbout() {
	fmt.Println(colorize("关于程序", FgCyan))
	fmt.Printf("\n%s %s\n", colorize("程序名称:", FgWhite), AppName)
	fmt.Printf("%s %s\n", colorize("版本:", FgWhite), Version)
	fmt.Printf("%s\n", colorize(Author, FgWhite))
	fmt.Printf("%s\n", colorize(Website, FgBlue))
	fmt.Printf("%s\n", colorize(License, FgWhite))
	fmt.Printf("%s\n", colorize(Copyright, FgWhite))

	fmt.Printf("\n%s\n", colorize("使用说明", FgYellow))
	fmt.Println(colorize("\n启动模式：", FgYellow))
	for _, item := range mainMenu {
		if item.Key != "4" && item.Key != "5" && item.Key != "0" {
			fmt.Printf("\n%s. %s\n", item.Key, colorize(item.Description, FgWhite))
			for _, detail := range item.Details {
				fmt.Printf("   %s\n", detail)
			}
		}
	}

	fmt.Printf("\n%s\n", colorize("注意事项：", FgRed))
	for _, notice := range noticeHelp {
		fmt.Printf("• %s\n", notice)
	}
	fmt.Println()
}

func drawStatus(status string, color int) {
	fmt.Printf("\n%s\n\n", colorize(status, color))
}

func readInput() string {
	var input string
	fmt.Print(colorize("请选择操作: ", FgGreen))
	fmt.Scanln(&input)
	return input
}

func confirmAction(message string) bool {
	fmt.Print(colorize(message+" (Y/N): ", FgYellow))
	var input string
	fmt.Scanln(&input)
	return strings.ToUpper(input) == "Y"
}

func inputPath(prompt string) string {
	fmt.Print(colorize(prompt+": ", FgGreen))
	var path string
	fmt.Scanln(&path)
	path = strings.Trim(path, "\"")
	return path
}
