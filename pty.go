package main

import (
	"context"
	"sync"

	"github.com/UserExistsError/conpty"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type PtyManager struct {
	ctx  context.Context
	cpty *conpty.ConPty
	mu   sync.Mutex
}

func NewPtyManager(ctx context.Context) *PtyManager {
	pm := &PtyManager{ctx: ctx}
	runtime.EventsOn(ctx, "pty:input", func(data ...interface{}) {
		if len(data) == 0 {
			return
		}
		s, ok := data[0].(string)
		if !ok {
			return
		}
		pm.mu.Lock()
		defer pm.mu.Unlock()
		if pm.cpty != nil {
			pm.cpty.Write([]byte(s))
		}
	})
	return pm
}

func (pm *PtyManager) Start(cols, rows int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cpty != nil {
		pm.cpty.Close()
		pm.cpty = nil
	}

	cpty, err := conpty.Start("powershell.exe", conpty.ConPtyDimensions(cols, rows))
	if err != nil {
		return err
	}
	pm.cpty = cpty

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := cpty.Read(buf)
			if n > 0 {
				runtime.EventsEmit(pm.ctx, "pty:data", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	return nil
}

func (pm *PtyManager) Resize(cols, rows int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.cpty != nil {
		pm.cpty.Resize(cols, rows)
	}
}

func (pm *PtyManager) Close() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.cpty != nil {
		pm.cpty.Close()
		pm.cpty = nil
	}
}
