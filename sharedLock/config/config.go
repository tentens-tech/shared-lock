package config

import (
	"os"
	"strconv"
)

const (
	DefaultServerPort               = ":8080"
	DefaultEtcdAddr                 = "localhost:2379"
	DefaultEtcdTLSEnabled           = false
	DefaultEtcdServerCACertPath     = "/etc/etcd/ca.crt"
	DefaultEtcdServerClientCertPath = "/etc/etcd/client.crt"
	DefaultEtcdServerClientKeyPath  = "/etc/etcd/client.key"
)

type Config struct {
	ServerPort string
	EtcdCfg    EtcdCfg
}

type EtcdCfg struct {
	EtcdAddr             string
	TLSEnabled           bool
	ServerCACertPath     string
	ServerClientCertPath string
	ServerClientKeyPath  string
}

func Load() *Config {
	return &Config{
		ServerPort: getEnv("SHARED_LOCK_SERVER_PORT", DefaultServerPort),
		EtcdCfg: EtcdCfg{
			EtcdAddr:             getEnv("SHARED_LOCK_ETCD_ADDR", DefaultEtcdAddr),
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
