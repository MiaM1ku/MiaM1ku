//go:build windows

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

const (
	logFileName    = "miam1ku-status.log"
	sampleInterval = 2 * time.Second
	offlineTimeout = 4 * time.Second
)

func logf(format string, a ...any) { log.Printf(format, a...) }

func main() {
	install := flag.Bool("install", false, "写入开机自启（HKCU Run）并在后台启动")
	uninstall := flag.Bool("uninstall", false, "删除开机自启并结束正在运行的实例")
	once := flag.Bool("once", false, "采样一次并把 JSON 打印到 stdout（调试用，不发送）")
	console := flag.Bool("console", false, "前台运行，日志输出到 stdout")
	flag.Parse()

	exePath, err := os.Executable()
	if err == nil {
		exePath, err = filepath.Abs(exePath)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "无法获取 exe 路径：", err)
		os.Exit(1)
	}
	dir := filepath.Dir(exePath)

	interactive := *install || *uninstall || *once
	if interactive || *console {
		AttachParentConsole()
	}
	closeLog := setupLogging(dir, *console, *once, interactive)
	defer closeLog()

	code := 0
	switch {
	case *once:
		code = runOnce(dir)
	case *install:
		code = runInstall(dir, exePath)
	case *uninstall:
		if err := Uninstall(exePath); err != nil {
			logf("卸载失败：%v", err)
			code = 1
		} else {
			logf("卸载完成")
		}
	default:
		code = runAgent(dir)
	}
	closeLog()
	os.Exit(code)
}

// setupLogging 配置日志输出：-console 写 stdout；-once 写 stderr；-install/-uninstall 同时写控制台与日志文件；默认写日志文件。
func setupLogging(dir string, console, once, interactive bool) func() {
	log.SetFlags(log.LstdFlags)
	switch {
	case console:
		log.SetOutput(os.Stdout)
		return func() {}
	case once:
		log.SetOutput(os.Stderr)
		return func() {}
	}
	f, err := OpenRotatingFile(filepath.Join(dir, logFileName), MaxLogSize)
	if err != nil {
		log.SetOutput(os.Stderr)
		logf("无法打开日志文件：%v", err)
		return func() {}
	}
	if interactive {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	} else {
		log.SetOutput(f)
	}
	var closeOnce sync.Once
	return func() { closeOnce.Do(func() { f.Close() }) }
}

func configPath(dir string) string { return filepath.Join(dir, ConfigFileName) }

// loadConfigOrExplain 读取配置；缺失或 token 为空时记录提示并返回 nil。
func loadConfigOrExplain(dir string) *Config {
	path := configPath(dir)
	cfg, err := LoadConfig(path)
	switch {
	case err == nil:
		return cfg
	case errors.Is(err, ErrConfigCreated):
		logf("未找到配置文件，已生成模板 %s；请填写 token 后重新运行", path)
	case errors.Is(err, ErrTokenEmpty):
		logf("配置文件 %s 中 token 为空；请填写 token 后重新运行", path)
	default:
		logf("读取配置失败：%v", err)
	}
	return nil
}

func runOnce(dir string) int {
	var cfg *Config
	data, err := os.ReadFile(configPath(dir))
	if err != nil {
		data = []byte(ConfigTemplate)
	}
	if cfg, err = ParseConfig(data); err != nil {
		logf("%v", err)
		return 1
	}
	s := NewSampler().Sample()
	r := BuildReport(cfg, BuildActivity(cfg, s), s, DeviceModel(cfg), OSVersion(), true)
	fmt.Fprintln(os.Stdout, string(r.JSON()))
	return 0
}

func runInstall(dir, exePath string) int {
	if loadConfigOrExplain(dir) == nil {
		return 1
	}
	if err := Install(exePath); err != nil {
		logf("安装失败：%v", err)
		return 1
	}
	logf("安装完成：登录 Windows 时自动启动")
	return 0
}

func runAgent(dir string) int {
	mutex, ok, err := AcquireSingleInstance()
	if err != nil {
		logf("创建单实例互斥量失败：%v", err)
		return 1
	}
	if !ok {
		logf("已有实例在运行，本进程退出")
		return 0
	}
	defer windows.CloseHandle(mutex)

	cfg := loadConfigOrExplain(dir)
	if cfg == nil {
		return 1
	}
	sender := NewSender(cfg)
	model, osName := DeviceModel(cfg), OSVersion()
	logf("启动：endpoint=%s device=%s model=%q os=%q", cfg.Endpoint, cfg.Device, model, osName)

	var stopping atomic.Bool
	var offlineOnce sync.Once
	sendOffline := func(reason string) {
		offlineOnce.Do(func() {
			stopping.Store(true)
			ctx, cancel := context.WithTimeout(context.Background(), offlineTimeout)
			defer cancel()
			if err := sender.Send(ctx, OfflineReport(cfg.Device)); err != nil {
				logf("%s，发送 offline 失败：%v", reason, err)
			} else {
				logf("%s，已发送 offline", reason)
			}
		})
	}

	stopCh := make(chan string, 4)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() { stopCh <- fmt.Sprintf("收到信号 %v", <-sigCh) }()
	WatchStopEvent(func() { stopCh <- "收到 -uninstall/-install 的停止请求" })
	WatchEndSession(func() { sendOffline("系统结束会话") })

	sampler := NewSampler()
	th := NewThrottle(cfg.MinInterval(), cfg.Heartbeat())
	failing := true // 首次成功上报也写一条日志
	var waitC <-chan time.Time

	step := func() {
		if stopping.Load() {
			return
		}
		s := sampler.Sample()
		a := BuildActivity(cfg, s)
		key, now := a.Key(), time.Now()
		d := th.Decide(now, key)
		switch d.Action {
		case ActionWait:
			if waitC == nil {
				waitC = time.After(time.Until(d.WaitUntil))
			}
			return
		case ActionNone:
			return
		}
		th.Attempted(now)
		// 服务端每次上报整体覆盖记录，所以变化上报同样附带 cpu/mem/idle
		r := BuildReport(cfg, a, s, model, osName, true)
		ctx, cancel := context.WithTimeout(context.Background(), HTTPTimeout)
		err := sender.Send(ctx, r)
		cancel()
		if err != nil {
			logf("上报失败（%v）：%v；%d 秒后重试", d.Action, err, cfg.MinIntervalSeconds)
			failing = true
			return
		}
		th.Succeeded(now, key, d.Action)
		if failing || d.Action == ActionChange {
			logf("上报成功（%v）：app=%q pkg=%q", d.Action, a.App, a.Pkg)
		}
		failing = false
	}

	ticker := time.NewTicker(sampleInterval)
	defer ticker.Stop()
	step()
	for {
		select {
		case <-ticker.C:
			step()
		case <-waitC:
			waitC = nil
			step()
		case reason := <-stopCh:
			sendOffline(reason)
			logf("退出")
			return 0
		}
	}
}
