package config

import (
	"os"
	"strconv"
	"time"
)

const (
	DefaultServerPort               = "8080"
	DefaultServerReadTimeout        = 10 * time.Second
	DefaultServerWriteTimeout       = 10 * time.Second
	DefaultServerIdleTimeout        = 120 * time.Second
	DefaultServerShutdownTimeout    = 10 * time.Second
	DefaultEtcdAddrList             = "localhost:2379"
	DefaultEtcdTLSEnabled           = false
	DefaultEtcdServerCACertPath     = "/etc/etcd/ca.crt"
	DefaultEtcdServerClientCertPath = "/etc/etcd/client.crt"
	DefaultEtcdServerClientKeyPath  = "/etc/etcd/client.key"
)

type Config struct {
	Server ServerCfg
	Etcd   EtcdCfg
}

type ServerCfg struct {
	Port    string
	Timeout ServerTimeout
}

type ServerTimeout struct {
	Read     time.Duration
	Write    time.Duration
	Idle     time.Duration
	Shutdown time.Duration
}

type EtcdCfg struct {
	EtcdAddrList         string
	TLSEnabled           bool
	ServerCACertPath     string
	ServerClientCertPath string
	ServerClientKeyPath  string
}

func NewConfig() *Config {
	return &Config{
		Server: ServerCfg{
			Port: getEnv("SHARED_LOCK_SERVER_PORT", DefaultServerPort),
			Timeout: ServerTimeout{
				Read:     getEnv("SHARED_LOCK_SERVER_READ_TIMEOUT", DefaultServerReadTimeout),
				Write:    getEnv("SHARED_LOCK_SERVER_WRITE_TIMEOUT", DefaultServerWriteTimeout),
				Idle:     getEnv("SHARED_LOCK_SERVER_IDLE_TIMEOUT", DefaultServerIdleTimeout),
				Shutdown: getEnv("SHARED_LOCK_SERVER_SHUTDOWN_TIMEOUT", DefaultServerShutdownTimeout),
			},
		},
		Etcd: EtcdCfg{
			EtcdAddrList:         getEnv("SHARED_LOCK_ETCD_ADDR_LIST", DefaultEtcdAddrList),
			TLSEnabled:           getEnv("SHARED_LOCK_ETCD_TLS", DefaultEtcdTLSEnabled),
			ServerCACertPath:     getEnv("SHARED_LOCK_CA_CERT_PATH", DefaultEtcdServerCACertPath),
			ServerClientCertPath: getEnv("SHARED_LOCK_CA_CERT_PATH", DefaultEtcdServerClientCertPath),
			ServerClientKeyPath:  getEnv("SHARED_LOCK_CA_CERT_PATH", DefaultEtcdServerClientKeyPath),
		},
	}
}

func getEnv[T any](key string, defaultVal T) T {
	if value, exists := os.LookupEnv(key); exists {
		switch any(defaultVal).(type) {
		case string:
			return any(value).(T)
		case int:
			if intVal, err := strconv.Atoi(value); err == nil {
				return any(intVal).(T)
			}
		case bool:
			if boolVal, err := strconv.ParseBool(value); err == nil {
				return any(boolVal).(T)
			}
		}
	}

	return defaultVal
}
