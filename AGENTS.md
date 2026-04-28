# agents.md — Codex Agent Instructions

This project uses three practical agents to avoid unnecessary token burn:

1. Engineer Agent
2. QA Agent
3. Security Agent

The agents are roles/perspectives. They do not need to run as separate processes unless the tooling supports it cleanly.

## Global rules for all agents

- Do not assume missing product behavior.
- Ask focused questions before implementing ambiguous major behavior.
- Follow the product decisions in `plans.md`.
- Follow TDD for new business behavior.
- Keep changes small and reviewable.
- Prefer explicit, boring, maintainable code.
- Do not introduce new frameworks or external services without explaining why.
- Do not commit any code.
- Do not bypass the Go backend for business mutations.
- Preserve i18n for all user-facing strings.
- Keep alpha constraints distinct from roadmap items.

## Engineer Agent

### Mission

Implement the application feature-by-feature, separating backend and frontend concerns while maintaining a working monorepo.

### Responsibilities

- Create and maintain project structure.
- Implement backend use cases.
- Implement frontend features.
- Maintain OpenAPI contracts.
- Generate clients/types.
- Write tests before implementation for new business behavior.
- Keep Docker/local development working.
- Update docs when architecture or behavior changes.

### Working style

For each feature:

1. Restate the target behavior briefly.
2. Identify backend contracts first.
3. Write failing tests.
4. Implement minimal code.
5. Run relevant tests.
6. Refactor.
7. Update docs and OpenAPI if needed.

### Backend implementation rules

Use:

- Go
- chi
- PostgreSQL
- pgx
- sqlc
- goose
- OpenAPI
- oapi-codegen

Follow this dependency direction:

```txt
transport/http -> application/usecase -> domain
infrastructure -> application interfaces
```

Avoid:

- large global interfaces
- framework-heavy abstractions
- business logic in handlers
- SQL logic leaking into frontend assumptions
- floats for money
- stringly typed status handling where enums are appropriate

### Frontend implementation rules

Use:

- React
- TypeScript
- Vite
- TanStack Router
- TanStack Query
- ShadCN UI
- React Hook Form
- Zod
- IndexedDB
- i18n
- MSW
- Vitest
- React Testing Library
- Cypress

Rules:

- User-facing strings go through i18n.
- Forms validate with Zod.
- Backend validation still required.
- Server state goes through TanStack Query.
- Offline state goes through IndexedDB/outbox.
- Tests should assert user-visible behavior, not implementation details.

### First assigned task

The first Engineer Agent task is the MercadoPago backend spike.

Output:

- `docs/spikes/mercadopago.md`
- optional proof-of-concept code under `apps/api/internal/infrastructure/mercadopago/spike/`
- recommendation on Pix, webhook verification, polling, and Point/card reader feasibility

Do not implement production payment code before the spike is done.

## QA Agent

### Mission

Ensure implemented behavior matches product rules and edge cases.

### Responsibilities

- Translate requirements into test scenarios.
- Review test coverage quality.
- Identify missing edge cases.
- Design Cypress flows.
- Verify offline behavior.
- Verify i18n coverage.
- Verify accessibility and responsive behavior enough for alpha.

### QA focus areas

Inventory:

- add item
- edit item
- delete item
- duplicate product rejection
- duplicate variant rejection
- negative price/cost/quantity rejection
- sold-out visibility
- required photo
- required cost
- audit log write behavior

Merch Booth:

- empty cart checkout disabled
- sold-out add-to-cart disabled
- cart add/remove
- successful cash sale
- successful Pix
- successful card sale, conditional on the MercadoPago spike confirming a viable alpha card path
- failed Pix sale
- failed card sale, conditional on the MercadoPago spike confirming a viable alpha card path
- retry payment verification
- cancel payment and release inventory
- offline cash sale queued
- pending sync banner
- inventory decremented after finalized sale
- transaction created per sold item

Permissions:

- owner can write
- viewer cannot write
- unauthenticated users see login only
- protected route redirect works

Internationalization:

- Portuguese strings exist
- English strings exist
- enum labels are translated
- validation messages are translated

### QA output format

For each feature, produce:

```md
## QA Checklist: <Feature>

### Happy paths
- [ ] ...

### Error paths
- [ ] ...

### Edge cases
- [ ] ...

### Regression risks
- [ ] ...

### Required automated tests
- [ ] Unit:
- [ ] Integration:
- [ ] E2E:
```

## Security Agent

### Mission

Ensure the alpha does not normalize unsafe habits even though it is low-traffic.

### Responsibilities

- Review auth flow.
- Review authorization checks.
- Review CORS config.
- Review rate limiting.
- Review idempotency.
- Review validation.
- Review secret handling.
- Review audit logging.
- Review payment webhook handling.
- Review object storage access.

### Required checks

Auth:

- Email verification enabled.
- Password reset enabled.
- Google login configured.
- Meta login deferred.
- Viewer users cannot mutate data in alpha.
- Backend enforces roles, not frontend only.

Authorization:

- Every protected API route resolves user and band context.
- Every write checks role.
- Alpha write operations require owner.
- Future roles are modeled but not enabled accidentally.

Validation:

- All request bodies validated before use case logic.
- Backend rejects negative money/quantity.
- Backend rejects invalid enums.
- Backend rejects duplicate product/variant.

Idempotency:

- Every mutating route requires idempotency key.
- Idempotency key is scoped by band and operation.
- Duplicate checkout does not double-sell.
- Duplicate webhook does not double-finalize.

Rate limiting:

- Login: 5 attempts/min/IP.
- Internal routes: IP + userId.
- Global baseline: 500 req/min.
- Redis or fallback persistence is documented.

CORS:

- Allowlist only.
- No wildcard production origins.
- Localhost only in development.

Secrets:

- No secrets committed.
- `.env.example` only contains placeholders.
- Deployment provider variables documented.
- MercadoPago credentials are not logged.

Payments:

- Webhook payload verification documented.
- Payment status re-checks call provider before finalization.
- Manual confirmation, if implemented, is owner-only and audited.
- Failed/cancelled payments release reservations.
- Pending payments do not create final transactions.

Object storage:

- Uploads are restricted by auth.
- File size/type constraints exist.
- Stored object keys are not predictable if public access is unsafe.
- FullHD internal images are enough.

### Security output format

```md
## Security Review: <Feature>

### Findings
- ...

### Required fixes
- ...

### Acceptable alpha risks
- ...

### Do not ship if
- ...
```

## Escalation rules

Stop and ask the product owner if:

- a requirement conflicts with another requirement
- MercadoPago API does not support a required flow
- a free-tier deployment limit blocks required alpha behavior
- offline behavior can cause irreversible data corruption
- a security requirement cannot be implemented within the selected stack
- implementing a roadmap item would delay alpha scope

Do not stop for minor implementation details that can be solved with standard engineering judgment.
