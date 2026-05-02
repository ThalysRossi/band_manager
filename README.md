# Band Manager

This is a POC that will be discarded after validation in favour of organic hand rolled code. For now, I let Sam take the wheel and gave up reviewing or trying to steer the agent in a better direction.

Band Manager is a mobile-first web app for Brazilian underground bands to manage merch inventory and merch booth sales.

The alpha is organized as a monorepo with a React/Vite frontend, Go API backend, shared OpenAPI contract, and local PostgreSQL/Redis dependencies.

## Workspace

```txt
apps/
  api/      Go backend
  web/      React frontend
packages/
  api-contract/
  config/
  i18n/
docs/
  spikes/
```

## Local setup

```bash
pnpm install
docker compose up -d postgres redis
pnpm dev:web
```

Confirm local Postgres is healthy before running database-backed API work:

```bash
docker compose exec postgres pg_isready -U band_manager -d band_manager
```

The local database URL is:

```bash
postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable
```

Run the API with explicit local environment variables:

```bash
APP_ENV=local \
API_ADDR=:8080 \
API_ALLOWED_ORIGINS=http://localhost:5173 \
DATABASE_URL=postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable \
REDIS_URL=redis://localhost:6379/0 \
SUPABASE_JWT_SECRET=replace-me \
MERCADOPAGO_ACCESS_TOKEN=replace-me \
MERCADOPAGO_WEBHOOK_SECRET=replace-me \
MERCADOPAGO_POINT_TERMINAL_ID=replace-me \
pnpm dev:api
```

Run Postgres integration tests only after the Compose database is up and healthy:

```bash
cd apps/api
DATABASE_URL='postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable' \
GOCACHE=/tmp/go-build-cache \
go test ./internal/infrastructure/postgres/... -v
```

If local port `5432` is already in use, stop the conflicting service or change the Compose port mapping and use the matching `DATABASE_URL`.

## Validation

```bash
pnpm lint
pnpm test
cd apps/api && go test ./...
```

## Current implementation state

- MercadoPago spike completed in `docs/spikes/mercadopago.md`.
- Monorepo skeleton is in place.
- API foundation exposes `GET /healthz`.
- Frontend foundation renders the initial app shell with Portuguese/English translation support.
- Auth/account foundation defines owner signup contracts, alpha role rules, and account tables.

## License

Copyright (C) 2026 Band Manager contributors.

Band Manager is licensed under the GNU General Public License version 3 only. See `LICENSE` for the full license text.

Third-party dependency license notes are tracked in `docs/legal/third-party-notices.md`.

## Original planning bundle

- `initial_codex_prompt.md`: prompt to give Codex before implementation.
- `plans.md`: implementation plan and architecture.
- `agents.md`: lean agent role instructions.
- `roadmap.md`: product and implementation roadmap.
