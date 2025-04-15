# Shared Lock

This repository contains a Go-based implementation of a shared lock service using etcd. The service provides endpoints for creating and maintaining leases, ensuring distributed coordination and synchronization.

## How to build this project
To build it:
``` bash
go build -o shared-lock main.go
```

To build docker image:
``` bash
docker build -t shared-lock:latest -f deployment/Dockerfile .
```

## How to run this project
To run this project, use:
``` bash
./shared-lock serve
```

## Service configuration
| Environment Variable                  | Default Value                     | Description                                      |
|---------------------------------------|-----------------------------------|--------------------------------------------------|
| SHARED_LOCK_SERVER_PORT               | 8080                              | Port on which the server will run                |
| SHARED_LOCK_SERVER_READ_TIMEOUT       | 10s                               | Server read timeout duration                     |
| SHARED_LOCK_SERVER_WRITE_TIMEOUT      | 10s                               | Server write timeout duration                    |
| SHARED_LOCK_SERVER_IDLE_TIMEOUT       | 120s                              | Server idle timeout duration                     |
| SHARED_LOCK_SERVER_SHUTDOWN_TIMEOUT   | 10s                               | Server shutdown timeout duration                 |
| SHARED_LOCK_PPROF_ENABLED             | false                             | Enable pprof for debugging                       |
| SHARED_LOCK_STORAGE_TYPE              | etcd                              | Storage type to use (`etcd` or `mock`)           |
| SHARED_LOCK_ETCD_ADDR_LIST            | http://localhost:2379             | Comma-separated list of etcd endpoints           |
| SHARED_LOCK_ETCD_TLS                  | false                             | Enable TLS for etcd connections                  |
| SHARED_LOCK_CA_CERT_PATH              | /etc/etcd/ca.crt                  | Path to the CA certificate for etcd              |
| SHARED_LOCK_CLIENT_CERT_PATH          | /etc/etcd/client.crt              | Path to the client certificate for etcd          |
| SHARED_LOCK_CLIENT_KEY_PATH           | /etc/etcd/client.key              | Path to the client key for etcd                  |
| SHARED_LOCK_CACHE_ENABLED             | false                             | Enable in-memory cache for leases                |
| SHARED_LOCK_CACHE_SIZE                | 1000                              | Maximum number of items in the cache             |
| SHARED_LOCK_DEBUG                     | false                             | Toggle for debug mode                            |

## How to deploy this project
For this tool to work, you'll need live etcd installation.

As long as etcd mostly used as a part of Kubernetes cluster, we provide examplar installation manifest for the shared lock in `deployment/kubernetes-example.yaml`.

## How to use shared-lock server

### Example

Exampler app that demonstrates shared-lock usage in case of need to guarantee that some app will run only in one instance can be found at the `example` dir.

### Endpoints

1. **Create Lease**
   - **URL**: `/lease`
   - **Method**: `POST`
   - **Headers**:
     - `x-lease-ttl`: (Optional) The TTL (Time To Live) for the lease.
   - **Request Body**:
     - JSON object representing the lease details.
   - **Responses**:
     - `202 Accepted`: Lease request accepted but lease not granted (already present).
     - `201 Created`: Lease successfully created.
     - `500 Internal Server Error`: Failed to create lease.
   - **Example**:
     ```sh
     curl -X POST http://localhost:8080/lease \
          -H "Content-Type: application/json" \
          -H "x-lease-ttl: 60s" \
          -d '{"key": "value"}'
     ```

2. **Keep Alive Lease**
   - **URL**: `/keepalive`
   - **Method**: `POST`
   - **Request Body**:
     - JSON object representing the lease details.
   - **Responses**:\
     - `200 OK`: Lease successfully renewed.
     - `204 No Content`: Failed to prolong lease.
     - `400 Bad Request`: Failed to unmarshal request body.
     - `500 Internal Server Error`: Failed to parse lease ID or prolong lease.
   - **Example**:
     ```sh
     curl -X POST http://localhost:8080/keepalive \
          -H "Content-Type: application/json" \
          -d "12345"
     ```

3. **Health Check**
   - **URL**: `/health`
   - **Method**: `GET`
   - **Responses**:
     - `200 OK`: Server is healthy.
   - **Example**:
     ```sh
     curl -X GET http://localhost:8080/health
     ```

### Error Handling

- The server will respond with appropriate HTTP status codes and error messages in case of failures.
- Common error responses include:
  - `400 Bad Request`: Invalid request body.
  - `500 Internal Server Error`: Internal server error.

### Example Usage

1. **Create a Lease**:
   ```sh
   curl -X POST http://localhost:8080/lease \
        -H "Content-Type: application/json" \
        -H "x-lease-ttl: 60s" \
        -d '{"key": "value"}'