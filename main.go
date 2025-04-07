package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	PROCESS_ALL_ACCESS = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0xFFF
	SE_DEBUG_PRIVILEGE = "SeDebugPrivilege"
)

const (
	FgBlack = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

var (
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	setProcessAffinityMask = kernel32.NewProc("SetProcessAffinityMask")
	ntdll                  = windows.NewLazySystemDLL("ntdll.dll")
	ntSuspendProcess       = ntdll.NewProc("NtSuspendProcess")
)

func colorize(text string, color int) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", color, text)
}

func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

func enableDebugPrivilege() error {
	var token windows.Token
	currentProcess, _ := windows.GetCurrentProcess()

	err := windows.OpenProcessToken(currentProcess, windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token)
	if err != nil {
		return fmt.Errorf("OpenProcessToken failed: %v", err)
	}
	defer token.Close()

	var sedebugnamePtr = syscall.StringToUTF16Ptr(SE_DEBUG_PRIVILEGE)
	var luid windows.LUID
	err = windows.LookupPrivilegeValue(nil, sedebugnamePtr, &luid)
	if err != nil {
		return fmt.Errorf("LookupPrivilegeValue failed: %v", err)
	}

	privileges := windows.Tokenprivileges{
		PrivilegeCount: 1,
		Privileges: [1]windows.LUIDAndAttributes{
			{
				Luid:       luid,
				Attributes: windows.SE_PRIVILEGE_ENABLED,
			},
		},
	}

	err = windows.AdjustTokenPrivileges(token, false, &privileges, 0, nil, nil)
	if err != nil {
		return fmt.Errorf("AdjustTokenPrivileges failed: %v", err)
	}

	return nil
}

func getSystemCPUCount() int {
	return runtime.NumCPU()
}

func findOptimalCPUCore() int {
	cpuCount := getSystemCPUCount()
	// 选择倒数第二个CPU核心，避免使用最后一个核心
	return cpuCount - 2
}

func calculateAffinityMask(cpu int) uint64 {
	return 1 << uint64(cpu)
}

func setAffinity(pid uint32, cpu int) error {
	if err := enableDebugPrivilege(); err != nil {
		return fmt.Errorf("无法启用调试权限: %v", err)
	}

	mask := calculateAffinityMask(cpu)
	handle, err := windows.OpenProcess(PROCESS_ALL_ACCESS, false, pid)
	if err != nil {
		return fmt.Errorf("无法打开进程(错误码:%v)", err)
	}
	defer windows.CloseHandle(handle)

	ret, _, err := setProcessAffinityMask.Call(
		uintptr(handle),
		uintptr(mask),
	)
	if ret == 0 {
		return fmt.Errorf("设置亲和性失败(错误码:%v)", err)
	}
	return nil
}

func suspendProcess(pid uint32) error {
	if err := enableDebugPrivilege(); err != nil {
		return fmt.Errorf("无法启用调试权限: %v", err)
	}

	handle, err := windows.OpenProcess(PROCESS_ALL_ACCESS, false, pid)
	if err != nil {
		return fmt.Errorf("无法打开进程: %v", err)
	}
	defer windows.CloseHandle(handle)

	ret, _, err := ntSuspendProcess.Call(uintptr(handle))
	if ret != 0 {
		return fmt.Errorf("挂起进程失败: %v", err)
	}
	return nil
}

func findProcess(name string) (uint32, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(snapshot)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	err = windows.Process32First(snapshot, &pe)
	if err != nil {
		return 0, err
	}

	for {
		if strings.EqualFold(windows.UTF16ToString(pe.ExeFile[:]), name) {
			return pe.ProcessID, nil
		}

		err = windows.Process32Next(snapshot, &pe)
		if err != nil {
			break
		}
	}
	return 0, fmt.Errorf("进程未找到: %s", name)
}

func isProcessRunning(name string) bool {
	_, err := findProcess(name)
	return err == nil
}

func launchGame(config *Config) error {
	if config.GamePath == "" {
		return fmt.Errorf("游戏路径未设置")
	}

	cmd := exec.Command(config.GamePath)
	cmd.Dir = filepath.Dir(config.GamePath)
	return cmd.Start()
}

func handleNormalLaunch(config *Config) {
	drawStatus("正在启动游戏...", FgYellow)
	if err := launchGame(config); err != nil {
		drawStatus(fmt.Sprintf("启动失败: %v", err), FgRed)
		return
	}
	drawStatus("游戏已启动", FgGreen)
}

func showProcessStatus() {
	gameRunning := isProcessRunning("DeltaForceClient-Win64-Shipping.exe")
	aceRunning := isProcessRunning("SGuard64.exe")

	gameStatus := "✗ 未运行"
	gameColor := FgRed
	if gameRunning {
		gameStatus = "✓ 已运行"
		gameColor = FgGreen
	}

	aceStatus := "✗ 未运行"
	aceColor := FgRed
	if aceRunning {
		aceStatus = "✓ 已运行"
		aceColor = FgGreen
	}

	fmt.Printf("\n%s %s", colorize("游戏状态:", FgWhite), colorize(gameStatus, gameColor))
	fmt.Printf("\n%s %s\n", colorize("反作弊状态:", FgWhite), colorize(aceStatus, aceColor))
}

func waitForProcess(name string, timeout int, message string) bool {
	fmt.Printf("\n%s", colorize(message, FgYellow))
	for i := 0; i < timeout; i++ {
		if isProcessRunning(name) {
			return true
		}
		fmt.Printf("\r%s [%d/%d]", colorize(message, FgYellow), i+1, timeout)
		time.Sleep(1 * time.Second)
	}
	fmt.Printf("\r%s", colorize("等待超时，进程未启动", FgRed))
	return false
}

func handleSuspendLaunch(config *Config) {
	clearScreen()
	drawTitle()
	fmt.Println(colorize("挂起启动模式", FgCyan))
	showProcessStatus()

	// 检查游戏是否已经运行
	if !isProcessRunning("DeltaForceClient-Win64-Shipping.exe") {
		fmt.Printf("\n%s\n", colorize("错误: 请先启动游戏后再使用此功能", FgRed))
		return
	}

	if !confirmAction("\n警告: 挂起反作弊可能导致游戏闪退，是否继续?") {
		return
	}

	// 确保ACE进程正在运行
	if !isProcessRunning("SGuard64.exe") {
		fmt.Printf("\n%s\n", colorize("错误: ACE反作弊未运行", FgRed))
		return
	}

	// 执行挂起操作
	fmt.Println(colorize("\n正在执行挂起操作...", FgYellow))

	// 先关闭ACE服务
	fmt.Println(colorize("正在停止ACE服务...", FgYellow))
	exec.Command("sc", "stop", "AntiCheatExpert Service").Run()
	time.Sleep(2 * time.Second)

	// 强制结束SGuard进程
	fmt.Println(colorize("正在关闭ACE进程...", FgYellow))
	exec.Command("taskkill", "/f", "/im", "SGuard64.exe").Run()
	time.Sleep(2 * time.Second)

	// 检查ACE进程是否已经结束
	if !isProcessRunning("SGuard64.exe") {
		fmt.Printf("\n%s\n", colorize("✓ ACE反作弊已成功关闭", FgGreen))
		showProcessStatus()
		return
	} else {
		fmt.Printf("\n%s\n", colorize("错误: 无法关闭ACE进程", FgRed))
		return
	}
}

func handleAffinityLaunch(config *Config) {
	clearScreen()
	drawTitle()
	fmt.Println(colorize("亲和启动模式", FgCyan))
	showProcessStatus()

	// 检查游戏是否已经运行
	if !isProcessRunning("DeltaForceClient-Win64-Shipping.exe") {
		fmt.Printf("\n%s\n", colorize("错误: 请先启动游戏后再使用此功能", FgRed))
		return
	}

	// 等待ACE进程启动
	if !isProcessRunning("SGuard64.exe") {
		fmt.Printf("\n%s\n", colorize("错误: ACE反作弊未运行", FgRed))
		return
	}

	pid, err := findProcess("SGuard64.exe")
	if err != nil {
		fmt.Printf("\n%s\n", colorize("无法获取ACE进程ID: "+err.Error(), FgRed))
		return
	}

	cpu := findOptimalCPUCore()
	if err := setAffinity(pid, cpu); err != nil {
		fmt.Printf("\n%s\n", colorize("设置CPU亲和性失败: "+err.Error(), FgRed))
		return
	}

	fmt.Printf("\n%s\n", colorize(fmt.Sprintf("✓ 已将ACE反作弊限制在CPU %d上运行", cpu), FgGreen))
	showProcessStatus()
}

func handleSettings(config *Config) {
	for {
		clearScreen()
		drawTitle()
		drawSettingsMenu(config)

		switch readInput() {
		case "1":
			clearScreen()
			drawTitle()
			fmt.Println(colorize("设置游戏路径", FgCyan))
			fmt.Println(colorize("\n选择设置方式：", FgYellow))
			fmt.Println(colorize("[1] 自动搜索", FgWhite), "- 自动在常见位置搜索游戏启动器")
			fmt.Println(colorize("[2] 浏览文件", FgWhite), "- 使用文件选择器选择游戏启动器")
			fmt.Println(colorize("[3] 手动输入", FgWhite), "- 手动输入或粘贴游戏路径")
			fmt.Println(colorize("[0] 返回", FgWhite))
			fmt.Println()

			var path string
			switch readInput() {
			case "1":
				path = autoDetectGamePath()
			case "2":
				path = openFileDialog("选择游戏启动器(delta_force_launcher.exe)")
			case "3":
				path = inputPath("请输入游戏启动器路径（可直接拖拽文件到此处）")
			case "0":
				continue
			}

			if path != "" {
				// 验证文件是否存在且是正确的文件
				if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
					if strings.EqualFold(filepath.Base(path), "delta_force_launcher.exe") {
						config.GamePath = path
						saveConfig(config)
						drawStatus("游戏路径已保存", FgGreen)
					} else {
						drawStatus("错误：请选择delta_force_launcher.exe文件", FgRed)
					}
				} else {
					drawStatus("错误：无效的文件路径", FgRed)
				}
			}
		case "2":
			config.AutoClose = !config.AutoClose
			saveConfig(config)
			drawStatus(fmt.Sprintf("运行后自动关闭已设为: %v", config.AutoClose), FgGreen)
		case "0":
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func showHelp() {
	clearScreen()
	drawTitle()
	fmt.Println(colorize("使用说明：", FgCyan))
	fmt.Println()
	fmt.Println(colorize("启动模式：", FgYellow))
	fmt.Println("1. 正常启动")
	fmt.Println("   • 直接启动游戏，不做任何修改")
	fmt.Println("   • 适合正常玩家使用")
	fmt.Println()
	fmt.Println("2. 亲和启动")
	fmt.Println("   • 将ACE反作弊限制在单个CPU核心上运行")
	fmt.Println("   • 自动选择最优核心，避免影响游戏性能")
	fmt.Println("   • 可以在游戏运行前或运行后使用")
	fmt.Println()
	fmt.Println("3. 挂起启动")
	fmt.Println("   • 启动游戏时自动关闭ACE服务")
	fmt.Println("   • 等待游戏和ACE完全启动后挂起进程")
	fmt.Println("   • 当游戏运行的时候，并且面板提示反作弊未运行的时候，说明挂起成功")
	fmt.Println("   • ⚠️ 警告：部分电脑在运行。程序的时候可能会导致游戏闪退。严重导致极小可能封号，请谨慎使用")
	fmt.Println()
	fmt.Println(colorize("配置说明：", FgYellow))
	fmt.Println("• 游戏路径：支持自动搜索、浏览和手动输入")
	fmt.Println("• 自动关闭：操作完成后是否自动退出程序")
	fmt.Println()
	fmt.Println(colorize("注意事项：", FgRed))
	fmt.Println("1. 必须以管理员身份运行此程序")
	fmt.Println("2. 首次使用请先设置游戏路径")
	fmt.Println("3. 如果启动失败，请适当增加启动延时")
	fmt.Println("4. 挂起模式有被检测的风险，请谨慎使用")
	fmt.Println()
	fmt.Print(colorize("按任意键返回主菜单...", FgGreen))
	fmt.Scanln()
}

func main() {
	if !isAdmin() {
		clearScreen()
		drawTitle()
		fmt.Println(colorize("错误: 请以管理员身份运行此程序！", FgRed))
		fmt.Println(colorize("\n请右键点击程序 -> 选择'以管理员身份运行'", FgYellow))
		fmt.Print(colorize("\n按任意键退出...", FgGreen))
		fmt.Scanln()
		return
	}

	// 启用控制台ANSI转义序列
	kernel32.NewProc("SetConsoleMode").Call(uintptr(os.Stdout.Fd()), 0x0001|0x0004)

	config, err := loadConfig()
	if err != nil {
		fmt.Printf(colorize("加载配置失败: %v\n", FgRed), err)
		fmt.Println("将使用默认配置")
		config = &Config{
			AutoClose: false,
		}
	}

	// 如果游戏路径未设置，自动尝试查找
	if config.GamePath == "" {
		clearScreen()
		drawTitle()
		fmt.Println(colorize("首次运行设置", FgCyan))
		fmt.Println(colorize("\n正在自动查找游戏路径...", FgYellow))

		path := autoDetectGamePath()
		if path == "" {
			fmt.Println(colorize("\n未能自动找到游戏，请手动设置路径", FgYellow))
			fmt.Println(colorize("\n选择设置方式：", FgYellow))
			fmt.Println(colorize("[1] 浏览文件", FgWhite), "- 使用文件选择器选择游戏启动器")
			fmt.Println(colorize("[2] 手动输入", FgWhite), "- 手动输入或粘贴游戏路径")
			fmt.Println()

			switch readInput() {
			case "1":
				path = openFileDialog("选择游戏启动器(delta_force_launcher.exe)")
			case "2":
				path = inputPath("请输入游戏启动器路径（可直接拖拽文件到此处）")
			}

			if path != "" && strings.EqualFold(filepath.Base(path), "delta_force_launcher.exe") {
				config.GamePath = path
				saveConfig(config)
			}
		} else {
			config.GamePath = path
			saveConfig(config)
		}
	}

	for {
		clearScreen()
		drawTitle()
		drawMenu(mainMenu)
		showProcessStatus()

		switch readInput() {
		case "1":
			handleNormalLaunch(config)
		case "2":
			handleAffinityLaunch(config)
		case "3":
			handleSuspendLaunch(config)
		case "4":
			handleSettings(config)
		case "5":
			clearScreen()
			drawTitle()
			drawAbout()
			fmt.Print(colorize("\n按任意键返回主菜单...", FgGreen))
			fmt.Scanln()
		case "0":
			clearScreen()
			drawTitle()
			fmt.Println(colorize("感谢使用！", FgCyan))
			time.Sleep(1 * time.Second)
			return
		}

		if config.AutoClose {
			return
		}

		time.Sleep(2 * time.Second)
	}
}
