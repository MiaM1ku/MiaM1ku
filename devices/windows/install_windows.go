//go:build windows

package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValueName = "MiaM1kuStatus"
)

// Install 写入开机自启注册表项，结束旧实例后在后台启动自身。
func Install(exePath string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if err := k.SetStringValue(runValueName, `"`+exePath+`"`); err != nil {
		return err
	}
	logf("已写入 HKCU\\%s\\%s = \"%s\"", runKeyPath, runValueName, exePath)

	// 重复安装（例如升级 exe）时先让旧实例退出，确保新版本能拿到单实例锁。
	StopInstances(exePath)

	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	if err := cmd.Start(); err != nil {
		return err
	}
	logf("已在后台启动，PID %d", cmd.Process.Pid)
	return cmd.Process.Release()
}

// Uninstall 删除开机自启注册表项并结束正在运行的实例。
func Uninstall(exePath string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err == nil {
		err = k.DeleteValue(runValueName)
		k.Close()
	}
	switch {
	case err == nil:
		logf("已删除 HKCU\\%s\\%s", runKeyPath, runValueName)
	case errors.Is(err, registry.ErrNotExist):
		logf("注册表中没有 %s，跳过删除", runValueName)
	default:
		return err
	}
	StopInstances(exePath)
	return nil
}

// StopInstances 先通过命名事件请求运行中的实例优雅退出（它会发送 offline），
// 8 秒内仍未退出时按进程名强制结束（跳过当前进程）。
func StopInstances(exePath string) {
	if !instanceRunning() {
		return
	}
	evName, _ := windows.UTF16PtrFromString(stopEvent)
	if h, err := windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, evName); err == nil {
		windows.SetEvent(h)
		windows.CloseHandle(h)
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			if !instanceRunning() {
				logf("运行中的实例已退出")
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	names := map[string]bool{
		"miam1ku-status.exe":                    true,
		"miam1ku-status-console.exe":            true,
		strings.ToLower(filepath.Base(exePath)): true,
	}
	if n := killByName(names); n > 0 {
		logf("已强制结束 %d 个进程", n)
	}
}

// instanceRunning 通过单实例互斥量判断是否有实例在运行。
func instanceRunning() bool {
	name, _ := windows.UTF16PtrFromString(mutexName)
	h, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return !errors.Is(err, windows.ERROR_FILE_NOT_FOUND)
	}
	windows.CloseHandle(h)
	return true
}

func killByName(names map[string]bool) int {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return 0
	}
	defer windows.CloseHandle(snap)
	self := windows.GetCurrentProcessId()
	killed := 0
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	for err = windows.Process32First(snap, &pe); err == nil; err = windows.Process32Next(snap, &pe) {
		if pe.ProcessID == self || !names[strings.ToLower(windows.UTF16ToString(pe.ExeFile[:]))] {
			continue
		}
		h, oerr := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pe.ProcessID)
		if oerr != nil {
			continue
		}
		if windows.TerminateProcess(h, 1) == nil {
			killed++
		}
		windows.CloseHandle(h)
	}
	return killed
}
