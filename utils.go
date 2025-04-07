package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func clearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

func getPossibleGameDirs() []string {
	var dirs []string

	for _, drive := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
		drivePath := string(drive) + ":\\"
		if _, err := os.Stat(drivePath); err == nil {
			dirs = append(dirs,
				drivePath+"Program Files (x86)",
				drivePath+"Program Files",
				drivePath+"Games",
				drivePath+"ACE",
				drivePath+"Delta Force",
			)
		}
	}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs,
			filepath.Join(home, "Documents", "Games"),
			filepath.Join(home, "Documents", "My Games"),
			filepath.Join(home, "Games"),
		)
	}

	return dirs
}

func autoDetectGamePath() string {
	targetFile := "delta_force_launcher.exe"
	searchDirs := getPossibleGameDirs()

	fmt.Println(colorize("正在搜索游戏路径...", FgYellow))

	for _, dir := range searchDirs {
		fmt.Printf("\r%s", colorize("搜索目录: "+dir, FgCyan))

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return filepath.SkipDir
			}

			if info.IsDir() && (strings.Contains(strings.ToLower(info.Name()), "windows") ||
				strings.Contains(strings.ToLower(info.Name()), "system") ||
				strings.Contains(strings.ToLower(info.Name()), "$recycle.bin")) {
				return filepath.SkipDir
			}

			if strings.EqualFold(info.Name(), targetFile) {
				fmt.Printf("\n%s\n", colorize("找到游戏启动器: "+path, FgGreen))
				if confirmAction("是否使用此路径?") {
					return fmt.Errorf("FOUND:%s", path)
				}
			}
			return nil
		})

		if err != nil && strings.HasPrefix(err.Error(), "FOUND:") {
			return strings.TrimPrefix(err.Error(), "FOUND:")
		}
	}

	fmt.Printf("\n%s\n", colorize("未找到游戏启动器", FgYellow))
	return ""
}

func openFileDialog(title string) string {
	var args = []string{
		"powershell",
		"-Command",
		`Add-Type -AssemblyName System.Windows.Forms;
		$f=New-Object System.Windows.Forms.OpenFileDialog;
		$f.Filter='游戏启动器 (*.exe)|*.exe';
		$f.Title='` + title + `';
		$f.ShowDialog();
		$f.FileName`,
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	path := strings.TrimSpace(string(output))
	if path == "" {
		return ""
	}

	return path
}
