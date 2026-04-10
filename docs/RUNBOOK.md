# Runbook

This runbook gives you a single documentation entry point for:

- local development mode
- production-style local simulation
- cloud or VPS deployment
- Swagger UI
- load-balancing proof

## Quick Choices

Choose one mode:

1. Local mode
   Runs PostgreSQL, Redis, two app instances, and Nginx from [`docker-compose.yml`](../docker-compose.yml).
2. Cloud or VPS mode
   Runs only two app instances and Nginx from [`docker-compose.prod.yml`](../docker-compose.prod.yml), while PostgreSQL and Redis come from external providers such as Neon and Upstash.

## Swagger UI

Start Swagger UI:

```bash
docker compose -f docker-compose.docs.yml up -d
```

Open:

- `http://localhost:8090/docs`

The UI reads [`docs/openapi.yaml`](openapi.yaml), which already includes both local and cloud server options.

## Diagrams

### Deployment view

```mermaid
flowchart LR
    Client[Client Browser or API Client] --> Nginx[Nginx Reverse Proxy]
    Nginx --> App1[app-1]
    Nginx --> App2[app-2]
    App1 --> Postgres[(PostgreSQL)]
    App2 --> Postgres
    App1 --> Redis[(Redis)]
    App2 --> Redis
    Swagger[Swagger UI] -. reads spec .-> OpenAPI[OpenAPI YAML]
```

### Local vs cloud decision

```mermaid
flowchart TD
    Start[Need to run the auth service] --> Decision{Where are Postgres and Redis?}
    Decision -->|Inside Docker on this machine| Local[Use docker-compose.yml]
    Decision -->|Managed or remote services| Cloud[Use docker-compose.prod.yml]
    Local -. ships .-> PG[Postgres container]
    Local -. ships .-> RD[Redis container]
    Cloud -. expects .-> Neon[External PostgreSQL]
    Cloud -. expects .-> Upstash[External Redis]
```

### Load-balancing proof flow

```mermaid
flowchart LR
    Request1[Request 1] --> Proxy[Nginx]
    Request2[Request 2] --> Proxy
    Request10[Request 10] --> Proxy
    Proxy --> App1[app-1]
    Proxy --> App2[app-2]
    App1 -. returns instance_name .-> Result[Proof output]
    App2 -. returns instance_name .-> Result
```

## Local Mode

Use local mode when you want Docker to run everything for you.

1. Copy `.env.example` to `.env`.
2. Start the stack:

```bash
docker compose up -d --build
```

3. Check health:

```bash
curl http://localhost:8080/health
```

Local mode services:

- Nginx on `http://localhost:8080`
- PostgreSQL on `localhost:5432`
- Redis on `localhost:6379`

## Production-Style Local Simulation

Use this if you want to test the production compose file on your own machine while PostgreSQL and Redis are still running outside that compose file.

1. Start local PostgreSQL and Redis only:

```bash
docker compose up -d postgres redis
```

2. Export production env vars. Example:

```bash
export COOKIE_DOMAIN=example.com
export DATABASE_URL='postgres://postgres:postgres@host.docker.internal:5432/auth_service?sslmode=disable'
export REDIS_URL='redis://host.docker.internal:6379/0'
export JWT_ACCESS_SECRET='replace-with-a-real-random-secret-at-least-32-characters'
export CORS_ALLOWED_ORIGINS='https://example.com'
docker compose -f docker-compose.prod.yml up -d --build
```

On Windows PowerShell, use `$env:NAME="value"` instead of `export`.

## Cloud or VPS Mode

Use this when PostgreSQL and Redis are external services.

Example providers:

- PostgreSQL: Neon
- Redis: Upstash

Example `.env` values:

```env
APP_ENV=production
DATABASE_URL=postgresql://USER:PASSWORD@HOST/DBNAME?sslmode=require&channel_binding=require
REDIS_URL=rediss://default:PASSWORD@HOST:6379/0
JWT_ACCESS_SECRET=replace-with-a-real-random-secret-at-least-32-characters
COOKIE_DOMAIN=auth.example.com
CORS_ALLOWED_ORIGINS=https://app.example.com
```

Start:

```bash
docker compose -f docker-compose.prod.yml --env-file .env up -d --build
```

## Load-Balancing Proof

This project exposes `instance_name` from `/health` and also sends `X-App-Instance`, so you can prove requests are reaching both app instances.

### Linux or macOS

```bash
chmod +x docs/scripts/prove-load-balancing.sh
./docs/scripts/prove-load-balancing.sh http://localhost:8080/health 10
```

### PowerShell

```powershell
./docs/scripts/prove-load-balancing.ps1 -Url http://localhost:8080/health -Requests 10
```

Expected result:

- repeated requests should show a mix of `app-1` and `app-2`
- if you only see one instance, inspect Nginx config and container health

## Recommended Verification

Run these after deployment:

```bash
curl http://localhost/health
curl http://localhost/auth/login
docker ps
docker compose -f docker-compose.prod.yml logs --tail=100
```

For a public host:

```bash
curl http://auth.example.com/health
```

## Notes

- Production compose intentionally does not create PostgreSQL or Redis.
- Upstash Redis requires `rediss://` because TLS is required.
- If `COOKIE_SECURE=true`, browser cookie flows should be tested over HTTPS.
