package main

import (
	"context"
	"sync"

	"github.com/UserExistsError/conpty"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const defaultShell = "powershell.exe"

// PtyManager 管理多个 PTY 会话，每个 Tab 对应一个，以 id 为键。
type PtyManager struct {
	ctx  context.Context
	mu   sync.Mutex
	ptys map[string]*conpty.ConPty
}

func NewPtyManager(ctx context.Context) *PtyManager {
	return &PtyManager{
		ctx:  ctx,
		ptys: make(map[string]*conpty.ConPty),
	}
}

func (m *PtyManager) Start(id string, cols, rows int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.ptys[id]; ok {
		existing.Close()
		delete(m.ptys, id)
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
	m.ptys[id] = pty
	go m.pump(id, pty)
	return nil
}

// pump 持续读取 PTY 输出，通过 pty:data:{id} 事件转发到前端。
func (m *PtyManager) pump(id string, pty *conpty.ConPty) {
	buf := make([]byte, 4096)
	event := "pty:data:" + id
	for {
		n, err := pty.Read(buf)
		if n > 0 {
			wruntime.EventsEmit(m.ctx, event, string(buf[:n]))
		}
		if err != nil {
			return
		}
	}
}

func (m *PtyManager) Write(id string, data string) {
	m.mu.Lock()
	pty, ok := m.ptys[id]
	m.mu.Unlock()
	if ok && pty != nil {
		_, _ = pty.Write([]byte(data))
	}
}

func (m *PtyManager) Resize(id string, cols, rows int) error {
	m.mu.Lock()
	pty, ok := m.ptys[id]
	m.mu.Unlock()
	if !ok || pty == nil {
		return nil
	}
	return pty.Resize(cols, rows)
}

func (m *PtyManager) Close(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pty, ok := m.ptys[id]; ok {
		pty.Close()
		delete(m.ptys, id)
	}
}

func (m *PtyManager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, pty := range m.ptys {
		pty.Close()
	}
	m.ptys = make(map[string]*conpty.ConPty)
}
