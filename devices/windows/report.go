package main

import (
	"encoding/json"
	"math"
	"strings"
)

// 协议规定的字段长度上限（按 rune 计）。
const (
	MaxApp   = 64
	MaxPkg   = 128
	MaxTitle = 120
	MaxModel = 64
	MaxOS    = 64
)

// LockScreenApp 是锁屏或拿不到前台窗口时上报的 app 名称。
const LockScreenApp = "锁屏"

// UnknownApp 是读不到前台进程信息时上报的 app 名称。
const UnknownApp = "未知应用"

// Sample 是一次采样的原始结果，由平台相关代码填充。
type Sample struct {
	Locked      bool   // 锁屏或拿不到前台窗口
	Exe         string // 前台进程 exe 文件名，例如 Code.exe
	Description string // exe 版本信息里的 FileDescription
	Title       string // 前台窗口标题

	Idle int     // 距上次键鼠输入的秒数
	CPU  float64 // CPU 占用百分比
	Mem  float64 // 内存占用百分比

	HasBattery bool
	Battery    int
	Charging   bool
}

// Activity 是经过隐私处理和截断后的前台信息，用于变化检测。
type Activity struct {
	App   string
	Pkg   string
	Title *string // nil 表示 show_title=false，整体省略
}

// Key 返回用于变化比较的字符串。
func (a Activity) Key() string {
	t := "\x00"
	if a.Title != nil {
		t = *a.Title
	}
	return a.App + "\x1f" + a.Pkg + "\x1f" + t
}

// Report 是 POST /api/report 的请求体。
type Report struct {
	Device   string   `json:"device"`
	App      *string  `json:"app,omitempty"`
	Pkg      *string  `json:"pkg,omitempty"`
	Title    *string  `json:"title,omitempty"`
	Model    string   `json:"model,omitempty"`
	OS       string   `json:"os,omitempty"`
	Battery  *int     `json:"battery,omitempty"`
	Charging *bool    `json:"charging,omitempty"`
	CPU      *float64 `json:"cpu,omitempty"`
	Mem      *float64 `json:"mem,omitempty"`
	Idle     *int     `json:"idle,omitempty"`
	Offline  bool     `json:"offline,omitempty"`
}

// JSON 序列化请求体。
func (r Report) JSON() []byte {
	b, _ := json.Marshal(r)
	return b
}

// TruncateRunes 把字符串截断到最多 n 个 rune。
func TruncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		if count == n {
			return s[:i]
		}
		count++
	}
	return s
}

// FriendlyAppName 按优先级选择显示名：app_aliases > FileDescription > 去掉 .exe 的文件名。
func FriendlyAppName(cfg *Config, exe, description string) string {
	if alias, ok := cfg.Alias(exe); ok {
		return alias
	}
	if d := strings.TrimSpace(description); d != "" {
		return d
	}
	name := exe
	if len(name) >= 4 && strings.EqualFold(name[len(name)-4:], ".exe") {
		name = name[:len(name)-4]
	}
	return name
}

// BuildActivity 对采样结果应用锁屏规则、隐私配置与截断。
func BuildActivity(cfg *Config, s Sample) Activity {
	var a Activity
	title := ""
	if s.Locked {
		a.App = LockScreenApp
	} else {
		a.App = TruncateRunes(FriendlyAppName(cfg, s.Exe, s.Description), MaxApp)
		if a.App == "" {
			// 有前台窗口但读不到进程路径（例如受保护进程）时使用占位名。
			a.App = UnknownApp
		}
		a.Pkg = TruncateRunes(s.Exe, MaxPkg)
		if !cfg.TitleHiddenFor(s.Exe) {
			title = TruncateRunes(strings.TrimSpace(s.Title), MaxTitle)
		}
	}
	if cfg.TitleEnabled() {
		a.Title = &title
	}
	return a
}

// Round1 保留 1 位小数并限制在 0–100。
func Round1(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 100 {
		v = 100
	}
	return math.Round(v*10) / 10
}

// BuildReport 构造上报请求体；full 为 true（心跳）时附带 cpu/mem/idle。
func BuildReport(cfg *Config, a Activity, s Sample, model, osName string, full bool) Report {
	app, pkg := a.App, a.Pkg
	r := Report{
		Device: cfg.Device,
		App:    &app,
		Pkg:    &pkg,
		Title:  a.Title,
		Model:  TruncateRunes(model, MaxModel),
		OS:     TruncateRunes(osName, MaxOS),
	}
	if s.HasBattery {
		b := s.Battery
		if b < 0 {
			b = 0
		} else if b > 100 {
			b = 100
		}
		ch := s.Charging
		r.Battery, r.Charging = &b, &ch
	}
	if full {
		cpu, mem, idle := Round1(s.CPU), Round1(s.Mem), s.Idle
		if idle < 0 {
			idle = 0
		}
		r.CPU, r.Mem, r.Idle = &cpu, &mem, &idle
	}
	return r
}

// OfflineReport 构造主动下线请求体：{"device":"pc","offline":true}。
func OfflineReport(device string) Report {
	return Report{Device: device, Offline: true}
}
