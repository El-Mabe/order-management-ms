# Orders Service
Professional microservice for delivery order management, developed in Go using the Gin framework, implementing Clean Architecture, caching with Redis, and asynchronous messaging with Apache Kafka.

## 📚 Table of Contents
- [Features](#-features)
- [Architecture](#-architecture)
- [Prerequisites](#-prerequisites)
- [Installation & Execution](#-installation)
- [Usage Examples](#api-usage-examples)
- [Technical Decisions](#-tecnical_decisions)
- [Testing](#6-testing)
- [Documentation](#-documentation)

## ✨ Features

### Core Functionality

- ✅ Order CRUD: Create, retrieve, and update delivery orders
- ✅ State Management: Validated transitions (NEW → IN_PROGRESS → DELIVERED/CANCELLED)
- ✅ Filtering & Pagination: Search by status or customer with pagination
- ✅ Concurrency Control: Optimistic locking with versioning
- ✅ Redis Cache: Cache-aside pattern with a 60-second TTL
- ✅ Kafka Events: Publishes state change events asynchronously
- ✅ Health Check: Monitors the status of service dependencies

### Technical Features

- 🏗️ Clean Architecture: Layered separation (Domain, Application, Infrastructure)
- 🔒 Robust Validation: Input validation with descriptive error messages
- 📊 Structured Logging: JSON logs with Zap and request ID tracking
- 🐳 Containerization: Production-ready with Docker and Docker Compose
- 🧪 Comprehensive Testing: Unit tests with mocks and coverage >80%
- 🔄 Graceful Shutdown: Proper handling of connections and service termination

## 🧱 Architecture
```
┌─────────────────────────────────────┐
│      HTTP Layer (Gin)               │
│   Handlers + Middleware + Routes    │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│     Application Layer               │
│   OrderService (Business Logic)     │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│       Domain Layer                  │
│  Order, Events (Models + Rules)     │
└─────────────────────────────────────┘
               │
┌──────────────▼──────────────────────┐
│    Infrastructure Layer             │
│  MongoDB + Redis + Kafka            │
└─────────────────────────────────────┘
```

### Order States
```
    NEW
     │
     ├──────► IN_PROGRESS
     │             │
     │             ├──────► DELIVERED (final)
     │             │
     │             └──────► CANCELLED (final)
     │
     └──────► CANCELLED (final)
```
## 📦 Prerequisites

- Docker 20.10+
- Docker Compose 2.0+

## 🚀 Installation & Execution

### 1. Clone the repository
- git clone https://github.com/El-Mabe/order-management-ms.git
cd orders-service

### 2. Start all services (MongoDB, Redis, Kafka, API)
- docker-compose up -d

### 3. Wait ~30 seconds for all services to initialize

### 4. Verify that everything is running
- docker-compose ps

### 5. Check the service health
- curl http://localhost:3000/health


### 🧩 Available Services

- API: http://localhost:3000
- MongoDB: localhost:27017
- Redis: localhost:6379
- Kafka: localhost:9092
- Kafka UI: http://localhost:8080
- Mongo Express: http://localhost:8081 (username: admin, password: admin)

### 🩺 Health Check
- curl http://localhost:3000/health


### Expected response:
```
{
  "status": "healthy",
  "timestamp": "2025-10-25T10:00:00Z",
  "dependencies": {
    "mongodb": "connected",
    "redis": "connected",
    "kafka": "connected"
  }
}
```
## 📡 API Usage Examples
🟢 Create a New Order
```
curl -X POST http://localhost:3000/api/v1/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customerId": "123e4567-e89b-12d3-a456-426614174000",
    "items": [
      { "sku": "LAPTOP-001", "quantity": 2, "price": 999.99 },
      { "sku": "MOUSE-002", "quantity": 1, "price": 29.99 }
    ]
  }'
```

Response (201 Created):
```
{
  "orderId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "NEW",
  "totalAmount": 2029.97,
  "version": 1
}
```

🟠 Get Order by ID 
- curl http://localhost:3000/api/orders/550e8400-e29b-41d4-a716-446655440000

🟣 List Orders (with Filters & Pagination)
- curl "http://localhost:3000/api/orders?status=NEW&page=1&limit=10"

🔵 Update Order Status
- curl -X PATCH http://localhost:3000/api/orders/550e8400-e29b-41d4-a716-446655440000/status \
  -H "Content-Type: application/json" \
  -d '{ "status": "IN_PROGRESS" }'


Kafka Event (topic: orders.events):
```
{
  "eventType": "ORDER_STATUS_CHANGED",
  "orderId": "550e8400-e29b-41d4-a716-446655440000",
  "oldStatus": "NEW",
  "newStatus": "IN_PROGRESS"
}
```

## 🧠 Technical Decisions
### 🧩 1. Architecture

- Clean Architecture pattern: clear separation between HTTP, Application, Domain, and Infrastructure layers.

- Dependency Injection: the application layer depends only on abstractions (interfaces).

- Decoupled persistence: swapping MongoDB for another database requires minimal changes.

### 💾 2. Data Persistence

- **MongoDB** used for flexibility in storing nested order structures.

- Each document stores:

    - _id: UUID
    - status: Enum
    - version: for optimistic locking
    - createdAt / updatedAt

### ⚡ 3. Caching

- **Redis** follows the cache-aside pattern:

    - First read from cache
    - On miss → fetch from DB, cache the result with TTL 60s
    - On update → invalidate cache

### 📬 4. Messaging

- **Kafka** handles domain events (e.g., ORDER_CREATED, ORDER_STATUS_CHANGED).
- Producers in the application layer emit messages asynchronously after transaction commits.

### 🧱 5. Concurrency & Locking

- **Optimistic Locking** ensures safe concurrent updates using a version field.
- Updates require matching the current version — otherwise return conflict (409).

## 🧰 Testing

- Unit Tests using stretchr/testify for repositories, services, and handlers.
- Integration Tests with testcontainers-go to spin up Mongo, Redis, Kafka.
- Coverage report:
```
go test ./...
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📖 Documentation

### **with Docsify**

This project uses [Docsify](https://docsify.js.org/) to serve project documentation directly in the browser.

### Prerequisites

- Node.js installed
- npm (comes with Node.js)
- Docsify CLI installed globally:
  - npm i docsify-cli -g
### Serve Docsify locally
- Navigate to the project root:
  - cd orders-service
- Start Docsify from the docs folder:
  - docsify serve docs
- Open the browser at:
  - http://localhost:3000

### **with Swagger**
This project includes interactive API documentation powered by Swagger UI
 and generated automatically using swaggo/swag.

### 🚀 Accessing Swagger UI

Once the service is running (using Docker Compose):
- docker-compose up

You can open the Swagger UI in your browser at:

👉 http://localhost:3000/api/swagger/index.html