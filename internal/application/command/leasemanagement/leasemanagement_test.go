package leasemanagement

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tentens-tech/shared-lock/internal/infrastructure/storage"
)

type MockStorage struct {
	checkLeasePresenceFunc func(ctx context.Context, key string) (int64, error)
	createLeaseFunc        func(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error)
	keepLeaseOnceFunc      func(ctx context.Context, leaseID int64) error
}

func (m *MockStorage) CheckLeasePresence(ctx context.Context, key string) (int64, error) {
	if m.checkLeasePresenceFunc != nil {
		return m.checkLeasePresenceFunc(ctx, key)
	}
	return 0, nil
}

func (m *MockStorage) CreateLease(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error) {
	if m.createLeaseFunc != nil {
		return m.createLeaseFunc(ctx, key, leaseTTL, data)
	}
	return storage.StatusCreated, 123, nil
}

func (m *MockStorage) KeepLeaseOnce(ctx context.Context, leaseID int64) error {
	if m.keepLeaseOnceFunc != nil {
		return m.keepLeaseOnceFunc(ctx, leaseID)
	}
	return nil
}

func TestCreateLease(t *testing.T) {
	tests := []struct {
		name              string
		lease             Lease
		leaseTTL          time.Duration
		data              []byte
		checkLeaseID      int64
		checkLeaseError   error
		createLeaseStatus string
		createLeaseID     int64
		createLeaseError  error
		keepLeaseError    error
		expectedStatus    string
		expectedID        int64
		expectedError     error
	}{
		{
			name: "Lease already exists",
			lease: Lease{
				Key: "test-key",
			},
			leaseTTL:        10 * time.Second,
			data:            []byte("test-data"),
			checkLeaseID:    456,
			checkLeaseError: nil,
			expectedStatus:  "accepted",
			expectedID:      456,
			expectedError:   nil,
		},
		{
			name: "Lease creation successful",
			lease: Lease{
				Key: "test-key",
			},
			leaseTTL:          10 * time.Second,
			data:              []byte("test-data"),
			checkLeaseID:      0,
			checkLeaseError:   nil,
			createLeaseStatus: storage.StatusCreated,
			createLeaseID:     123,
			createLeaseError:  nil,
			keepLeaseError:    nil,
			expectedStatus:    storage.StatusCreated,
			expectedID:        123,
			expectedError:     nil,
		},
		{
			name: "Error checking lease presence",
			lease: Lease{
				Key: "test-key",
			},
			leaseTTL:        10 * time.Second,
			data:            []byte("test-data"),
			checkLeaseID:    0,
			checkLeaseError: errors.New("check error"),
			expectedStatus:  "",
			expectedID:      0,
			expectedError:   errors.New("failed to check lease presence: check error"),
		},
		{
			name: "Error creating lease",
			lease: Lease{
				Key: "test-key",
			},
			leaseTTL:          10 * time.Second,
			data:              []byte("test-data"),
			checkLeaseID:      0,
			checkLeaseError:   nil,
			createLeaseStatus: "",
			createLeaseID:     0,
			createLeaseError:  errors.New("create error"),
			expectedStatus:    "",
			expectedID:        0,
			expectedError:     errors.New("create error"),
		},
		{
			name: "Error keeping lease alive",
			lease: Lease{
				Key: "test-key",
			},
			leaseTTL:          10 * time.Second,
			data:              []byte("test-data"),
			checkLeaseID:      0,
			checkLeaseError:   nil,
			createLeaseStatus: storage.StatusCreated,
			createLeaseID:     123,
			createLeaseError:  nil,
			keepLeaseError:    errors.New("keep error"),
			expectedStatus:    "",
			expectedID:        0,
			expectedError:     errors.New("failed to prolong lease with leaseID: 123, keep error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{
				checkLeasePresenceFunc: func(ctx context.Context, key string) (int64, error) {
					return tt.checkLeaseID, tt.checkLeaseError
				},
				createLeaseFunc: func(ctx context.Context, key string, leaseTTL int64, data []byte) (string, int64, error) {
					return tt.createLeaseStatus, tt.createLeaseID, tt.createLeaseError
				},
				keepLeaseOnceFunc: func(ctx context.Context, leaseID int64) error {
					return tt.keepLeaseError
				},
			}

			status, id, err := CreateLease(context.Background(), mockStorage, tt.leaseTTL, tt.lease)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestReviveLease(t *testing.T) {
	tests := []struct {
		name           string
		leaseID        int64
		keepLeaseError error
		expectedError  error
	}{
		{
			name:           "Successful lease revival",
			leaseID:        123,
			keepLeaseError: nil,
			expectedError:  nil,
		},
		{
			name:           "Error keeping lease alive",
			leaseID:        456,
			keepLeaseError: errors.New("keep error"),
			expectedError:  errors.New("keep error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{
				keepLeaseOnceFunc: func(ctx context.Context, leaseID int64) error {
					return tt.keepLeaseError
				},
			}

			err := ReviveLease(context.Background(), mockStorage, tt.leaseID)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
