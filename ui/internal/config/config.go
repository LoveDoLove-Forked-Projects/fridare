package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 应用程序配置结构
type Config struct {
	// 全局配置
	AppVersion string `json:"app_version"`
	WorkDir    string `json:"work_dir"`

	// 网络配置
	Proxy   string `json:"proxy"`
	Timeout int    `json:"timeout"` // 秒
	Retries int    `json:"retries"`

	// Frida 配置
	DefaultPort int    `json:"default_port"`
	MagicName   string `json:"magic_name"`
	AutoConfirm bool   `json:"auto_confirm"`

	// UI 配置
	Theme        string `json:"theme"` // "light", "dark", "auto"
	WindowWidth  int    `json:"window_width"`
	WindowHeight int    `json:"window_height"`
	DebugMode    bool   `json:"debug_mode"`
	NoShowNotice bool   `json:"no_show_notice"` // 是否不再显示通知

	// 下载配置
	DownloadDir         string `json:"download_dir"`
	ConcurrentDownloads int    `json:"concurrent_downloads"`

	// 最近使用
	RecentVersions  []string `json:"recent_versions"`
	RecentPlatforms []string `json:"recent_platforms"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		AppVersion: "1.0.0",
		WorkDir:    filepath.Join(homeDir, ".fridare"),

		Proxy:   "",
		Timeout: 30,
		Retries: 3,

		DefaultPort: 27042,
		MagicName:   "frida",
		AutoConfirm: false,

		Theme:        "auto",
		WindowWidth:  1200,
		WindowHeight: 800,
		DebugMode:    false,
		NoShowNotice: false,

		DownloadDir:         filepath.Join(homeDir, "Downloads", "fridare"),
		ConcurrentDownloads: 3,

		RecentVersions:  []string{},
		RecentPlatforms: []string{},
	}
}

// ConfigPath 返回配置文件路径
func ConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	appConfigDir := filepath.Join(configDir, "fridare")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(appConfigDir, "config.json"), nil
}

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, fmt.Errorf("获取配置路径失败: %w", err)
	}

	// 如果配置文件不存在，返回默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		// 尝试保存默认配置
		if saveErr := cfg.Save(); saveErr != nil {
			// 保存失败但不影响使用默认配置
			fmt.Printf("保存默认配置失败: %v\n", saveErr)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证并补充默认值
	cfg.validate()

	return &cfg, nil
}

// Save 保存配置到文件
func (c *Config) Save() error {
	configPath, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("获取配置路径失败: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// validate 验证配置并补充默认值
func (c *Config) validate() {
	if c.Timeout <= 0 {
		c.Timeout = 30
	}
	if c.Retries <= 0 {
		c.Retries = 3
	}
	if c.DefaultPort <= 0 || c.DefaultPort > 65535 {
		c.DefaultPort = 27042
	}
	if c.WindowWidth <= 0 {
		c.WindowWidth = 1200
	}
	if c.WindowHeight <= 0 {
		c.WindowHeight = 800
	}
	if c.ConcurrentDownloads <= 0 {
		c.ConcurrentDownloads = 3
	}
	if c.Theme == "" {
		c.Theme = "auto"
	}
	if c.WorkDir == "" {
		homeDir, _ := os.UserHomeDir()
		c.WorkDir = filepath.Join(homeDir, ".fridare")
	}
	if c.DownloadDir == "" {
		homeDir, _ := os.UserHomeDir()
		c.DownloadDir = filepath.Join(homeDir, "Downloads", "fridare")
	}
}

// AddRecentVersion 添加最近使用的版本
func (c *Config) AddRecentVersion(version string) {
	// 移除重复项
	for i, v := range c.RecentVersions {
		if v == version {
			c.RecentVersions = append(c.RecentVersions[:i], c.RecentVersions[i+1:]...)
			break
		}
	}

	// 添加到开头
	c.RecentVersions = append([]string{version}, c.RecentVersions...)

	// 限制数量
	if len(c.RecentVersions) > 10 {
		c.RecentVersions = c.RecentVersions[:10]
	}
}

// AddRecentPlatform 添加最近使用的平台
func (c *Config) AddRecentPlatform(platform string) {
	// 移除重复项
	for i, p := range c.RecentPlatforms {
		if p == platform {
			c.RecentPlatforms = append(c.RecentPlatforms[:i], c.RecentPlatforms[i+1:]...)
			break
		}
	}

	// 添加到开头
	c.RecentPlatforms = append([]string{platform}, c.RecentPlatforms...)

	// 限制数量
	if len(c.RecentPlatforms) > 10 {
		c.RecentPlatforms = c.RecentPlatforms[:10]
	}
}

// EnsureWorkDir 确保工作目录存在
func (c *Config) EnsureWorkDir() error {
	return os.MkdirAll(c.WorkDir, 0755)
}

// EnsureDownloadDir 确保下载目录存在
func (c *Config) EnsureDownloadDir() error {
	return os.MkdirAll(c.DownloadDir, 0755)
}
