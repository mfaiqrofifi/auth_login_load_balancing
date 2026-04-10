# Auth Login Load Balancing

A production-ready Go authentication service with PostgreSQL, Redis, Docker, and Nginx load balancing. The app stays stateless so multiple containers can sit behind a reverse proxy, while PostgreSQL stores durable auth data and Redis stores refresh-token/runtime state.

## Features

- Register, login, refresh, logout, logout-all, profile, and session management endpoints
- JWT access tokens with issuer/audience validation
- HttpOnly refresh-token cookies with rotation and reuse detection
- PostgreSQL-backed users, sessions, and audit logs
- Redis-backed refresh-token state and auth rate limiting
- JSON responses, request IDs, structured logs, timeouts, and recovery middleware
- Docker image healthcheck and Nginx reverse proxy/load balancer

## Documentation

- OpenAPI spec: [`docs/openapi.yaml`](docs/openapi.yaml)
- Runbook: [`docs/RUNBOOK.md`](docs/RUNBOOK.md)
- Load-balancing proof script for Bash: [`docs/scripts/prove-load-balancing.sh`](docs/scripts/prove-load-balancing.sh)
- Load-balancing proof script for PowerShell: [`docs/scripts/prove-load-balancing.ps1`](docs/scripts/prove-load-balancing.ps1)

Run Swagger UI locally:

```powershell
docker compose -f docker-compose.docs.yml up -d
```

Then open:

```text
http://localhost:8090
```

## Project Structure

```text
.
|-- cmd/server
|-- db
|-- deploy/aws
|-- infra/nginx
|-- internal/config
|-- internal/database
|-- internal/handler
|-- internal/middleware
|-- internal/model
|-- internal/repository
|-- internal/service
|-- Dockerfile
|-- docker-compose.yml
|-- docker-compose.prod.yml
|-- Makefile
`-- .env.example
```

## Environment

Create your local `.env` from the example:

```powershell
Copy-Item .env.example .env
```

Important variables:

- `APP_ENV`: use `development` locally and `production` on a VPS
- `PORT`: app container listens on this port; default is `8080`
- `DATABASE_URL`: preferred PostgreSQL connection string, for example external RDS/managed PostgreSQL
- `REDIS_URL`: preferred Redis connection string, for example external ElastiCache/managed Redis
- `JWT_ACCESS_SECRET`: must be a strong random value in production, at least 32 characters
- `COOKIE_DOMAIN`: set to your real domain in production
- `COOKIE_SECURE`: forced to `true` when `APP_ENV=production`
- `CORS_ALLOWED_ORIGINS`: set explicit frontend origins when cookies/credentials are used

The older host/port fields (`DB_HOST`, `REDIS_HOST`, and related variables) still work for local/manual runs, but URL-based config is the cleaner production path.

## Run Locally Without Docker

Start PostgreSQL and Redis on your machine, then apply the schema:

```powershell
psql -h localhost -U postgres -d auth_service -f db/schema.sql
go run ./cmd/server
```

Health check:

```powershell
curl.exe http://localhost:8080/health
```

With Make:

```powershell
make run
```

## Docker Local Run

The default `docker-compose.yml` is for local development. It runs PostgreSQL, Redis, two app containers, and Nginx.

```powershell
Copy-Item .env.example .env
docker compose up --build
```

Public endpoints:

- `http://localhost:8080` -> Nginx load balancer
- PostgreSQL -> `localhost:5432`
- Redis -> `localhost:6379`

Load-balancer check:

```powershell
curl.exe -i http://localhost:8080/health
```

With Make:

```powershell
make docker-local
make logs
make stop
```

## Production-Style Docker Run

`docker-compose.prod.yml` mirrors a VPS deployment where PostgreSQL and Redis are external services. It only runs:

- `nginx`
- `app-1`
- `app-2`

Set production secrets in the VPS environment or `.env` file. Do not commit the real `.env`.

Required production values:

```env
APP_ENV=production
DATABASE_URL=postgres://USER:PASSWORD@HOST:5432/DBNAME?sslmode=require
REDIS_URL=redis://:PASSWORD@HOST:6379/0
JWT_ACCESS_SECRET=replace-with-a-real-random-secret-at-least-32-chars
COOKIE_DOMAIN=auth.example.com
CORS_ALLOWED_ORIGINS=https://example.com
```

Start production-style:

```powershell
docker compose -f docker-compose.prod.yml up --build -d
```

With Make:

```powershell
make docker-prod
```

## Nginx

Nginx config lives in:

- `infra/nginx/nginx.conf`
- `infra/nginx/conf.d/auth.conf`

The upstream load balances traffic across:

- `app-1:8080`
- `app-2:8080`

The app binds to `0.0.0.0:${PORT}`, so it works inside containers and behind a reverse proxy. `/health` returns a small JSON response and does not expose secrets, making it safe for container and proxy health checks.

## Contabo VPS and AWS Mapping

This VPS layout intentionally mirrors the later AWS version:

- Contabo Nginx -> AWS Application Load Balancer
- Contabo app containers -> ECS tasks
- External PostgreSQL URL -> RDS PostgreSQL
- External Redis URL -> ElastiCache Redis
- VPS `.env`/environment variables -> Secrets Manager
- Docker healthcheck and `/health` -> ECS/ALB health checks

On a real VPS, put Nginx behind HTTPS using your domain and certificate tooling, keep database and Redis private, and expose only the reverse proxy publicly.

## Deployment Notes

- Use `APP_ENV=production` on the VPS.
- Keep `COOKIE_SECURE=true` in production. The app also forces this when `APP_ENV=production`.
- Set `COOKIE_DOMAIN` to the domain that should receive the refresh-token cookie.
- Use `DATABASE_URL` for the external PostgreSQL connection. Use `sslmode=require` when your provider supports TLS.
- Use `REDIS_URL` for the external Redis connection.
- Keep secrets in environment variables or a server-side `.env` file that is never committed.
- Configure the domain and HTTPS at the Nginx/VPS layer before using secure cookies from a browser.

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

Refresh:

```powershell
curl.exe -X POST http://localhost:8080/auth/refresh `
  -b cookies.txt `
  -c cookies.txt
```

## Test

```powershell
go test ./...
go build ./cmd/server
```

or:

```powershell
make test
```
