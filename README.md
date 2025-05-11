# frappuccino

### entity relationship diagram:

<img src="img/ERD.png" alt="entity relationship diagram">

# Frappuccino Order Management System

## Overview

Frappuccino is a comprehensive order management system designed for coffee shops and cafes. It provides robust APIs for handling orders, inventory, reporting, and search functionality.

## Features

- **Order Management**: Create, read, update, and delete orders
- **Inventory Tracking**: Real-time inventory updates with transactions
- **Reporting**: Sales reports, popular items, and period-based analytics
- **Search**: Full-text search across menu items, orders, and ingredients
- **Batch Processing**: Handle multiple orders in a single transaction

## API Documentation

### Base URL

`http://localhost:9090/`

### Authentication

[Describe your authentication method here]

### Endpoints

#### Order Endpoints

    "POST /orders"
    "GET /orders/{id}"
    "PUT /orders/{id}"
    "DELETE /orders/{id}"
    "POST /orders/{id}/close"
    "GET /orders"
    "POST /orders/batch-process"
    "GET /orders/numberOfOrderedItems"

#### Inventory Endpoints

    "POST /inventory"
    "GET /inventory/{id}"
    "PUT /inventory/{id}"
    "DELETE /inventory/{id}"
    "GET /inventory"
    "GET /inventory/getLeftOvers"

#### Menu routes

    "POST /menu"
    "GET /menu/{id}"
    "PUT /menu/{id}"
    "DELETE /menu/{id}"
    "GET /menu"

#### Report Endpoints

```

"GET /reports/orderedItemsByPeriod"
"GET /reports/search"
"GET /reports/total-sales"
"GET /reports/popular-items"

```

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 15+

### Docker Setup

```bash
docker-compose up
```

## Testing

Run unit tests:

```bash
make test
```

Run integration tests:

```bash
make test-integration
```

## Examples

### Create an Order

```bash
curl -X POST http://localhost:9090/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": 1,
    "items": [
      {
        "menu_item_id": 4,
        "quantity": 2,
        "customizations": {"milk": "almond"}
      }
    ]
  }'
```

### Get Sales Report

```bash
curl "http://localhost:9090/reports/sales?start_date=2023-01-01&end_date=2023-01-31"
```

## Database Schema

![Database Schema Diagram](docs/db_schema.png

[Describe key tables and relationships]

## Configuration

Environment variables:

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=frappuccino
SERVER_PORT=9090
```

## License

[MIT License](LICENSE

## Support

For support, please open an issue or contact [your email].
