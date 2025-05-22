# Ollama Proxy

[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/ollama-proxy)](https://goreportcard.com/report/github.com/yourusername/ollama-proxy)
[![License](https://img.shields.io/github/license/yourusername/ollama-proxy)](LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/yourusername/ollama-proxy)](https://hub.docker.com/r/yourusername/ollama-proxy)
[![Helm Chart](https://img.shields.io/badge/helm-chart-blue)](https://yourusername.github.io/ollama-proxy)

A production-ready proxy service for Ollama, providing enhanced features like rate limiting, metrics collection, and request validation. Built with Go, this proxy helps you manage and monitor your Ollama API usage in a more controlled and observable way.

## 🌟 Features

- 🔄 **Request Proxying**: Seamlessly proxy requests to Ollama API
- 🚦 **Rate Limiting**: Protect your Ollama instance from abuse
- 📊 **Metrics Collection**: Prometheus metrics for monitoring
- 🔍 **Request Validation**: Validate requests before forwarding
- 🔐 **Environment-based Configuration**: Flexible configuration through environment variables
- 🐳 **Docker Support**: Ready-to-use Docker image
- 📦 **Helm Chart**: Easy deployment to Kubernetes
- 📝 **Structured Logging**: Comprehensive logging for debugging and monitoring

## 📋 Prerequisites

- Go 1.21 or higher
- Docker (optional)
- Kubernetes cluster (optional, for Helm deployment)
- Ollama instance running

## 🚀 How to Run

### 1. Local Development

First, ensure you have Ollama running locally:

```bash
# Start Ollama (if not already running)
ollama serve
```

Then run the proxy:

```bash
# Clone the repository
git clone https://github.com/yourusername/ollama-proxy.git
cd ollama-proxy

# Build and run
go build -o ollama-proxy
./ollama-proxy
```

The proxy will start on `http://localhost:8080` by default.

### 2. Using Docker

```bash
# Pull the image
docker pull ghcr.io/yourusername/ollama-proxy:latest

# Run the container
docker run -p 8080:8080 \
  -e OLLAMA_HOST=http://host.docker.internal:11434 \
  -e PORT=8080 \
  -e LOG_LEVEL=debug \
  ghcr.io/yourusername/ollama-proxy:latest
```

Note: Use `host.docker.internal` to connect to Ollama running on your host machine.

### 3. Using Docker Compose

Create a `docker-compose.yml`:

```yaml
version: '3'
services:
  ollama-proxy:
    image: ghcr.io/yourusername/ollama-proxy:latest
    ports:
      - "8080:8080"
    environment:
      - OLLAMA_HOST=http://ollama:11434
      - PORT=8080
      - LOG_LEVEL=debug
    depends_on:
      - ollama

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
```

Run with:
```bash
docker-compose up -d
```

### 4. Using Helm

```bash
# Add the Helm repository
helm repo add ollama-proxy https://yourusername.github.io/ollama-proxy

# Update the repository
helm repo update

# Install the chart
helm install ollama-proxy ollama-proxy/ollama-proxy \
  --set config.ollamaHost=http://ollama:11434 \
  --set config.logLevel=debug
```

## 🔍 Testing the Proxy

Once running, you can test the proxy using curl:

```bash
# Test the proxy
curl -X POST http://localhost:8080/api/generate \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "prompt": "Hello, how are you?"
  }'
```

## ⚙️ Configuration

The proxy can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `OLLAMA_HOST` | Ollama service URL | `http://localhost:11434` |
| `PORT` | Proxy server port | `8080` |
| `READ_TIMEOUT` | Read timeout in seconds | `30` |
| `WRITE_TIMEOUT` | Write timeout in seconds | `30` |
| `IDLE_TIMEOUT` | Idle timeout in seconds | `120` |
| `METRICS_ENABLED` | Enable metrics collection | `true` |
| `METRICS_PATH` | Metrics endpoint path | `/metrics` |
| `LOG_LEVEL` | Logging level | `info` |
| `RATE_LIMIT` | Requests per second | `100` |
| `RATE_LIMIT_BURST` | Burst limit | `200` |

## 📊 Metrics

The proxy exposes Prometheus metrics at `/metrics` (configurable). Available metrics include:

- Request count
- Request duration
- Error count
- Rate limit hits
- Token usage

## 🛠️ Development

### Building

```bash
# Build the application
go build -o ollama-proxy

# Run tests
go test -v ./...
```

### Docker Build

```bash
# Build the Docker image
docker build -t ollama-proxy .

# Run the container
docker run -p 8080:8080 ollama-proxy
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Ollama](https://github.com/ollama/ollama) - The original Ollama project
- [Go](https://golang.org/) - The programming language
- [Prometheus](https://prometheus.io/) - For metrics collection
- [Helm](https://helm.sh/) - For Kubernetes deployment

## 📫 Contact

Your Name - [@yourusername](https://twitter.com/yourusername)

Project Link: [https://github.com/yourusername/ollama-proxy](https://github.com/yourusername/ollama-proxy)

## External Services

The proxy requires two external services for validation and metrics collection:

### Validation Service
- **POST** `/validate` - Validates incoming requests
  - Accepts JSON payload with request details
  - Returns validation response with `valid` and `rateLimited` flags
- **GET** `/validate` - Health check endpoint
  - Returns 200 OK if service is available
  - Used for startup validation

### Metrics Service
- **POST** `/log_metrics` - Collects usage metrics
  - Accepts JSON payload with metrics data
  - Returns 200 OK on successful metrics collection
- **GET** `/log_metrics` - Health check endpoint
  - Returns 200 OK if service is available
  - Used for startup validation

Both services must:
- Accept requests with `X-API-Key` header for authentication
- Return appropriate HTTP status codes
- Be accessible at the URLs specified in the configuration 