package lease

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"tentens-tech/shared-lock/sharedLock/config"
	l "tentens-tech/shared-lock/sharedLock/lock"
	"time"

	log "github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	DefaultLeaseTTLHeader       = "x-lease-ttl"
	DefaultPrefix               = "/shared-lock/"
	DefaultLeaseDurationSeconds = 10
)

type Lease struct {
	Key     string            `json:"key"`
	Value   string            `json:"value"`
	Labels  map[string]string `json:"labels"`
	Created time.Time         `json:"timestamp"`
}

func (l *Lease) ToJSON() ([]byte, error) {
	return json.Marshal(l)
}

func CreateLease(cfg *config.Config, etcdConnection *l.Lock, writer http.ResponseWriter, request *http.Request) {
	data := readHTTPBody(request, writer)
	var err error
	lease := Lease{}
	log.Debugf("Load body: %v", string(data))
	err = json.Unmarshal(data, &lease)
	if err != nil {
		log.Errorf("Failed parse json body %v", err)
	}

	key := DefaultPrefix + lease.Key

	var leaseTTL time.Duration
	leaseTTLString := request.Header.Get(DefaultLeaseTTLHeader)
	leaseTTL, err = time.ParseDuration(leaseTTLString)
	if err != nil {
		log.Warnf("Use defaultLeaseDurationSeconds for %v", key)
		leaseTTL = DefaultLeaseDurationSeconds
	}

	log.Debugf("Get lease for key: %v, with ttl: %v", key, leaseTTL)
	var leaseID clientv3.LeaseID
	leaseID, err = etcdConnection.GetLease(key, writer, data, int64(leaseTTL.Seconds()))
	if err != nil {
		log.Errorf("%v", err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	_, err = writer.Write([]byte(fmt.Sprintf("%v", leaseID)))
	if err != nil {
		log.Errorf("Failed to write response for /lease endpoint, %v", err)
	}
}

func ReviveLease(cfg *config.Config, etcdConnection *l.Lock, writer http.ResponseWriter, request *http.Request) {
	var err error
	data := readHTTPBody(request, writer)
	leaseIDString := string(data)

	var leaseIDInt64 int64
	leaseIDInt64, err = strconv.ParseInt(leaseIDString, 10, 64)
	if err != nil {
		log.Errorf("Failed to parse lease id from string, leaseIDString: %v, %v", leaseIDString, err)
		writer.WriteHeader(http.StatusInternalServerError)
	}

	err = etcdConnection.KeepLeaseOnce(etcdConnection.Cli, clientv3.LeaseID(leaseIDInt64))
	if err != nil {
		log.Warnf("Failed to prolong lease: %v", err)
		writer.WriteHeader(http.StatusNoContent)
	}
}

func readHTTPBody(request *http.Request, writer http.ResponseWriter) []byte {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusBadRequest)
		return nil
	}
	data, err := io.ReadAll(request.Body)
	if err != nil {
		log.Errorf("%v", err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	err = request.Body.Close()
	if err != nil {
		log.Errorf("%v", err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	return data
}
