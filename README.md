# Internal Wallet Service

A production-grade, ACID-compliant wallet service built with Go and PostgreSQL for managing application-specific credits in gaming platforms and loyalty reward systems.

## 🚀 Features

- **Double-Entry Ledger**: All transactions recorded with immutable audit trail
- **ACID Compliance**: PostgreSQL transactions with SERIALIZABLE isolation
- **Idempotency**: Safe request retries with unique idempotency keys
- **Concurrency Safe**: Handles high-traffic loads with proper locking
- **Production-Grade**: No balance column, account-level locking, proper error handling
- **RESTful API**: JSON-based HTTP endpoints

## 🏗️ Architecture

- **Go 1.21+**: High-performance backend with clean architecture
- **PostgreSQL 14+**: ACID-compliant database with row-level locking
- **pgx/v5**: High-performance PostgreSQL driver with connection pooling
- **Gorilla Mux**: HTTP router for API endpoints
- **Docker**: Containerized deployment with docker-compose

## 🚀 Quick Start

### Prerequisites

- Docker and Docker Compose (for containerized setup)
- OR Go 1.21+ and PostgreSQL 14+ (for local development)

### Option 1: Docker (Recommended)

```bash
# 1. Clone the repository
git clone <repository-url>
cd Dino-Ventures-Backend_Engineer_Assignment

# 2. Create environment file
cp .env.example .env
# Edit .env and set your PostgreSQL password

# 3. Start PostgreSQL container
docker-compose up -d postgres

# 4. Wait for PostgreSQL to be ready (5-10 seconds)
docker-compose logs -f postgres

# 5. Initialize database schema and seed data
docker exec -i wallet-postgres psql -U postgres -d wallet_service < migrations/000001_initial_schema.up.sql
docker exec -i wallet-postgres psql -U postgres -d wallet_service < scripts/seed_all.sql

# 6. Build and run the wallet service
go build -o wallet-service.exe cmd/wallet-service/main.go
./wallet-service.exe

# 7. Access the application
# API: http://localhost:8080
# Frontend Demo: http://localhost:8080/
```

### Option 2: Local Development (Windows)

```bash
# 1. Install PostgreSQL locally or use Docker for PostgreSQL only
docker run -d --name wallet-postgres -e POSTGRES_PASSWORD=your_password -p 5432:5432 postgres:15-alpine

# 2. Create database
docker exec -it wallet-postgres psql -U postgres -c "CREATE DATABASE wallet_service;"

# 3. Setup environment
cp .env.example .env
# Edit .env with your database credentials

# 4. Initialize database
docker exec -i wallet-postgres psql -U postgres -d wallet_service < migrations/000001_initial_schema.up.sql
docker exec -i wallet-postgres psql -U postgres -d wallet_service < scripts/seed_all.sql

# 5. Install Go dependencies
go mod download

# 6. Build and run
go build -o wallet-service.exe cmd/wallet-service/main.go
./wallet-service.exe

# 7. Open browser to http://localhost:8080
```

### Verify Installation

```bash
# Check service health
curl http://localhost:8080/health

# Check user balance
curl http://localhost:8080/api/v1/accounts/3/balances/gold_coins

# Open frontend demo
# Navigate to http://localhost:8080 in your browser
```

## 🎨 Frontend Demo

The service includes a modern web interface for testing and demonstration:

- **URL**: http://localhost:8080
- **Features**:
  - Account switching (User 1, User 2, User 3)
  - Real-time balance display
  - Transaction execution (Top-up, Bonus, Spend)
  - Transaction history with pagination
  - Responsive design

## 📡 API Endpoints

### Execute Transaction

```bash
POST /api/v1/transactions
Content-Type: application/json
Idempotency-Key: unique-key-123

{
  "type": "topup",
  "asset_type_id": "gold_coins",
  "amount": 1000,
  "from_account_id": 1,
  "to_account_id": 3,
  "metadata": {
    "payment_reference": "PAY-12345",
    "description": "Purchase 1000 gold coins"
  }
}
```

**Transaction Types**:

- `topup`: Purchase credits (from treasury to user)
- `bonus`: Free credits (from treasury to user)
- `spend`: Use credits (from user to revenue account)

### Get Balance

```bash
GET /api/v1/accounts/{account_id}/balances/{asset_type_id}
```

### Get All Balances

```bash
GET /api/v1/accounts/{account_id}/balances
```

### Get Transaction History

```bash
GET /api/v1/accounts/{account_id}/transactions?limit=10&offset=0
```

### Health Check

```bash
GET /health
```

## 🧪 Testing the API

```bash
# 1. Check service health
curl http://localhost:8080/health

# 2. Get user balance (should show initial seed data)
curl http://localhost:8080/api/v1/accounts/3/balances/gold_coins

# 3. Execute a spend transaction
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-spend-123" \
  -d '{
    "type": "spend",
    "asset_type_id": "gold_coins",
    "amount": 100,
    "from_account_id": 3,
    "to_account_id": 2,
    "metadata": {"description": "Buy premium item"}
  }'

# 4. Check balance again (should be reduced by 100)
curl http://localhost:8080/api/v1/accounts/3/balances/gold_coins

# 5. Test idempotency (same request should return existing transaction)
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-spend-123" \
  -d '{
    "type": "spend",
    "asset_type_id": "gold_coins",
    "amount": 100,
    "from_account_id": 3,
    "to_account_id": 2,
    "metadata": {"description": "Buy premium item"}
  }'
```

## 🏛️ Database Schema

The service uses a production-grade double-entry ledger system:

- **asset_types**: Credit types (Gold Coins, Diamonds, etc.)
- **accounts**: User and system accounts
- **transactions**: Transaction records with idempotency
- **ledger_entries**: Immutable ledger (NO balance column!)

### Key Design Decisions

1. **No Balance Column**: Balances computed from `SUM(amount)` in ledger_entries
2. **Account-Level Locking**: Prevents race conditions on zero-balance accounts
3. **Idempotency Inside Transaction**: Uses `INSERT ... ON CONFLICT` pattern
4. **Pending → Completed Status**: Transaction only completed after ledger entries exist
5. **Immutable Ledger**: Database triggers prevent UPDATE/DELETE on ledger_entries

## 🌱 Seed Data

The service comes with pre-populated test data:

**Asset Types**:

- `gold_coins`: Primary in-game currency
- `diamonds`: Premium currency
- `loyalty_points`: Reward points

**System Accounts**:

- Treasury (ID: 1): Source for top-ups and bonuses (1M gold_coins, 100K diamonds, 100K loyalty_points)
- Revenue (ID: 2): Destination for user spending

**User Accounts**:

- user_001 (ID: 3): 5,000 gold_coins, 100 diamonds, 1,000 loyalty_points
- user_002 (ID: 4): 3,000 gold_coins, 50 diamonds, 500 loyalty_points
- user_003 (ID: 5): 10,000 gold_coins, 200 diamonds, 2,000 loyalty_points

## ⚙️ Environment Variables

| Variable             | Default                                               | Description                      |
| -------------------- | ----------------------------------------------------- | -------------------------------- |
| `DATABASE_URL`       | `postgres://localhost/wallet_service?sslmode=disable` | PostgreSQL connection string     |
| `SERVER_PORT`        | `8080`                                                | HTTP server port                 |
| `MAX_OPEN_CONNS`     | `25`                                                  | Database connection pool size    |
| `MAX_IDLE_CONNS`     | `5`                                                   | Idle connections in pool         |
| `CONN_MAX_LIFETIME`  | `5m`                                                  | Maximum connection lifetime      |
| `CONN_MAX_IDLE_TIME` | `5m`                                                  | Maximum connection idle time     |
| `IDEMPOTENCY_TTL`    | `24h`                                                 | Idempotency key retention period |

## 🏗️ Project Structure

```
Dino-Ventures-Backend_Engineer_Assignment/
├── cmd/wallet-service/          # Main application entry point
│   └── main.go
├── internal/                    # Private application code
│   ├── config/                  # Configuration management
│   │   ├── config.go           # Environment variables
│   │   └── database.go         # Database connection
│   ├── models/                  # Data structures
│   │   └── models.go           # Domain models
│   ├── repository/              # Database access layer
│   │   └── repository.go       # PostgreSQL implementation
│   ├── service/                 # Business logic layer
│   │   └── service.go          # Transaction service
│   └── handler/                 # HTTP handlers (API endpoints)
│       └── handler.go          # REST API handlers
├── migrations/                  # Database migration files
│   └── 000001_initial_schema.up.sql
├── scripts/                     # Utility scripts (seed data, setup)
│   ├── seed_asset_types.sql
│   ├── seed_accounts.sql
│   ├── seed_initial_balances.sql
│   └── seed_all.sql            # Combined seed script
├── web/static/                  # Frontend demo
│   └── index.html              # Single-page application
├── .env.example                 # Environment template
├── Dockerfile                   # Docker container definition
├── docker-compose.yml           # Multi-container setup
├── go.mod                       # Go dependencies
└── README.md                    # This file
```

## 🔒 Production-Grade Features

### Concurrency Safety

- SERIALIZABLE isolation level
- Deterministic lock ordering (prevents deadlocks)
- Account-level locking before balance computation
- Retry logic with exponential backoff

### Data Integrity

- Double-entry ledger with immutable records
- Database triggers prevent ledger tampering
- Foreign key constraints and check constraints
- Balances computed from ledger entries (no balance column)

### Idempotency

- Unique idempotency keys prevent duplicate transactions
- Idempotency check inside database transaction
- Safe for network retries and client errors

### Error Handling

- Comprehensive error types and HTTP status codes
- Graceful degradation and proper error messages
- Request validation and sanitization

## 🚢 Deployment

### Docker Deployment

```bash
# Production deployment
docker-compose -f docker-compose.yml up -d

# Scale the service
docker-compose up -d --scale wallet-service=3
```

### Manual Deployment

1. Set up PostgreSQL database
2. Run migrations: `psql -f migrations/000001_initial_schema.up.sql`
3. Run seed data: `psql -f scripts/seed_all.sql`
4. Set environment variables
5. Deploy the binary: `go build -o wallet-service cmd/wallet-service/main.go`

## 🧪 Load Testing

The service is designed for high-traffic scenarios:

- Connection pooling handles concurrent requests
- SERIALIZABLE isolation prevents race conditions
- Retry logic handles temporary failures
- Proper indexing ensures fast queries

## 🤝 Assignment Requirements Coverage

This implementation addresses all core requirements and brownie points:

### ✅ Core Requirements

- **Data Seeding**: Complete seed scripts with asset types, system accounts, and user accounts
- **API Endpoints**: RESTful API with transaction execution, balance queries, and transaction history
- **Functional Logic**: All three transaction flows implemented (top-up, bonus, spend)
- **Concurrency**: SERIALIZABLE isolation with account-level locking
- **Idempotency**: INSERT ... ON CONFLICT pattern inside transactions

### 🌟 Brownie Points

- **Deadlock Avoidance**: Deterministic lock ordering by account ID
- **Ledger-Based Architecture**: Double-entry ledger with NO balance column
- **Containerization**: Complete Docker and docker-compose setup
- **Frontend Demo**: Modern web interface for testing
- **Production-Grade**: Immutable ledger, retry logic, comprehensive error handling

## 📝 Technology Choices

**Why Go?**

- High performance and low latency
- Excellent concurrency primitives
- Strong typing and compile-time safety
- Great PostgreSQL driver (pgx/v5)

**Why PostgreSQL?**

- ACID compliance with SERIALIZABLE isolation
- Row-level locking for concurrency control
- Robust constraint system
- Excellent performance for financial data

**Concurrency Strategy**:

1. SERIALIZABLE isolation level prevents anomalies
2. Account-level locking before balance computation
3. Deterministic lock ordering (sorted by account ID) prevents deadlocks
4. Retry logic with exponential backoff handles transient failures
5. Idempotency check inside transaction prevents duplicates

## 📝 License

This project is part of the Dino Ventures Backend Engineer Assignment.
