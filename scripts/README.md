# Load Testing Script

This directory contains a simple load testing script for the shared-lock application.

## Load Test Script

The `load.go` script is a simple HTTP load testing tool that can be used to test the performance of the shared-lock application.

### Features

- Configurable number of concurrent clients
- Configurable test duration
- Real-time statistics (requests per second, success/failure rate, average latency)
- Support for different HTTP methods and endpoints
- Customizable request body

### Usage

```bash
# Build the script
go build -o load scripts/load.go

# Run with default settings (150 concurrent clients for 30 seconds)
./load

# Run with custom settings
./load -url=http://localhost:8080 -endpoint=/lease -concurrency=20 -duration=1m

# Run with a custom request body
./load -body='{"key":"custom-key","value":"custom-value","labels":{"test":"load-test","priority":"high"}}'

# Enable verbose output
./load -verbose=true
```

### Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `-url` | Base URL of the application | `http://localhost:8080` |
| `-endpoint` | API endpoint to test | `/lease` |
| `-method` | HTTP method to use | `POST` |
| `-concurrency` | Number of concurrent clients | `150` |
| `-duration` | Duration of the load test | `30s` |
| `-body` | Request body (JSON) | `{"key":"test-key","value":"test-value","labels":{"test":"load-test"}}` |
| `-verbose` | Enable verbose output | `false` |

### Example Output

``` text
Requests: 150, Success: 148, Failed: 2, RPS: 5.00, Avg Latency: 45.32 ms

Load test completed in 30.00 seconds
Total requests: 150
Successful requests: 148
Failed requests: 2
Requests per second: 5.00
Average latency: 45.32 ms
```

## Tips for Load Testing

1. Start with a small number of concurrent clients and gradually increase to find the optimal performance.
2. Monitor the application's resource usage (CPU, memory, network) during the test.
3. Look for patterns in failed requests to identify potential bottlenecks.
4. Consider testing different endpoints and request payloads to simulate real-world usage.
5. For production testing, consider using a dedicated load testing machine to avoid affecting the application server. 