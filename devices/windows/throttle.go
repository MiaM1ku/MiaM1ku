package main

import "time"

// Action 是节流器对当前采样给出的决定。
type Action int

const (
	ActionNone      Action = iota // 本周期无需上报
	ActionChange                  // 前台变化，立即上报（不含 cpu/mem/idle）
	ActionHeartbeat               // 心跳或首次上报，附带 cpu/mem/idle
	ActionWait                    // 有待发内容，但距上次发送未满最小间隔；在 WaitUntil 时再决策
)

func (a Action) String() string {
	switch a {
	case ActionChange:
		return "change"
	case ActionHeartbeat:
		return "heartbeat"
	case ActionWait:
		return "wait"
	default:
		return "none"
	}
}

// Decision 是 Throttle.Decide 的返回值。
type Decision struct {
	Action    Action
	WaitUntil time.Time // 仅 ActionWait 时有效
}

// Throttle 是上报节奏状态机，与具体时钟和网络解耦，便于单元测试。
//
// 规则：
//   - 首次上报按心跳处理（附带全部指标）。
//   - 前台 key 与上次成功上报的 key 不同时立即上报；距上次发送尝试未满 MinInterval 时返回
//     ActionWait，调用方在 WaitUntil 时用最新采样再次决策。
//   - 距上次成功心跳满 Heartbeat 时发送心跳。
//   - 发送失败计入“上次尝试时间”，下一个满足最小间隔的周期自动重试。
type Throttle struct {
	MinInterval time.Duration
	Heartbeat   time.Duration

	attempted     bool
	lastAttempt   time.Time
	sent          bool
	sentKey       string
	lastHeartbeat time.Time
}

// NewThrottle 创建节流器。
func NewThrottle(minInterval, heartbeat time.Duration) *Throttle {
	return &Throttle{MinInterval: minInterval, Heartbeat: heartbeat}
}

// Decide 根据当前时间与前台 key 给出决定，不修改状态。
func (t *Throttle) Decide(now time.Time, key string) Decision {
	want := ActionNone
	switch {
	case !t.sent:
		want = ActionHeartbeat
	case now.Sub(t.lastHeartbeat) >= t.Heartbeat:
		want = ActionHeartbeat
	case key != t.sentKey:
		want = ActionChange
	}
	if want == ActionNone {
		return Decision{Action: ActionNone}
	}
	if t.attempted {
		if next := t.lastAttempt.Add(t.MinInterval); now.Before(next) {
			return Decision{Action: ActionWait, WaitUntil: next}
		}
	}
	return Decision{Action: want}
}

// Attempted 记录一次发送尝试（无论成败）。
func (t *Throttle) Attempted(now time.Time) {
	t.attempted = true
	t.lastAttempt = now
}

// Succeeded 记录一次成功上报。
func (t *Throttle) Succeeded(now time.Time, key string, action Action) {
	t.sent = true
	t.sentKey = key
	if action == ActionHeartbeat {
		t.lastHeartbeat = now
	}
}
