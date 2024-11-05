# Mini-Scan Take-Home Project

This project is a simple application simulating a scanning and message processing system using Google Cloud Pub/Sub and PostgreSQL. It pulls scan results from a Pub/Sub subscription, processes them, and maintains an up-to-date record of each unique `(ip, port, service)`. This information is then stored in a PostgreSQL database, normalized between two tables: `ip_addresses` and `messages`.

## Table of Contents

- [Requirements](#requirements)
- [Setup and Installation](#setup-and-installation)
- [Running the Application](#running-the-application)
- [Database Schema](#database-schema)
- [Testing](#testing)
- [Acknowledgments](#acknowledgments)

## Requirements

- **Go** (version 1.16+)
- **Docker** and **Docker Compose** for running PostgreSQL and Pub/Sub emulator
- **Google Cloud SDK** for Pub/Sub emulation

## Setup and Installation

### 1. Clone the Repository

```bash
git clone https://github.com/DisKiDKelp/mini-scan-takehome.git
cd mini-scan-takehome
```

### 2. Environment Variables

Create an `.env` file in the project root with the following environment variables for the application:

```bash
# Pub/Sub
PUBSUB_EMULATOR_HOST=localhost:8085
PUBSUB_PROJECT_ID=test-project

# PostgreSQL Database
POSTGRES_HOST=localhost
POSTGRES_USER=user
POSTGRES_PASSWORD=password
POSTGRES_DB=mini_scan_db
```

### 3. Run Services with Docker Compose

Use Docker Compose to spin up PostgreSQL and the Pub/Sub emulator:

```bash
docker-compose up --build
```

This starts the following services:
- **PostgreSQL**: Used for storing IP address and message data.
- **Google Pub/Sub Emulator**: Used to simulate Pub/Sub for local development.
- **Scanner**: Used to simulate inserts into Pub/Sub for local development.
- **Consumer**: Service inserting into local PostgreSQL

### 4. Initialize the Database

The application will automatically create the required tables (`ip_addresses` and `messages`) if they don't exist. You can modify or reset the database by updating the `NewPostgresConnection` function in `internal/db/database.go`.

## Running the Application

Once the environment is set up, you can start the main consumer application locally with:

```bash
go run cmd/consumer/main.go
```

This will start listening for messages from the Pub/Sub subscription and store processed information in the database.

## Database Schema

The project uses a normalized schema with two tables:

- **ip_addresses**: Stores unique `(ip, port)` pairs, along with `service`, `last_updated`, and a `lock_id` for concurrency control.
- **messages**: Stores individual messages with a foreign key reference to `ip_addresses(id)`, along with fields for `message_content`, `service`, `response`, and `insertion_time`.

### Schema Definition

```sql
CREATE TABLE ip_addresses (
    id SERIAL PRIMARY KEY,
    ip TEXT,
    port INTEGER,
    service TEXT,
    last_updated TIMESTAMP,
    lock_id TEXT,
    UNIQUE (ip, port)
);

CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    ip_address_id INTEGER REFERENCES ip_addresses(id) ON DELETE CASCADE,
    message_content TEXT,
    service TEXT,
    response TEXT,
    insertion_time TIMESTAMP
);
```

### Testing notes

The docker logs will also have spit out of the messages being tracked and inserted, to verify you should be able to hook up to the locally created PostgreSQL and see data being inserted and used.

## Acknowledgments

This project was created as a coding exercise. It uses Go, Google Cloud Pub/Sub emulator, and PostgreSQL.
