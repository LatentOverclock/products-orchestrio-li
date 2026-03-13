# products-orchestio-li

Product management app implemented with the req-trace flow (`github.com/trace-code-org/req-trace`).

## Requirements source
- `requirements/project.v1.md`
- `requirements/project.v2.md`
- `requirements/project.v3.md`
- `requirements/project.v4.md`

## Stack
- Frontend: TypeScript + React
- Backend: Go + GraphQL
- Database: PostgreSQL
- Deployment: Docker Compose + Traefik

## Local run
```bash
make up
```

Frontend: `http://localhost:5174`
Backend health: `http://localhost:8180/health`

## Tests
```bash
make test
```

## Authentication

Required backend auth env variables:
- `AUTH_JWT_SECRET`
- `AUTH_ADMIN_EMAIL`
- `AUTH_ADMIN_PASSWORD`

At startup, the backend ensures the configured admin user exists.

## Deploy
```bash
make deploy
```

Set deployment config in `.env`:
- `APP_HOST`
- `AUTH_JWT_SECRET`
- `AUTH_ADMIN_EMAIL`
- `AUTH_ADMIN_PASSWORD`
