# plans.md — Band Manager Alpha Implementation Plan

## 1. Product objective

Build a mobile-first web app for Brazilian underground bands to manage merch inventory and merch booth sales.

The alpha must be fully functional, not static-only. It should support a real backend, PostgreSQL persistence, authentication, object storage, payment-provider integration, and deployment from a monorepo.

The alpha focuses on the most business-rule-heavy areas:

1. Inventory
2. Merch Booth

The backend should expose the complete alpha API surface before non-critical UI work. The alpha frontend is limited to login, inventory, and merch booth. Financial reports, calendar, and offline UI are deferred after their backend contracts are stable.

## 2. Key product decisions

### Account model

- One band per account.
- A user may belong to multiple bands in the future.
- Alpha UI assumes one active band.
- Data model must support future multi-band memberships.

### Roles

Roles:

- owner
- admin
- member
- viewer

Alpha permissions:

- The sign-up user is the owner.
- Invited users are viewers.
- Only the owner can create, update, and delete in alpha.
- Viewers can read only.

Future permissions:

- owner/admin/member can manage inventory and operate merch booth.
- owner/admin can delete.
- viewer can read only.

### Auth

Alpha must support:

- email/password
- email verification
- password reset
- Google login

Deferred:

- Meta login through Facebook/Instagram

Preferred alpha auth approach:

- Use Supabase Auth if the deployment target is free/managed.
- Keep auth access behind an adapter.
- Do not let auth-provider choice leak into domain logic.

Alternative later:

- Keycloak if the project moves toward full self-hosting.

### Language

- Portuguese and English supported from alpha.
- Use i18n from the beginning.
- No hardcoded user-facing strings.

### Currency

- Alpha currency is BRL.
- Store money as integer minor units plus currency code.
- Example: R$ 42,00 => `amount = 4200`, `currency = "BRL"`.
- Do not use floating-point numbers for money.

### Timezone

- Store timestamps as UTC.
- Use browser timezone for alpha display and date-range selection.
- Store detected `band_timezone` at signup to avoid future report/calendar ambiguity.

## 3. Monorepo deployment strategy

A monorepo does not require deploying backend and frontend to the same service.

Use one GitHub repository with separate deploy targets:

```txt
band-manager/
  apps/
    web/      # React/Vite frontend
    api/      # Go backend
  packages/
    api-contract/
    i18n/
    config/
```

Deployment services should be configured with root directories or build commands:

| Service | Repo root setting | Build/start behavior |
|---|---|---|
| Frontend | `apps/web` | Build Vite app and publish static assets |
| Backend | `apps/api` | Build/run Go API or Docker image |
| Shared OpenAPI | `packages/api-contract` | Used during CI/codegen, not deployed alone |

If a deploy provider cannot see shared packages when `rootDir` is set, use one of these patterns:

1. Build from repository root with service-specific commands.
2. Use Docker with repository root as context and `apps/api/Dockerfile`.
3. Copy generated artifacts into each app during CI.
4. Keep shared code minimal and generated.

Preferred for this project:

- Frontend deploys from `apps/web`.
- Backend deploys with Docker using root build context so it can access shared OpenAPI files.
- GitHub Actions validates the entire monorepo before deploy.

## 4. Free/low-cost alpha deployment options

The alpha has at most 3 concurrent users, so free-tier cold starts and small storage limits are acceptable during exploration.

### Recommended free alpha stack

| Layer | Recommendation | Reason |
|---|---|---|
| Frontend | Cloudflare Pages | Free static hosting, GitHub integration, monorepo root directory support |
| Backend | Render Free Web Service | GitHub integration, Go/Docker support, monorepo root directory support |
| Database | Supabase Free Postgres | Free managed PostgreSQL, enough for alpha |
| Auth | Supabase Auth | Free enough for alpha, supports email/password and OAuth |
| Object storage | Supabase Storage | Free enough for internal FullHD merch photos |
| Redis | Upstash Redis Free or Redis Cloud Free | Enough for low-volume rate limit/idempotency |
| CI/CD | GitHub Actions | Native to GitHub monorepo |

### Render caveats

Render Free Web Services can spin down after inactivity. For 3 alpha users, this is acceptable, but the first request after inactivity may be slow.

Do not store uploads or SQLite files on the Render service filesystem. Use managed database/object storage.

### Supabase caveats

Supabase Free limits are enough for alpha, but keep the backend as the business-rule authority. The frontend must not bypass the Go API to mutate business tables.

### Redis caveats

If a free Redis provider is unavailable or too limited, use PostgreSQL-backed idempotency and rate-limit tables for alpha, then move to Redis before broader release.

## 5. Local development architecture

Use Docker Compose for local development:

Services:

- Go API
- PostgreSQL
- Redis
- optional local object storage via MinIO
- optional local auth provider only if not using Supabase Auth

Recommended local commands:

```bash
pnpm install
pnpm --filter web dev
docker compose up -d postgres redis
cd apps/api && go test ./...
cd apps/api && go run ./cmd/api
```

PostgreSQL-backed integration tests require an explicitly running local database. Start and verify it before setting `DATABASE_URL`:

```bash
docker compose up -d postgres redis
docker compose exec postgres pg_isready -U band_manager -d band_manager
cd apps/api
DATABASE_URL='postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable' go test ./internal/infrastructure/postgres/... -v
```

Do not treat skipped Postgres integration tests as a failure when `DATABASE_URL` is unset.

Suggested root scripts:

```json
{
  "scripts": {
    "dev:web": "pnpm --filter web dev",
    "dev:api": "cd apps/api && go run ./cmd/api",
    "lint": "pnpm -r lint",
    "format": "prettier --check .",
    "format:write": "prettier --write .",
    "test": "pnpm -r test",
    "test:e2e": "pnpm --filter web cypress:run",
    "audit": "pnpm audit",
    "codegen": "pnpm --filter api-contract generate",
    "ci": "pnpm lint && pnpm format && pnpm test && pnpm audit"
  }
}
```

## 6. Backend architecture

Use a lightweight Clean Architecture-inspired structure.

```txt
apps/api/
  cmd/
    api/
      main.go
  internal/
    domain/
      merch/
      inventory/
      sales/
      payments/
      finance/
      auth/
    application/
      inventory/
      merchbooth/
      payments/
      auth/
    infrastructure/
      postgres/
      redis/
      mercadopago/
      supabase/
      storage/
    transport/
      http/
      middleware/
    platform/
      config/
      logger/
      clock/
  migrations/
  openapi/
  test/
```

Dependency direction:

```txt
transport/http -> application/usecase -> domain
infrastructure/postgres -> application interfaces
infrastructure/payment -> application interfaces
infrastructure/auth -> application interfaces
```

Rules:

- Domain must not import HTTP, SQL, Redis, MercadoPago, Supabase, or filesystem/object storage code.
- Keep interfaces small.
- Declare interfaces near the package that consumes them.
- Avoid over-abstracting simple code.

## 7. Backend stack

Use:

- Go
- chi
- PostgreSQL
- pgx
- sqlc
- goose
- OpenAPI
- oapi-codegen
- Redis-compatible store
- Docker

## 8. Frontend architecture

```txt
apps/web/src/
  app/
    router/
    providers/
    layouts/
  features/
    auth/
    inventory/
    merch-booth/
  shared/
    api/
    components/
    i18n/
    lib/
    ui/
  test/
```

Rules:

- Feature folders own their pages, hooks, schemas, tests, and components.
- Shared UI primitives go under `shared`.
- Domain-specific reusable logic goes under the relevant feature.
- Use TanStack Query for server state.
- Use React Hook Form + Zod for forms.
- Use React Testing Library with user-visible behavior.

## 9. Database design direction

Use PostgreSQL.

Core tables:

- bands
- users
- band_memberships
- merch_products
- merch_variants
- inventory_movements
- carts
- cart_items
- inventory_reservations
- sales
- sale_items
- payments
- payment_events
- transactions
- audit_logs
- idempotency_keys
- outbox_events

Product uniqueness:

```txt
band_id + category + normalized_name
```

Variant uniqueness:

```txt
product_id + size + colour
```

Use soft deletes:

- `deleted_at`
- `deleted_by`

Use timestamps:

- `created_at`
- `updated_at`

All write tables should have audit coverage.

## 10. Inventory requirements

### Functional requirements

The owner can:

- create inventory products/variants
- upload required merch photo
- edit inventory data
- delete inventory data
- view inventory list
- see sold-out items
- see total quantity per variant
- see price, cost, and expected profit

Each merch variant includes:

- UUID
- product name
- category
- size when applicable
- colour when applicable
- price amount
- cost amount
- currency
- photo
- quantity

### Business rules

- Price cannot be negative.
- Cost cannot be negative.
- Quantity cannot be negative.
- Photo is required.
- Cost is required.
- Product category is fixed enum in alpha.
- Size is fixed enum in alpha.
- Category + normalized name must be unique per band at product level.
- Size and colour identify product variants.
- Inventory changes create inventory movement records.
- Sold-out items remain visible.

### Acceptance criteria

- Owner can create a Shirt product with multiple sizes.
- Duplicate product name/category is rejected.
- Duplicate variant size/colour is rejected.
- Invalid negative values are rejected by frontend and backend.
- Sold-out item is visible in inventory.
- Audit log is written for create/update/delete.
- Unit tests cover pure domain rules where they add signal.
- Integration tests cover DB constraints.

## 11. Merch Booth requirements

### Functional requirements

The owner can:

- view all merch variants in a mobile-first grid
- see sold-out status
- add available items to cart
- remove items from cart
- checkout with cash
- checkout with Pix
- checkout with card if the MercadoPago spike confirms a viable Point/card-reader implementation path
- retry failed payment verification
- cancel failed/pending purchase

### Business rules

- Empty cart checkout is disabled.
- Sold-out items cannot be added.
- Checkout reserves inventory first.
- Sale finalizes only after valid payment confirmation, except online cash sales.
- Payment cancellation releases inventory.
- Failed payment prompts retry or cancel.
- Mixed full payment methods are allowed.
- Each sold item creates one transaction.
- Historical sale price is preserved.
- Payment events are stored separately and linked to transactions.

### Acceptance criteria

- Successful cash sale updates inventory and creates item-level transactions.
- Successful Pix sale updates inventory only after confirmation.
- Successful card sale updates inventory only after confirmation when Point/card support is confirmed viable for alpha.
- Failed Pix payment does not finalize sale.
- Failed card payment does not finalize sale when Point/card support is confirmed viable for alpha.
- Cancelled payment releases inventory.
- Empty cart cannot checkout.
- Sold-out Add to Cart button is disabled.
- Idempotent checkout retries do not duplicate sales.

## 12. MercadoPago spike

This is the first task.

Create `docs/spikes/mercadopago.md`.

The spike must answer:

- Pix charge creation flow.
- QR code generation response fields.
- Required stored payment identifiers.
- Webhook verification.
- Webhook retry behavior.
- Manual status polling.
- MercadoPago Point/card-reader alpha feasibility and implementation design.
- Local webhook development.
- Idempotency requirements.
- Payment state machine.
- Errors the frontend must handle.

Card support is alpha-critical if MercadoPago Point/card-reader integration is feasible. If the spike shows it is blocked for alpha, document the concrete blocker and escalation path before deferring card support or choosing an alternative flow.

Do not implement production payment flows until the spike is complete.

## 13. Deferred offline model

Offline support is not part of the alpha frontend or backend implementation. A future offline phase should use IndexedDB for:

- cached inventory
- cached booth items
- local cart state
- offline cash sales
- outbox queue
- sync status

Future offline constraints:

- Offline checkout supports cash only.
- Offline Pix is disabled.
- Offline card is disabled.
- Offline sale is marked pending sync.
- User sees a visible pending sync banner.
- Multi-device conflict resolution.
- Manual conflict review.
- Fan-facing offline limitations.

## 14. Security plan

### Request security

- Validate all input before logic/database/API calls.
- Use Zod on frontend.
- Use backend validation for every request.
- Never rely on frontend validation alone.

### Rate limiting

- Login: 5 attempts/minute per IP.
- Unauthenticated routes: IP-based.
- Authenticated routes: IP + userId.
- General internal routes: start at 500 requests/minute.

### Idempotency

Every mutating route requires idempotency.

Retention:

| Operation | Retention |
|---|---:|
| Normal create/update/delete | 5–15 minutes |
| Checkout | 30–60 minutes |
| Payment creation | 24 hours |
| Sale finalization | 24 hours |
| Webhook processing | 7–30 days |

### CORS

- Allowlist only.
- Localhost only in development.
- No wildcard origins.

### Secrets

- No secrets in code.
- Use `.env.example`.
- Use provider environment variables for deployed services.

### Audit logs

Audit all writes.

Fields:

- userId
- bandId
- action
- entity type
- entity ID
- createdOn
- updatedOn
- request ID
- idempotency key

Audit logs are immutable and backend-only.

## 15. Testing plan

### Backend

Use Go unit tests when they add signal for pure domain and application rules, including money validation, inventory reservation decisions, idempotency decisions, payment state transitions, and permission rules.

Use integration tests for:

- PostgreSQL constraints
- API behavior
- inventory reservation
- sale finalization
- idempotency
- webhook handling
- permissions
- Supabase and MercadoPago adapters when real calls or provider sandboxes are practical

### Frontend

Use:

- Vitest
- React Testing Library
- MSW
- Cypress

Cypress critical flows:

- Add inventory item.
- Update inventory item.
- Remove inventory item.
- Successful sale.
- Failed sale.
- Successful card sale when the MercadoPago spike confirms a viable alpha card path.
- Failed card sale when the MercadoPago spike confirms a viable alpha card path.
- Sale with empty cart disabled.
- Remove all items from cart.
- Sold-out item cannot be added.
- Inventory reflects sale.
- Transaction created per sold item.

### TDD

New business behavior must follow:

1. RED: failing test.
2. GREEN: minimal implementation.
3. REFACTOR: improve while tests stay green.

Coverage target:

- 80% minimum.
- Meaningful behavior coverage is more important than raw percentage.
- Avoid unit tests that exist only to inflate coverage or mock implementation details.
- Prefer integration, E2E, smoke, and real-behavior tests for database constraints, API behavior, Supabase/MercadoPago adapters, checkout flows, and UI behavior.

## 16. CI/CD plan

Use GitHub Actions.

Required checks:

- pnpm install with lockfile
- frontend lint
- frontend format check
- frontend unit tests
- frontend build
- frontend package audit
- backend gofmt/go vet
- backend tests
- backend build
- OpenAPI/codegen drift check
- Cypress critical tests when feasible

Path filters:

- Frontend workflow triggered by `apps/web/**`, `packages/**`, `pnpm-lock.yaml`.
- Backend workflow triggered by `apps/api/**`, `packages/api-contract/**`.
- Full workflow triggered by PR to main.

## 17. Implementation sequence

### Step 1 — Repository skeleton

- Create monorepo.
- Add pnpm workspace.
- Add frontend app.
- Add backend app.
- Add shared packages.
- Add GitHub Actions skeleton.
- Add Docker Compose.
- Add README.

### Step 2 — MercadoPago spike

- Explore Pix.
- Explore webhook verification.
- Explore status polling.
- Explore Point/card reader integration feasibility and implementation design.
- Write spike document.
- Confirm the Pix/Card alpha implementation path, or document the card blocker and escalation path before changing alpha scope.

### Step 3 — Backend foundation

- Config.
- Logger.
- Healthcheck.
- CORS.
- Security headers.
- Auth middleware placeholder.
- PostgreSQL connection.
- Redis/idempotency abstraction.
- OpenAPI skeleton.
- Migration setup.

### Step 4 — Auth integration

- Supabase Auth adapter.
- Protected routes.
- Owner signup model.
- Viewer invite model.
- Role enforcement.

### Step 5 — Inventory backend

- Migrations.
- Domain rules.
- Use cases.
- HTTP routes.
- Tests.

### Step 6 — Merch Booth backend

- Cart.
- Reservations.
- Cash checkout.
- Pix checkout.
- Card checkout when the MercadoPago spike confirms a viable alpha card path.
- Webhooks.
- Transactions.
- Tests.

### Step 7 — Remaining backend API surface

- Financial reports API.
- Calendar API.
- Account-management API contracts needed for alpha operations.
- OpenAPI paths and schemas for every backend route.
- Backend tests.

### Step 8 — Frontend foundation

- Router.
- Auth pages.
- Protected layouts.
- i18n.
- ShadCN.
- Bottom tab navigation.
- Header.
- MSW.

### Step 9 — Inventory frontend

- List.
- Create.
- Edit.
- Delete.
- Upload.
- Tests.

### Step 10 — Merch Booth frontend

- Grid.
- Cart.
- Cash checkout.
- Pix checkout.
- Card checkout when the MercadoPago spike confirms a viable alpha card path.
- Tests.

### Step 11 — Alpha hardening

- Security review.
- Responsive review.
- Accessibility review.
- Deployment.
- Manual test script.
- Backup/export instructions.
