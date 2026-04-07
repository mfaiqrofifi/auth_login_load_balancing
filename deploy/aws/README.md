# AWS Production Architecture

This document describes a practical AWS production architecture for this authentication system.

The goal is to keep the design:

- production-ready
- beginner-friendly
- cost-aware
- easy to operate

## Recommended choice: ECS Fargate

Use **ECS Fargate** instead of EC2.

Why Fargate is the better fit for this project:

- the app is already containerized with Docker
- no EC2 server management is needed
- easier scaling behind an Application Load Balancer
- simpler operational model for a beginner production setup
- better focus on the application instead of VM patching and capacity planning

Choose EC2 only if you need:

- lower cost at larger steady-state scale
- deeper host-level control
- special sidecars or kernel-level customization

For this project, **Fargate is the practical default**.

## High-level architecture diagram

```text
Client
  |
  v
Route 53 (optional custom domain)
  |
  v
Application Load Balancer (public subnets)
  |
  v
ECS Service on Fargate (private subnets, multiple tasks)
  |                    \
  |                     \
  v                      v
ElastiCache Redis      RDS PostgreSQL
(private subnets)      (private subnets)
  |
  v
CloudWatch Logs / Metrics / Alarms

Secrets Manager
  |
  v
ECS task runtime injects secrets into containers
```

## Request flow

```text
client -> ALB -> ECS Fargate app tasks -> Redis / PostgreSQL
```

Detailed flow:

1. Client sends HTTPS request to the public ALB endpoint
2. ALB forwards the request to a healthy ECS task in the target group
3. The Go auth service handles the request
4. The app reads shared runtime state from ElastiCache Redis
5. The app reads and writes durable data in RDS PostgreSQL
6. Logs and metrics go to CloudWatch

## Core AWS services

### 1. Application Load Balancer

Use ALB as the public entry point.

Responsibilities:

- terminate HTTPS with ACM certificate
- route traffic to ECS tasks
- run health checks against `/health`
- distribute traffic across multiple app tasks

Recommended health check:

- path: `/health`
- success code: `200`
- interval: 15-30 seconds
- unhealthy threshold: 2-3
- healthy threshold: 2

### 2. ECS Fargate

Run the Go app as an ECS service with multiple tasks.

Recommended first production shape:

- 2 tasks minimum across 2 availability zones
- 0.25 vCPU or 0.5 vCPU to start
- 512 MB or 1 GB RAM to start
- autoscaling later based on CPU and memory

Why multiple tasks matter:

- one task can fail while another keeps serving traffic
- ALB only routes traffic to healthy tasks
- rolling deployments are easier

### 3. RDS PostgreSQL

Use RDS PostgreSQL for durable data:

- users
- sessions
- audit logs

Recommended beginner-friendly setup:

- PostgreSQL managed by RDS
- private subnets only
- automatic backups enabled
- Multi-AZ optional at the start, recommended later for stronger availability

### 4. ElastiCache Redis

Use ElastiCache Redis for shared fast-changing auth state:

- refresh token metadata
- refresh token rotation state
- rate limit counters

Recommended beginner-friendly setup:

- single Redis node first for cost control
- private subnets only
- enable auth token / in-transit encryption when moving to stricter production posture

### 5. Secrets Manager

Use Secrets Manager for sensitive values:

- `DB_PASSWORD`
- `JWT_ACCESS_SECRET`
- `REDIS_PASSWORD`
- future API keys or third-party credentials

Do not store secrets in:

- Docker image
- committed `.env`
- ECS task definition plaintext where avoidable

### 6. CloudWatch

Use CloudWatch for:

- application logs
- ECS task logs
- CPU and memory metrics
- ALB metrics
- alarms for unhealthy targets or restart spikes

Recommended alarms:

- ALB unhealthy host count > 0
- ECS task restart spikes
- RDS CPU or storage pressure
- Redis memory pressure

## Stateless app instances behind ALB

This app is a good fit for ALB because the application layer is stateless.

Important idea:

- app tasks do not store critical auth state only in memory
- PostgreSQL stores durable records
- Redis stores shared runtime auth state

That means:

- login can hit task A
- refresh can hit task B
- logout can hit task C

and the auth flow still works correctly.

This is exactly why multiple ECS tasks behind an ALB are practical for this project.

## Environment variables and secret management

### Safe to keep as normal environment variables

These are configuration values, not secrets:

- `APP_ENV`
- `PORT`
- `APP_NAME`
- `APP_INSTANCE_NAME`
- `DB_HOST`
- `DB_PORT`
- `DB_NAME`
- `DB_SSLMODE`
- `REDIS_HOST`
- `REDIS_PORT`
- `REDIS_DB`
- `JWT_ACCESS_TTL_MINUTES`
- `REFRESH_TOKEN_TTL_HOURS`
- `JWT_ISSUER`
- `JWT_AUDIENCE`
- `COOKIE_DOMAIN`
- `COOKIE_SECURE`
- `COOKIE_SAMESITE`
- `REQUEST_TIMEOUT_SECONDS`
- `SHUTDOWN_TIMEOUT_SECONDS`
- `TRUST_PROXY_HEADERS`
- `LOG_LEVEL`
- `CORS_ALLOWED_ORIGINS`
- `CORS_ALLOWED_METHODS`
- `CORS_ALLOWED_HEADERS`
- `CORS_ALLOW_CREDENTIALS`
- rate limit config values

### Store in Secrets Manager

These should be treated as secrets:

- `DB_PASSWORD`
- `JWT_ACCESS_SECRET`
- `REDIS_PASSWORD`

### Runtime pattern

Recommended pattern:

1. keep non-secret config in ECS task definition environment variables
2. keep secrets in Secrets Manager
3. inject secrets into ECS task at runtime
4. restrict IAM permissions so only the ECS task role can read those secrets

## Health checks

### ALB health check

Use:

- path: `/health`
- matcher: `200`

The endpoint should stay lightweight and quick.

### ECS container health

Use the same `/health` endpoint inside the container if needed.

### What healthy means in this project

At minimum:

- process is alive
- app can serve HTTP

Later improvements:

- add deeper readiness checks for Redis and PostgreSQL if desired
- keep liveness and readiness concerns separate if complexity grows

## Networking layout

Recommended VPC layout:

- public subnets:
  - ALB
- private subnets:
  - ECS tasks
  - RDS PostgreSQL
  - ElastiCache Redis

Recommended security group flow:

- internet -> ALB on `443`
- ALB -> ECS service on app port
- ECS -> RDS on PostgreSQL port
- ECS -> Redis on Redis port

Do not expose RDS or Redis publicly.

## Deployment checklist

- container image builds successfully
- image pushed to ECR
- VPC with at least 2 AZs prepared
- ALB created with HTTPS listener
- ACM certificate attached to ALB
- ECS cluster and Fargate service created
- ECS service spans multiple subnets / AZs
- target group health check points to `/health`
- RDS PostgreSQL created in private subnets
- ElastiCache Redis created in private subnets
- Secrets stored in Secrets Manager
- ECS task role allowed to read required secrets
- non-secret env vars configured in ECS task definition
- CloudWatch log group configured for ECS tasks
- alarms configured for unhealthy targets and task instability
- cookie and CORS settings reviewed for real domain usage
- production `JWT_ACCESS_SECRET` is strong and rotated safely
- smoke test register, login, refresh, logout after deployment

## Beginner-friendly recommendation

If you want the simplest practical first AWS deployment:

- 1 ALB
- 1 ECS Fargate service
- 2 app tasks
- 1 RDS PostgreSQL instance
- 1 ElastiCache Redis node
- Secrets Manager for passwords and JWT secret
- CloudWatch logs and 2-3 basic alarms

This gives you:

- no VM management
- real load balancing
- managed database
- managed Redis
- a clean path to scale later

## Cost-aware recommendation

To control cost early:

- choose ECS Fargate with small task size first
- start with 2 app tasks only
- use a modest RDS instance size
- use a small single-node ElastiCache Redis setup first
- avoid over-provisioning alarms and observability tooling at the beginning

What not to cut too early:

- HTTPS on ALB
- Secrets Manager for real secrets
- multiple app tasks
- private networking for RDS and Redis

## How to validate after deployment

1. Open the ALB DNS name in requests to `/health`
2. Register a test user
3. Login and capture `access_token` and refresh cookie
4. Call `/auth/me` with the token
5. Call `/auth/refresh`
6. Check CloudWatch logs for request flow and errors
7. Confirm ALB target group shows healthy tasks

## Mapping from local architecture to AWS

Local:

- Nginx -> app containers -> PostgreSQL / Redis

AWS:

- ALB -> ECS Fargate tasks -> RDS PostgreSQL / ElastiCache Redis

So the architecture pattern stays the same.
Only the infrastructure becomes managed and production-oriented.
