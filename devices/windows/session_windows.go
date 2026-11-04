//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// 会话结束与外部停止通知：
//   - 隐藏顶层窗口接收 WM_QUERYENDSESSION / WM_ENDSESSION，GUI 程序在关机、注销时借此发送 offline。
//   - 命名事件 Local\MiaM1kuStatusStop 供 -uninstall 通知运行中的实例优雅退出。

const (
	mutexName = `Local\MiaM1kuStatusAgent`
	stopEvent = `Local\MiaM1kuStatusStop`

	wmQueryEndSession = 0x0011
	wmEndSession      = 0x0016
)

var (
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procAttachConsole    = kernel32.NewProc("AttachConsole")
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     windows.Handle
	hIcon         windows.Handle
	hCursor       windows.Handle
	hbrBackground windows.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       windows.Handle
}

type winMsg struct {
	hwnd     uintptr
	message  uint32
	wParam   uintptr
	lParam   uintptr
	time     uint32
	ptX      int32
	ptY      int32
	lPrivate uint32
}

var endSessionHandler func()

// WatchEndSession 在独立的 OS 线程上创建隐藏窗口；系统结束会话时同步调用 onEnd。
func WatchEndSession(onEnd func()) {
	endSessionHandler = onEnd
	go func() {
		runtime.LockOSThread()
		hInst, _, _ := procGetModuleHandleW.Call(0)
		className, _ := windows.UTF16PtrFromString("MiaM1kuStatusHiddenWindow")
		wndProc := windows.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
			switch msg {
			case wmQueryEndSession:
				return 1
			case wmEndSession:
				if wParam != 0 && endSessionHandler != nil {
					endSessionHandler()
				}
				return 0
			}
			r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
			return r
		})
		wc := wndClassEx{
			cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			lpfnWndProc:   wndProc,
			hInstance:     windows.Handle(hInst),
			lpszClassName: className,
		}
		if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
			logf("注册隐藏窗口类失败：%v", err)
			return
		}
		hwnd, _, err := procCreateWindowExW.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(className)),
			0, 0, 0, 0, 0, 0, 0, hInst, 0)
		if hwnd == 0 {
			logf("创建隐藏窗口失败：%v", err)
			return
		}
		var m winMsg
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
			if int32(r) <= 0 {
				return
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}()
}

// WatchStopEvent 创建命名自动重置事件，被 -uninstall 置位时调用 onStop。
func WatchStopEvent(onStop func()) {
	name, _ := windows.UTF16PtrFromString(stopEvent)
	h, err := windows.CreateEvent(nil, 0, 0, name)
	if err != nil && h == 0 {
		logf("创建停止事件失败：%v", err)
		return
	}
	go func() {
		if ev, _ := windows.WaitForSingleObject(h, windows.INFINITE); ev == windows.WAIT_OBJECT_0 {
			onStop()
		}
	}()
}

// AcquireSingleInstance 创建命名互斥量；已有实例运行时返回 false。返回的句柄需在进程生命周期内保持打开。
func AcquireSingleInstance() (windows.Handle, bool, error) {
	name, _ := windows.UTF16PtrFromString(mutexName)
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			windows.CloseHandle(h)
		}
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return h, true, nil
}

// AttachParentConsole 让 GUI 子系统的 exe 在命令行中运行时也能输出到父控制台。
// stdout 已重定向（句柄有效）时沿用现有句柄。
func AttachParentConsole() {
	if h, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE); err == nil && h != 0 && h != windows.InvalidHandle {
		return
	}
	const attachParentProcess = ^uintptr(0) // ATTACH_PARENT_PROCESS = (DWORD)-1
	if r, _, _ := procAttachConsole.Call(attachParentProcess); r == 0 {
		return
	}
	if f, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout, os.Stderr = f, f
		// 父控制台的提示符已先行输出，补一个换行让后续内容从新行开始。
		fmt.Fprintln(f)
	}
}
