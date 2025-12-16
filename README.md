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
```
{
  "url": "https://example.com/very-long-url",
  "custom_alias": "mylink", (optional)
  "expiry_hours": 24 (optional)
}
```

**Response**: `201 Created`
```
{
  "short_url": "http://localhost:8080/abc123",
  "short_code": "abc123",
  "original_url": "https://example.com/long-long-long-url",
  "expires_at": "2025-12-15T16:55:13Z"  (null if no expiry set)
}
```

**Field Details**:
- `url` (required): Valid URL to shorten
- `custom_alias` (optional): Custom short code
- `expiry_hours` (optional): URL expiration time in hours

**Error Responses**:
```
// 400 Bad Request - Invalid input
{
  "error": "invalid URL format"
}

// 500 Internal Server Error
{
  "error": "failed to create short URL"
}
```

### 2. Redirect to Original URL

**Endpoint**: `GET /:shortCode`

**Example**: `GET /abc123`

**Response**: `301 Moved Permanently`
- Redirects to original URL
- Increments click counter
- Serves from cache if available

**Error Response**: `404 Not Found`
```json
{
  "error": "URL not found or expired"
}
```

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

```bash
# Server
SERVER_PORT=

DB_HOST=
DB_PORT=
DB_USER=
DB_PASSWORD=
DB_NAME=

REDIS_HOST=
REDIS_PORT=
REDIS_PASSWORD=
REDIS_DB=
```