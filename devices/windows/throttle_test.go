package main

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

func at(sec float64) time.Time {
	return t0.Add(time.Duration(sec * float64(time.Second)))
}

// send 模拟一次成功发送。
func send(th *Throttle, now time.Time, key string, a Action) {
	th.Attempted(now)
	th.Succeeded(now, key, a)
}

func TestThrottleFirstReportIsHeartbeat(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	if d := th.Decide(at(0), "a"); d.Action != ActionHeartbeat {
		t.Fatalf("first decision = %v, want heartbeat", d.Action)
	}
}

func TestThrottleNoChangeNoReport(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	send(th, at(0), "a", ActionHeartbeat)
	for s := 2.0; s < 60; s += 2 {
		if d := th.Decide(at(s), "a"); d.Action != ActionNone {
			t.Fatalf("t=%v decision = %v, want none", s, d.Action)
		}
	}
}

func TestThrottleHeartbeatEvery60s(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	send(th, at(0), "a", ActionHeartbeat)
	if d := th.Decide(at(60), "a"); d.Action != ActionHeartbeat {
		t.Fatalf("t=60 decision = %v, want heartbeat", d.Action)
	}
	send(th, at(60), "a", ActionHeartbeat)
	if d := th.Decide(at(118), "a"); d.Action != ActionNone {
		t.Fatalf("t=118 decision = %v, want none", d.Action)
	}
	if d := th.Decide(at(120), "a"); d.Action != ActionHeartbeat {
		t.Fatalf("t=120 decision = %v, want heartbeat", d.Action)
	}
}

func TestThrottleChangeImmediateAfterMinInterval(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	send(th, at(0), "a", ActionHeartbeat)
	if d := th.Decide(at(12), "b"); d.Action != ActionChange {
		t.Fatalf("t=12 decision = %v, want change", d.Action)
	}
}

func TestThrottleChangeWithinMinIntervalWaits(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	send(th, at(0), "a", ActionHeartbeat)
	d := th.Decide(at(4), "b")
	if d.Action != ActionWait {
		t.Fatalf("t=4 decision = %v, want wait", d.Action)
	}
	if !d.WaitUntil.Equal(at(10)) {
		t.Fatalf("WaitUntil = %v, want %v", d.WaitUntil, at(10))
	}
	// 等待期间前台又变成 c：补发时使用最新状态。
	if d := th.Decide(at(6), "c"); d.Action != ActionWait {
		t.Fatalf("t=6 decision = %v, want wait", d.Action)
	}
	if d := th.Decide(at(10), "c"); d.Action != ActionChange {
		t.Fatalf("t=10 decision = %v, want change", d.Action)
	}
	send(th, at(10), "c", ActionChange)
	if d := th.Decide(at(12), "c"); d.Action != ActionNone {
		t.Fatalf("t=12 decision = %v, want none", d.Action)
	}
}

func TestThrottleChangeRevertedBeforeIntervalNoReport(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	send(th, at(0), "a", ActionHeartbeat)
	if d := th.Decide(at(4), "b"); d.Action != ActionWait {
		t.Fatalf("t=4 decision = %v, want wait", d.Action)
	}
	// 满 10 秒时又切回 a，与上次上报一致，无需补发。
	if d := th.Decide(at(10), "a"); d.Action != ActionNone {
		t.Fatalf("t=10 decision = %v, want none", d.Action)
	}
}

func TestThrottleChangeDoesNotResetHeartbeat(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	send(th, at(0), "a", ActionHeartbeat)
	send(th, at(30), "b", ActionChange)
	// 心跳按上次心跳计时，保证 cpu/mem/idle 至少每 60 秒刷新一次。
	if d := th.Decide(at(60), "b"); d.Action != ActionHeartbeat {
		t.Fatalf("t=60 decision = %v, want heartbeat", d.Action)
	}
}

func TestThrottleHeartbeatRespectsMinInterval(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	send(th, at(0), "a", ActionHeartbeat)
	send(th, at(55), "b", ActionChange)
	d := th.Decide(at(60), "b")
	if d.Action != ActionWait || !d.WaitUntil.Equal(at(65)) {
		t.Fatalf("t=60 decision = %v until %v, want wait until %v", d.Action, d.WaitUntil, at(65))
	}
	if d := th.Decide(at(65), "b"); d.Action != ActionHeartbeat {
		t.Fatalf("t=65 decision = %v, want heartbeat", d.Action)
	}
}

func TestThrottleFailureRetriesAfterMinInterval(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	send(th, at(0), "a", ActionHeartbeat)
	// t=20 前台变化，发送失败。
	if d := th.Decide(at(20), "b"); d.Action != ActionChange {
		t.Fatalf("t=20 decision = %v, want change", d.Action)
	}
	th.Attempted(at(20))
	if d := th.Decide(at(22), "b"); d.Action != ActionWait {
		t.Fatalf("t=22 decision = %v, want wait", d.Action)
	}
	if d := th.Decide(at(30), "b"); d.Action != ActionChange {
		t.Fatalf("t=30 decision = %v, want change (retry)", d.Action)
	}
}

func TestThrottleFirstReportFailureRetries(t *testing.T) {
	th := NewThrottle(10*time.Second, 60*time.Second)
	th.Attempted(at(0))
	if d := th.Decide(at(2), "a"); d.Action != ActionWait {
		t.Fatalf("t=2 decision = %v, want wait", d.Action)
	}
	if d := th.Decide(at(10), "a"); d.Action != ActionHeartbeat {
		t.Fatalf("t=10 decision = %v, want heartbeat", d.Action)
	}
}
