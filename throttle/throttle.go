// Package throttle 提供轻量节流执行器：以 key 首次提交的时刻起计时，
// 等待窗口内的重复提交会被丢弃；窗口结束回调执行一次后，方可接收下一次提交。
package throttle

import (
	"sync"
	"time"
)

// Throttler 按 key 节流执行回调。适用于低频场景（如延迟保存），不做队列/取消等复杂机制。
type Throttler struct {
	mu      sync.Mutex
	pending map[string]struct{}
}

// New 创建一个空的节流执行器
func New() *Throttler {
	return &Throttler{pending: make(map[string]struct{})}
}

// Submit 以 key 提交一次执行：等待 delay 后执行一次 cb。
//   - 同 key 在等待窗口内再次提交会被直接丢弃（以第一次提交为准）
//   - 回调执行完毕后该 key 恢复可提交状态，再次提交重新计时
//   - key 为空、delay <= 0 或 cb 为 nil 时直接忽略
func (t *Throttler) Submit(key string, delay time.Duration, cb func()) {
	if key == "" || delay <= 0 || cb == nil {
		return
	}

	t.mu.Lock()
	if _, ok := t.pending[key]; ok {
		t.mu.Unlock()
		return // 窗口内重复提交：丢弃
	}
	t.pending[key] = struct{}{}
	t.mu.Unlock()

	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		<-timer.C
		// 先移除再执行回调，使冷却一结束就能重新提交
		t.mu.Lock()
		delete(t.pending, key)
		t.mu.Unlock()
		cb()
	}()
}
