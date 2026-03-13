# products-orchestio-li

Product management app implemented with the req-trace flow (`github.com/trace-code-org/req-trace`).

## Requirements source
- `requirements/project.v1.md`

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

## Deploy
```bash
make deploy
```

Set `APP_HOST` in `.env` (deployment config boundary).
