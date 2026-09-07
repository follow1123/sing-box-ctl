package throttle

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 窗口内多次提交只执行一次
func TestDuplicateWithinWindowDropped(t *testing.T) {
	th := New()
	var n atomic.Int32
	cb := func() { n.Add(1) }

	th.Submit("k", 80*time.Millisecond, cb)
	th.Submit("k", 80*time.Millisecond, cb) // 窗口内：丢弃
	th.Submit("k", 80*time.Millisecond, cb) // 窗口内：丢弃

	require.Eventually(t, func() bool { return n.Load() == 1 }, time.Second, 5*time.Millisecond)
}

// 回调执行后，再次提交可以重新计时并再执行一次
func TestAfterExecutionCanSubmitAgain(t *testing.T) {
	th := New()
	var n atomic.Int32
	cb := func() { n.Add(1) }

	th.Submit("k", 80*time.Millisecond, cb)
	require.Eventually(t, func() bool { return n.Load() == 1 }, time.Second, 5*time.Millisecond)

	// 执行完后的提交不再被丢弃
	th.Submit("k", 80*time.Millisecond, cb)
	require.Eventually(t, func() bool { return n.Load() == 2 }, time.Second, 5*time.Millisecond)
}

// 不同 key 互不影响
func TestDifferentKeysIndependent(t *testing.T) {
	th := New()
	var a, b atomic.Int32

	th.Submit("key-a", 60*time.Millisecond, func() { a.Add(1) })
	th.Submit("key-b", 60*time.Millisecond, func() { b.Add(1) })
	// key-b 在窗口内重复提交，key-a 不受影响
	th.Submit("key-b", 60*time.Millisecond, func() { b.Add(1) })

	require.Eventually(t, func() bool { return a.Load() == 1 && b.Load() == 1 }, time.Second, 5*time.Millisecond)
}

// 非法参数直接忽略，不影响其它任务
func TestInvalidArgsIgnored(t *testing.T) {
	th := New()
	var n atomic.Int32

	th.Submit("", time.Second, func() { n.Add(1) })   // 空 key
	th.Submit("k", 0, func() { n.Add(1) })            // 零延时
	th.Submit("k", time.Second, nil)                  // 空回调
	th.Submit("k", -time.Second, func() { n.Add(1) }) // 负延时

	th.Submit("k", 40*time.Millisecond, func() { n.Add(1) })
	require.Eventually(t, func() bool { return n.Load() == 1 }, time.Second, 5*time.Millisecond)
}
