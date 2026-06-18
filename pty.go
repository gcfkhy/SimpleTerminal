package main

import (
	"context"
	"sync"

	"github.com/UserExistsError/conpty"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// defaultShell 当前为单会话 PowerShell；v2.0 多 Tab 时改为按 tabID 管理的进程池。
const defaultShell = "powershell.exe"

// PtyManager 用 Mutex 保护、持有单个 ConPty 会话。
type PtyManager struct {
	ctx context.Context
	mu  sync.Mutex
	pty *conpty.ConPty
}

// NewPtyManager 创建管理器，ctx 用于向前端发送 pty:data 事件。
func NewPtyManager(ctx context.Context) *PtyManager {
	return &PtyManager{ctx: ctx}
}

// Start (重新)启动 powershell.exe，并用 goroutine 持续抽取 PTY 输出。
func (m *PtyManager) Start(cols, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 已有会话则先关闭，旧的 pump goroutine 会在 Read 出错后自行退出。
	if m.pty != nil {
		m.pty.Close()
		m.pty = nil
	}

	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	pty, err := conpty.Start(defaultShell, conpty.ConPtyDimensions(cols, rows))
	if err != nil {
		return err
	}
	m.pty = pty

	go m.pump(pty)
	return nil
}

// pump 持续读取 PTY 输出并转发到前端（pty:data）。pty 由参数传入，
// 避免重启会话后误用新句柄。
func (m *PtyManager) pump(pty *conpty.ConPty) {
	buf := make([]byte, 4096)
	for {
		n, err := pty.Read(buf)
		if n > 0 {
			wruntime.EventsEmit(m.ctx, "pty:data", string(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

// Write 把键盘输入/粘贴写入 PTY。
func (m *PtyManager) Write(data string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pty != nil {
		_, _ = m.pty.Write([]byte(data))
	}
}

// Resize 同步 PTY 尺寸（分隔线拖动/窗口缩放后必须调用，否则输出换行错乱）。
func (m *PtyManager) Resize(cols, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pty == nil {
		return nil
	}
	return m.pty.Resize(cols, rows)
}

// Close 释放 PTY，进程退出时调用。
func (m *PtyManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pty != nil {
		m.pty.Close()
		m.pty = nil
	}
}
