package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
	pty *PtyManager
}

type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.pty = NewPtyManager(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	a.pty.Close()
}

func (a *App) StartPty(cols, rows int) error {
	return a.pty.Start(cols, rows)
}

func (a *App) ResizePty(cols, rows int) {
	a.pty.Resize(cols, rows)
}

func (a *App) ReadDir(path string) []FileEntry {
	entries, err := os.ReadDir(path)
	if err != nil {
		return []FileEntry{}
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
		return result[i].Name < result[j].Name
	})
	return result
}

func (a *App) OpenFolderDialog() string {
	dir, _ := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择目录",
	})
	return dir
}
