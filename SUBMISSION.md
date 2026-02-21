# Dino Ventures Backend Engineer Assignment - Submission

## 📦 Repository Information

**GitHub Repository**: [Your GitHub URL here]

## 🎯 Assignment Completion Summary

This submission implements a production-grade internal wallet service for high-traffic applications with complete coverage of all core requirements and brownie points.

## ✅ Core Requirements

### A. Data Seeding & Setup

- ✅ Asset Types: `gold_coins`, `diamonds`, `loyalty_points`
- ✅ System Accounts: Treasury (ID: 1), Revenue (ID: 2)
- ✅ User Accounts: user_001, user_002, user_003 with initial balances
- ✅ Seed Script: `scripts/seed_all.sql` (combined) or individual scripts

### B. API Endpoints

- ✅ `POST /api/v1/transactions` - Execute transactions
- ✅ `GET /api/v1/accounts/{id}/balances` - Get all balances
- ✅ `GET /api/v1/accounts/{id}/balances/{asset}` - Get specific balance
- ✅ `GET /api/v1/accounts/{id}/transactions` - Transaction history
- ✅ `GET /health` - Health check

### C. Functional Logic

- ✅ **Top-up**: Treasury → User (purchase credits)
- ✅ **Bonus**: Treasury → User (free credits)
- ✅ **Spend**: User → Revenue (use credits)

### D. Critical Constraints

- ✅ **Concurrency**: SERIALIZABLE isolation + account-level locking
- ✅ **Race Conditions**: Deterministic lock ordering prevents deadlocks
- ✅ **Idempotency**: `INSERT ... ON CONFLICT` pattern inside transactions

## 🌟 Brownie Points

### 1. Deadlock Avoidance ✅

- Deterministic lock ordering (sorted by account ID)
- Accounts always locked in ascending order
- Prevents circular wait conditions

### 2. Ledger-Based Architecture ✅

- Double-entry ledger system
- NO balance column in database
- Balances computed from `SUM(amount)` in ledger_entries
- Immutable ledger with database triggers

### 3. Containerization ✅

- Complete `Dockerfile` for application
- `docker-compose.yml` for PostgreSQL + app
- One-command setup: `docker-compose up -d`

### 4. Frontend Demo ✅

- Modern web interface at `http://localhost:8080`
- Account switching, balance display, transaction execution
- Real-time transaction history

## 🏗️ Architecture Highlights

### Production-Grade Patterns

1. **No Balance Column**: Prevents race conditions and ensures audit trail
2. **Account-Level Locking**: Locks account row before computing balance
3. **Idempotency Inside Transaction**: Uses `INSERT ... ON CONFLICT DO NOTHING`
4. **Transaction Status Flow**: `pending` → `completed` (only after ledger entries)
5. **Immutable Ledger**: Database triggers prevent UPDATE/DELETE

### Concurrency Strategy

- SERIALIZABLE isolation level
- Account-level locking with `FOR UPDATE`
- Deterministic lock ordering
- Retry logic with exponential backoff
- Handles serialization failures and deadlocks

## 🚀 Quick Start

```bash
# 1. Clone repository
git clone [your-repo-url]
cd Dino-Ventures-Backend_Engineer_Assignment

# 2. Setup environment
cp .env.example .env
# Edit .env with your PostgreSQL password

# 3. Start PostgreSQL
docker-compose up -d postgres

# 4. Initialize database
docker exec -i wallet-postgres psql -U postgres -d wallet_service < migrations/000001_initial_schema.up.sql
docker exec -i wallet-postgres psql -U postgres -d wallet_service < scripts/seed_all.sql

# 5. Build and run
go build -o wallet-service.exe cmd/wallet-service/main.go
./wallet-service.exe

# 6. Test
curl http://localhost:8080/health
# Open browser: http://localhost:8080
```

## 📊 Test Scenarios

### 1. Basic Transaction

```bash
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-001" \
  -d '{
    "type": "spend",
    "asset_type_id": "gold_coins",
    "amount": 100,
    "from_account_id": 3,
    "to_account_id": 2
  }'
```

### 2. Idempotency Test

```bash
# Run same request twice - should return same transaction
curl -X POST http://localhost:8080/api/v1/transactions \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: test-001" \
  -d '{...same payload...}'
```

### 3. Balance Check

```bash
curl http://localhost:8080/api/v1/accounts/3/balances/gold_coins
```

### 4. Transaction History

```bash
curl http://localhost:8080/api/v1/accounts/3/transactions?limit=10
```

## 🛠️ Technology Stack

- **Backend**: Go 1.21+ (high performance, excellent concurrency)
- **Database**: PostgreSQL 15 (ACID compliance, row-level locking)
- **Driver**: pgx/v5 (high-performance PostgreSQL driver)
- **Router**: Gorilla Mux (HTTP routing)
- **Container**: Docker + Docker Compose

## 📁 Key Files

- `cmd/wallet-service/main.go` - Application entry point
- `internal/service/service.go` - Business logic with concurrency handling
- `internal/repository/repository.go` - Database operations
- `internal/handler/handler.go` - REST API endpoints
- `migrations/000001_initial_schema.up.sql` - Database schema
- `scripts/seed_all.sql` - Seed data
- `web/static/index.html` - Frontend demo
- `README.md` - Complete documentation

## 🎓 Learning Outcomes

This implementation demonstrates:

- Production-grade transaction handling
- Advanced concurrency control
- Idempotency patterns
- Ledger-based accounting
- Clean architecture principles
- Docker containerization
- RESTful API design

## 📝 Notes

- All temporary/test files removed from repository
- `.env` excluded via `.gitignore` (use `.env.example`)
- Complete documentation in README.md
- Frontend demo included for easy testing
- Production-ready error handling and validation

---

**Submitted by**: [Your Name]
**Date**: February 22, 2026
**Assignment**: Dino Ventures Backend Engineer - Internal Wallet Service
