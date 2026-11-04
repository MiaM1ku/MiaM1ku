//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procGetWindowTextW       = user32.NewProc("GetWindowTextW")
	procGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	procGetLastInputInfo     = user32.NewProc("GetLastInputInfo")
	procGetTickCount         = kernel32.NewProc("GetTickCount")
	procGetSystemTimes       = kernel32.NewProc("GetSystemTimes")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetSystemPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
)

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

type systemPowerStatus struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

// Sampler 负责采集前台窗口与系统指标，保存 CPU 采样基线与友好名称缓存。
type Sampler struct {
	descCache map[string]string // exe 绝对路径（小写）→ FileDescription

	cpuValid               bool
	prevIdle, prevKernUser uint64
}

// NewSampler 创建采样器，并记录一次 CPU 基线，短暂等待后首次采样即可得到有效占用率。
func NewSampler() *Sampler {
	s := &Sampler{descCache: map[string]string{}}
	s.cpuPercent()
	time.Sleep(500 * time.Millisecond)
	return s
}

// Sample 采集一次完整状态。
func (s *Sampler) Sample() Sample {
	var out Sample
	s.foreground(&out)
	out.Idle = idleSeconds()
	out.CPU = s.cpuPercent()
	out.Mem = memPercent()
	out.HasBattery, out.Battery, out.Charging = battery()
	return out
}

func (s *Sampler) foreground(out *Sample) {
	hwnd := windows.GetForegroundWindow()
	if hwnd == 0 {
		out.Locked = true
		return
	}
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err != nil || pid == 0 {
		out.Locked = true
		return
	}
	path := processImagePath(pid)
	exe := filepath.Base(path)
	if path != "" && strings.EqualFold(exe, "ApplicationFrameHost.exe") {
		// UWP 应用的顶层窗口属于 ApplicationFrameHost，真实进程是其子窗口的所有者。
		if childPid := uwpChildPid(hwnd, pid); childPid != 0 {
			if p := processImagePath(childPid); p != "" {
				path, exe = p, filepath.Base(p)
			}
		}
	}
	if strings.EqualFold(exe, "LockApp.exe") {
		out.Locked = true
		return
	}
	out.Title = windowText(hwnd)
	if path == "" {
		return
	}
	out.Exe = exe
	out.Description = s.description(path)
}

func windowText(hwnd windows.HWND) string {
	n, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if n == 0 {
		return ""
	}
	buf := make([]uint16, n+1)
	r, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return windows.UTF16ToString(buf[:r])
}

func processImagePath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:size])
}

// EnumChildWindows 回调通过包级变量传递结果；采样只在主循环单线程中进行，用互斥锁保证安全。
var (
	uwpMu       sync.Mutex
	uwpHostPid  uint32
	uwpFoundPid uint32
	uwpCallback = windows.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var pid uint32
		windows.GetWindowThreadProcessId(windows.HWND(hwnd), &pid)
		if pid != 0 && pid != uwpHostPid {
			uwpFoundPid = pid
			return 0
		}
		return 1
	})
)

func uwpChildPid(hwnd windows.HWND, hostPid uint32) uint32 {
	uwpMu.Lock()
	defer uwpMu.Unlock()
	uwpHostPid, uwpFoundPid = hostPid, 0
	windows.EnumChildWindows(hwnd, uwpCallback, nil)
	return uwpFoundPid
}

// description 返回 exe 的 FileDescription，按路径缓存（包括空结果）。
func (s *Sampler) description(path string) string {
	key := strings.ToLower(path)
	if d, ok := s.descCache[key]; ok {
		return d
	}
	d := fileDescription(path)
	if len(s.descCache) > 512 {
		s.descCache = map[string]string{}
	}
	s.descCache[key] = d
	return d
}

func fileDescription(path string) string {
	var zero windows.Handle
	size, err := windows.GetFileVersionInfoSize(path, &zero)
	if err != nil || size == 0 {
		return ""
	}
	buf := make([]byte, size)
	if err := windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buf[0])); err != nil {
		return ""
	}
	block := unsafe.Pointer(&buf[0])

	var subBlocks []string
	var transPtr uintptr
	var transLen uint32
	if err := windows.VerQueryValue(block, `\VarFileInfo\Translation`, unsafe.Pointer(&transPtr), &transLen); err == nil && transLen >= 4 && transPtr != 0 {
		// Translation 是 {WORD lang, WORD codepage} 数组；transPtr 指向 buf 内部。
		off := transPtr - uintptr(block)
		if off+4 <= uintptr(len(buf)) {
			lang := uint16(buf[off]) | uint16(buf[off+1])<<8
			cp := uint16(buf[off+2]) | uint16(buf[off+3])<<8
			subBlocks = append(subBlocks, fmt.Sprintf(`\StringFileInfo\%04x%04x\FileDescription`, lang, cp))
		}
	}
	subBlocks = append(subBlocks, `\StringFileInfo\040904b0\FileDescription`)

	for _, sb := range subBlocks {
		var strPtr uintptr
		var strLen uint32
		if err := windows.VerQueryValue(block, sb, unsafe.Pointer(&strPtr), &strLen); err != nil || strLen == 0 || strPtr == 0 {
			continue
		}
		off := strPtr - uintptr(block)
		end := off + uintptr(strLen)*2
		if end > uintptr(len(buf)) {
			continue
		}
		u := make([]uint16, strLen)
		for i := range u {
			u[i] = uint16(buf[off+uintptr(i)*2]) | uint16(buf[off+uintptr(i)*2+1])<<8
		}
		if d := strings.TrimSpace(windows.UTF16ToString(u)); d != "" {
			return d
		}
	}
	return ""
}

func idleSeconds() int {
	lii := lastInputInfo{cbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	if r, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&lii))); r == 0 {
		return 0
	}
	now, _, _ := procGetTickCount.Call()
	return int((uint32(now) - lii.dwTime) / 1000)
}

func filetimeU64(ft windows.Filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

// cpuPercent 用两次 GetSystemTimes 的差值计算 CPU 占用率（kernel 时间包含 idle）。
func (s *Sampler) cpuPercent() float64 {
	var idle, kernel, user windows.Filetime
	r, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return 0
	}
	i, ku := filetimeU64(idle), filetimeU64(kernel)+filetimeU64(user)
	defer func() { s.prevIdle, s.prevKernUser, s.cpuValid = i, ku, true }()
	if !s.cpuValid || ku <= s.prevKernUser {
		return 0
	}
	total := ku - s.prevKernUser
	idleD := i - s.prevIdle
	if idleD > total {
		return 0
	}
	return Round1(float64(total-idleD) * 100 / float64(total))
}

func memPercent() float64 {
	ms := memoryStatusEx{dwLength: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if r, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms))); r == 0 || ms.ullTotalPhys == 0 {
		return 0
	}
	used := ms.ullTotalPhys - ms.ullAvailPhys
	return Round1(float64(used) * 100 / float64(ms.ullTotalPhys))
}

// battery 返回是否有电池、电量与充电状态；BatteryFlag 为 128（无电池）或 255（未知）时视为无电池。
func battery() (bool, int, bool) {
	var ps systemPowerStatus
	if r, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&ps))); r == 0 {
		return false, 0, false
	}
	if ps.BatteryFlag == 128 || ps.BatteryFlag == 255 || ps.BatteryLifePercent > 100 {
		return false, 0, false
	}
	charging := ps.ACLineStatus == 1 || ps.BatteryFlag&8 != 0
	return true, int(ps.BatteryLifePercent), charging
}

// DeviceModel 返回配置的 model，为空时使用主机名。
func DeviceModel(cfg *Config) string {
	if cfg != nil && cfg.Model != "" {
		return cfg.Model
	}
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

// OSVersion 从注册表读取 ProductName + DisplayVersion；CurrentBuild ≥ 22000 时修正为 Windows 11。
func OSVersion() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "Windows"
	}
	defer k.Close()
	product, _, _ := k.GetStringValue("ProductName")
	display, _, _ := k.GetStringValue("DisplayVersion")
	if display == "" {
		display, _, _ = k.GetStringValue("ReleaseId")
	}
	buildStr, _, _ := k.GetStringValue("CurrentBuild")
	if build, err := strconv.Atoi(strings.TrimSpace(buildStr)); err == nil && build >= 22000 {
		product = strings.Replace(product, "Windows 10", "Windows 11", 1)
	}
	if product == "" {
		product = "Windows"
	}
	return strings.TrimSpace(product + " " + display)
}
