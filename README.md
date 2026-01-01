# URL Shortener

> A high-performance URL shortening service with Redis caching and PostgreSQL persistence. Built with clean architecture for optimal scalability.

## Performance & Scale

**Load Test Results** (2 CPU cores, 2GB RAM, 10M records)

| Metric | Baseline  | Optimized   | Improvement |
|--------|-----------|-------------|-------------|
| **Throughput (RPS)** | 757 req/s | 1,062 req/s | **+40%** |
| **P95 Latency** | 529ms     | 83ms        | **-84%** |
| **P99 Latency** | 1,100ms   | 400ms       | **-64%** |

**Key Optimizations:**
- **Cache-aside pattern with Redis** - Reducing DB load by 6x for read-heavy workload (90% reads)
- **Connection pooling** - Reduced connection overhead, +40% throughput
- **Database indexing** - Optimized queries on 10M+ records for sub-100ms P95 latency
- **Dependency injection with clean architecture** - Repository pattern enables easy horizontal scaling

**Test Methodology:** k6 load testing with 1,000+ concurrent virtual users, 8-minute duration, and realistic traffic distribution (zipfian)

## Code Quality

- **Test Coverage:** >95% (unit + integration tests)
- **Architecture:** Clean architecture with dependency injection
- **Containerized:** Docker + Docker Compose for consistent environments

## API Endpoints

### 1. Create Short URL
**Endpoint**: `POST /api/shorten`

**Request Body**:
```json
{
  "url": "https://example.com/very-long-url",
  "custom_alias": "mylink",
  "expiry_hours": 24
}
```

**Field Details**:
- `url` (required): Valid URL to shorten
- `custom_alias` (optional): Custom short code (alphanumeric)
- `expiry_hours` (optional): URL expiration time in hours

**Success Response**: `201 Created`
```json
{
  "short_url": "http://localhost:8080/abc123",
  "short_code": "abc123",
  "original_url": "https://example.com/very-long-url",
  "expires_at": "2025-12-27T10:30:00Z"
}
```

**Error Responses**:
- `400 Bad Request`: Invalid URL format or custom alias already taken
- `500 Internal Server Error`: Failed to create short URL

---

### 2. Redirect to Original URL
**Endpoint**: `GET /:shortCode`

**Example**: `GET /abc123`

**Response**: `301 Moved Permanently`
- Redirects to original URL
- Tracks click analytics (timestamp, user agent, IP)
- Utilizes Redis cache for faster lookups

**Error Response**: `404 Not Found` - URL not found or expired

---

### 3. Get URL Analytics
**Endpoint**: `GET /api/analytics/:shortCode`

**Query Parameters**:
- `days` (optional): Number of days for analytics (default: 30, max: 365)

**Example**: `GET /api/analytics/abc123?days=7`

**Success Response**: `200 OK`
```json
{
  "status": "success",
  "message": "Analytics retrieved successfully",
  "data": {
    "short_code": "abc123",
    "original_url": "https://example.com/very-long-url",
    "total_clicks": 150,
    "created_at": "2025-12-26T08:00:00Z",
    "expires_at": "2025-12-27T10:30:00Z"
  }
}
```

**Error Responses**:
- `400 Bad Request`: Short code is required
- `404 Not Found`: URL not found

---

### 4. Get Click History
**Endpoint**: `GET /api/analytics/:shortCode/clicks`

**Query Parameters**:
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 20, max: 100)

**Example**: `GET /api/analytics/abc123/clicks?page=1&page_size=20`

**Success Response**: `200 OK`
```json
{
  "status": "success",
  "message": "Click history retrieved successfully",
  "data": {
    "short_code": "abc123",
    "clicks": [
      {
        "clicked_at": "2025-12-26T14:30:00Z",
        "user_agent": "Mozilla/5.0...",
        "ip_address": "192.168.1.1"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 150
    }
  }
}
```

**Error Responses**:
- `400 Bad Request`: Short code is required
- `404 Not Found`: URL not found

---

### 5. Health Check Endpoints

#### Liveness Check
**Endpoint**: `GET /healthz`

**Response**: `200 OK`
```json
{
  "status": "ok",
  "timestamp": "2025-12-26T14:30:00Z"
}
```

#### Readiness Check
**Endpoint**: `GET /readyz`

**Success Response**: `200 OK`
```json
{
  "status": "up",
  "checks": {
    "database": {
      "status": "up",
      "message": "connected"
    },
    "redis": {
      "status": "up",
      "message": "connected"
    }
  },
  "metadata": {
    "version": "1.0.0",
    "timestamp": "2025-12-26T14:30:00Z"
  }
}
```

**Error Response**: `503 Service Unavailable`
```json
{
  "status": "down",
  "checks": {
    "database": {
      "status": "down",
      "message": "connection timeout"
    },
    "redis": {
      "status": "up",
      "message": "connected"
    }
  },
  "metadata": {
    "version": "1.0.0",
    "timestamp": "2025-12-26T14:30:00Z"
  }
}
```

## Quick Start
```bash
# Clone repository
git clone https://github.com/yourusername/url-shortener.git

# Start services
docker-compose up

# Run migrations
make migrate-up

# Run
make run

```

## Testing
```bash
# Unit tests
make test

# Integration tests
make test-integration
```

## Configuration

Create a `.env` file in the root directory with the following variables:
```bash
# Server Configuration
SERVER_PORT=8080
SERVER_SHUTDOWN_TIMEOUT=10s
SERVER_READ_TIMEOUT=10s
SERVER_WRITE_TIMEOUT=10s

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=urlshortener
DB_MAX_CONNS=25
DB_MIN_CONNS=5
DB_CONN_MAX_LIFETIME=1h
DB_CONN_MAX_IDLE_TIME=30m

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONNS=5
REDIS_MAX_RETRIES=3

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT_PATH=logs/app.log
LOG_MAX_SIZE=100
LOG_MAX_BACKUPS=3
LOG_MAX_AGE=28
LOG_COMPRESS=true
```

> Or copy `.env.example` to `.env` and adjust values for your environment.
