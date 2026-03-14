# Sanchay IMS — From Godown to Dashboard

> **Digitizing India's Warehouses — Real-time. Centralized. Zero Guesswork.**

Sanchay is a modular Inventory Management System built to replace manual registers, Excel sheets, and scattered tracking methods with a single centralized, real-time platform. Built as part of the **Odoo Hackathon** virtual round.

---

## Table of Contents

- [Overview](#overview)
- [Tech Stack](#tech-stack)
- [Features](#features)
- [Project Structure](#project-structure)
- [Database Schema](#database-schema)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Backend Setup (GoLang)](#backend-setup-golang)
  - [Frontend Setup (React)](#frontend-setup-react)
  - [Database Setup (PostgreSQL)](#database-setup-postgresql)
- [Environment Variables](#environment-variables)
- [API Endpoints](#api-endpoints)
- [Build & Run](#build--run)
- [Screenshots](#screenshots)
- [Inventory Flow](#inventory-flow)
- [Hackathon Context](#hackathon-context)
- [License](#license)

---

## Overview

Sanchay IMS solves a real problem faced by thousands of Indian warehouse businesses — stock tracked in notebooks, Excel files that go out of sync, and no single source of truth.

**What Sanchay does:**
- Tracks every inbound and outbound stock movement in real time
- Provides a live dashboard with key inventory KPIs
- Supports receipts (goods IN), deliveries (goods OUT), internal transfers, and stock adjustments
- Logs every movement in a full stock ledger with timestamps

**Target Users:**
- Inventory Managers — manage stock, approve flows, view reports
- Warehouse Staff — perform transfers, picking, shelving, and counting

---

## Tech Stack

| Layer      | Technology              | Reason                                              |
|------------|-------------------------|-----------------------------------------------------|
| Frontend   | React.js                | Component-based UI, fast rendering                  |
| Backend    | GoLang (Golang)         | High concurrency, parallel connections, low latency |
| Database   | PostgreSQL              | Relational integrity for stock ledger accuracy      |
| Auth       | JWT + OTP               | Stateless auth, secure password reset               |
| Fonts      | Plus Jakarta Sans       | Clean, modern Indian SaaS aesthetic                 |
| Mono Font  | JetBrains Mono          | SKU codes, batch IDs, ledger entries                |

---

## Features

### Authentication
- [x] User Sign Up / Login
- [x] Forgot password flow with email existence check
- [x] 6-digit OTP email verification
- [x] Password reset with bcrypt hashing
- [x] JWT token authentication
- [x] Redirect to Dashboard on success

### Dashboard
- [x] Total Products in Stock
- [x] Low Stock / Out of Stock item count
- [x] Pending Receipts count
- [x] Pending Deliveries count
- [x] Internal Transfers scheduled

### Products Module
- [x] Create product (Name, SKU, Category, Unit of Measure, Initial Stock)
- [x] View all products with current stock quantity
- [x] Search by SKU or product name
- [x] Product categories

### Receipts — Goods IN
- [x] Create new receipt
- [x] Add supplier and products with quantities
- [x] Validate receipt → stock auto-increments
- [x] Status flow: Draft → Waiting → Ready → Done

### Delivery Orders — Goods OUT
- [x] Create delivery order
- [x] Pick and pack items
- [x] Validate → stock auto-decrements
- [x] Status flow: Draft → Waiting → Ready → Done

### Internal Transfers
- [x] Move stock from Location A → Location B
- [x] Total stock quantity unchanged
- [x] Location updated in ledger

### Stock Adjustments
- [x] Select product and location
- [x] Enter physically counted quantity
- [x] System auto-corrects and logs the adjustment with reason

### Move History / Stock Ledger
- [x] Every stock event listed chronologically
- [x] Timestamp, movement type, quantity delta, and user recorded
- [x] Filter by product, date, or movement type

### Additional
- [x] Low stock alerts
- [x] SKU search and smart filters
- [x] Multi-warehouse / location support

---

## Project Structure

```
sanchay-ims/
├── frontend/                   # React.js application
│   ├── public/
│   ├── src/
│   │   ├── components/
│   │   │   ├── SanchayLogo.jsx
│   │   │   ├── Navbar.jsx
│   │   │   ├── Sidebar.jsx
│   │   │   └── StatusBadge.jsx
│   │   ├── pages/
│   │   │   ├── Login.jsx
│   │   │   ├── Signup.jsx
│   │   │   ├── Dashboard.jsx
│   │   │   ├── Products.jsx
│   │   │   ├── Receipts.jsx
│   │   │   ├── Deliveries.jsx
│   │   │   ├── Transfers.jsx
│   │   │   ├── Adjustments.jsx
│   │   │   └── MoveHistory.jsx
│   │   ├── styles/
│   │   │   ├── global.css
│   │   │   ├── dashboard.css
│   │   │   ├── products.css
│   │   │   └── ...
│   │   ├── hooks/
│   │   │   ├── useAuth.js
│   │   │   └── useStock.js
│   │   ├── api/
│   │   │   └── client.js
│   │   ├── App.jsx
│   │   └── main.jsx
│   ├── package.json
│   └── vite.config.js
│
├── backend/                    # GoLang API server
│   ├── cmd/
│   │   └── main.go             # Entry point
│   ├── internal/
│   │   ├── auth/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── otp.go
│   │   ├── products/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   ├── receipts/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   ├── deliveries/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   ├── transfers/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   ├── adjustments/
│   │   │   ├── handler.go
│   │   │   ├── service.go
│   │   │   └── repository.go
│   │   ├── ledger/
│   │   │   ├── handler.go
│   │   │   └── repository.go
│   │   ├── middleware/
│   │   │   └── auth.go
│   │   └── db/
│   │       └── postgres.go
│   ├── go.mod
│   └── go.sum
│
├── database/
│   ├── migrations/
│   │   ├── 001_create_users.sql
│   │   ├── 002_create_warehouses.sql
│   │   ├── 003_create_products.sql
│   │   ├── 004_create_receipts.sql
│   │   ├── 005_create_deliveries.sql
│   │   ├── 006_create_transfers.sql
│   │   ├── 007_create_adjustments.sql
│   │   └── 008_create_stock_ledger.sql
│   └── seed.sql
│
├── .env.example
├── docker-compose.yml
└── README.md
```

---

## Database Schema

### Core Tables

```sql
-- Users
users (id, name, email, password_hash, otp_secret, created_at)

-- Warehouses / Locations
warehouses (id, name, code, address, created_at)
locations  (id, warehouse_id, name, code, created_at)

-- Products
categories (id, name)
products   (id, name, sku, category_id, unit_of_measure,
            reorder_threshold, created_at)

-- Stock (current quantity per product per location)
stock      (id, product_id, location_id, quantity, updated_at)

-- Receipts (goods IN)
receipts      (id, supplier_name, status, created_at, validated_at)
receipt_lines (id, receipt_id, product_id, expected_qty, received_qty)

-- Deliveries (goods OUT)
deliveries      (id, customer_name, status, created_at, validated_at)
delivery_lines  (id, delivery_id, product_id, ordered_qty, delivered_qty)

-- Internal Transfers
transfers      (id, from_location_id, to_location_id, status, created_at)
transfer_lines (id, transfer_id, product_id, quantity)

-- Stock Adjustments
adjustments    (id, product_id, location_id, old_qty,
                new_qty, reason, adjusted_at)

-- Stock Ledger (append-only event log)
stock_ledger   (id, product_id, location_id, movement_type,
                quantity_delta, reference_id, reference_type,
                note, created_at)
-- movement_type: RECEIPT | DELIVERY | TRANSFER_OUT | TRANSFER_IN | ADJUSTMENT
```

---

## Getting Started

### Prerequisites

Make sure you have the following installed:

- **Go** 1.21+
- **Node.js** 18+
- **PostgreSQL** 15+
- **Git**

```bash
go version    # go1.21+
node -v       # v18+
psql --version
```

---

### Backend Setup (GoLang)

```bash
# 1. Clone the repository
git clone https://github.com/your-username/sanchay-ims.git
cd sanchay-ims/backend

# 2. Install Go dependencies
go mod tidy

# 3. Copy environment file
cp ../.env.example ../.env
# Edit .env with your database credentials and JWT secret

# 4. Run the server
go run cmd/server/main.go
```

The API server starts at `http://localhost:8080`

---

### Frontend Setup (React)

```bash
# From the project root
# Install dependencies
npm install

# Start development server
npm run dev
```

The React app starts at `http://localhost:5173`

---

### Database Setup (PostgreSQL)

```bash
# Create the database
psql -U postgres -c "CREATE DATABASE sanchay_db;"

# Run migrations in order
psql -U postgres -d sanchay_db -f database/migrations/001_create_users.sql
psql -U postgres -d sanchay_db -f database/migrations/002_create_warehouses.sql
psql -U postgres -d sanchay_db -f database/migrations/003_create_products.sql
psql -U postgres -d sanchay_db -f database/migrations/004_create_receipts.sql
psql -U postgres -d sanchay_db -f database/migrations/005_create_deliveries.sql
psql -U postgres -d sanchay_db -f database/migrations/006_create_transfers.sql
psql -U postgres -d sanchay_db -f database/migrations/007_create_adjustments.sql
psql -U postgres -d sanchay_db -f database/migrations/008_create_stock_ledger.sql

# Optional: Load sample data
psql -U postgres -d sanchay_db -f database/seed.sql
```

---

## Environment Variables

Create a `.env` file in the project root based on `.env.example`:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=sanchay_db
DB_USER=postgres
DB_PASSWORD=your_password

# Server
SERVER_PORT=8080
SERVER_ENV=development

# Auth
JWT_SECRET=your_super_secret_key_here
JWT_EXPIRY_HOURS=24

# OTP (for password reset)
OTP_EXPIRY_MINUTES=10

# SMTP (forgot password emails)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=madhavbhayani21@gmail.com
SMTP_FROM=madhavbhayani21@gmail.com
SMTP_PASSWORD=your_gmail_app_password
```

For Gmail app passwords, values copied with spaces (for example `xxxx xxxx xxxx xxxx`) are accepted.

---

## API Endpoints

### Auth
| Method | Endpoint                  | Description              |
|--------|---------------------------|--------------------------|
| POST   | `/api/auth/signup`        | Create new account       |
| POST   | `/api/auth/login`         | Login, returns JWT       |
| POST   | `/api/auth/forgot-password/request` | Verify email exists and send 6-digit OTP |
| POST   | `/api/auth/forgot-password/verify`  | Verify the OTP for email |
| POST   | `/api/auth/forgot-password/reset`   | Reset password after OTP verification |

### Stocks
| Method | Endpoint               | Description              |
|--------|------------------------|--------------------------|
| GET    | `/api/stocks/meta`     | Categories + locations for stock forms |
| POST   | `/api/stocks/categories` | Create product category |
| GET    | `/api/stocks/products` | List/search stock products |
| POST   | `/api/stocks/products` | Create product |
| PUT    | `/api/stocks/products/{id}` | Update product |
| DELETE | `/api/stocks/products/{id}` | Delete product |

### Operations
| Method | Endpoint                        | Description              |
|--------|---------------------------------|--------------------------|
| GET    | `/api/operations/meta`          | Locations + products for operations |
| GET    | `/api/operations/receipts`      | List receipt orders |
| POST   | `/api/operations/receipts`      | Create receipt order |
| GET    | `/api/operations/delivery`      | List delivery orders |
| POST   | `/api/operations/delivery`      | Create delivery order |
| GET    | `/api/operations/orders/{operationType}/{referenceNumber}` | Get operation detail |
| PUT    | `/api/operations/orders/{operationType}/{referenceNumber}` | Update operation detail |
| POST   | `/api/operations/orders/{operationType}/{referenceNumber}/validate` | Validate order |
| POST   | `/api/operations/orders/{operationType}/{referenceNumber}/cancel` | Cancel order |
| DELETE | `/api/operations/orders/{id}`   | Delete operation order |

### Adjustments
| Method | Endpoint               | Description              |
|--------|------------------------|--------------------------|
| GET    | `/api/operations/adjustments` | List adjustment overview + history |
| POST   | `/api/operations/adjustments/transfer` | Internal transfer between locations |
| POST   | `/api/operations/adjustments/quantity` | Correct free-to-use quantity |

### Dashboard & Ledger
| Method | Endpoint               | Description              |
|--------|------------------------|--------------------------|
| GET    | `/api/dashboard/overview` | Dashboard counters + chart data |
| GET    | `/api/move-history`    | Stock ledger move history |

---

## Build & Run

### Run everything with Docker Compose

```bash
# From project root
docker-compose up --build
```

This starts PostgreSQL, runs migrations, and launches both the Go API and React frontend together.

### Build for production

```bash
# Frontend
npm run build

# Backend
cd backend && go build -o sanchay-server ./cmd/server
./sanchay-server
```

## Forgot Password Flow

1. User enters registered email on `/forgot-password`.
2. Backend checks if email exists in DB.
3. Backend sends a 6-digit OTP to the email.
4. User verifies OTP.
5. User sets a new password.
6. Backend stores new password as bcrypt hash.

---

## Inventory Flow

```
Vendor → [Receipt] → Stock +N → [Ledger Entry]
                                      ↓
                         [Internal Transfer] → Location Updated
                                      ↓
Customer ← [Delivery] ← Stock −N → [Ledger Entry]
                                      ↓
                        [Adjustment] → Mismatch Fixed → [Ledger Entry]
```

Every single operation writes to the Stock Ledger — an append-only log that serves as the single source of truth for all inventory history.

---

## Hackathon Context

**Event:** Odoo Hackathon — Virtual Round

**Problem Statement:** Build a modular Inventory Management System (IMS) that digitizes and streamlines all stock-related operations within a business.

**Approach Taken:**
- Single-role system (no access control) for clean, fast demo
- GoLang backend chosen specifically for its goroutine-based concurrency — ideal for real-time stock updates across parallel warehouse operations
- PostgreSQL chosen for ACID compliance — critical when stock numbers must be accurate
- Sanchay brand (Sanskrit: संचय = accumulation) designed with Indian warehouse context in mind

---

## License

MIT License — free to use, modify, and distribute.

---

> Built with care for India's warehouse workers.  
> **Sanchay** — *संचय · From Godown to Dashboard.*