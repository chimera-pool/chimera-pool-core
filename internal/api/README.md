# Chimera Pool REST API

This package implements the REST API for the Chimera Mining Pool, providing comprehensive endpoints for pool statistics, user management, and multi-factor authentication.

## Overview

The API is built following Test-Driven Development (TDD) principles and implements the requirements:
- **Requirement 7.1**: Real-time hashrate statistics
- **Requirement 7.2**: Block discovery metrics updates  
- **Requirement 21.1**: MFA setup with authenticator apps

## Architecture

### Components

- **Handlers** (`handlers.go`): HTTP request handlers and routing
- **Services** (`services.go`): Business logic and data access
- **Models** (`models.go`): Data structures and interfaces
- **Tests**: Comprehensive test suite covering unit, integration, E2E, performance, and security testing

### Key Features

1. **Authentication & Authorization**
   - JWT-based authentication
   - Secure token validation
   - Protected endpoints with middleware

2. **Real-time Statistics** (Requirement 7.1)
   - Current and average hashrate
   - Active miners count
   - Shares per second
   - Pool efficiency metrics

3. **Block Discovery Metrics** (Requirement 7.2)
   - Total blocks found
   - Recent block statistics (24h, 7d)
   - Average block time
   - Orphan block tracking

4. **Multi-Factor Authentication** (Requirement 21.1)
   - TOTP setup with QR codes
   - Backup codes generation
   - Secure verification flow

5. **User Management**
   - Profile management
   - Mining statistics
   - Miner device tracking

## API Endpoints

### Public Endpoints

```
GET  /health                    - Health check
GET  /api/v1/pool/stats         - Pool statistics
GET  /api/v1/pool/realtime      - Real-time statistics
GET  /api/v1/pool/blocks        - Block discovery metrics
```

### Protected Endpoints (Require Authentication)

```
GET  /api/v1/user/profile       - Get user profile
PUT  /api/v1/user/profile       - Update user profile
GET  /api/v1/user/stats         - Get user mining statistics
GET  /api/v1/user/miners        - Get user's miners
GET  /api/v1/user/miners/:id/stats - Get specific miner stats

POST /api/v1/user/mfa/setup     - Setup MFA
POST /api/v1/user/mfa/verify    - Verify MFA code
```

## Authentication

The API uses JWT (JSON Web Tokens) for authentication. Include the token in the Authorization header:

```
Authorization: Bearer <jwt-token>
```

### Token Structure

```json
{
  "user_id": 123,
  "username": "testuser",
  "email": "test@example.com",
  "iat": 1640995200,
  "exp": 1641081600
}
```

## Request/Response Examples

### Get Pool Statistics

```bash
curl -X GET http://localhost:8080/api/v1/pool/stats
```

Response:
```json
{
  "total_hashrate": 1000000.0,
  "connected_miners": 150,
  "total_shares": 50000,
  "valid_shares": 49500,
  "invalid_shares": 500,
  "blocks_found": 25,
  "last_block_time": "2024-01-01T12:00:00Z",
  "network_hashrate": 50000000.0,
  "network_difficulty": 1000000.0,
  "pool_fee": 1.0,
  "efficiency": 99.0
}
```

### Get Real-time Statistics

```bash
curl -X GET http://localhost:8080/api/v1/pool/realtime
```

Response:
```json
{
  "current_hashrate": 1500000.0,
  "average_hashrate": 1200000.0,
  "active_miners": 175,
  "shares_per_second": 25.5,
  "last_block_found": "2024-01-01T11:55:00Z",
  "network_difficulty": 1500000.0,
  "pool_efficiency": 99.2,
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Setup MFA

```bash
curl -X POST http://localhost:8080/api/v1/user/mfa/setup \
  -H "Authorization: Bearer <jwt-token>"
```

Response:
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code_url": "otpauth://totp/ChimeraPool:testuser?secret=JBSWY3DPEHPK3PXP&issuer=ChimeraPool",
  "backup_codes": ["12345678", "87654321", "11223344", "44332211", "55667788"]
}
```

### Verify MFA

```bash
curl -X POST http://localhost:8080/api/v1/user/mfa/verify \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"code": "123456"}'
```

Response:
```json
{
  "verified": true,
  "message": "MFA enabled successfully"
}
```

## Error Handling

The API returns consistent error responses:

```json
{
  "error": "error_code",
  "message": "Human readable error message",
  "code": 400
}
```

### Common Error Codes

- `missing_token` - Authorization header missing
- `invalid_token` - JWT token invalid or expired
- `validation_error` - Request validation failed
- `internal_error` - Server error occurred
- `invalid_mfa_code` - MFA verification failed

## Security Features

1. **JWT Authentication**: Secure token-based authentication
2. **Input Validation**: Comprehensive request validation
3. **SQL Injection Prevention**: Parameterized queries
4. **XSS Prevention**: Input sanitization
5. **Rate Limiting**: Protection against brute force attacks
6. **MFA Support**: Two-factor authentication with TOTP

## Performance

The API is designed to meet performance requirements:
- Sub-100ms response times for most endpoints
- Support for 1000+ concurrent connections
- Efficient real-time statistics updates
- Optimized database queries

## Testing

The implementation includes comprehensive testing:

### Test Types

1. **Unit Tests** (`handlers_test.go`)
   - Individual handler testing
   - Mock service integration
   - Error condition coverage

2. **End-to-End Tests** (`e2e_test.go`)
   - Complete workflow testing
   - Authentication flows
   - MFA setup and verification

3. **Performance Tests** (`performance_test.go`)
   - Concurrent request handling
   - Response time validation
   - Throughput testing

4. **Security Tests** (`security_test.go`)
   - Authentication bypass attempts
   - Input validation testing
   - SQL injection prevention
   - XSS prevention

5. **Integration Tests** (`integration_test.go`)
   - Real service integration
   - Database interaction
   - JWT token lifecycle

### Running Tests

```bash
# Run all tests
go test ./internal/api/... -v

# Run specific test types
go test ./internal/api/... -v -run TestUnit
go test ./internal/api/... -v -run TestE2E
go test ./internal/api/... -v -run TestPerformance
go test ./internal/api/... -v -run TestSecurity
go test ./internal/api/... -v -run TestIntegration

# Run benchmarks
go test ./internal/api/... -bench=.
```

## Configuration

The API can be configured through environment variables:

```bash
# JWT Configuration
JWT_SECRET=your-secret-key

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=chimera_pool
DB_USER=pool_user
DB_PASSWORD=pool_password

# Server Configuration
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60s
```

## Dependencies

- **Gin**: HTTP web framework
- **JWT-Go**: JWT token handling
- **Testify**: Testing framework
- **BCrypt**: Password hashing
- **PostgreSQL**: Database driver

## Development

### Adding New Endpoints

1. Define models in `models.go`
2. Add service methods in `services.go`
3. Implement handlers in `handlers.go`
4. Add routes in `SetupAPIRoutes`
5. Write comprehensive tests

### Testing Guidelines

- Follow TDD approach: write tests first
- Achieve 100% test coverage
- Include error condition testing
- Test security scenarios
- Validate performance requirements

## Deployment

The API is designed for containerized deployment:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api ./cmd/api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api .
CMD ["./api"]
```

## Monitoring

The API includes built-in monitoring capabilities:
- Health check endpoint
- Metrics collection
- Error logging
- Performance tracking

## Future Enhancements

1. **WebSocket Support**: Real-time data streaming
2. **GraphQL API**: Flexible query interface
3. **API Versioning**: Backward compatibility
4. **Advanced Rate Limiting**: Per-user limits
5. **Audit Logging**: Security event tracking