#!/bin/bash

# Setup script for the Internal Wallet Service
# This script sets up the database and runs migrations and seed data

set -e

echo "🚀 Setting up Internal Wallet Service..."

# Check if PostgreSQL is running
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL is not installed or not in PATH"
    echo "Please install PostgreSQL and try again"
    exit 1
fi

# Default database configuration
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-wallet_service}
DB_USER=${DB_USER:-wallet_user}
DB_PASSWORD=${DB_PASSWORD:-wallet_password}

echo "📊 Database Configuration:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"

# Create database and user if they don't exist
echo "🔧 Creating database and user..."
psql -h $DB_HOST -p $DB_PORT -U postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" 2>/dev/null || echo "User already exists"
psql -h $DB_HOST -p $DB_PORT -U postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>/dev/null || echo "Database already exists"
psql -h $DB_HOST -p $DB_PORT -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" 2>/dev/null

# Run migrations
echo "🗄️  Running database migrations..."
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/000001_initial_schema.up.sql

# Run seed data
echo "🌱 Running seed data..."
cd scripts
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f seed_all.sql
cd ..

echo "✅ Setup complete!"
echo ""
echo "🎯 Next steps:"
echo "1. Set environment variables:"
echo "   export DATABASE_URL='postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable'"
echo "2. Run the service:"
echo "   go run cmd/wallet-service/main.go"
echo "3. Test the API:"
echo "   curl http://localhost:8080/health"
echo ""
echo "📚 API Documentation:"
echo "  POST /api/v1/transactions - Execute transaction"
echo "  GET  /api/v1/accounts/{id}/balances/{asset_id} - Get balance"
echo "  GET  /api/v1/accounts/{id}/balances - Get all balances"
echo "  GET  /health - Health check"