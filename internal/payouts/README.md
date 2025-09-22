# PPLNS Payout System

This package implements a Pay Per Last N Shares (PPLNS) payout system for the Chimera Mining Pool. The system ensures fair distribution of block rewards based on miners' recent contributions while maintaining mathematical accuracy and preventing gaming.

## Overview

The PPLNS payout system distributes block rewards proportionally based on the difficulty of valid shares submitted by miners within a sliding window. This approach provides:

- **Fair Distribution**: Rewards are proportional to actual mining contribution
- **Gaming Resistance**: The sliding window prevents pool hopping and other gaming strategies
- **Mathematical Accuracy**: All calculations are validated for correctness
- **Configurable Pool Fees**: Supports configurable pool fee percentages

## Key Components

### PPLNSCalculator

The core calculator that implements the PPLNS algorithm:

```go
calculator, err := NewPPLNSCalculator(windowSize, poolFeePercent)
if err != nil {
    return err
}

payouts, err := calculator.CalculatePayouts(shares, blockReward, blockTime)
```

**Parameters:**
- `windowSize`: Total difficulty window for PPLNS calculation (e.g., 1000000)
- `poolFeePercent`: Pool fee percentage (0-100, e.g., 1.5 for 1.5%)

### PayoutService

High-level service that orchestrates the complete payout workflow:

```go
service := NewPayoutService(database, calculator)

// Process payouts for a confirmed block
err := service.ProcessBlockPayout(ctx, blockID)

// Calculate estimated payout for a user
estimate, err := service.CalculateEstimatedPayout(ctx, userID, estimatedReward)

// Get payout history
history, err := service.GetPayoutHistory(ctx, userID, limit, offset)
```

## PPLNS Algorithm

### Sliding Window

The PPLNS system uses a sliding window based on difficulty rather than time:

1. **Collect Shares**: Gather all valid shares sorted by timestamp (newest first)
2. **Apply Window**: Include shares until the total difficulty reaches the window size
3. **Partial Shares**: If a share partially fits, include only the portion that fits
4. **Calculate Proportions**: Distribute rewards based on each user's difficulty contribution

### Example

With a window size of 1000 difficulty and the following shares:

```
Share 1: User A, 300 difficulty, 10:00 AM
Share 2: User B, 400 difficulty, 10:05 AM  
Share 3: User C, 500 difficulty, 10:10 AM  <- Most recent
```

The window would include:
- Share 3: 500 difficulty (full)
- Share 2: 400 difficulty (full)  
- Share 1: 100 difficulty (partial, only 100 out of 300)

Total window: 1000 difficulty
- User A: 100/1000 = 10%
- User B: 400/1000 = 40%
- User C: 500/1000 = 50%

### Pool Fee Calculation

Pool fees are deducted before distribution:

```
Net Reward = Block Reward × (100 - Pool Fee Percent) / 100
User Payout = Net Reward × (User Difficulty / Total Window Difficulty)
```

## Database Integration

The system requires the following database operations:

```go
type PayoutDatabase interface {
    GetSharesForPayout(ctx context.Context, blockTime time.Time, windowSize int64) ([]Share, error)
    CreatePayouts(ctx context.Context, payouts []Payout) error
    GetBlock(ctx context.Context, blockID int64) (*Block, error)
    GetPayoutHistory(ctx context.Context, userID int64, limit, offset int) ([]Payout, error)
}
```

## Testing

The package includes comprehensive tests:

### Unit Tests (`pplns_test.go`)
- PPLNS calculator validation
- Edge cases (empty shares, zero rewards, etc.)
- Pool fee calculations
- Sliding window behavior

### Service Tests (`service_test.go`)
- Complete payout workflow
- Database integration
- Error handling
- Payout validation

### End-to-End Tests (`e2e_test.go`)
- Realistic mining scenarios
- Multiple blocks and users
- Mathematical accuracy validation
- Sliding window behavior with real data

Run all tests:
```bash
go test -v ./internal/payouts
```

## Usage Examples

### Basic Setup

```go
// Create calculator
calculator, err := NewPPLNSCalculator(1000000, 1.5) // 1M difficulty window, 1.5% fee
if err != nil {
    log.Fatal(err)
}

// Create service with database
service := NewPayoutService(database, calculator)
```

### Process Block Payout

```go
// When a block is found and confirmed
err := service.ProcessBlockPayout(context.Background(), blockID)
if err != nil {
    log.Printf("Failed to process payout: %v", err)
}
```

### Get User Statistics

```go
// Get payout history for a user
history, err := service.GetPayoutHistory(ctx, userID, 50, 0)
if err != nil {
    return err
}

// Get payout statistics
stats, err := service.GetPayoutStatistics(ctx, userID, time.Now().Add(-30*24*time.Hour))
if err != nil {
    return err
}

fmt.Printf("User earned %d satoshis from %d payouts", stats.TotalPayout, stats.PayoutCount)
```

### Validate Payout Fairness

```go
// Validate that payouts are mathematically correct
validation, err := service.ValidatePayoutFairness(ctx, blockID)
if err != nil {
    return err
}

if !validation.IsValid {
    log.Printf("Payout validation failed: %d discrepancies", len(validation.Discrepancies))
    for _, disc := range validation.Discrepancies {
        log.Printf("- %s: %s", disc.Type, disc.Description)
    }
}
```

## Configuration

### Recommended Settings

- **Window Size**: 1-10x average block difficulty
- **Pool Fee**: 0.5-3.0% (typical range)
- **Minimum Payout**: 0.001 coins (to reduce transaction costs)

### Performance Considerations

- The sliding window calculation is O(n) where n is the number of shares
- Database queries should be optimized with proper indexes on timestamp and user_id
- Consider caching recent share data for frequent payout estimations

## Security

The PPLNS system includes several security features:

1. **Input Validation**: All parameters are validated before processing
2. **Mathematical Verification**: Payouts are validated for correctness
3. **Audit Trail**: All payouts are logged with full context
4. **Error Handling**: Comprehensive error handling with detailed messages

## Requirements Satisfied

This implementation satisfies the following requirements:

- **6.2**: Fair distribution algorithm (PPLNS)
- **6.3**: Block reward distribution to contributing miners  
- **6.4**: Automatic payout when threshold is reached

The system ensures mathematical accuracy, fairness, and provides comprehensive testing to validate all functionality.