# Pool Manager

The Pool Manager is the central orchestration component of the Chimera Mining Pool. It coordinates all mining pool components and manages the complete mining workflow from miner connections to payout processing.

## Overview

The Pool Manager implements the requirements specified in:
- **Requirement 2.1**: Stratum Protocol Compatibility
- **Requirement 6.1**: Pool Mining Functionality (share recording and validation)
- **Requirement 6.2**: Pool Mining Functionality (fair payout distribution)

## Architecture

The Pool Manager follows a dependency injection pattern and coordinates the following components:

- **Stratum Server**: Handles miner connections and Stratum protocol communication
- **Share Processor**: Validates and processes mining shares using Blake2S algorithm
- **Authentication Service**: Manages user authentication and JWT tokens
- **Payout Service**: Calculates and processes PPLNS payouts

## Key Features

### Component Coordination
- Orchestrates startup and shutdown of all pool components
- Monitors component health and provides status reporting
- Handles graceful error recovery and component failover

### Mining Workflow Management
- Processes mining shares through the complete validation pipeline
- Coordinates share validation with the Blake2S algorithm engine
- Manages real-time statistics and performance metrics

### Status Monitoring
- Provides real-time pool status and statistics
- Monitors component health with automatic health checks
- Tracks miner connections and mining performance

### End-to-End Integration
- Supports complete mining workflow from authentication to payout
- Validates all components work together correctly
- Provides comprehensive testing capabilities

## Usage

### Basic Usage

```go
// Create configuration
config := &PoolManagerConfig{
    StratumAddress: ":3333",
    MaxMiners:      1000,
    BlockReward:    5000000000, // 50 coins in satoshis
}

// Create component instances
stratumServer := stratum.NewStratumServer(config.StratumAddress)
shareProcessor := shares.NewShareProcessor()
authService := auth.NewAuthService(userRepo, "jwt_secret")
payoutService := payouts.NewPayoutService(db, calculator)

// Create pool manager
manager := poolmanager.NewPoolManager(
    config,
    stratumServer,
    shareProcessor,
    authService,
    payoutService,
)

// Start the pool
err := manager.Start()
if err != nil {
    log.Fatal("Failed to start pool:", err)
}

// Process shares
share := &poolmanager.Share{
    ID:         1,
    MinerID:    123,
    UserID:     456,
    JobID:      "job123",
    Nonce:      "deadbeef",
    Difficulty: 1.0,
    Timestamp:  time.Now(),
}

result := manager.ProcessShare(share)
if result.Success {
    log.Printf("Share processed: %+v", result.ProcessedShare)
}

// Get pool statistics
stats := manager.GetPoolStatistics()
log.Printf("Pool stats: %+v", stats)

// Stop the pool
err = manager.Stop()
if err != nil {
    log.Error("Failed to stop pool:", err)
}
```

### Status Monitoring

```go
// Get current pool status
status := manager.GetStatus()
fmt.Printf("Pool Status: %s\n", status.Status)
fmt.Printf("Connected Miners: %d\n", status.ConnectedMiners)
fmt.Printf("Total Shares: %d\n", status.TotalShares)
fmt.Printf("Valid Shares: %d\n", status.ValidShares)

// Check component health
health := manager.GetComponentHealth()
fmt.Printf("Stratum Server: %s\n", health.StratumServer.Status)
fmt.Printf("Share Processor: %s\n", health.ShareProcessor.Status)
fmt.Printf("Auth Service: %s\n", health.AuthService.Status)
fmt.Printf("Payout Service: %s\n", health.PayoutService.Status)
```

## Testing

The Pool Manager includes comprehensive test coverage:

### Unit Tests
- Component creation and configuration
- Start/stop lifecycle management
- Share processing workflow
- Status and statistics reporting
- Component health monitoring

### Integration Tests
- Real component integration testing
- End-to-end workflow validation
- Component coordination verification

### End-to-End Tests
- Complete mining workflow from authentication to payout
- Multi-share processing scenarios
- Statistics and health monitoring validation

Run tests:
```bash
go test ./internal/poolmanager -v
go test ./internal/poolmanager -v -run TestEndToEndMiningWorkflow
go test ./internal/poolmanager -v -cover
```

## Configuration

### PoolManagerConfig

```go
type PoolManagerConfig struct {
    StratumAddress string `json:"stratum_address"` // Stratum server listen address
    MaxMiners      int    `json:"max_miners"`      // Maximum concurrent miners
    BlockReward    int64  `json:"block_reward"`    // Block reward in satoshis
}
```

## Status Types

### PoolStatus
- `PoolStatusStopped`: Pool is not running
- `PoolStatusStarting`: Pool is starting up
- `PoolStatusRunning`: Pool is fully operational
- `PoolStatusStopping`: Pool is shutting down
- `PoolStatusError`: Pool encountered an error

### Component Health
Each component reports health status:
- `healthy`: Component is operating normally
- `unhealthy`: Component has issues but may recover
- `error`: Component has failed and needs intervention

## Error Handling

The Pool Manager implements comprehensive error handling:

- **Startup Errors**: If any component fails to start, the pool manager stops all components and reports the error
- **Runtime Errors**: Component failures are logged and monitored, with automatic recovery attempts
- **Shutdown Errors**: Clean shutdown is attempted for all components, with errors logged but not blocking

## Performance

The Pool Manager is designed for high performance:

- **Non-blocking Operations**: All operations use non-blocking I/O where possible
- **Concurrent Processing**: Share processing and component monitoring run concurrently
- **Efficient Statistics**: Statistics are updated incrementally with minimal overhead
- **Resource Management**: Proper cleanup and resource management prevent memory leaks

## Thread Safety

All Pool Manager operations are thread-safe:

- **Status Management**: Protected by read-write mutexes
- **Component Access**: Safe concurrent access to all components
- **Statistics Updates**: Atomic operations for performance counters
- **Health Monitoring**: Background monitoring with proper synchronization