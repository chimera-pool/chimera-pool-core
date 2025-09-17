# Technical Specifications

## BlockDAG Network Integration

### Network Connection
```go
type BlockDAGClient struct {
    endpoint    string
    apiKey      string
    timeout     time.Duration
    retryCount  int
}

type BlockDAGConfig struct {
    NetworkType     string `json:"network_type"`     // "mainnet", "testnet"
    RPCEndpoint     string `json:"rpc_endpoint"`     // "https://api.blockdag.network"
    WebSocketURL    string `json:"websocket_url"`    // "wss://ws.blockdag.network"
    APIKey          string `json:"api_key"`
    ChainID         int    `json:"chain_id"`
    BlockTime       int    `json:"block_time"`       // seconds
    Confirmations   int    `json:"confirmations"`    // required confirmations
}
```

### Block Structure
```go
type Block struct {
    Hash         string    `json:"hash"`
    Height       uint64    `json:"height"`
    Timestamp    int64     `json:"timestamp"`
    Difficulty   uint64    `json:"difficulty"`
    Nonce        uint64    `json:"nonce"`
    PrevHash     string    `json:"prev_hash"`
    MerkleRoot   string    `json:"merkle_root"`
    Transactions []Transaction `json:"transactions"`
    Reward       uint64    `json:"reward"`
}

type Transaction struct {
    Hash      string  `json:"hash"`
    From      string  `json:"from"`
    To        string  `json:"to"`
    Amount    uint64  `json:"amount"`
    Fee       uint64  `json:"fee"`
    Signature string  `json:"signature"`
}
```

### RPC Methods
```go
// Required BlockDAG RPC methods
type BlockDAGRPC interface {
    GetBlockTemplate() (*BlockTemplate, error)
    SubmitBlock(block *Block) (*SubmitResult, error)
    GetBalance(address string) (uint64, error)
    SendTransaction(tx *Transaction) (string, error)
    GetBlockByHeight(height uint64) (*Block, error)
    GetNetworkInfo() (*NetworkInfo, error)
}
```

## Stratum Protocol Implementation

### Message Formats
```json
// Mining.subscribe
{
  "id": 1,
  "method": "mining.subscribe",
  "params": ["BlockDAGMiner/1.0.0", null, "blockdag.pool.com", 3333]
}

// Mining.authorize
{
  "id": 2,
  "method": "mining.authorize",
  "params": ["wallet_address.worker_name", "password"]
}

// Mining.notify (server to client)
{
  "id": null,
  "method": "mining.notify",
  "params": [
    "job_id",
    "prev_hash",
    "coinbase1",
    "coinbase2",
    ["merkle_branch"],
    "version",
    "nbits",
    "ntime",
    true
  ]
}

// Mining.submit
{
  "id": 4,
  "method": "mining.submit",
  "params": ["wallet_address.worker_name", "job_id", "extranonce2", "ntime", "nonce"]
}
```

### Work Generation
```go
type WorkTemplate struct {
    JobID        string    `json:"job_id"`
    PrevHash     string    `json:"prev_hash"`
    Coinbase1    string    `json:"coinbase1"`
    Coinbase2    string    `json:"coinbase2"`
    MerkleBranch []string  `json:"merkle_branch"`
    Version      string    `json:"version"`
    NBits        string    `json:"nbits"`
    NTime        string    `json:"ntime"`
    CleanJobs    bool      `json:"clean_jobs"`
    Target       string    `json:"target"`
    Difficulty   float64   `json:"difficulty"`
}

func GenerateWork(blockTemplate *BlockTemplate, extraNonce1 string) *WorkTemplate {
    // Implementation details for work generation
    return &WorkTemplate{
        JobID:     generateJobID(),
        PrevHash:  blockTemplate.PreviousBlockHash,
        Coinbase1: buildCoinbase1(blockTemplate, extraNonce1),
        Coinbase2: buildCoinbase2(),
        // ... other fields
    }
}
```

## Mining Economics

### PPLNS Implementation
```go
type PPLNSCalculator struct {
    windowSize    int64   // Number of shares in window
    shareWindow   []Share // Sliding window of shares
    blockReward   uint64  // Current block reward
    poolFee       float64 // Pool fee percentage (0.01 = 1%)
}

type Share struct {
    MinerID    string    `json:"miner_id"`
    Difficulty uint64    `json:"difficulty"`
    Timestamp  int64     `json:"timestamp"`
    Valid      bool      `json:"valid"`
    BlockHash  string    `json:"block_hash,omitempty"`
}

func (p *PPLNSCalculator) CalculatePayouts(foundBlock *Block) map[string]uint64 {
    totalShares := uint64(0)
    minerShares := make(map[string]uint64)
    
    // Calculate total shares in window
    for _, share := range p.shareWindow {
        if share.Valid {
            totalShares += share.Difficulty
            minerShares[share.MinerID] += share.Difficulty
        }
    }
    
    // Calculate payouts
    payouts := make(map[string]uint64)
    rewardAfterFee := uint64(float64(p.blockReward) * (1.0 - p.poolFee))
    
    for minerID, shares := range minerShares {
        payout := (shares * rewardAfterFee) / totalShares
        payouts[minerID] = payout
    }
    
    return payouts
}
```

### Fee Structure
```go
type FeeConfig struct {
    PoolFee        float64 `json:"pool_fee"`         // 1.0 = 1%
    WithdrawalFee  uint64  `json:"withdrawal_fee"`   // Fixed fee in smallest unit
    MinPayout      uint64  `json:"min_payout"`       // Minimum payout threshold
    PayoutInterval int     `json:"payout_interval"`  // Hours between payouts
}
```

## Database Schema

### PostgreSQL Tables
```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_address VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(255),
    password_hash VARCHAR(255),
    mfa_secret VARCHAR(32),
    mfa_enabled BOOLEAN DEFAULT FALSE,
    balance BIGINT DEFAULT 0,
    total_paid BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Miners table
CREATE TABLE miners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    worker_name VARCHAR(50) NOT NULL,
    ip_address INET,
    user_agent TEXT,
    difficulty BIGINT DEFAULT 1,
    last_share TIMESTAMP,
    shares_accepted BIGINT DEFAULT 0,
    shares_rejected BIGINT DEFAULT 0,
    hashrate_1h BIGINT DEFAULT 0,
    hashrate_24h BIGINT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'offline',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Shares table (partitioned by date)
CREATE TABLE shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    miner_id UUID REFERENCES miners(id),
    job_id VARCHAR(64) NOT NULL,
    difficulty BIGINT NOT NULL,
    share_diff BIGINT NOT NULL,
    block_hash VARCHAR(64),
    is_block BOOLEAN DEFAULT FALSE,
    valid BOOLEAN DEFAULT TRUE,
    error_code VARCHAR(20),
    ip_address INET,
    timestamp TIMESTAMP DEFAULT NOW()
) PARTITION BY RANGE (timestamp);

-- Blocks table
CREATE TABLE blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    height BIGINT UNIQUE NOT NULL,
    hash VARCHAR(64) UNIQUE NOT NULL,
    prev_hash VARCHAR(64) NOT NULL,
    finder_id UUID REFERENCES miners(id),
    difficulty BIGINT NOT NULL,
    reward BIGINT NOT NULL,
    fees BIGINT DEFAULT 0,
    confirmations INT DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending', -- pending, confirmed, orphaned
    found_at TIMESTAMP DEFAULT NOW(),
    confirmed_at TIMESTAMP
);

-- Payouts table
CREATE TABLE payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    block_id UUID REFERENCES blocks(id),
    amount BIGINT NOT NULL,
    fee BIGINT DEFAULT 0,
    transaction_hash VARCHAR(64),
    status VARCHAR(20) DEFAULT 'pending', -- pending, processing, completed, failed
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_miners_user_id ON miners(user_id);
CREATE INDEX idx_miners_status ON miners(status);
CREATE INDEX idx_shares_miner_timestamp ON shares(miner_id, timestamp DESC);
CREATE INDEX idx_shares_block_hash ON shares(block_hash) WHERE block_hash IS NOT NULL;
CREATE INDEX idx_blocks_height ON blocks(height DESC);
CREATE INDEX idx_blocks_status ON blocks(status);
CREATE INDEX idx_payouts_user_status ON payouts(user_id, status);
```

## Configuration Files

### Pool Configuration
```yaml
# config/pool.yaml
pool:
  name: "BlockDAG Mining Pool"
  website: "https://pool.blockdag.network"
  fee: 1.0  # 1% pool fee
  
network:
  type: "mainnet"  # mainnet, testnet
  rpc_endpoint: "https://api.blockdag.network"
  websocket_url: "wss://ws.blockdag.network"
  api_key: "${BLOCKDAG_API_KEY}"
  chain_id: 1
  block_time: 30  # seconds
  confirmations: 10

stratum:
  host: "0.0.0.0"
  port: 3333
  difficulty: 1000
  var_diff: true
  var_diff_min: 100
  var_diff_max: 100000
  var_diff_target: 15  # seconds
  var_diff_retarget: 90  # seconds

database:
  host: "${DB_HOST:-localhost}"
  port: 5432
  name: "${DB_NAME:-blockdag_pool}"
  user: "${DB_USER:-pool}"
  password: "${DB_PASSWORD}"
  max_connections: 100
  ssl_mode: "require"

redis:
  host: "${REDIS_HOST:-localhost}"
  port: 6379
  password: "${REDIS_PASSWORD}"
  db: 0
  max_connections: 50

payouts:
  min_payout: 1000000000  # 10 BDAG (assuming 8 decimals)
  payout_interval: 24  # hours
  withdrawal_fee: 100000000  # 1 BDAG
  auto_payout: true

security:
  jwt_secret: "${JWT_SECRET}"
  mfa_issuer: "BlockDAG Pool"
  rate_limit_requests: 100
  rate_limit_window: 60  # seconds
  max_login_attempts: 5
  lockout_duration: 300  # seconds

monitoring:
  prometheus_port: 9090
  log_level: "info"
  metrics_interval: 30  # seconds
```

### Docker Compose
```yaml
# docker-compose.yml
version: '3.8'

services:
  pool-manager:
    build: .
    ports:
      - "3333:3333"  # Stratum
      - "8080:8080"  # Web API
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
      - BLOCKDAG_API_KEY=${BLOCKDAG_API_KEY}
      - JWT_SECRET=${JWT_SECRET}
    depends_on:
      - postgres
      - redis
    volumes:
      - ./config:/app/config
      - ./logs:/app/logs
    restart: unless-stopped

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=blockdag_pool
      - POSTGRES_USER=pool
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./sql/init.sql:/docker-entrypoint-initdb.d/init.sql
    ports:
      - "5432:5432"
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl/certs
    depends_on:
      - pool-manager
    restart: unless-stopped

  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    restart: unless-stopped

  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    volumes:
      - grafana_data:/var/lib/grafana
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  prometheus_data:
  grafana_data:
```

## Performance Specifications

### Target Metrics
```yaml
performance_targets:
  concurrent_miners: 10000
  shares_per_second: 1000
  api_response_time: 100ms  # 95th percentile
  stratum_latency: 50ms     # average
  database_queries: 500ms   # 99th percentile
  memory_usage: 2GB         # maximum per instance
  cpu_usage: 80%            # maximum sustained

scaling:
  horizontal_scaling: true
  load_balancer: nginx
  session_affinity: false
  auto_scaling_trigger: 70%  # CPU threshold
  min_instances: 2
  max_instances: 10
```

### Caching Strategy
```go
type CacheConfig struct {
    UserSessions    time.Duration `json:"user_sessions"`    // 24h
    MinerStats      time.Duration `json:"miner_stats"`      // 5m
    PoolStats       time.Duration `json:"pool_stats"`       // 1m
    BlockTemplates  time.Duration `json:"block_templates"`  // 30s
    Payouts         time.Duration `json:"payouts"`          // 1h
}

// Redis key patterns
const (
    KeyUserSession    = "session:%s"
    KeyMinerStats     = "miner:%s:stats"
    KeyPoolStats      = "pool:stats"
    KeyBlockTemplate  = "block:template"
    KeyUserBalance    = "user:%s:balance"
)
```

## Security Implementation

### API Security
```go
type SecurityConfig struct {
    JWTSecret           string        `json:"jwt_secret"`
    JWTExpiration      time.Duration `json:"jwt_expiration"`      // 24h
    RefreshExpiration  time.Duration `json:"refresh_expiration"`  // 7d
    MFAIssuer          string        `json:"mfa_issuer"`
    RateLimitRequests  int           `json:"rate_limit_requests"` // per window
    RateLimitWindow    time.Duration `json:"rate_limit_window"`   // 1m
    MaxLoginAttempts   int           `json:"max_login_attempts"`  // 5
    LockoutDuration    time.Duration `json:"lockout_duration"`    // 5m
}

// Rate limiting middleware
func RateLimitMiddleware(config SecurityConfig) gin.HandlerFunc {
    limiter := rate.NewLimiter(
        rate.Every(config.RateLimitWindow/time.Duration(config.RateLimitRequests)),
        config.RateLimitRequests,
    )
    
    return gin.HandlerFunc(func(c *gin.Context) {
        if !limiter.Allow() {
            c.JSON(429, gin.H{"error": "Rate limit exceeded"})
            c.Abort()
            return
        }
        c.Next()
    })
}
```

### Encryption Standards
```go
// Password hashing
func HashPassword(password string) (string, error) {
    return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

// Data encryption (AES-256-GCM)
func EncryptData(data []byte, key []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    
    ciphertext := gcm.Seal(nonce, nonce, data, nil)
    return ciphertext, nil
}
```

This technical specification provides the concrete implementation details needed to build the mining pool software successfully.