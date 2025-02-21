package storage

type Connection interface {
	GetLease(key string, data []byte, leaseTTL int64) (string, int64, error)
	KeepLeaseOnce(leaseID int64) error
}
