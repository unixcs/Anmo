// Package config loads YAML configuration with ANMO_-prefixed env overrides.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   Server   `yaml:"server"`
	MySQL    MySQL    `yaml:"mysql"`
	Auth     Auth     `yaml:"auth"`
	SMS      SMS      `yaml:"sms"`
	Log      Log      `yaml:"log"`
	Business Business `yaml:"business"`
}

type Server struct {
	Addr string `yaml:"addr"`
}

type MySQL struct {
	DSN string `yaml:"dsn"`
}

type Auth struct {
	JWTSecret          string `yaml:"jwt_secret"`
	AdminTokenHours    int    `yaml:"admin_token_hours"`
	CustomerTokenHours int    `yaml:"customer_token_hours"`
	AdminPhone         string `yaml:"admin_phone"`
	AdminPasswordSeed  string `yaml:"admin_password_seed"`
}

type SMS struct {
	Mode    string `yaml:"mode"` // dev: fixed code 123456
	CodeTTL int    `yaml:"code_ttl_seconds"`
}

type Log struct {
	Level string `yaml:"level"` // debug|info|warn|error
}

type Business struct {
	OpenTime          string `yaml:"open_time"`            // "09:00"
	CloseTime         string `yaml:"close_time"`           // "21:00"
	SlotMinutes       int    `yaml:"slot_minutes"`         // 30
	BookAheadDays     int    `yaml:"book_ahead_days"`      // 30
	BookMinAheadHours int    `yaml:"book_min_ahead_hours"` // 2
	CancelMinAheadHrs int    `yaml:"cancel_min_ahead_hours"`
	BufferMinutes     int    `yaml:"buffer_minutes"`
	LowBalanceCount   int    `yaml:"low_balance_count"` // <=2 低余额
	DormantDays       int    `yaml:"dormant_days"`      // 60 沉睡
	ExpiringDays      int    `yaml:"expiring_days"`     // 7 临期
}

// Load reads path (optional) and applies ANMO_SECTION_KEY env overrides,
// e.g. ANMO_MYSQL_DSN, ANMO_AUTH_JWTSECRET, ANMO_SERVER_ADDR.
func Load(path string) (*Config, error) {
	cfg := &Config{}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(b, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}
	applyEnv(cfg)
	setDefaults(cfg)
	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	for env, set := range envOverrides {
		if v, ok := os.LookupEnv(env); ok && strings.TrimSpace(v) != "" {
			set(cfg, v)
		}
	}
}

// envOverrides maps ANMO_* environment variables to config setters.
var envOverrides = map[string]func(*Config, string){
	"ANMO_SERVER_ADDR":                   func(c *Config, v string) { c.Server.Addr = v },
	"ANMO_MYSQL_DSN":                     func(c *Config, v string) { c.MySQL.DSN = v },
	"ANMO_AUTH_JWT_SECRET":               func(c *Config, v string) { c.Auth.JWTSecret = v },
	"ANMO_AUTH_JWTSECRET":                func(c *Config, v string) { c.Auth.JWTSecret = v },
	"ANMO_AUTH_ADMIN_PHONE":              func(c *Config, v string) { c.Auth.AdminPhone = v },
	"ANMO_AUTH_ADMIN_PASSWORD_SEED":      func(c *Config, v string) { c.Auth.AdminPasswordSeed = v },
	"ANMO_SMS_MODE":                      func(c *Config, v string) { c.SMS.Mode = v },
	"ANMO_LOG_LEVEL":                     func(c *Config, v string) { c.Log.Level = v },
	"ANMO_BUSINESS_OPEN_TIME":            func(c *Config, v string) { c.Business.OpenTime = v },
	"ANMO_BUSINESS_CLOSE_TIME":           func(c *Config, v string) { c.Business.CloseTime = v },
	"ANMO_BUSINESS_BOOK_AHEAD_DAYS":      func(c *Config, v string) { c.Business.BookAheadDays = atoi(v) },
	"ANMO_BUSINESS_BOOK_MIN_AHEAD_HOURS": func(c *Config, v string) { c.Business.BookMinAheadHours = atoi(v) },
}

func atoi(v string) int {
	n, _ := strconv.Atoi(v)
	return n
}

func setDefaults(cfg *Config) {
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Auth.AdminTokenHours == 0 {
		cfg.Auth.AdminTokenHours = 12
	}
	if cfg.Auth.CustomerTokenHours == 0 {
		cfg.Auth.CustomerTokenHours = 24 * 7
	}
	if cfg.SMS.Mode == "" {
		cfg.SMS.Mode = "dev"
	}
	if cfg.SMS.CodeTTL == 0 {
		cfg.SMS.CodeTTL = 300
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Business.OpenTime == "" {
		cfg.Business.OpenTime = "09:00"
	}
	if cfg.Business.CloseTime == "" {
		cfg.Business.CloseTime = "21:00"
	}
	if cfg.Business.SlotMinutes == 0 {
		cfg.Business.SlotMinutes = 30
	}
	if cfg.Business.BookAheadDays == 0 {
		cfg.Business.BookAheadDays = 30
	}
	if cfg.Business.BookMinAheadHours == 0 {
		cfg.Business.BookMinAheadHours = 2
	}
	if cfg.Business.CancelMinAheadHrs == 0 {
		cfg.Business.CancelMinAheadHrs = 2
	}
	if cfg.Business.LowBalanceCount == 0 {
		cfg.Business.LowBalanceCount = 2
	}
	if cfg.Business.DormantDays == 0 {
		cfg.Business.DormantDays = 60
	}
	if cfg.Business.ExpiringDays == 0 {
		cfg.Business.ExpiringDays = 7
	}
}

func validate(cfg *Config) error {
	if cfg.MySQL.DSN == "" {
		return fmt.Errorf("mysql.dsn is required")
	}
	if cfg.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret is required")
	}
	return nil
}
