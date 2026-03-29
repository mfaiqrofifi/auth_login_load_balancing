# Auth Login Load Balancing

A production-oriented authentication service built with Go, PostgreSQL, Redis, Docker, and Nginx.

This project started as a clean login foundation and grew into a stateless authentication system with:

- user registration and login
- JWT access tokens
- refresh token rotation
- session management
- logout and logout-all
- audit logging
- Redis-backed rate limiting
- local multi-instance deployment behind Nginx

## Tech Stack

- Go (`net/http`)
- PostgreSQL
- Redis
- Docker
- Nginx

## Why `net/http`

The project uses the Go standard library HTTP stack to keep the service lightweight, explicit, and easy to scale without framework lock-in. That makes the codebase easier to reason about while still being production-friendly.

## Features

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `POST /auth/logout-all`
- `GET /auth/me`
- `GET /auth/sessions`
- `DELETE /auth/sessions/:id`
- `GET /health`

Security-related features:

- bcrypt password hashing
- JWT short-lived access token
- opaque refresh token stored in `HttpOnly` cookie
- refresh token rotation and reuse detection
- Redis-backed token/session state
- Redis-backed rate limiting for register, login, and refresh
- PostgreSQL audit logs for important auth events

## Project Structure

```text
.
|-- cmd/server
|-- db
|-- deploy/nginx
|-- internal/config
|-- internal/database
|-- internal/handler
|-- internal/middleware
|-- internal/model
|-- internal/repository
|-- internal/service
|-- Dockerfile
`-- docker-compose.yml
```

Folder summary:

- `cmd/server`: application bootstrap and HTTP server startup
- `internal/config`: environment variable loading
- `internal/database`: PostgreSQL and Redis initialization
- `internal/handler`: request/response layer
- `internal/middleware`: auth, logging, recovery, rate limiting, instance headers
- `internal/model`: request, response, and domain models
- `internal/repository`: PostgreSQL and Redis access
- `internal/service`: authentication business logic
- `db`: SQL schema
- `deploy/nginx`: local load balancer configuration

## Auth Flow

### Register

1. Client sends email and password to `POST /auth/register`
2. Handler validates the request body
3. Service hashes the password with bcrypt
4. Repository stores the user in PostgreSQL
5. Audit log is written

### Login

1. Client sends email and password to `POST /auth/login`
2. Service verifies credentials against PostgreSQL
3. A new session is stored in PostgreSQL
4. A short-lived JWT access token is generated
5. A refresh token is generated and stored in Redis
6. Refresh token is returned as an `HttpOnly` cookie
7. Access token is returned in JSON

### Refresh

1. Client calls `POST /auth/refresh`
2. Handler reads refresh token from cookie
3. Service checks refresh token state in Redis
4. Old refresh token is marked used/revoked during rotation
5. A new access token and new refresh token are issued
6. Session `last_used_at` is updated

### Logout

- `POST /auth/logout`: revoke current refresh token and clear cookie
- `POST /auth/logout-all`: revoke all refresh tokens and all active sessions for the current user

### Session Management

- `GET /auth/sessions`: list active and revoked sessions for the current user
- `DELETE /auth/sessions/:id`: revoke one session/device

## Data Design

### PostgreSQL

PostgreSQL stores durable data:

- `users`
- `sessions`
- `audit_logs`

Schema file:

- [`db/schema.sql`](db/schema.sql)

### Redis

Redis stores fast-changing auth state:

- refresh token metadata
- refresh token indexes by user and session
- rate limiting counters

This separation keeps the application instances stateless and ready for horizontal scaling.

## Environment Variables

Core app:

- `PORT`
- `APP_NAME`
- `APP_INSTANCE_NAME`

PostgreSQL:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSLMODE`

Redis:

- `REDIS_HOST`
- `REDIS_PORT`
- `REDIS_PASSWORD`
- `REDIS_DB`

JWT and refresh tokens:

- `JWT_ACCESS_SECRET`
- `JWT_ACCESS_TTL_MINUTES`
- `REFRESH_TOKEN_TTL_HOURS`

Cookie settings:

- `COOKIE_DOMAIN`
- `COOKIE_SECURE`
- `COOKIE_SAMESITE`

Rate limiting:

- `LOGIN_RATE_LIMIT_REQUESTS`
- `LOGIN_RATE_LIMIT_WINDOW_SECONDS`
- `REGISTER_RATE_LIMIT_REQUESTS`
- `REGISTER_RATE_LIMIT_WINDOW_SECONDS`
- `REFRESH_RATE_LIMIT_REQUESTS`
- `REFRESH_RATE_LIMIT_WINDOW_SECONDS`

## Run Locally Without Docker

1. Create PostgreSQL database
2. Apply schema
3. Make sure Redis is running
4. Configure `.env`
5. Start the server

Example:

```powershell
psql -h localhost -U postgres -d "auth_service" -f db/schema.sql
go run ./cmd/server
```

Health check:

```powershell
curl.exe http://localhost:8080/health
```

## Run With Docker Compose

This project includes:

- `postgres`
- `redis`
- `app-1`
- `app-2`
- `app-3`
- `nginx`

Start everything:

```powershell
docker compose up --build
```

Public ports:

- `http://localhost:8080` -> Nginx load balancer
- `http://localhost:8081` -> app-1
- `http://localhost:8082` -> app-2
- `http://localhost:8083` -> app-3

Direct instance health checks:

```powershell
curl.exe http://localhost:8081/health
curl.exe http://localhost:8082/health
curl.exe http://localhost:8083/health
```

Load balancer health check:

```powershell
curl.exe -i http://localhost:8080/health
```

## Nginx Load Balancing

Nginx is configured in:

- [`deploy/nginx/nginx.conf`](deploy/nginx/nginx.conf)

The `upstream` block groups the three app instances:

- `app-1:8080`
- `app-2:8080`
- `app-3:8080`

Because no balancing algorithm is specified, Nginx uses default round robin behavior.

Round robin test:

```powershell
1..10 | ForEach-Object { curl.exe -s http://localhost:8080/health }
```

Readable version:

```powershell
1..10 | ForEach-Object { curl.exe -s http://localhost:8080/health | ConvertFrom-Json | Select-Object instance_name,timestamp }
```

You should see requests distributed across `app-1`, `app-2`, and `app-3`.

## Example Requests

Register:

```powershell
curl.exe -X POST http://localhost:8080/auth/register `
  -H "Content-Type: application/json" `
  -d "{\"email\":\"user@example.com\",\"password\":\"plainpassword\"}"
```

Login:

```powershell
curl.exe -X POST http://localhost:8080/auth/login `
  -H "Content-Type: application/json" `
  -c cookies.txt `
  -d "{\"email\":\"user@example.com\",\"password\":\"plainpassword\"}"
```

Get profile:

```powershell
curl.exe http://localhost:8080/auth/me `
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

Refresh access token:

```powershell
curl.exe -X POST http://localhost:8080/auth/refresh `
  -b cookies.txt `
  -c cookies.txt
```

Logout current session:

```powershell
curl.exe -X POST http://localhost:8080/auth/logout `
  -b cookies.txt `
  -c cookies.txt
```

Logout all sessions:

```powershell
curl.exe -X POST http://localhost:8080/auth/logout-all `
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" `
  -b cookies.txt `
  -c cookies.txt
```

## Scaling Idea

This service is designed to be stateless at the application layer:

- app instances do not store auth state in memory
- PostgreSQL stores durable records
- Redis stores shared runtime auth state

That means:

- login can hit `app-1`
- refresh can hit `app-3`
- logout can hit `app-2`

and the system still behaves consistently.

## Current Status

This project is a strong local foundation for:

- JWT auth
- refresh token rotation
- session/device management
- load balancing
- future Redis hardening
- future cloud deployment

Good next steps:

- HTTPS and secure cookie behavior for non-local environments
- migration tooling
- access-token revocation strategy
- observability and tracing
- deployment to AWS or Kubernetes
