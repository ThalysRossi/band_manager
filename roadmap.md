# roadmap.md — Band Manager Product Roadmap

## Roadmap principle

The alpha must prove the core operational loop:

```txt
Inventory item exists -> item appears in Merch Booth -> item is sold -> inventory changes -> transaction is created
```

Everything else should support or protect this loop.

## Phase 0 — Product and architecture lock

### Goals

- Freeze alpha scope.
- Confirm stack.
- Confirm deployment path.
- Confirm domain model.
- Confirm payment spike plan.

### Decisions

- Single-band account for alpha.
- Future-compatible multi-band membership model.
- Inventory and Merch Booth are alpha-critical.
- PostgreSQL over MongoDB.
- Go backend.
- React frontend.
- Monorepo.
- i18n from alpha.
- Offline support is deferred until after alpha.
- BRL only in alpha, currency abstraction from the start.
- Supabase Free acceptable for alpha database/auth/storage.
- Go backend remains business-rule authority.
- MercadoPago only payment provider in alpha.

### Deliverables

- `plans.md`
- `agents.md`
- `roadmap.md`
- initial Codex prompt
- initial architecture decision records

## Phase 1 — MercadoPago spike

### Why this comes first

Payment feasibility affects backend design, checkout state machine, inventory reservation behavior, and frontend checkout UX.

### Scope

- Pix QR code generation.
- Pix payment confirmation.
- Webhook verification.
- Status polling fallback.
- MercadoPago Point/card reader feasibility.
- Local webhook testing strategy.
- Payment state machine.
- Idempotency strategy.
- Alpha card-payment implementation path if Point/card reader integration is feasible.

### Deliverables

- `docs/spikes/mercadopago.md`
- proof-of-concept backend code if useful
- recommendation on Pix, webhook verification, polling, and Point/card reader feasibility

### Exit criteria

- Pix flow is understood.
- Webhook verification approach is documented.
- Polling fallback is documented.
- Point/card reader support is confirmed feasible for alpha, or the blocker and escalation path are documented for product-owner decision before deferral or alternative selection.
- Required persisted payment fields are known.

## Phase 2 — Monorepo and local infrastructure

### Scope

- pnpm workspace.
- React/Vite app.
- Go API app.
- Docker Compose.
- PostgreSQL.
- Redis or Redis-compatible fallback.
- OpenAPI package.
- Shared config packages.
- GitHub Actions baseline.

### Deliverables

- Repository skeleton.
- Working local frontend.
- Working local backend healthcheck.
- Working local database.
- CI skeleton.

### Exit criteria

- `pnpm install` works.
- Frontend dev server starts.
- Backend starts.
- Backend healthcheck passes.
- Tests run locally.
- Docker Compose starts dependencies.

## Phase 3 — Free alpha deployment path

### Goal

Deploy from one monorepo to separate services.

### Recommended alpha deployment

| Layer | Service |
|---|---|
| Frontend | Cloudflare Pages |
| Backend | Render Free Web Service |
| Database | Supabase Free Postgres |
| Auth | Supabase Auth |
| Storage | Supabase Storage |
| Redis | Upstash Redis Free or Redis Cloud Free |
| CI/CD | GitHub Actions |

### Monorepo deployment model

- Frontend service watches/builds `apps/web`.
- Backend service watches/builds `apps/api`.
- Shared OpenAPI/codegen is validated in GitHub Actions.
- Backend Docker build may use repository root as context when needed.
- Frontend receives backend URL through environment variable.
- Backend receives frontend allowed origin through environment variable.

### Environment targets

- `local`
- `staging`
- `production`

For alpha, staging and production may use the same provider family but different env vars.

### Exit criteria

- Frontend deploys from GitHub.
- Backend deploys from GitHub.
- Backend can receive MercadoPago webhooks.
- Frontend can call backend.
- CORS allowlist works.
- Secrets are stored outside code.

## Phase 4 — Auth and band account

### Scope

- Email/password login.
- Email verification.
- Password reset.
- Google login.
- Band sign-up.
- Band-related email instruction.
- Owner role creation.
- Invite users as viewers.
- Protected routes.

### Deliverables

- Auth pages.
- Auth backend middleware.
- Role enforcement.
- Invite flow skeleton.
- Viewer read-only behavior.

### Exit criteria

- Owner can sign up.
- Owner can log in.
- Owner can invite viewer.
- Viewer can log in.
- Viewer cannot write.
- Unauthenticated user only sees login route.

## Phase 5 — Inventory backend

### Scope

- Products.
- Variants.
- Inventory movements.
- Photo metadata.
- Product uniqueness.
- Variant uniqueness.
- Soft deletes.
- Audit logs.
- Tests.

### Deliverables

- Database migrations.
- Inventory domain rules.
- Inventory use cases.
- Inventory HTTP routes.
- OpenAPI paths.
- Backend tests.

### Exit criteria

- Owner can create product/variant.
- Owner can update product/variant.
- Owner can delete product/variant.
- Viewer cannot mutate.
- Duplicate product is rejected.
- Duplicate variant is rejected.
- Negative values are rejected.
- Audit logs are created.

## Phase 6 — Merch Booth backend

### Scope

- Booth catalog query.
- Cart.
- Cart items.
- Inventory reservation.
- Checkout.
- Cash payment.
- Pix payment.
- Card payment when the MercadoPago spike confirms a viable Point/card reader path.
- Webhooks.
- Status polling.
- Sale finalization.
- Item-level transactions.

### Deliverables

- Cart API.
- Checkout API.
- Payment API.
- Webhook endpoint.
- Transaction generation.
- Integration tests.

### Exit criteria

- Empty cart cannot checkout.
- Sold-out items cannot be added.
- Checkout reserves inventory.
- Cash sale finalizes.
- Pix sale finalizes after confirmation.
- Card sale finalizes after confirmation when the MercadoPago spike confirms a viable alpha card path.
- Failed Pix does not finalize.
- Failed card payment does not finalize when the MercadoPago spike confirms a viable alpha card path.
- Cancelled payment releases reservation.
- Duplicate checkout does not double-sell.
- Sale creates one transaction per item.

## Phase 7 — Remaining backend API surface

### Scope

- Financial reports API.
- Calendar API.
- Account-management API contracts needed for alpha operations.
- OpenAPI coverage for every backend route.
- Backend tests for new endpoint behavior.

### Deliverables

- Report API.
- Calendar API.
- Account-management routes where needed.
- OpenAPI updates.
- Backend tests.

### Exit criteria

- Backend exposes the complete alpha API surface.
- Report endpoints derive data from finalized transactions.
- Calendar endpoints persist local events and recurrence.
- Protected backend routes enforce owner/viewer alpha permissions.
- No report, calendar, or account-management UI is required for alpha.

## Phase 8 — Frontend foundation

### Scope

- React app shell.
- TanStack Router.
- TanStack Query.
- ShadCN.
- i18n.
- Auth pages.
- Protected route redirect.
- Header.
- Bottom tab navigation.
- Dark mode default.
- Band logo/photo in header.

### Deliverables

- Frontend shell.
- Login/sign-up UI.
- Navigation shell.
- Translation files.
- MSW setup.
- Base tests.

### Exit criteria

- Portuguese and English work.
- Protected route redirects work.
- Header highlights current page.
- Bottom mobile navigation works.
- Frontend can call backend in staging.

## Phase 9 — Inventory frontend

### Scope

- Inventory list.
- Create item.
- Edit item.
- Delete item.
- Photo upload.
- Product/variant grouping.
- Sold-out visibility.
- Price/cost/profit display.
- Validation.

### Deliverables

- Inventory pages.
- Forms.
- Tests.
- Cypress flows.

### Exit criteria

- Owner can add inventory item.
- Owner can update inventory item.
- Owner can remove inventory item.
- Viewer sees read-only inventory.
- Sold-out item remains visible.
- Invalid values are blocked.
- Required fields are enforced.

## Phase 10 — Merch Booth frontend

### Scope

- Mobile-first product grid.
- Sold-out visual state.
- Cart drawer/page.
- Cash payment.
- Pix QR flow.
- Card payment flow when the MercadoPago spike confirms a viable Point/card reader path.
- Payment retry/cancel prompt.

### Deliverables

- Merch Booth UI.
- Checkout UI.
- Tests.
- Cypress flows.

### Exit criteria

- User can complete cash sale.
- User can complete Pix sale after confirmation.
- User can complete card sale after confirmation when the MercadoPago spike confirms a viable alpha card path.
- User can handle failed Pix.
- User can handle failed card payment when the MercadoPago spike confirms a viable alpha card path.
- User can cancel and release inventory.
- User can remove all items from cart.
- Sold-out Add to Cart is disabled.

## Phase 11 — Alpha release hardening

### Scope

- Security review.
- Accessibility pass.
- Responsive QA.
- Backup/export instructions.
- Error monitoring.
- Deployment guide.
- Manual QA script.
- Known limitations document.

### Deliverables

- Alpha release candidate.
- Deployment runbook.
- Rollback notes.
- Manual testing checklist.
- Known limitations.

### Exit criteria

- Critical Cypress flows pass.
- Backend integration tests pass when local or CI Postgres is available.
- Security review has no blocking issues.
- Owner can use login, inventory, and merch booth end-to-end.

## Future roadmap

### Offline/PWA

- Installable PWA.
- Service worker strategy.
- Static asset caching.
- API data caching policy.
- Offline banner.
- Offline cash mode.
- Pending sync banner.
- IndexedDB sync hardening.

### Reports UI

- Report page.

### Calendar UI

- Calendar page.

### Payments

- Additional payment providers.
- Payment provider settings page.
- Band-specific payment credentials UI.
- Manual payment confirmation controls.
- Refunds.
- Cancellations.
- Receipts.
- WhatsApp/email receipt delivery.

### Merch Booth

- Fan-facing booth.
- Barcode/QR scanning.
- Discounts.
- Better multi-device support.
- Manual conflict resolution.
- Multiple operators at shows.

### Inventory

- Low-stock warnings.
- Configurable categories.
- Configurable sizes.
- Configurable colours.
- Bulk import/export.
- Purchase planning.

### Finance

- CSV export.
- Expense receipt attachments.
- Configurable transaction categories.
- Event-based financial reports.
- Gig settlement flow.
- Profit/loss by merch item.
- Profit/loss by event.

### Calendar

- Google Calendar integration.
- Email notifications.
- RSVP.
- Event reminders.
- Gig-specific fields:
  - fee
  - expected costs
  - settlement status
  - venue contact

### Product/account

- Multiple active bands per user.
- Full SaaS multi-tenant support.
- Owner/admin/member write permissions.
- Band settings page.
- Re-authentication for sensitive operations.
- Subscription/billing if SaaS direction becomes relevant.

### UX/design

- Underground/punk visual identity phase.
- Custom theme.
- Dashboard actionable widgets.
- Improved onboarding.
- Better analytics widgets.

### Platform

- Full self-hosted mode with Keycloak.
- S3-compatible storage abstraction.
- Paid production deployment path.
- Backup automation.
- Observability.
- Structured logs.
- Error tracking.
