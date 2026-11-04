package main

import (
	"os"
	"sync"
)

// MaxLogSize 是日志文件轮转阈值（1 MB）。
const MaxLogSize = 1 << 20

// RotatingFile 是按大小轮转的日志文件：超过 MaxLogSize 时重命名为 <path>.1 并新建文件。
type RotatingFile struct {
	mu   sync.Mutex
	path string
	max  int64
	f    *os.File
	size int64
}

// OpenRotatingFile 以追加方式打开日志文件。
func OpenRotatingFile(path string, max int64) (*RotatingFile, error) {
	r := &RotatingFile{path: path, max: max}
	if err := r.open(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RotatingFile) open() error {
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	r.f, r.size = f, st.Size()
	return nil
}

func (r *RotatingFile) rotate() error {
	r.f.Close()
	backup := r.path + ".1"
	os.Remove(backup)
	if err := os.Rename(r.path, backup); err != nil {
		// 重命名失败时清空当前文件，确保日志大小保持在阈值附近。
		if f, terr := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); terr == nil {
			f.Close()
		}
	}
	return r.open()
}

// Write 实现 io.Writer。
func (r *RotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.size > 0 && r.size+int64(len(p)) > r.max {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

// Close 关闭文件。
func (r *RotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.f.Close()
}
