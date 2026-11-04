package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

// ConfigFileName 是与 exe 同目录的配置文件名。
const ConfigFileName = "miam1ku-status.json"

// Config 对应 miam1ku-status.json。
type Config struct {
	Endpoint           string            `json:"endpoint"`
	Token              string            `json:"token"`
	Device             string            `json:"device"`
	Model              string            `json:"model"`
	ShowTitle          *bool             `json:"show_title"`
	HideTitleApps      []string          `json:"hide_title_apps"`
	AppAliases         map[string]string `json:"app_aliases"`
	HeartbeatSeconds   int               `json:"heartbeat_seconds"`
	MinIntervalSeconds int               `json:"min_interval_seconds"`
}

// ConfigTemplate 是配置文件缺失时写出的模板，token 留空由用户填写。
const ConfigTemplate = `{
  "endpoint": "https://miam1ku.cn/api/report",
  "token": "",
  "device": "pc",
  "model": "",
  "show_title": true,
  "hide_title_apps": ["KeePassXC.exe", "1Password.exe"],
  "app_aliases": { "Code.exe": "Visual Studio Code" },
  "heartbeat_seconds": 60,
  "min_interval_seconds": 10
}
`

// ErrConfigCreated 表示配置文件原本缺失，已写出模板。
var ErrConfigCreated = errors.New("config template created")

// ErrTokenEmpty 表示配置中 token 为空。
var ErrTokenEmpty = errors.New("token is empty")

// ParseConfig 解析配置内容并补齐默认值。
func ParseConfig(data []byte) (*Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.applyDefaults()
	return &c, nil
}

func (c *Config) applyDefaults() {
	c.Endpoint = strings.TrimSpace(c.Endpoint)
	if c.Endpoint == "" {
		c.Endpoint = "https://miam1ku.cn/api/report"
	}
	c.Token = strings.TrimSpace(c.Token)
	// 设备 ID 固定为 pc，配置里的 device 字段仅作说明用途。
	c.Device = "pc"
	c.Model = strings.TrimSpace(c.Model)
	if c.ShowTitle == nil {
		t := true
		c.ShowTitle = &t
	}
	if c.HeartbeatSeconds <= 0 {
		c.HeartbeatSeconds = 60
	}
	if c.MinIntervalSeconds <= 0 {
		c.MinIntervalSeconds = 10
	}
	if c.AppAliases == nil {
		c.AppAliases = map[string]string{}
	}
}

// LoadConfig 读取配置；文件缺失时写出模板并返回 ErrConfigCreated，token 为空时返回 ErrTokenEmpty。
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if werr := os.WriteFile(path, []byte(ConfigTemplate), 0o600); werr != nil {
			return nil, fmt.Errorf("write config template: %w", werr)
		}
		return nil, ErrConfigCreated
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	c, err := ParseConfig(data)
	if err != nil {
		return nil, err
	}
	if c.Token == "" {
		return c, ErrTokenEmpty
	}
	return c, nil
}

// TitleEnabled 返回是否上报 title 字段（show_title）。
func (c *Config) TitleEnabled() bool {
	return c.ShowTitle == nil || *c.ShowTitle
}

// TitleHiddenFor 返回该 exe 是否在 hide_title_apps 中（大小写不敏感）。
func (c *Config) TitleHiddenFor(exe string) bool {
	if exe == "" {
		return false
	}
	for _, name := range c.HideTitleApps {
		if strings.EqualFold(strings.TrimSpace(name), exe) {
			return true
		}
	}
	return false
}

// Alias 按 exe 名（大小写不敏感）查找配置的显示名。
func (c *Config) Alias(exe string) (string, bool) {
	if exe == "" {
		return "", false
	}
	for k, v := range c.AppAliases {
		if strings.EqualFold(strings.TrimSpace(k), exe) && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v), true
		}
	}
	return "", false
}

// Heartbeat 返回心跳间隔。
func (c *Config) Heartbeat() time.Duration {
	return time.Duration(c.HeartbeatSeconds) * time.Second
}

// MinInterval 返回两次上报的最小间隔。
func (c *Config) MinInterval() time.Duration {
	return time.Duration(c.MinIntervalSeconds) * time.Second
}
