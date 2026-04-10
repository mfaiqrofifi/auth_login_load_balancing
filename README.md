# Auth Login Load Balancing

Production-ready Go authentication service with PostgreSQL, Redis, Docker, and Nginx load balancing. The service is intentionally stateless at the app layer, so multiple containers can serve the same API behind a reverse proxy while PostgreSQL stores durable auth data and Redis stores refresh-token state plus rate-limit counters.

## Repository Description

Suggested GitHub repository description:

`Stateless Go authentication service with PostgreSQL, Redis, Docker, and Nginx load balancing for local and VPS/cloud deployment.`

## What This Project Does

- user registration and login
- JWT access token issuing and validation
- HttpOnly refresh token cookie rotation
- logout current session and logout all sessions
- session listing and session revocation
- PostgreSQL-backed users, sessions, and audit logs
- Redis-backed refresh token state and rate limiting
- Nginx reverse proxy in front of multiple app instances
- local Docker setup and production-style Docker setup

## Architecture

### High-Level View

```mermaid
flowchart LR
    Client[Browser or API Client] --> Nginx[Nginx Reverse Proxy]
    Nginx --> App1[Go API app-1]
    Nginx --> App2[Go API app-2]
    App1 --> Postgres[(PostgreSQL)]
    App2 --> Postgres
    App1 --> Redis[(Redis)]
    App2 --> Redis
    Swagger[Swagger UI] -. local docs .-> OpenAPI[OpenAPI spec]
```

### Request Flow

```mermaid
flowchart TD
    Start[Incoming request] --> Proxy[Nginx]
    Proxy --> Instance1[app-1]
    Proxy --> Instance2[app-2]
    Instance1 -. reads and writes .-> PG[(PostgreSQL)]
    Instance2 -. reads and writes .-> PG
    Instance1 -. stores tokens and rate limits .-> RD[(Redis)]
    Instance2 -. stores tokens and rate limits .-> RD
    Instance1 -. returns instance_name .-> ClientResult[Client response]
    Instance2 -. returns instance_name .-> ClientResult
```

### Local vs Production Mode

```mermaid
flowchart TD
    Need[Need to run the project] --> Choice{Where do Postgres and Redis live?}
    Choice -->|Run everything in Docker| Local[Use docker-compose.yml]
    Choice -->|Use managed or external services| Prod[Use docker-compose.prod.yml]
    Local -. starts .-> LocalPg[Postgres container]
    Local -. starts .-> LocalRedis[Redis container]
    Prod -. expects .-> Neon[External PostgreSQL]
    Prod -. expects .-> Upstash[External Redis]
```

## Main Components

### App Instances

- two Go API containers: `app-1` and `app-2`
- both run the same binary
- both expose the same endpoints
- each response can include `instance_name` so you can prove load balancing

### Nginx

- sits in front of the Go app containers
- forwards requests to both app instances
- exposes a single public port
- performs simple reverse proxy and load balancing duties

### PostgreSQL

- stores users
- stores sessions
- stores audit logs
- schema lives in `db/schema.sql`

### Redis

- stores refresh token state
- stores refresh token reuse information
- stores rate-limiting data

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
|-- cmd/server                 # application entrypoint
|-- internal/config            # configuration loading and validation
|-- internal/database          # PostgreSQL and Redis connection setup
|-- internal/handler           # HTTP handlers and route wiring
|-- internal/middleware        # request ID, CORS, auth, timeout, logging, recovery
|-- internal/repository        # PostgreSQL and Redis repositories
|-- internal/service           # business logic
|-- db/schema.sql              # database schema
|-- infra/nginx               # Nginx config
|-- deploy/aws                # AWS deployment notes
|-- docker-compose.yml        # local all-in-one stack
|-- docker-compose.prod.yml   # production-style stack with external DB/Redis
|-- docker-compose.docs.yml   # Swagger UI docs stack
|-- docs/openapi.yaml         # OpenAPI documentation
|-- docs/RUNBOOK.md           # extra runbook
|-- Dockerfile                # app container image
`-- .env.example              # local environment template
```

## API Overview

### Public Endpoints

- `GET /`
- `GET /health`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`

### Protected Endpoints

- `POST /auth/logout-all`
- `GET /auth/me`
- `GET /auth/sessions`
- `DELETE /auth/sessions/{sessionId}`

### Token Model

- access token:
  returned in JSON response body
- refresh token:
  returned as HttpOnly cookie
- `/auth/me` and other protected endpoints:
  require `Authorization: Bearer <access_token>`
- `/auth/refresh`:
  requires the `refresh_token` cookie

## Environment Variables

Important variables:

- `APP_ENV`
  `development` locally, `production` on VPS or cloud
- `PORT`
  internal app container port, default `8080`
- `DATABASE_URL`
  preferred PostgreSQL DSN
- `REDIS_URL`
  preferred Redis DSN
- `JWT_ACCESS_SECRET`
  must be strong in production, minimum 32 characters
- `COOKIE_DOMAIN`
  domain for refresh-token cookie in production
- `COOKIE_SECURE`
  forced to `true` in production
- `COOKIE_SAMESITE`
  cookie same-site mode
- `CORS_ALLOWED_ORIGINS`
  explicit browser origins allowed to call the API
- `TRUST_PROXY_HEADERS`
  whether to trust Nginx forwarded headers

The older host-based variables such as `DB_HOST`, `DB_PORT`, `REDIS_HOST`, and `REDIS_PORT` still work for manual runs, but `DATABASE_URL` and `REDIS_URL` are the cleaner production path.

## Quick Start

### Option 1: Local Docker Stack

This is the easiest way to run everything on one machine.

1. copy the env file

```powershell
Copy-Item .env.example .env
```

2. start the stack

```powershell
docker compose up -d --build
```

3. open the API through Nginx

```text
http://localhost:8080
```

4. verify health

```powershell
curl.exe http://localhost:8080/health
```

Local services:

- API through Nginx: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

### Option 2: Run Without Docker

Use this if PostgreSQL and Redis are already running on your machine.

1. apply database schema

```powershell
psql -h localhost -U postgres -d auth_service -f db/schema.sql
```

2. start the app

```powershell
go run ./cmd/server
```

3. verify health

```powershell
curl.exe http://localhost:8080/health
```

## Swagger and API Docs

OpenAPI and Swagger are included for local documentation and testing.

Files:

- `docs/openapi.yaml`
- `docs/RUNBOOK.md`
- `docker-compose.docs.yml`

Start Swagger UI:

```powershell
docker compose -f docker-compose.docs.yml up -d
```

Open:

```text
http://localhost:8090
```

Notes:

- browser-based Swagger testing needs CORS to allow `http://localhost:8090`
- login stores the access token for protected bearer endpoints in the local Swagger helper
- refresh still depends on the refresh-token cookie flow

## Docker Files Explained

### `docker-compose.yml`

Use this for local development.

It starts:

- `postgres`
- `redis`
- `app-1`
- `app-2`
- `nginx`

This file is best when you want one command to boot the whole stack.

### `docker-compose.prod.yml`

Use this for VPS or cloud-style deployment.

It starts only:

- `app-1`
- `app-2`
- `nginx`

It does not start PostgreSQL or Redis. Those must come from:

- Neon
- Upstash
- cloud-managed services
- any other external PostgreSQL or Redis provider

## Production-Style Run

Example production `.env`:

```env
APP_ENV=production
DATABASE_URL=postgresql://USER:PASSWORD@HOST/DBNAME?sslmode=require&channel_binding=require
REDIS_URL=rediss://default:PASSWORD@HOST:6379/0
JWT_ACCESS_SECRET=replace-with-a-real-random-secret-at-least-32-characters
COOKIE_DOMAIN=auth.example.com
CORS_ALLOWED_ORIGINS=https://app.example.com
```

Start production-style stack:

```bash
docker compose -f docker-compose.prod.yml --env-file .env up -d --build
```

Important:

- `REDIS_URL` should use `rediss://` when your provider requires TLS
- `COOKIE_DOMAIN` should match your auth domain
- `COOKIE_SECURE=true` means browser refresh-cookie flows should be tested over HTTPS

## VPS Deployment Guide

### Deployment Shape

```mermaid
flowchart LR
    Public[Public Internet] --> Domain[Domain or nip.io]
    Domain --> NginxVPS[Nginx on VPS]
    NginxVPS --> App1VPS[app-1 container]
    NginxVPS --> App2VPS[app-2 container]
    App1VPS -. external connection .-> PgCloud[(Neon or PostgreSQL)]
    App2VPS -. external connection .-> PgCloud
    App1VPS -. external connection .-> RedisCloud[(Upstash or Redis)]
    App2VPS -. external connection .-> RedisCloud
```

### Typical VPS Steps

1. clone the repository on the VPS
2. create a production `.env`
3. make sure your Nginx server name or domain is correct
4. start the production stack
5. test `/health`
6. add HTTPS before testing browser cookie flows

Example command:

```bash
docker compose -f docker-compose.prod.yml --env-file .env up -d --build
```

Verify:

```bash
docker ps
curl http://localhost/health
docker compose -f docker-compose.prod.yml logs --tail=100
```

## Nginx Notes

Nginx config lives in:

- `infra/nginx/nginx.conf`
- `infra/nginx/conf.d/auth.conf`

The upstream points traffic to:

- `app-1:8080`
- `app-2:8080`

If you use a public domain or `nip.io`, make sure:

- `server_name` matches the public hostname
- `COOKIE_DOMAIN` matches the hostname used by the browser
- `CORS_ALLOWED_ORIGINS` matches the frontend origin

## Example Requests

### Register

```powershell
curl.exe -X POST http://localhost:8080/auth/register `
  -H "Content-Type: application/json" `
  -d "{\"email\":\"user@example.com\",\"password\":\"plainpassword\"}"
```

### Login

```powershell
curl.exe -X POST http://localhost:8080/auth/login `
  -H "Content-Type: application/json" `
  -c cookies.txt `
  -d "{\"email\":\"user@example.com\",\"password\":\"plainpassword\"}"
```

### Get Current User

```powershell
curl.exe http://localhost:8080/auth/me `
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Refresh Access Token

```powershell
curl.exe -X POST http://localhost:8080/auth/refresh `
  -b cookies.txt `
  -c cookies.txt
```

### Logout

```powershell
curl.exe -X POST http://localhost:8080/auth/logout `
  -b cookies.txt `
  -c cookies.txt
```

## Load-Balancing Proof

Because the service exposes `instance_name`, you can prove Nginx is distributing requests between both app containers.

PowerShell:

```powershell
./docs/scripts/prove-load-balancing.ps1 -Url http://localhost:8080/health -Requests 10
```

Bash:

```bash
chmod +x docs/scripts/prove-load-balancing.sh
./docs/scripts/prove-load-balancing.sh http://localhost:8080/health 10
```

Expected result:

- a mix of `app-1` and `app-2`
- if all requests go to one instance only, inspect Nginx config and health checks

## Testing

Run unit and package tests:

```powershell
go test ./...
go build ./cmd/server
```

or:

```powershell
make test
```

## Useful Make Targets

```powershell
make run
make docker-local
make docker-prod
make logs
make stop
make test
```

## Troubleshooting

### `COOKIE_DOMAIN is missing`

Production compose requires `COOKIE_DOMAIN`. Set it in the environment or in your production `.env`.

### App container is unhealthy

Common causes:

- invalid `DATABASE_URL`
- invalid `REDIS_URL`
- PostgreSQL password mismatch
- managed service host unreachable
- JWT secret too short in production

### `Bind for 0.0.0.0:80 failed`

Another process is already using port `80`. Change the host mapping or stop the conflicting service.

### Swagger says unauthorized on `/auth/me`

- login again so Swagger gets a fresh access token
- hard refresh the browser
- if needed, clear local storage for the Swagger site

### Refresh token returns unauthorized

- `/auth/refresh` depends on the `refresh_token` cookie
- login again first
- cookie-based flows are best tested over proper HTTP or HTTPS with matching cookie settings

## Cloud Mapping

This project can map cleanly to a later cloud deployment:

- VPS Nginx -> cloud load balancer
- app containers -> ECS tasks or container service
- external PostgreSQL -> RDS or managed PostgreSQL
- external Redis -> ElastiCache or managed Redis
- production `.env` -> secret manager

For AWS-specific notes, see:

- `deploy/aws/README.md`

## Final Notes

- local mode is best for development
- production mode is best for VPS or cloud
- run Swagger only when you need interactive documentation
- secure-cookie browser flows should be tested behind HTTPS in real deployments
