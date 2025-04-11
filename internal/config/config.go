package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultDebugMode                = false
	DefaultServerPort               = "8080"
	DefaultServerReadTimeout        = 10 * time.Second
	DefaultServerWriteTimeout       = 10 * time.Second
	DefaultServerIdleTimeout        = 120 * time.Second
	DefaultServerShutdownTimeout    = 10 * time.Second
	DefaultServerPPROFEnabled       = false
	DefaultStorageType              = "etcd"
	DefaultEtcdAddrList             = "http://localhost:2379"
	DefaultEtcdTLSEnabled           = false
	DefaultEtcdServerCACertPath     = "/etc/etcd/ca.crt"
	DefaultEtcdServerClientCertPath = "/etc/etcd/client.crt"
	DefaultEtcdServerClientKeyPath  = "/etc/etcd/client.key"
	DefaultCacheEnabled             = false
	DefaultCacheSize                = 1000
)

type Config struct {
	Server  ServerCfg
	Storage StorageCfg
	Cache   CacheCfg
	Debug   bool
}

type ServerCfg struct {
	Port         string
	PPROFEnabled bool
	Timeout      ServerTimeout
}

type ServerTimeout struct {
	Read     time.Duration
	Write    time.Duration
	Idle     time.Duration
	Shutdown time.Duration
}

type StorageCfg struct {
	Type string `validate:"required" oneof:"etcd mock"`
	Etcd EtcdCfg
	Mock MockCfg
}

type MockCfg struct {
}

type EtcdCfg struct {
	EtcdAddrList         []string
	TLSEnabled           bool
	ServerCACertPath     string
	ServerClientCertPath string
	ServerClientKeyPath  string
}

type CacheCfg struct {
	Enabled bool
	Size    int
}

func NewConfig() *Config {
	etcdEndpointsList, err := checkEtcdEndpointsList(getEnv("SHARED_LOCK_ETCD_ADDR_LIST", DefaultEtcdAddrList))
	if err != nil {
		log.Fatal(err)
	}

	return &Config{
		Server: ServerCfg{
			Port:         getEnv("SHARED_LOCK_SERVER_PORT", DefaultServerPort),
			PPROFEnabled: getEnv("SHARED_LOCK_PPROF_ENABLED", bool(DefaultServerPPROFEnabled)),
			Timeout: ServerTimeout{
				Read:     getEnv("SHARED_LOCK_SERVER_READ_TIMEOUT", DefaultServerReadTimeout),
				Write:    getEnv("SHARED_LOCK_SERVER_WRITE_TIMEOUT", DefaultServerWriteTimeout),
				Idle:     getEnv("SHARED_LOCK_SERVER_IDLE_TIMEOUT", DefaultServerIdleTimeout),
				Shutdown: getEnv("SHARED_LOCK_SERVER_SHUTDOWN_TIMEOUT", DefaultServerShutdownTimeout),
			},
		},
		Storage: StorageCfg{
			Type: getEnv("SHARED_LOCK_STORAGE_TYPE", DefaultStorageType),
			Etcd: EtcdCfg{
				EtcdAddrList:         etcdEndpointsList,
				TLSEnabled:           getEnv("SHARED_LOCK_ETCD_TLS", DefaultEtcdTLSEnabled),
				ServerCACertPath:     getEnv("SHARED_LOCK_CA_CERT_PATH", DefaultEtcdServerCACertPath),
				ServerClientCertPath: getEnv("SHARED_LOCK_CLIENT_CERT_PATH", DefaultEtcdServerClientCertPath),
				ServerClientKeyPath:  getEnv("SHARED_LOCK_CLIENT_KEY_PATH", DefaultEtcdServerClientKeyPath),
			},
		},
		Cache: CacheCfg{
			Enabled: getEnv("SHARED_LOCK_CACHE_ENABLED", DefaultCacheEnabled),
			Size:    getEnv("SHARED_LOCK_CACHE_SIZE", DefaultCacheSize),
		},
		Debug: getEnv("SHARED_LOCK_DEBUG", bool(DefaultDebugMode)),
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

func checkEtcdEndpointsList(etcdEndpointsList string) ([]string, error) {
	etcdEndpoints := strings.Split(etcdEndpointsList, ",")
	if len(etcdEndpoints) == 0 {
		return nil, fmt.Errorf("no etcd endpoints provided")
	}
	if strings.ContainsAny(etcdEndpointsList, ";|") {
		return nil, fmt.Errorf("invalid separator in etcd endpoints. Use comma (,) to separate endpoints")
	}

	for _, endpoint := range etcdEndpoints {
		if endpoint = strings.TrimSpace(endpoint); endpoint == "" {
			return nil, fmt.Errorf("empty etcd endpoint provided")
		}
	}

	return etcdEndpoints, nil
}
