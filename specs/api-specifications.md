# API Specifications

## REST API Endpoints

### Authentication Endpoints

#### POST /api/auth/register
```json
// Request
{
  "wallet_address": "bdag1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
  "email": "user@example.com",
  "password": "SecurePassword123!"
}

// Response (201 Created)
{
  "success": true,
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "wallet_address": "bdag1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
    "email": "user@example.com",
    "created_at": "2024-01-15T10:30:00Z"
  },
  "message": "User registered successfully"
}
```

#### POST /api/auth/login
```json
// Request
{
  "wallet_address": "bdag1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
  "password": "SecurePassword123!",
  "totp_code": "123456"  // Optional, required if MFA enabled
}

// Response (200 OK)
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400,
    "user": {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "wallet_address": "bdag1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
      "mfa_enabled": true
    }
  }
}
```

#### POST /api/auth/refresh
```json
// Request
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}

// Response (200 OK)
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400
  }
}
```

### MFA Endpoints

#### POST /api/auth/mfa/setup
```json
// Request (Authenticated)
{}

// Response (200 OK)
{
  "success": true,
  "data": {
    "secret": "JBSWY3DPEHPK3PXP",
    "qr_code": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
    "backup_codes": [
      "12345678",
      "87654321",
      "11223344",
      "44332211",
      "55667788"
    ]
  }
}
```

#### POST /api/auth/mfa/verify
```json
// Request (Authenticated)
{
  "totp_code": "123456"
}

// Response (200 OK)
{
  "success": true,
  "data": {
    "mfa_enabled": true
  },
  "message": "MFA enabled successfully"
}
```

### Pool Statistics Endpoints

#### GET /api/pool/stats
```json
// Response (200 OK)
{
  "success": true,
  "data": {
    "hashrate": {
      "current": 1500000000,      // H/s
      "1h": 1450000000,
      "24h": 1400000000,
      "7d": 1350000000
    },
    "miners": {
      "active": 1250,
      "total": 2500
    },
    "blocks": {
      "found_24h": 48,
      "found_7d": 336,
      "last_block": {
        "height": 123456,
        "hash": "0x1234567890abcdef...",
        "found_at": "2024-01-15T10:25:00Z",
        "reward": 5000000000
      }
    },
    "network": {
      "difficulty": 1000000,
      "block_time": 30,
      "height": 123456
    },
    "pool": {
      "fee": 1.0,
      "min_payout": 1000000000,
      "total_paid": 50000000000000
    }
  }
}
```

#### GET /api/pool/blocks
```json
// Query parameters: ?page=1&limit=50&status=confirmed
// Response (200 OK)
{
  "success": true,
  "data": {
    "blocks": [
      {
        "height": 123456,
        "hash": "0x1234567890abcdef...",
        "finder": {
          "worker_name": "miner001",
          "wallet_address": "bdag1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh"
        },
        "difficulty": 1000000,
        "reward": 5000000000,
        "confirmations": 15,
        "status": "confirmed",
        "found_at": "2024-01-15T10:25:00Z",
        "confirmed_at": "2024-01-15T10:30:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 50,
      "total": 1000,
      "pages": 20
    }
  }
}
```

### User Account Endpoints

#### GET /api/user/profile
```json
// Response (200 OK) - Authenticated
{
  "success": true,
  "data": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "wallet_address": "bdag1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh",
    "email": "user@example.com",
    "balance": 2500000000,
    "total_paid": 10000000000,
    "mfa_enabled": true,
    "created_at": "2024-01-01T00:00:00Z",
    "miners": {
      "active": 5,
      "total": 8
    },
    "hashrate": {
      "current": 50000000,
      "24h": 48000000
    }
  }
}
```

#### GET /api/user/miners
```json
// Response (200 OK) - Authenticated
{
  "success": true,
  "data": {
    "miners": [
      {
        "id": "miner-uuid-1",
        "worker_name": "rig001",
        "status": "online",
        "hashrate": {
          "current": 25000000,
          "1h": 24500000,
          "24h": 24000000
        },
        "difficulty": 5000,
        "shares": {
          "accepted": 1500,
          "rejected": 25,
          "invalid": 2
        },
        "last_share": "2024-01-15T10:29:00Z",
        "connected_at": "2024-01-15T08:00:00Z",
        "ip_address": "192.168.1.100"
      }
    ]
  }
}
```

#### GET /api/user/payouts
```json
// Query parameters: ?page=1&limit=50&status=completed
// Response (200 OK) - Authenticated
{
  "success": true,
  "data": {
    "payouts": [
      {
        "id": "payout-uuid-1",
        "amount": 1000000000,
        "fee": 50000000,
        "net_amount": 950000000,
        "transaction_hash": "0xabcdef1234567890...",
        "status": "completed",
        "block": {
          "height": 123450,
          "hash": "0x1234567890abcdef..."
        },
        "created_at": "2024-01-15T06:00:00Z",
        "processed_at": "2024-01-15T06:05:00Z"
      }
    ],
    "summary": {
      "total_paid": 10000000000,
      "pending_amount": 500000000,
      "next_payout": "2024-01-16T06:00:00Z"
    },
    "pagination": {
      "page": 1,
      "limit": 50,
      "total": 100,
      "pages": 2
    }
  }
}
```

### Algorithm Management Endpoints (Admin)

#### GET /api/admin/algorithms/status
```json
// Response (200 OK) - Admin authenticated
{
  "success": true,
  "data": {
    "active_algorithm": {
      "name": "blake2s",
      "version": "1.0.0",
      "loaded_at": "2024-01-15T00:00:00Z",
      "performance": {
        "hashrate": 1500000000,
        "efficiency": 0.95
      },
      "health": "healthy"
    },
    "staged_algorithm": {
      "name": "blake3",
      "version": "1.1.0",
      "staged_at": "2024-01-15T10:00:00Z",
      "validation": {
        "signature_valid": true,
        "compatibility_check": "passed",
        "benchmark_score": 0.98
      },
      "status": "ready"
    },
    "migration_state": null
  }
}
```

#### POST /api/admin/algorithms/stage
```json
// Request (Admin authenticated) - Multipart form data
// File: algorithm_package (ZIP file)

// Response (202 Accepted)
{
  "success": true,
  "data": {
    "staging_id": "staging-uuid-1",
    "algorithm": {
      "name": "blake3",
      "version": "1.1.0"
    },
    "status": "validating",
    "estimated_time": 300
  },
  "message": "Algorithm staging initiated"
}
```

#### POST /api/admin/algorithms/deploy
```json
// Request (Admin authenticated)
{
  "strategy": "gradual",
  "shadow_duration": 300,
  "phase_duration": 600,
  "rollback_on_error": true,
  "notification_channels": ["email", "webhook"]
}

// Response (202 Accepted)
{
  "success": true,
  "data": {
    "deployment_id": "deploy-uuid-1",
    "status": "starting",
    "estimated_duration": 3600,
    "phases": [
      {
        "name": "shadow_mode",
        "duration": 300,
        "traffic_percentage": 1
      },
      {
        "name": "gradual_rollout",
        "duration": 3000,
        "traffic_percentage": 100
      },
      {
        "name": "finalization",
        "duration": 300,
        "traffic_percentage": 100
      }
    ]
  }
}
```

#### GET /api/admin/algorithms/deployment/{deployment_id}
```json
// Response (200 OK) - Admin authenticated
{
  "success": true,
  "data": {
    "deployment_id": "deploy-uuid-1",
    "status": "in_progress",
    "current_phase": "gradual_rollout",
    "progress": 45,
    "traffic_split": {
      "blake2s": 55,
      "blake3": 45
    },
    "metrics": {
      "error_rate": 0.001,
      "performance_delta": 0.03,
      "miner_disconnections": 2
    },
    "started_at": "2024-01-15T10:30:00Z",
    "estimated_completion": "2024-01-15T11:30:00Z"
  }
}
```

#### POST /api/admin/algorithms/rollback
```json
// Request (Admin authenticated)
{
  "deployment_id": "deploy-uuid-1",
  "reason": "Performance degradation detected"
}

// Response (200 OK)
{
  "success": true,
  "data": {
    "rollback_id": "rollback-uuid-1",
    "status": "initiated",
    "estimated_duration": 180
  },
  "message": "Algorithm rollback initiated"
}
```

## WebSocket API

### Connection
```javascript
// Connect to WebSocket
const ws = new WebSocket('wss://pool.blockdag.network/ws');

// Authentication after connection
ws.send(JSON.stringify({
  type: 'auth',
  token: 'jwt_token_here'
}));
```

### Real-time Events

#### Pool Statistics Updates
```json
{
  "type": "pool_stats",
  "data": {
    "hashrate": 1500000000,
    "active_miners": 1250,
    "difficulty": 1000000,
    "last_block": {
      "height": 123457,
      "found_at": "2024-01-15T10:35:00Z"
    }
  },
  "timestamp": "2024-01-15T10:35:05Z"
}
```

#### Miner Status Updates
```json
{
  "type": "miner_status",
  "data": {
    "miner_id": "miner-uuid-1",
    "worker_name": "rig001",
    "status": "online",
    "hashrate": 25000000,
    "shares_accepted": 1505,
    "last_share": "2024-01-15T10:35:00Z"
  },
  "timestamp": "2024-01-15T10:35:05Z"
}
```

#### Block Found Notification
```json
{
  "type": "block_found",
  "data": {
    "height": 123457,
    "hash": "0x1234567890abcdef...",
    "finder": {
      "worker_name": "rig001",
      "wallet_address": "bdag1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh"
    },
    "reward": 5000000000,
    "difficulty": 1000000
  },
  "timestamp": "2024-01-15T10:35:00Z"
}
```

#### Algorithm Migration Progress
```json
{
  "type": "migration_progress",
  "data": {
    "deployment_id": "deploy-uuid-1",
    "phase": "gradual_rollout",
    "progress": 50,
    "traffic_split": {
      "blake2s": 50,
      "blake3": 50
    },
    "eta": "2024-01-15T11:00:00Z"
  },
  "timestamp": "2024-01-15T10:35:05Z"
}
```

## Error Responses

### Standard Error Format
```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid wallet address or password",
    "details": {
      "field": "password",
      "reason": "authentication_failed"
    },
    "suggested_actions": [
      "Check your wallet address and password",
      "Reset your password if forgotten",
      "Contact support if issue persists"
    ],
    "documentation": "https://docs.pool.blockdag.network/auth-errors"
  },
  "timestamp": "2024-01-15T10:35:05Z",
  "request_id": "req-uuid-1"
}
```

### HTTP Status Codes
- `200 OK` - Success
- `201 Created` - Resource created
- `202 Accepted` - Request accepted for processing
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource conflict
- `422 Unprocessable Entity` - Validation errors
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service temporarily unavailable

### Common Error Codes
- `INVALID_CREDENTIALS` - Authentication failed
- `MFA_REQUIRED` - Two-factor authentication required
- `RATE_LIMIT_EXCEEDED` - Too many requests
- `INSUFFICIENT_BALANCE` - Not enough balance for operation
- `MINER_NOT_FOUND` - Miner not found
- `ALGORITHM_NOT_STAGED` - No algorithm staged for deployment
- `MIGRATION_IN_PROGRESS` - Algorithm migration already in progress
- `VALIDATION_ERROR` - Request validation failed
- `INTERNAL_ERROR` - Internal server error

## Rate Limiting

### Limits by Endpoint Type
- **Authentication**: 5 requests per minute per IP
- **Pool Stats**: 60 requests per minute per user
- **User Data**: 100 requests per minute per user
- **Admin Operations**: 10 requests per minute per admin
- **WebSocket**: 1 connection per user, 100 messages per minute

### Rate Limit Headers
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642248000
X-RateLimit-Retry-After: 60
```