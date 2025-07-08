# IoT Metrics Service

![Go Coverage](https://img.shields.io/badge/coverage-95.2%25-brightgreen)

This service collects and processes metrics (temperature and battery level) from IoT devices. It provides both gRPC and HTTP endpoints for data ingestion and management.

## System Behavior

The service receives metrics from IoT devices and stores them in a database. Additionally, all incoming metrics data is logged. By default, logs are generated in JSON format and stored in `./tmp/app.log`. This logging behavior can be configured via environment variables to send logs to various log analysis platforms.

The service also allows for device-specific configurations, such as setting thresholds for temperature and battery levels. When these thresholds are exceeded, the service generates alerts.

A rate limiter is in place to control the request rate from each device, preventing system overload. The service can be configured to use either an in-memory or a file-based SQLite database.

## Local Run Instructions

0. **Dev system requirement:**

- Ubuntu Linux 22.04
- Golang > 1.24.0
- Nodejs > 22.15.1

**Caution**: not tested on Macos, specifically `npm run postinstall` could fail. Recommend to use devcontainer of VSCode if on Macos (just open project in VSCode it will prompt open in devcontainer)

1.  **Install dependencies:**

    ```bash
    npm install
    ```

2.  **Configure environment:**

    Copy the `.env.example` file to `.env` and update the environment variables as needed.

    ```bash
    cp .env.example .env
    ```

    the meaning of entries in env are

    ```
    GO_ENV=development # can be development or production
    IOT_DB_TYPE=file # file is to use sqlite, otherwise will use in-memory sqlite
    IOT_DB_PATH=./iot.db # where is the sqlite db file
    IOT_HTTP_HOST_PORT=:1080 # default http restful server host:port
    IOT_GRPC_HOST_PORT=:10801 # default grpc server host:port, if leave empty will not start grpc server
    IOT_DEFAULT_RATE=64 # default rate, float value, # of req/second, zero disalbe all access
    IOT_DEFAULT_BURST=8 # default burst, int value, # of reqs, zero disable all access
    ```

3.  **Run the service:**

    - For development with hot-reloading:

      ```bash
      npm run dev
      ```

## Example Request/Response

Before trying the examples below, ensure the service is running. You can start it in development mode using:

```bash
npm install
npm run dev
```

### HTTP Restful Examples

Here are some examples of how to interact with the HTTP endpoints:

### Post Metrics

- **Request:**

  ```bash
  curl -X POST http://localhost:1080/devices/device-1/metrics \
  -H "Content-Type: application/json" \
  -d '{
      "timestamp": "2024-07-22T10:00:00Z",
      "temperature": 25.5,
      "battery": 80.2
  }'
  ```

- **Response:**

  `200 OK`

### Update Configuration

- **Request:**

  ```bash
  curl -X PUT http://localhost:1080/devices/device-1/config \
  -H "Content-Type: application/json" \
  -d '{
      "temperature_threshold": 30.0,
      "battery_threshold": 20.0
  }'
  ```

- **Response:**

  `200 OK`

### Get Alerts

- **Request:**

  ```bash
  curl http://localhost:1080/devices/device-1/alerts
  ```

- **Response:**

  ```json
  [
    {
      "ID": 1,
      "DeviceID": "device-1",
      "Timestamp": "2024-07-22T10:05:00Z",
      "Type": "temperature",
      "Message": "Temperature threshold exceeded"
    }
  ]
  ```

### Set Rate Limiter

- **Request:**

  ```bash
  curl -X POST http://localhost:1080/devices/device-1/limiter \
  -H "Content-Type: application/json" \
  -d '{
      "rate": 10,
      "burst": 5
  }'
  ```

- **Response:**

  `200 OK`

### Health Check

- **Request:**

  ```bash
  curl http://localhost:1080/health
  ```

- **Response:**

  ```json
  {
    "status": "ok"
  }
  ```

### gRPC Examples

The gRPC server starts on port `10801`.

To interact with the gRPC endpoints, you can use a tool like `grpcurl`. First, ensure you have `grpcurl` installed and the `service.proto` file available.

#### Update Configuration (gRPC)

```bash
grpcurl -plaintext -d '{"deviceId": "device-1", "config": {"temperatureThreshold": 30.0, "batteryThreshold": 20.0}}' localhost:10801 IOTService/UpdateConfig
```

response

```
{
  "status": {
    "success": true,
    "message": "OK"
  }
}
```

#### Post Metrics (gRPC)

```bash
grpcurl -plaintext -d '{"deviceId": "device-1", "metric": {"timestamp": "2024-07-22T10:00:00Z", "temperature": 35.5, "battery": 10.2}}' localhost:10801 IOTService/PostMetrics
```

response

```
{
  "status": {
    "success": true,
    "message": "OK"
  }
}
```

#### Get Alerts (gRPC)

```bash
grpcurl -plaintext -d '{"deviceId": "device-1"}' localhost:10801 IOTService/GetAlerts
```

response

```
{
  "status": {
    "success": true,
    "message": "OK"
  },
  "alerts": [
    {
      "id": "5222",
      "deviceId": "device-1",
      "timestamp": "2025-07-08T11:03:46.468635863Z",
      "type": "temperature",
      "message": "Temperature 35.50 exceeded threshold 30.00"
    },
    {
      "id": "5223",
      "deviceId": "device-1",
      "timestamp": "2025-07-08T11:03:46.468635863Z",
      "type": "battery",
      "message": "Battery 10.20 below threshold 20.00"
    }
  ]
}
```

## Testing and Coverage

### Running Unit Tests

To run all unit tests for the project, use the following command:

```bash
npm run test
```

### Generating Coverage Report

To generate a code coverage report locally, use the following command:

```bash
npm run cover
```

This will generate `coverage.filtered.out` and `coverage.html` files in the project root. You can open `coverage.html` in your browser to view a detailed report.

### Continuous Integration and Coverage Badge

This project uses GitHub Actions for continuous integration. Whenever changes are pushed to the `main` branch or a pull request is opened, the CI workflow will automatically:

1.  Run all unit tests.
2.  Generate a code coverage report.
3.  Update the coverage badge displayed at the top of this `README.md` file.

This ensures that the coverage percentage is always up-to-date with the latest codebase.

## Benchmarking

The `device1k` benchmark simulates concurrent reports from 1000+ IoT devices interacting with the service. To run this benchmark, you must have a working instance of the service running.

1.  **Start the IoT Metrics Service:**
    You can start the service using one of the following commands:

    ```bash
    npm run dev
    # or
    go run ./cmd/server
    ```

2.  **Run the Benchmark:**
    Once the service is running, execute the benchmark from the project root directory:
    ```bash
    go run ./benchmark/device1k
    ```

This will execute the benchmark, simulating concurrent device interactions, and print throughput results to your console.

### Example Run Log

```bash
yli@pvedev:~/iot-metrics-service$ go run ./benchmark/device1k/
generated 2000 device IDs
http server verified
gRPC server verified and connected
inserted config for 2000 devices: used time=1.343432474 seconds, throughput=1488.7238761209221 action/second
executed action GetAlerts for device 9ad66685-648a-4009-b0db-06d06b738c0b46d
did actions for 2000 devices: used time=5.333908352 seconds, throughput=1124.8787200759164 action/second
```

## System Design

The IoT Metrics Service is designed to collect, process, and manage metrics from IoT devices, providing both gRPC and HTTP interfaces. The architecture is modular, with distinct packages handling specific functionalities.

### Core Components

- **`cmd/server/main.go`**: The main entry point of the application. It initializes and configures the service, including:

  - Loading environment variables for configuration (e.g., database type, host/port for gRPC and HTTP).
  - Setting up the database connection (supporting file-based SQLite and in-memory SQLite).
  - Initializing and starting both gRPC and HTTP servers.
  - Configuring global rate limiting for incoming requests.

- **`pkg/iot`**: This package encapsulates the core business logic related to IoT device management. It includes functionalities for:

  - **Metrics**: Handling the ingestion and storage of device metrics (e.g., temperature, battery level).
  - **Configuration**: Managing device-specific configurations, such as thresholds for metrics.
  - **Alerts**: Generating alerts when configured thresholds are exceeded.
  - **Rate Limiting**: Implementing device-specific rate limiting to prevent system overload.

- **`pkg/db`**: Responsible for all database interactions. It provides an abstraction layer for data persistence, currently supporting SQLite (both file-based and in-memory).

- **`pkg/grpc`**: Implements the gRPC server and its handlers. It defines the protobuf service (`service.proto`) for efficient, high-performance communication with IoT devices and other services. It also includes an interceptor for rate limiting gRPC requests.

- **`pkg/http`**: Implements the RESTful HTTP server using the Gin framework. It provides HTTP endpoints for data ingestion, configuration updates, alert retrieval, and rate limiter settings.

- **`pkg/models`**: Contains the data structures (Go structs) that represent the various entities within the system, such as device metrics, configurations, and alerts.

- **`pkg/common`**: Provides common utilities and shared functionalities, such as logging and constants, used across different parts of the application.

### Data Flow

1.  **Data Ingestion**: IoT devices send metrics and configuration updates to the service via either gRPC or HTTP endpoints.
2.  **Rate Limiting**: Incoming requests are subjected to rate limiting to ensure system stability and prevent abuse.
3.  **Business Logic**: The `pkg/iot` layer processes the incoming data, applies business rules (e.g., checking thresholds), and generates alerts if necessary.
4.  **Data Persistence**: Processed data (metrics, configurations, alerts) is stored in the database via the `pkg/db` layer.
5.  **Data Retrieval**: Clients can query the service via gRPC or HTTP to retrieve device configurations, alerts, or other relevant information.
