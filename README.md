# 🚀 Microservice Framework

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](https://docker.com)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)

A production-ready Go microservice framework with clean architecture, comprehensive monitoring, and DevOps best
practices.

> 🎯 **Perfect starting point** for building scalable microservices with observability built-in

## Features

- 🏗️ **Clean Architecture** with DDD principles
- 🐳 **Docker & Docker Compose** ready
- 📊 **Complete Monitoring Stack** (Prometheus + Grafana)
- 🔍 **OpenTelemetry** metrics with proper histograms
- 🔄 **Database Migrations** with automatic setup
- 💾 **PostgreSQL & Redis** integration
- ⚡ **Health Checks** for Kubernetes/Docker
- 🛡️ **Security** with CORS, rate limiting
- 🔧 **Environment-based Configuration**

## 🚀 Quick Start

```bash
# 1. Copy environment configuration
cp .env.example .env

# 2. Start full stack with monitoring
make docker-up

# 3. Access services
# API: http://localhost:${HTTP_SERVER_PORT}
# Grafana: http://localhost:3000 (admin/admin123)
# Prometheus: http://localhost:9090
```

## 🏛️ Architecture

```
├── cmd/                    # Application entry points
├── internal/
│   ├── adapters/           # External interfaces (HTTP, DB, etc.)
│   ├── core/               # Business logic & domain
│   ├── config/             # Configuration management
│   └── platform/           # Shared infrastructure
├── infrastructure/         # Infrastructure configs & monitoring
├── migrations/             # Database schema changes
└── scripts/                # Automation scripts
```

### Key Architectural Principles

- **Clean Architecture** with dependency inversion
- **Domain-Driven Design** for business logic
- **Hexagonal Architecture** pattern
- **OpenTelemetry** observability

## 💻 Development

### Local Development

```bash
# Run without Docker (fast development)
go run cmd/http-server/main.go

# With full stack (database + monitoring)
make docker-up
```

### Available Commands

```bash
make help               # Show all available commands
make build              # Build the application
make test               # Run tests
make lint               # Run linter
make docker-up          # Start all services
make docker-down        # Stop all services
make migrate-up         # Run database migrations
```

## 🌐 API Endpoints

| Method | Endpoint             | Description        | Status  |
|--------|----------------------|--------------------|---------|
| GET    | `/health/live`       | Liveness probe     | ✅ Ready |
| GET    | `/health/ready`      | Readiness probe    | ✅ Ready |
| GET    | `/metrics`           | Prometheus metrics | ✅ Ready |
| POST   | `/api/examples`      | Create example     | ✅ Ready |
| GET    | `/api/examples/{id}` | Get example        | ✅ Ready |

## 📊 Monitoring & Observability

### Metrics (Prometheus)

- **HTTP request duration** (proper histograms)
- **Request count** by status code
- **Requests in flight** counter
- **Database connection pool** metrics

### Dashboards (Grafana)

- **HTTP Request Overview** - Response times, throughput
- **Database Performance** - Connection pools, query times
- **System Resources** - CPU, memory, disk usage
- **Custom business metrics** - Your domain-specific KPIs

## ⚙️ Configuration

Environment variables (copy `.env.example` to `.env`):

| Variable            | Default        | Description        |
|---------------------|----------------|--------------------|
| `ENV`               | `development`  | Environment mode   |
| `HTTP_SERVER_PORT`  | `8080`         | API server port    |
| `POSTGRES_HOST`     | `postgres`     | Database host      |
| `POSTGRES_PASSWORD` | -              | Database password  |
| `SERVICE_NAME`      | `microservice` | Service identifier |

## 🔧 Extending the Framework

### Adding New Domain

1. Create domain entity: `internal/core/domain/newdomain/`
2. Define ports: `internal/core/ports/newdomain.go`
3. Implement use cases: `internal/core/usecase/newdomain/`
4. Add repositories: `internal/adapters/repository/newdomain_*/`
5. Create HTTP handlers: `internal/adapters/http/newdomain/`
6. Wire up in dependency injection

### Adding New Storage Backend

1. Create adapter: `internal/adapters/repository/example_newstorage/`
2. Implement repository interface
3. Add to dependency injection modules

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Generate mocks
make mocks
```

## 🚀 Deployment

### Docker

```bash
# Build image
make docker-build

# Deploy with compose
docker-compose -f docker-compose.yml up
```

## ✅ Production Checklist

- [ ] Set `ENV=production`
- [ ] Configure PostgreSQL connection
- [ ] Set up proper logging level
- [ ] Configure CORS for your domain
- [ ] Set appropriate rate limits
- [ ] Set up health check monitoring
- [ ] Configure metrics collection
- [ ] Set up log aggregation
- [ ] Configure alerts in Grafana
- [ ] Set up backup strategies

## 📄 License

This project is licensed under the MIT License. Feel free to use it as a template for your microservices.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

---

<div align="center">

**⭐ Star this repo if it helped you build amazing microservices! ⭐**

[![GitHub stars](https://img.shields.io/github/stars/yourusername/microservice-framework?style=social)](https://github.com/yourusername/microservice-framework/stargazers)

</div>
