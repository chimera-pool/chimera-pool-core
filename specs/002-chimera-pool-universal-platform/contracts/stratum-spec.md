# Stratum Protocol Specification for Chimera Pool

## Overview

This document specifies the Stratum protocol implementation for the Chimera Pool Universal Platform, including standard Stratum v1 compatibility and custom extensions for multi-cryptocurrency and hot-swappable algorithm support.

## Standard Stratum v1 Protocol

### Connection Establishment

```json
// Client connects to stratum+tcp://pool.example.com:3333
// Server sends mining.notify immediately after connection
```

### Core Methods

#### mining.subscribe
**Client Request:**
```json
{
  "id": 1,
  "method": "mining.subscribe",
  "params": ["cpuminer/2.5.0", null, "pool.example.com", 3333]
}
```

**Server Response:**
```json
{
  "id": 1,
  "result": [
    [
      ["mining.set_difficulty", "subscription_id_1"],
      ["mining.notify", "subscription_id_2"]
    ],
    "extranonce1",
    4
  ],
  "error": null
}
```

#### mining.authorize
**Client Request:**
```json
{
  "id": 2,
  "method": "mining.authorize",
  "params": ["wallet_address.worker_name", "password"]
}
```

**Server Response:**
```json
{
  "id": 2,
  "result": true,
  "error": null
}
```

#### mining.submit
**Client Request:**
```json
{
  "id": 3,
  "method": "mining.submit",
  "params": [
    "wallet_address.worker_name",
    "job_id",
    "extranonce2",
    "ntime",
    "nonce"
  ]
}
```

**Server Response:**
```json
{
  "id": 3,
  "result": true,
  "error": null
}
```

### Server Notifications

#### mining.set_difficulty
```json
{
  "id": null,
  "method": "mining.set_difficulty",
  "params": [1024]
}
```

#### mining.notify
```json
{
  "id": null,
  "method": "mining.notify",
  "params": [
    "job_id",
    "prevhash",
    "coinb1",
    "coinb2",
    ["merkle_branch"],
    "version",
    "nbits",
    "ntime",
    true
  ]
}
```

## Chimera Pool Extensions

### Multi-Cryptocurrency Support

#### mining.set_coin
**Description:** Allows miners to specify which cryptocurrency to mine

**Client Request:**
```json
{
  "id": 4,
  "method": "mining.set_coin",
  "params": ["bitcoin"]
}
```

**Server Response:**
```json
{
  "id": 4,
  "result": {
    "coin": "bitcoin",
    "algorithm": "sha256",
    "difficulty": 1000000,
    "block_reward": 6.25
  },
  "error": null
}
```

**Supported Coins:**
- `bitcoin` - Bitcoin (SHA-256)
- `ethereum-classic` - Ethereum Classic (Ethash)
- `blockdag` - BlockDAG (Blake3)
- `litecoin` - Litecoin (Scrypt)
- `dash` - Dash (X11)
- `monero` - Monero (RandomX)
- `zcash` - Zcash (Equihash)

#### mining.multi_version
**Description:** Enables support for multiple algorithm versions simultaneously

**Client Request:**
```json
{
  "id": 5,
  "method": "mining.multi_version",
  "params": {
    "supported_algorithms": ["sha256", "blake3", "scrypt"],
    "preferred_algorithm": "blake3"
  }
}
```

**Server Response:**
```json
{
  "id": 5,
  "result": {
    "active_algorithm": "blake3",
    "fallback_algorithms": ["sha256"],
    "algorithm_switching": true
  },
  "error": null
}
```

### Enhanced Statistics and Monitoring

#### mining.get_statistics
**Description:** Retrieves detailed mining statistics for the worker

**Client Request:**
```json
{
  "id": 6,
  "method": "mining.get_statistics",
  "params": ["wallet_address.worker_name"]
}
```

**Server Response:**
```json
{
  "id": 6,
  "result": {
    "worker_name": "worker_name",
    "hashrate": 1500000000,
    "hashrate_1h": 1450000000,
    "hashrate_24h": 1400000000,
    "shares_accepted": 1250,
    "shares_rejected": 15,
    "shares_stale": 5,
    "last_share_time": "2025-09-21T10:30:00Z",
    "difficulty": 1024,
    "estimated_earnings": {
      "hourly": 0.00012,
      "daily": 0.00288,
      "weekly": 0.02016
    },
    "pool_stats": {
      "pool_hashrate": 50000000000000,
      "active_miners": 1500,
      "blocks_found_24h": 12,
      "current_difficulty": 25000000000000
    }
  },
  "error": null
}
```

#### mining.configure_notifications
**Description:** Configures custom notifications for the miner

**Client Request:**
```json
{
  "id": 7,
  "method": "mining.configure_notifications",
  "params": {
    "block_found": true,
    "difficulty_change": true,
    "algorithm_switch": true,
    "payout_processed": true,
    "high_reject_rate": {
      "enabled": true,
      "threshold": 5.0
    }
  }
}
```

**Server Response:**
```json
{
  "id": 7,
  "result": {
    "notifications_configured": true,
    "supported_notifications": [
      "block_found",
      "difficulty_change", 
      "algorithm_switch",
      "payout_processed",
      "high_reject_rate"
    ]
  },
  "error": null
}
```

### Hot-Swappable Algorithm Support

#### mining.algorithm_switch_notify
**Description:** Server notification when algorithm is being switched

**Server Notification:**
```json
{
  "id": null,
  "method": "mining.algorithm_switch_notify",
  "params": {
    "new_algorithm": "blake3",
    "switch_time": "2025-09-21T11:00:00Z",
    "preparation_time": 300,
    "reason": "scheduled_upgrade",
    "migration_id": "migration_001"
  }
}
```

#### mining.algorithm_switch_confirm
**Description:** Client confirmation of algorithm switch readiness

**Client Request:**
```json
{
  "id": 8,
  "method": "mining.algorithm_switch_confirm",
  "params": {
    "migration_id": "migration_001",
    "ready": true,
    "supported_algorithms": ["blake3", "sha256"]
  }
}
```

**Server Response:**
```json
{
  "id": 8,
  "result": {
    "confirmed": true,
    "switch_scheduled": "2025-09-21T11:00:00Z"
  },
  "error": null
}
```

### Enhanced Error Handling

#### Error Codes
```json
{
  "id": 1,
  "result": null,
  "error": [
    20, // Error code
    "Job not found", // Error message
    null // Additional data
  ]
}
```

**Standard Error Codes:**
- `20` - Other/Unknown error
- `21` - Job not found
- `22` - Duplicate share
- `23` - Low difficulty share
- `24` - Unauthorized worker
- `25` - Not subscribed

**Chimera Pool Extended Error Codes:**
- `100` - Unsupported algorithm
- `101` - Algorithm switching in progress
- `102` - Coin not supported
- `103` - Pool maintenance mode
- `104` - Rate limit exceeded
- `105` - Invalid worker configuration
- `106` - Wallet address validation failed

### Performance Optimizations

#### Connection Keep-Alive
```json
{
  "id": null,
  "method": "mining.ping",
  "params": []
}
```

**Client Response:**
```json
{
  "id": null,
  "method": "mining.pong",
  "params": []
}
```

#### Batch Operations
**Client Request (Multiple Submits):**
```json
[
  {
    "id": 10,
    "method": "mining.submit",
    "params": ["worker1", "job1", "extranonce2_1", "ntime1", "nonce1"]
  },
  {
    "id": 11,
    "method": "mining.submit", 
    "params": ["worker1", "job1", "extranonce2_2", "ntime2", "nonce2"]
  }
]
```

**Server Response:**
```json
[
  {"id": 10, "result": true, "error": null},
  {"id": 11, "result": false, "error": [22, "Duplicate share", null]}
]
```

## Protocol Flow Examples

### Standard Mining Flow
```
1. Client connects to stratum+tcp://pool.example.com:3333
2. Server sends mining.notify with initial job
3. Client sends mining.subscribe
4. Server responds with subscription details
5. Client sends mining.authorize
6. Server responds with authorization result
7. Server sends mining.set_difficulty
8. Client mines and sends mining.submit
9. Server responds with acceptance/rejection
10. Server sends new mining.notify when job changes
```

### Multi-Coin Mining Flow
```
1. Standard connection and subscription
2. Client sends mining.set_coin with desired cryptocurrency
3. Server responds with coin-specific parameters
4. Server sends mining.notify with coin-specific job
5. Client mines using appropriate algorithm
6. Standard submit/response cycle
```

### Algorithm Switch Flow
```
1. Server sends mining.algorithm_switch_notify
2. Client prepares for algorithm switch
3. Client sends mining.algorithm_switch_confirm
4. Server coordinates switch across all miners
5. Server sends new mining.notify with new algorithm
6. Client switches to new algorithm
7. Mining continues with new algorithm
```

## Security Considerations

### Connection Security
- Support for SSL/TLS encryption (stratum+ssl://)
- Certificate validation for secure connections
- Optional client certificate authentication

### Authentication
- Worker name validation against registered users
- Optional password-based authentication
- Rate limiting for failed authentication attempts

### Anti-Abuse Measures
- Connection rate limiting per IP address
- Share submission rate limiting per worker
- Automatic disconnection for invalid submissions
- IP-based blocking for malicious behavior

## Performance Specifications

### Connection Handling
- Support for 10,000+ concurrent connections per server
- Sub-100ms response times for all operations
- Automatic connection cleanup and resource management
- Efficient memory usage with connection pooling

### Protocol Efficiency
- Minimal bandwidth usage through efficient JSON encoding
- Optional compression for high-frequency data
- Batch operations support for reduced round trips
- Keep-alive mechanisms to reduce connection overhead

### Scalability
- Horizontal scaling across multiple Stratum servers
- Load balancing with session affinity
- Automatic failover and redundancy
- Geographic distribution support

## Implementation Notes

### Backward Compatibility
- Full compatibility with existing Stratum v1 miners
- Graceful degradation when extensions are not supported
- Optional feature negotiation during subscription
- Legacy miner support with standard protocol only

### Extension Detection
```json
// During mining.subscribe, client can indicate extension support
{
  "id": 1,
  "method": "mining.subscribe",
  "params": [
    "cpuminer/2.5.0",
    null,
    "pool.example.com",
    3333,
    {
      "extensions": ["multi_coin", "algorithm_switch", "enhanced_stats"]
    }
  ]
}
```

### Monitoring and Debugging
- Comprehensive logging of all protocol interactions
- Performance metrics collection and reporting
- Debug mode with detailed protocol tracing
- Health check endpoints for monitoring systems

This Stratum protocol specification ensures full compatibility with existing mining software while providing advanced features for the Chimera Pool Universal Platform's multi-cryptocurrency and hot-swappable algorithm capabilities.

