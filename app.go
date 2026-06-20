package main

import (
	"context"
	"encoding/base64"
	"io"
	"mime"
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

	// 监听前端键盘输入/粘贴（pty:input），参数为 (tabId, data)。
	wruntime.EventsOn(ctx, "pty:input", func(args ...interface{}) {
		if len(args) < 2 {
			return
		}
		id, ok1 := args[0].(string)
		data, ok2 := args[1].(string)
		if ok1 && ok2 {
			a.pty.Write(id, data)
		}
	})

	// Wails 的 WebView2Loader 不能稳定触发 window-show 事件，
	// 这里硬编码延迟 2 秒后再显示窗口；修改/移除前请先测试窗口仍能正常显示。
	go func() {
		time.Sleep(2 * time.Second)
		wruntime.WindowShow(ctx)
	}()
}

// shutdown 在应用退出时调用，释放所有 PTY 进程。
func (a *App) shutdown(ctx context.Context) {
	if a.pty != nil {
		a.pty.CloseAll()
	}
}

// StartPty (重新)启动指定 Tab 的 PowerShell 会话。
func (a *App) StartPty(id string, cols, rows int) error {
	return a.pty.Start(id, cols, rows)
}

// ResizePty 同步指定 Tab 的 PTY 尺寸。
func (a *App) ResizePty(id string, cols, rows int) error {
	return a.pty.Resize(id, cols, rows)
}

// ClosePty 关闭指定 Tab 的 PTY 进程（Tab 关闭时调用）。
func (a *App) ClosePty(id string) {
	a.pty.Close(id)
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

// ReadFileText 读取文本文件，最多返回 maxBytes 字节，供代码/文本预览。
func (a *App) ReadFileText(path string, maxBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxBytes))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ReadFileBase64 读取文件并返回 data URL，供图片预览。
func (a *App) ReadFileBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(path))
	mt := mime.TypeByExtension(ext)
	if mt == "" {
		mt = "application/octet-stream"
	}
	return "data:" + mt + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// SaveExportFile 弹出保存对话框并把 content 写入所选路径，返回路径（取消时空串）。
func (a *App) SaveExportFile(defaultName, content string) (string, error) {
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "导出",
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}
