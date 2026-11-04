package main

import (
	"strings"
	"testing"
)

func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"", 5, ""},
		{"abc", 5, "abc"},
		{"abcdef", 3, "abc"},
		{"abc", 3, "abc"},
		{"哔哩哔哩", 2, "哔哩"},
		{"a哔b", 2, "a哔"},
		{"😀😀😀", 1, "😀"},
		{"abc", 0, ""},
	}
	for _, c := range cases {
		if got := TruncateRunes(c.in, c.n); got != c.want {
			t.Errorf("TruncateRunes(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
		}
	}
}

func testConfig(t *testing.T, js string) *Config {
	t.Helper()
	c, err := ParseConfig([]byte(js))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestBuildActivityTruncatesByRune(t *testing.T) {
	cfg := testConfig(t, ConfigTemplate)
	s := Sample{
		Exe:         strings.Repeat("长", 200) + ".exe",
		Description: strings.Repeat("名", 100),
		Title:       strings.Repeat("标", 300),
	}
	a := BuildActivity(cfg, s)
	if n := len([]rune(a.App)); n != MaxApp {
		t.Errorf("app runes = %d, want %d", n, MaxApp)
	}
	if n := len([]rune(a.Pkg)); n != MaxPkg {
		t.Errorf("pkg runes = %d, want %d", n, MaxPkg)
	}
	if a.Title == nil || len([]rune(*a.Title)) != MaxTitle {
		t.Errorf("title not truncated to %d runes", MaxTitle)
	}
}

func TestFriendlyAppNamePriority(t *testing.T) {
	cfg := testConfig(t, ConfigTemplate)
	if got := FriendlyAppName(cfg, "code.EXE", "Visual Studio Code Insiders"); got != "Visual Studio Code" {
		t.Errorf("alias (case-insensitive) = %q", got)
	}
	if got := FriendlyAppName(cfg, "chrome.exe", "Google Chrome"); got != "Google Chrome" {
		t.Errorf("description = %q", got)
	}
	if got := FriendlyAppName(cfg, "foo.EXE", "  "); got != "foo" {
		t.Errorf("fallback = %q", got)
	}
}

func TestBuildActivityPrivacy(t *testing.T) {
	cfg := testConfig(t, ConfigTemplate)
	a := BuildActivity(cfg, Sample{Exe: "keepassxc.exe", Description: "KeePassXC", Title: "secret.kdbx"})
	if a.Title == nil || *a.Title != "" {
		t.Errorf("hidden app title = %v, want empty string", a.Title)
	}
	cfg = testConfig(t, `{"token":"x","show_title":false}`)
	a = BuildActivity(cfg, Sample{Exe: "Code.exe", Title: "main.go"})
	if a.Title != nil {
		t.Errorf("show_title=false should omit title")
	}
	if strings.Contains(string(BuildReport(cfg, a, Sample{}, "", "", false).JSON()), "title") {
		t.Errorf("JSON should omit title when show_title=false")
	}
}

func TestBuildActivityLocked(t *testing.T) {
	cfg := testConfig(t, ConfigTemplate)
	a := BuildActivity(cfg, Sample{Locked: true, Exe: "LockApp.exe", Title: "Windows 默认锁屏界面"})
	if a.App != LockScreenApp || a.Pkg != "" || a.Title == nil || *a.Title != "" {
		t.Errorf("locked activity = %+v", a)
	}
}

func TestBuildReportJSON(t *testing.T) {
	cfg := testConfig(t, ConfigTemplate)
	s := Sample{Exe: "Code.exe", Title: "index.html", CPU: 12.345, Mem: 48.25, Idle: 3}
	a := BuildActivity(cfg, s)
	change := string(BuildReport(cfg, a, s, "DESKTOP-1", "Windows 11 23H2", false).JSON())
	want := `{"device":"pc","app":"Visual Studio Code","pkg":"Code.exe","title":"index.html","model":"DESKTOP-1","os":"Windows 11 23H2"}`
	if change != want {
		t.Errorf("change report = %s\nwant %s", change, want)
	}
	s.HasBattery, s.Battery, s.Charging = true, 85, false
	full := string(BuildReport(cfg, a, s, "DESKTOP-1", "Windows 11 23H2", true).JSON())
	for _, frag := range []string{`"battery":85`, `"charging":false`, `"cpu":12.3`, `"mem":48.3`, `"idle":3`} {
		if !strings.Contains(full, frag) {
			t.Errorf("heartbeat report %s missing %s", full, frag)
		}
	}
	if got := string(OfflineReport("pc").JSON()); got != `{"device":"pc","offline":true}` {
		t.Errorf("offline report = %s", got)
	}
}

func TestParseConfigDefaults(t *testing.T) {
	cfg := testConfig(t, `{"token":"t"}`)
	if cfg.Device != "pc" || !cfg.TitleEnabled() || cfg.HeartbeatSeconds != 60 || cfg.MinIntervalSeconds != 10 {
		t.Errorf("defaults = %+v", cfg)
	}
	if cfg.Endpoint != "https://miam1ku.cn/api/report" {
		t.Errorf("endpoint = %q", cfg.Endpoint)
	}
}
