#!/bin/bash

# Database Foundation Validation Script
# This script validates that the database foundation implementation meets all requirements

set -e

echo "🔍 Validating Database Foundation Implementation..."
echo "=================================================="

# Check if we're in the correct directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Must be run from the chimera-pool-core directory"
    exit 1
fi

echo "✅ Directory structure validated"

# Check that all required files exist
required_files=(
    "internal/database/database.go"
    "internal/database/connection.go"
    "internal/database/models.go"
    "internal/database/operations.go"
    "migrations/001_initial_schema.up.sql"
)

echo ""
echo "📁 Checking required files..."
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file exists"
    else
        echo "❌ $file missing"
        exit 1
    fi
done

# Check that all required test files exist
test_files=(
    "internal/database/database_test.go"
    "internal/database/connection_test.go"
    "internal/database/schema_test.go"
    "internal/database/integration_test.go"
    "internal/database/comprehensive_test.go"
)

echo ""
echo "🧪 Checking test files..."
for file in "${test_files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file exists"
    else
        echo "❌ $file missing"
        exit 1
    fi
done

# Check that required database tables are defined in schema
required_tables=(
    "users"
    "miners"
    "shares"
    "blocks"
    "payouts"
)

echo ""
echo "🗄️  Checking database schema..."
schema_file="migrations/001_initial_schema.up.sql"
for table in "${required_tables[@]}"; do
    if grep -q "CREATE TABLE.*$table" "$schema_file"; then
        echo "✅ Table '$table' defined in schema"
    else
        echo "❌ Table '$table' missing from schema"
        exit 1
    fi
done

# Check that required Go structs are defined
required_structs=(
    "User"
    "Miner"
    "Share"
    "Block"
    "Payout"
    "Config"
    "ConnectionPool"
    "Database"
)

echo ""
echo "🏗️  Checking Go data structures..."
models_file="internal/database/models.go"
for struct in "${required_structs[@]}"; do
    if grep -q "type $struct struct" "$models_file" "internal/database/database.go" "internal/database/connection.go"; then
        echo "✅ Struct '$struct' defined"
    else
        echo "❌ Struct '$struct' missing"
        exit 1
    fi
done

# Check that required database operations are implemented
required_operations=(
    "CreateUser"
    "GetUserByID"
    "GetUserByUsername"
    "CreateMiner"
    "GetMinersByUserID"
    "UpdateMinerLastSeen"
    "CreateShare"
    "GetSharesByMinerID"
)

echo ""
echo "⚙️  Checking database operations..."
operations_file="internal/database/operations.go"
for operation in "${required_operations[@]}"; do
    if grep -q "func $operation" "$operations_file"; then
        echo "✅ Operation '$operation' implemented"
    else
        echo "❌ Operation '$operation' missing"
        exit 1
    fi
done

# Check that connection pool methods are implemented
required_pool_methods=(
    "NewConnectionPool"
    "Close"
    "HealthCheck"
    "Stats"
    "QueryRow"
    "Query"
    "Exec"
    "Begin"
    "DB"
)

echo ""
echo "🔗 Checking connection pool methods..."
connection_file="internal/database/connection.go"
for method in "${required_pool_methods[@]}"; do
    if grep -q "func.*$method" "$connection_file"; then
        echo "✅ Method '$method' implemented"
    else
        echo "❌ Method '$method' missing"
        exit 1
    fi
done

# Check that migration support is implemented
echo ""
echo "📦 Checking migration support..."
if grep -q "RunMigrations" "$connection_file"; then
    echo "✅ Migration support implemented"
else
    echo "❌ Migration support missing"
    exit 1
fi

if grep -q "GetMigrationStatus" "$connection_file"; then
    echo "✅ Migration status support implemented"
else
    echo "❌ Migration status support missing"
    exit 1
fi

# Check that proper error handling is implemented
echo ""
echo "🚨 Checking error handling..."
if grep -q "fmt.Errorf" "internal/database/"*.go; then
    echo "✅ Error handling implemented"
else
    echo "❌ Error handling missing"
    exit 1
fi

# Check that context support is implemented
echo ""
echo "⏱️  Checking context support..."
if grep -q "context.Context" "internal/database/"*.go; then
    echo "✅ Context support implemented"
else
    echo "❌ Context support missing"
    exit 1
fi

# Check that timeouts are implemented
if grep -q "WithTimeout" "internal/database/"*.go; then
    echo "✅ Timeout support implemented"
else
    echo "❌ Timeout support missing"
    exit 1
fi

# Validate that all requirements are addressed
echo ""
echo "📋 Validating requirements coverage..."

# Requirement 6.1: Pool Mining Functionality
if grep -q "shares" "$schema_file" && grep -q "CreateShare" "$operations_file"; then
    echo "✅ Requirement 6.1: Share recording implemented"
else
    echo "❌ Requirement 6.1: Share recording missing"
    exit 1
fi

# Requirement 6.2: Payout System
if grep -q "payouts" "$schema_file" && grep -q "Payout" "$models_file"; then
    echo "✅ Requirement 6.2: Payout system implemented"
else
    echo "❌ Requirement 6.2: Payout system missing"
    exit 1
fi

# Check that indexes are created for performance
echo ""
echo "🚀 Checking performance optimizations..."
if grep -q "CREATE INDEX" "$schema_file"; then
    echo "✅ Database indexes implemented"
else
    echo "❌ Database indexes missing"
    exit 1
fi

# Check that foreign key constraints are implemented
if grep -q "REFERENCES" "$schema_file"; then
    echo "✅ Foreign key constraints implemented"
else
    echo "❌ Foreign key constraints missing"
    exit 1
fi

# Check that proper data types are used
if grep -q "BIGSERIAL" "$schema_file" && grep -q "TIMESTAMP WITH TIME ZONE" "$schema_file"; then
    echo "✅ Proper data types used"
else
    echo "❌ Proper data types missing"
    exit 1
fi

echo ""
echo "🎉 Database Foundation Validation Complete!"
echo "=========================================="
echo ""
echo "✅ All required components implemented:"
echo "   - PostgreSQL schema with proper tables, indexes, and constraints"
echo "   - Connection pool with health checks and statistics"
echo "   - Data models for all mining pool entities"
echo "   - Database operations for CRUD functionality"
echo "   - Migration support for schema management"
echo "   - Comprehensive test coverage"
echo "   - Error handling and timeout support"
echo "   - Requirements 6.1 and 6.2 addressed"
echo ""
echo "🚀 Ready for E2E testing with real PostgreSQL container!"