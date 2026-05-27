package tunnel

import (
	"errors"
	"sync"
	"time"
)

var (
	activeTunnel *Tunnel
	mu           sync.Mutex
	refCount     int
)

type Manager struct {
	mu       sync.Mutex
	tunnel   *Tunnel
	refCount int
	lastUsed time.Time
}

var manager = &Manager{}

func Acquire() (*Tunnel, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.refCount++
	manager.lastUsed = time.Now()

	// Reuse existing tunnel
	if manager.tunnel != nil {
		return manager.tunnel, nil
	}

	t, err := Start()
	if err != nil {
		manager.refCount--
		return nil, err
	}

	manager.tunnel = t

	return t, nil
}
func Release() error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.refCount > 0 {
		manager.refCount--
	}

	// Still active shares
	if manager.refCount > 0 {
		return nil
	}

	if manager.tunnel == nil {
		return nil
	}

	if manager.tunnel.Process == nil {
		manager.tunnel = nil
		return nil
	}

	if manager.tunnel.Process.Process == nil {
		manager.tunnel = nil
		return nil
	}

	err := manager.tunnel.Process.Process.Kill()

	manager.tunnel.Process.Wait()

	manager.tunnel = nil

	return err
}

func URL() (string, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	if manager.tunnel == nil {
		return "", errors.New("tunnel not running")
	}

	return manager.tunnel.URL, nil
}

func IsRunning() bool {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	return manager.tunnel != nil
}

func RefCount() int {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	return manager.refCount
}
