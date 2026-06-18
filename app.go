package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 结构体，其导出方法会被 Wails 暴露给前端 JS。
type App struct {
	ctx context.Context
	pty *PtyManager
}

// NewApp 创建 App 实例。
func NewApp() *App {
	return &App{}
}

// startup 在应用启动时调用，保存 ctx 并初始化 PTY、注册输入监听。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.pty = NewPtyManager(ctx)

	// 监听前端键盘输入/粘贴（pty:input），写入 PTY。
	wruntime.EventsOn(ctx, "pty:input", func(optionalData ...interface{}) {
		if len(optionalData) == 0 {
			return
		}
		if s, ok := optionalData[0].(string); ok {
			a.pty.Write(s)
		}
	})

	// Wails 的 WebView2Loader 不能稳定触发 window-show 事件，
	// 这里硬编码延迟 2 秒后再显示窗口；修改/移除前请先测试窗口仍能正常显示。
	go func() {
		time.Sleep(2 * time.Second)
		wruntime.WindowShow(ctx)
	}()
}

// shutdown 在应用退出时调用，释放 PTY 进程。
func (a *App) shutdown(ctx context.Context) {
	if a.pty != nil {
		a.pty.Close()
	}
}

// StartPty (重新)启动 PowerShell 会话。
func (a *App) StartPty(cols, rows int) error {
	return a.pty.Start(cols, rows)
}

// ResizePty 同步 PTY 尺寸。
func (a *App) ResizePty(cols, rows int) error {
	return a.pty.Resize(cols, rows)
}

// FileEntry 目录项。
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// ReadDir 读取目录，返回排序后的列表：目录在前，再按名称不区分大小写升序。
func (a *App) ReadDir(path string) ([]FileEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, FileEntry{
			Name:  e.Name(),
			Path:  filepath.Join(path, e.Name()),
			IsDir: e.IsDir(),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result, nil
}

// OpenFolderDialog 弹出选择目录对话框，返回所选目录绝对路径（取消时为空字符串）。
func (a *App) OpenFolderDialog() (string, error) {
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择目录",
	})
}

// HomeDir 返回用户主目录，供前端首次加载（无 lastDir 时）作为默认目录。
func (a *App) HomeDir() (string, error) {
	return os.UserHomeDir()
}
