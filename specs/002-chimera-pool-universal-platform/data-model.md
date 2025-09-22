# Data Model: Chimera Pool Universal Platform

## Overview

This document defines the comprehensive data model for the Chimera Pool Universal Platform, including database schemas, entity relationships, and data flow patterns for supporting multiple cryptocurrencies with hot-swappable algorithms.

## Database Architecture

### Primary Database: PostgreSQL
- **Purpose**: Transactional data, user accounts, pool configurations, payouts
- **Features**: ACID compliance, complex queries, JSON support, full-text search
- **Scaling**: Read replicas, connection pooling, query optimization

### Cache Layer: Redis
- **Purpose**: Session management, real-time statistics, temporary data
- **Features**: In-memory performance, pub/sub messaging, data structures
- **Scaling**: Redis Cluster, automatic failover, memory optimization

### Time-Series Database: InfluxDB
- **Purpose**: Mining statistics, performance metrics, historical data
- **Features**: Time-based queries, data retention policies, aggregations
- **Scaling**: Horizontal sharding, data compression, automated retention

## Core Entity Schemas

### Users and Authentication

#### users
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    salt VARCHAR(32) NOT NULL,
    
    -- Multi-factor authentication
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_secret VARCHAR(32),
    mfa_backup_codes TEXT[], -- Encrypted backup codes
    
    -- Profile information
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    timezone VARCHAR(50) DEFAULT 'UTC',
    language VARCHAR(10) DEFAULT 'en',
    
    -- Account status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'banned')),
    email_verified BOOLEAN DEFAULT FALSE,
    email_verification_token VARCHAR(64),
    
    -- Audit fields
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP,
    login_count INTEGER DEFAULT 0,
    
    -- Indexes
    CONSTRAINT users_username_length CHECK (length(username) >= 3),
    CONSTRAINT users_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
```

#### user_roles
```sql
CREATE TABLE user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'operator', 'miner', 'viewer')),
    granted_by UUID REFERENCES users(id),
    granted_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    UNIQUE(user_id, role)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role ON user_roles(role);
```

#### user_sessions
```sql
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    refresh_token_hash VARCHAR(64) UNIQUE,
    
    -- Session metadata
    ip_address INET,
    user_agent TEXT,
    device_fingerprint VARCHAR(64),
    
    -- Session lifecycle
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    
    -- Security flags
    is_mfa_verified BOOLEAN DEFAULT FALSE,
    requires_password_change BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_token_hash ON user_sessions(token_hash);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
```

### Cryptocurrency and Algorithm Management

#### supported_cryptocurrencies
```sql
CREATE TABLE supported_cryptocurrencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(10) NOT NULL UNIQUE, -- BTC, ETC, BDAG, LTC, etc.
    name VARCHAR(100) NOT NULL, -- Bitcoin, Ethereum Classic, etc.
    full_name VARCHAR(200) NOT NULL,
    
    -- Network information
    network_type VARCHAR(20) NOT NULL, -- mainnet, testnet
    block_time INTEGER NOT NULL, -- Target block time in seconds
    block_reward DECIMAL(20,8) NOT NULL,
    total_supply DECIMAL(30,8),
    
    -- Algorithm information
    primary_algorithm VARCHAR(50) NOT NULL,
    supported_algorithms TEXT[], -- Array of supported algorithm names
    
    -- Pool configuration
    default_difficulty BIGINT NOT NULL DEFAULT 1,
    minimum_difficulty BIGINT NOT NULL DEFAULT 1,
    maximum_difficulty BIGINT,
    difficulty_retarget_time INTEGER DEFAULT 600, -- seconds
    
    -- Payout configuration
    minimum_payout DECIMAL(20,8) NOT NULL DEFAULT 0.001,
    transaction_fee DECIMAL(20,8) NOT NULL DEFAULT 0.0001,
    confirmation_blocks INTEGER NOT NULL DEFAULT 6,
    
    -- Status and metadata
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'deprecated')),
    icon_url VARCHAR(500),
    website_url VARCHAR(500),
    explorer_url VARCHAR(500),
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_cryptocurrencies_symbol ON supported_cryptocurrencies(symbol);
CREATE INDEX idx_cryptocurrencies_status ON supported_cryptocurrencies(status);
```

#### algorithms
```sql
CREATE TABLE algorithms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE, -- sha256, blake3, ethash, etc.
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Algorithm metadata
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    author VARCHAR(100),
    license VARCHAR(50),
    
    -- Performance characteristics
    memory_requirements VARCHAR(20), -- low, medium, high
    cpu_intensive BOOLEAN DEFAULT FALSE,
    gpu_optimized BOOLEAN DEFAULT FALSE,
    asic_resistant BOOLEAN DEFAULT FALSE,
    
    -- Implementation details
    package_url VARCHAR(500),
    package_hash VARCHAR(64),
    package_signature TEXT,
    wasm_binary BYTEA,
    native_libraries JSONB, -- Platform-specific binaries
    
    -- Compatibility
    supported_platforms TEXT[], -- linux, windows, macos
    minimum_engine_version VARCHAR(20),
    
    -- Status
    status VARCHAR(20) DEFAULT 'available' CHECK (status IN ('available', 'installed', 'active', 'deprecated')),
    installed_at TIMESTAMP,
    activated_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_algorithms_name ON algorithms(name);
CREATE INDEX idx_algorithms_status ON algorithms(status);
```

#### algorithm_migrations
```sql
CREATE TABLE algorithm_migrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_algorithm_id UUID REFERENCES algorithms(id),
    to_algorithm_id UUID NOT NULL REFERENCES algorithms(id),
    
    -- Migration configuration
    strategy VARCHAR(20) NOT NULL CHECK (strategy IN ('immediate', 'gradual', 'scheduled')),
    shadow_duration INTEGER DEFAULT 300, -- seconds
    phase_duration INTEGER DEFAULT 600, -- seconds
    rollback_on_error BOOLEAN DEFAULT TRUE,
    
    -- Migration state
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'staging', 'in_progress', 'completed', 'failed', 'rolled_back')),
    progress INTEGER DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    current_phase VARCHAR(50),
    
    -- Timing
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Results
    error_message TEXT,
    performance_impact DECIMAL(5,2), -- Percentage impact
    rollback_reason TEXT,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_algorithm_migrations_status ON algorithm_migrations(status);
CREATE INDEX idx_algorithm_migrations_scheduled_at ON algorithm_migrations(scheduled_at);
```

### Pool Management

#### pools
```sql
CREATE TABLE pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    
    -- Cryptocurrency configuration
    cryptocurrency_id UUID NOT NULL REFERENCES supported_cryptocurrencies(id),
    algorithm_id UUID NOT NULL REFERENCES algorithms(id),
    
    -- Network configuration
    stratum_host VARCHAR(255) NOT NULL DEFAULT 'localhost',
    stratum_port INTEGER NOT NULL,
    stratum_ssl_port INTEGER,
    ssl_enabled BOOLEAN DEFAULT FALSE,
    
    -- Mining configuration
    difficulty BIGINT NOT NULL DEFAULT 1,
    variable_difficulty BOOLEAN DEFAULT TRUE,
    min_difficulty BIGINT DEFAULT 1,
    max_difficulty BIGINT,
    difficulty_retarget_time INTEGER DEFAULT 600,
    
    -- Pool economics
    fee_percentage DECIMAL(5,4) NOT NULL DEFAULT 1.0000, -- 1.0000 = 1%
    payout_method VARCHAR(20) DEFAULT 'PPLNS' CHECK (payout_method IN ('PPS', 'PPLNS', 'PROP')),
    payout_threshold DECIMAL(20,8) NOT NULL DEFAULT 0.01,
    payout_frequency VARCHAR(20) DEFAULT 'daily' CHECK (payout_frequency IN ('manual', 'hourly', 'daily', 'weekly')),
    
    -- Wallet configuration
    pool_wallet_address VARCHAR(100) NOT NULL,
    backup_wallet_address VARCHAR(100),
    
    -- Pool limits
    max_miners INTEGER DEFAULT 10000,
    max_hashrate BIGINT, -- Maximum pool hashrate
    
    -- Status and lifecycle
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'maintenance', 'deprecated')),
    auto_start BOOLEAN DEFAULT TRUE,
    
    -- Audit fields
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(stratum_port),
    UNIQUE(stratum_ssl_port)
);

CREATE INDEX idx_pools_cryptocurrency_id ON pools(cryptocurrency_id);
CREATE INDEX idx_pools_algorithm_id ON pools(algorithm_id);
CREATE INDEX idx_pools_status ON pools(status);
CREATE INDEX idx_pools_stratum_port ON pools(stratum_port);
```

#### pool_configurations
```sql
CREATE TABLE pool_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    
    -- Configuration data (JSON)
    configuration JSONB NOT NULL,
    
    -- Version control
    version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN DEFAULT FALSE,
    
    -- Change tracking
    changed_by UUID REFERENCES users(id),
    change_reason TEXT,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(pool_id, version)
);

CREATE INDEX idx_pool_configurations_pool_id ON pool_configurations(pool_id);
CREATE INDEX idx_pool_configurations_is_active ON pool_configurations(is_active);
```

### Miner Management

#### miners
```sql
CREATE TABLE miners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pool_id UUID NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    
    -- Miner identification
    worker_name VARCHAR(100) NOT NULL,
    display_name VARCHAR(200),
    
    -- Connection information
    ip_address INET,
    user_agent TEXT,
    mining_software VARCHAR(100),
    mining_software_version VARCHAR(50),
    
    -- Mining configuration
    difficulty BIGINT NOT NULL DEFAULT 1,
    auto_difficulty BOOLEAN DEFAULT TRUE,
    custom_difficulty BIGINT,
    
    -- Hardware information
    hardware_type VARCHAR(20) CHECK (hardware_type IN ('CPU', 'GPU', 'ASIC', 'FPGA', 'Unknown')),
    hardware_details JSONB,
    
    -- Performance tracking
    current_hashrate BIGINT DEFAULT 0,
    average_hashrate_1h BIGINT DEFAULT 0,
    average_hashrate_24h BIGINT DEFAULT 0,
    
    -- Share statistics
    shares_accepted BIGINT DEFAULT 0,
    shares_rejected BIGINT DEFAULT 0,
    shares_stale BIGINT DEFAULT 0,
    shares_duplicate BIGINT DEFAULT 0,
    
    -- Connection tracking
    first_seen TIMESTAMP DEFAULT NOW(),
    last_seen TIMESTAMP DEFAULT NOW(),
    connection_count INTEGER DEFAULT 0,
    total_uptime_seconds BIGINT DEFAULT 0,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'offline', 'banned')),
    
    -- Notifications
    notifications_enabled BOOLEAN DEFAULT TRUE,
    notification_preferences JSONB,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(user_id, pool_id, worker_name)
);

CREATE INDEX idx_miners_user_id ON miners(user_id);
CREATE INDEX idx_miners_pool_id ON miners(pool_id);
CREATE INDEX idx_miners_status ON miners(status);
CREATE INDEX idx_miners_last_seen ON miners(last_seen);
CREATE INDEX idx_miners_worker_name ON miners(worker_name);
```

#### miner_connections
```sql
CREATE TABLE miner_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    miner_id UUID NOT NULL REFERENCES miners(id) ON DELETE CASCADE,
    
    -- Connection details
    session_id VARCHAR(64) NOT NULL UNIQUE,
    ip_address INET NOT NULL,
    port INTEGER NOT NULL,
    
    -- Protocol information
    stratum_version VARCHAR(20),
    supported_extensions TEXT[],
    
    -- Connection lifecycle
    connected_at TIMESTAMP DEFAULT NOW(),
    disconnected_at TIMESTAMP,
    disconnect_reason VARCHAR(100),
    
    -- Performance metrics
    shares_submitted INTEGER DEFAULT 0,
    shares_accepted INTEGER DEFAULT 0,
    shares_rejected INTEGER DEFAULT 0,
    
    -- Network metrics
    bytes_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    average_latency_ms INTEGER,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'disconnected', 'timeout', 'error'))
);

CREATE INDEX idx_miner_connections_miner_id ON miner_connections(miner_id);
CREATE INDEX idx_miner_connections_session_id ON miner_connections(session_id);
CREATE INDEX idx_miner_connections_connected_at ON miner_connections(connected_at);
```

### Share and Block Management

#### shares
```sql
CREATE TABLE shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    miner_id UUID NOT NULL REFERENCES miners(id) ON DELETE CASCADE,
    pool_id UUID NOT NULL REFERENCES pools(id) ON DELETE CASCADE,
    
    -- Share identification
    job_id VARCHAR(64) NOT NULL,
    extranonce1 VARCHAR(16),
    extranonce2 VARCHAR(16),
    nonce VARCHAR(16) NOT NULL,
    
    -- Block information
    block_height BIGINT NOT NULL,
    block_hash VARCHAR(64),
    previous_block_hash VARCHAR(64),
    
    -- Difficulty and validation
    share_difficulty BIGINT NOT NULL,
    network_difficulty BIGINT NOT NULL,
    target VARCHAR(64) NOT NULL,
    
    -- Share result
    is_valid BOOLEAN NOT NULL,
    is_block BOOLEAN DEFAULT FALSE,
    is_stale BOOLEAN DEFAULT FALSE,
    is_duplicate BOOLEAN DEFAULT FALSE,
    
    -- Validation details
    validation_time_ms INTEGER,
    error_code INTEGER,
    error_message TEXT,
    
    -- Timing
    submitted_at TIMESTAMP DEFAULT NOW(),
    validated_at TIMESTAMP DEFAULT NOW(),
    
    -- Algorithm used
    algorithm_id UUID REFERENCES algorithms(id)
);

-- Partitioning by month for performance
CREATE TABLE shares_y2025m09 PARTITION OF shares
    FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');

CREATE INDEX idx_shares_miner_id ON shares(miner_id);
CREATE INDEX idx_shares_pool_id ON shares(pool_id);
CREATE INDEX idx_shares_submitted_at ON shares(submitted_at);
CREATE INDEX idx_shares_is_block ON shares(is_block) WHERE is_block = TRUE;
CREATE INDEX idx_shares_block_height ON shares(block_height);
```

#### blocks
```sql
CREATE TABLE blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES pools(id),
    miner_id UUID NOT NULL REFERENCES miners(id),
    share_id UUID NOT NULL REFERENCES shares(id),
    
    -- Block identification
    block_hash VARCHAR(64) NOT NULL UNIQUE,
    block_height BIGINT NOT NULL,
    previous_block_hash VARCHAR(64),
    
    -- Block details
    difficulty BIGINT NOT NULL,
    network_difficulty BIGINT NOT NULL,
    block_reward DECIMAL(20,8) NOT NULL,
    transaction_fees DECIMAL(20,8) DEFAULT 0,
    total_reward DECIMAL(20,8) NOT NULL,
    
    -- Block status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'confirmed', 'orphaned', 'rejected')),
    confirmations INTEGER DEFAULT 0,
    required_confirmations INTEGER NOT NULL DEFAULT 6,
    
    -- Timing
    found_at TIMESTAMP DEFAULT NOW(),
    confirmed_at TIMESTAMP,
    
    -- Payout information
    pool_fee DECIMAL(20,8),
    miner_reward DECIMAL(20,8),
    payout_processed BOOLEAN DEFAULT FALSE,
    payout_processed_at TIMESTAMP
);

CREATE INDEX idx_blocks_pool_id ON blocks(pool_id);
CREATE INDEX idx_blocks_miner_id ON blocks(miner_id);
CREATE INDEX idx_blocks_block_height ON blocks(block_height);
CREATE INDEX idx_blocks_status ON blocks(status);
CREATE INDEX idx_blocks_found_at ON blocks(found_at);
```

### Payout Management

#### user_wallets
```sql
CREATE TABLE user_wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cryptocurrency_id UUID NOT NULL REFERENCES supported_cryptocurrencies(id),
    
    -- Wallet information
    wallet_address VARCHAR(100) NOT NULL,
    wallet_label VARCHAR(100),
    
    -- Validation
    address_validated BOOLEAN DEFAULT FALSE,
    validation_transaction_hash VARCHAR(64),
    
    -- Payout configuration
    minimum_payout DECIMAL(20,8),
    automatic_payouts BOOLEAN DEFAULT TRUE,
    payout_frequency VARCHAR(20) DEFAULT 'daily',
    
    -- Status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'suspended')),
    
    -- Audit
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(user_id, cryptocurrency_id, wallet_address)
);

CREATE INDEX idx_user_wallets_user_id ON user_wallets(user_id);
CREATE INDEX idx_user_wallets_cryptocurrency_id ON user_wallets(cryptocurrency_id);
CREATE INDEX idx_user_wallets_wallet_address ON user_wallets(wallet_address);
```

#### payouts
```sql
CREATE TABLE payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    pool_id UUID NOT NULL REFERENCES pools(id),
    wallet_id UUID NOT NULL REFERENCES user_wallets(id),
    
    -- Payout details
    amount DECIMAL(20,8) NOT NULL,
    fee DECIMAL(20,8) NOT NULL DEFAULT 0,
    net_amount DECIMAL(20,8) NOT NULL,
    
    -- Transaction information
    transaction_hash VARCHAR(64),
    transaction_fee DECIMAL(20,8),
    block_height BIGINT,
    confirmations INTEGER DEFAULT 0,
    
    -- Payout calculation
    shares_included BIGINT NOT NULL,
    calculation_method VARCHAR(20) NOT NULL,
    calculation_period_start TIMESTAMP NOT NULL,
    calculation_period_end TIMESTAMP NOT NULL,
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'sent', 'confirmed', 'failed')),
    
    -- Error handling
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    error_message TEXT,
    
    -- Timing
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    confirmed_at TIMESTAMP,
    
    -- Audit
    processed_by UUID REFERENCES users(id),
    notes TEXT
);

CREATE INDEX idx_payouts_user_id ON payouts(user_id);
CREATE INDEX idx_payouts_pool_id ON payouts(pool_id);
CREATE INDEX idx_payouts_status ON payouts(status);
CREATE INDEX idx_payouts_created_at ON payouts(created_at);
CREATE INDEX idx_payouts_transaction_hash ON payouts(transaction_hash);
```

## Time-Series Data (InfluxDB)

### Mining Statistics
```
measurement: mining_stats
tags:
  - pool_id
  - miner_id
  - cryptocurrency
  - algorithm
fields:
  - hashrate (integer)
  - difficulty (integer)
  - shares_accepted (integer)
  - shares_rejected (integer)
  - shares_per_minute (float)
  - efficiency_percentage (float)
timestamp: nanosecond precision
```

### Pool Statistics
```
measurement: pool_stats
tags:
  - pool_id
  - cryptocurrency
  - algorithm
fields:
  - total_hashrate (integer)
  - active_miners (integer)
  - total_shares (integer)
  - blocks_found (integer)
  - network_difficulty (integer)
  - pool_luck_percentage (float)
timestamp: nanosecond precision
```

### System Metrics
```
measurement: system_metrics
tags:
  - server_id
  - component (pool_manager, stratum_server, algorithm_engine)
fields:
  - cpu_usage_percentage (float)
  - memory_usage_bytes (integer)
  - network_connections (integer)
  - response_time_ms (float)
  - error_rate (float)
timestamp: nanosecond precision
```

## Cache Data Structures (Redis)

### Session Management
```
Key: session:{token_hash}
Type: Hash
Fields:
  - user_id
  - expires_at
  - last_used_at
  - ip_address
  - mfa_verified
TTL: Session expiration time
```

### Real-Time Statistics
```
Key: pool_stats:{pool_id}
Type: Hash
Fields:
  - total_hashrate
  - active_miners
  - shares_per_minute
  - last_block_time
  - current_difficulty
TTL: 300 seconds (5 minutes)
```

### Miner Status
```
Key: miner_status:{miner_id}
Type: Hash
Fields:
  - current_hashrate
  - last_share_time
  - connection_status
  - difficulty
  - shares_accepted_1h
TTL: 3600 seconds (1 hour)
```

### Algorithm Migration State
```
Key: migration:{migration_id}
Type: Hash
Fields:
  - status
  - progress
  - current_phase
  - started_at
  - error_message
TTL: 86400 seconds (24 hours)
```

## Data Relationships

### Entity Relationship Diagram
```
Users (1) ←→ (N) Miners ←→ (1) Pools ←→ (1) Cryptocurrencies
  ↓                ↓           ↓              ↓
UserWallets    Shares      Blocks        Algorithms
  ↓              ↓           ↓              ↓
Payouts    ShareStats   BlockRewards   Migrations
```

### Key Relationships
- **Users** can have multiple **Miners** across different **Pools**
- **Pools** belong to one **Cryptocurrency** and use one **Algorithm**
- **Miners** submit **Shares** which may result in **Blocks**
- **Blocks** generate **Payouts** to **UserWallets**
- **Algorithms** can be migrated through **AlgorithmMigrations**

## Data Flow Patterns

### Mining Flow
1. Miner connects → Create/Update miner record
2. Share submitted → Validate and store share
3. Valid share → Update miner statistics
4. Block found → Create block record, trigger payout calculation
5. Payout calculated → Create payout record, process transaction

### Algorithm Migration Flow
1. New algorithm staged → Create migration record
2. Migration started → Update migration status
3. Gradual rollout → Track progress and performance
4. Migration completed → Update active algorithm
5. Cleanup → Archive old algorithm data

### Statistics Aggregation Flow
1. Real-time data → Store in Redis cache
2. Periodic aggregation → Calculate statistics
3. Historical data → Store in InfluxDB
4. Dashboard queries → Retrieve from appropriate storage

## Performance Considerations

### Database Optimization
- **Partitioning**: Shares table partitioned by month
- **Indexing**: Strategic indexes on frequently queried columns
- **Connection Pooling**: Efficient database connection management
- **Read Replicas**: Separate read and write workloads

### Cache Strategy
- **Write-Through**: Critical data written to both cache and database
- **Write-Behind**: Non-critical data written to cache first
- **TTL Management**: Appropriate expiration times for different data types
- **Cache Warming**: Pre-populate cache with frequently accessed data

### Data Retention
- **Shares**: Retain detailed data for 3 months, aggregated data for 2 years
- **Statistics**: Real-time data for 24 hours, hourly aggregates for 1 year
- **Logs**: Application logs for 30 days, audit logs for 7 years
- **Sessions**: Automatic cleanup of expired sessions

This comprehensive data model supports the full functionality of the Chimera Pool Universal Platform while maintaining performance, scalability, and data integrity across multiple cryptocurrencies and hot-swappable algorithms.

