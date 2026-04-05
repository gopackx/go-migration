package database

import "time"

// ConnectionConfig holds the configuration for a database connection.
type ConnectionConfig struct {
	Driver          string            `yaml:"driver" json:"driver"`
	Host            string            `yaml:"host" json:"host"`
	Port            int               `yaml:"port" json:"port"`
	Database        string            `yaml:"database" json:"database"`
	Username        string            `yaml:"username" json:"username"`
	Password        string            `yaml:"password" json:"password"`
	MaxOpenConns    int               `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int               `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration     `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	Options         map[string]string `yaml:"options" json:"options"`
}
