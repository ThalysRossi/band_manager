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

Run the API with explicit local environment variables:

```bash
APP_ENV=local \
API_ADDR=:8080 \
API_ALLOWED_ORIGINS=http://localhost:5173 \
DATABASE_URL=postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable \
REDIS_URL=redis://localhost:6379/0 \
SUPABASE_JWT_SECRET=replace-me \
MERCADOPAGO_ACCESS_TOKEN=replace-me \
pnpm dev:api
```

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
