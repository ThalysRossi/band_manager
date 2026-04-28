# Initial Codex Prompt — Band Manager Alpha

Act as an experienced senior software engineer working with a product owner. You are developing a greenfield monorepo for a web app that helps underground bands in Brazil self-manage merch inventory and merch booth sales.

The user is the product owner. Do not assume missing product behavior. Before implementing each major feature, ask focused clarification questions if the requirement is ambiguous. If the requirement is clear, proceed feature-by-feature using TDD.

## Product summary

Build a responsive, mobile-first web app for a single band/account. The alpha must prioritize Inventory and Merch Booth because they contain the most critical business rules.

The app must support Portuguese and English from alpha. Every user-facing string must go through i18n.

The app must support offline use. For alpha, offline checkout is allowed only for cash payments. Pix and card payments require online backend/payment-provider access.

The app is initially single-band/self-hosted, but the domain model must not block future SaaS/multi-band expansion.

## Alpha scope

In scope for alpha:

- Auth/sign-up flow.
- One owner account created during sign-up.
- Invited users can log in, but alpha users other than owner are viewers only.
- Inventory management.
- Merch Booth point-of-sale assistant.
- Cash checkout.
- Pix checkout through MercadoPago.
- Card checkout through MercadoPago Point/card reader if the MercadoPago spike confirms a viable alpha implementation path.
- MercadoPago payment confirmation through webhook and status polling.
- Backend-first implementation.
- PostgreSQL-backed persistence.
- Object storage for images.
- Offline support using IndexedDB and an outbox/sync model.
- Minimal generated financial transactions from merch sales.
- Responsive/mobile-first UI.
- CI/CD, tests, linting, formatting, audit checks.

Deferred to roadmap:

- Meta login.
- Fan-facing merch booth.
- Barcode/QR scanning.
- Discounts.
- Receipts.
- Refunds/cancellations beyond inventory reset on cancelled checkout.
- Configurable categories/sizes/colours.
- Google Calendar integration.
- Rich financial reports.
- Multiple concurrent merch booth operators.
- Full visual identity/design phase.
- Additional payment providers.

## Selected stack

Monorepo:

- pnpm workspace.
- Separate frontend and backend apps.
- Shared OpenAPI contract and generated client.
- GitHub Actions for CI.

Frontend:

- React.
- TypeScript.
- Vite.
- TanStack Router.
- TanStack Query.
- ShadCN UI.
- React Hook Form.
- Zod.
- IndexedDB.
- i18n.
- ESLint.
- Prettier.
- Husky.
- Vitest.
- React Testing Library.
- Cypress.
- MSW.

Backend:

- Go.
- chi router.
- PostgreSQL.
- pgx.
- sqlc.
- goose migrations.
- Redis-compatible store for rate limiting/idempotency where available.
- OpenAPI.
- oapi-codegen.
- Docker.
- Clean Architecture-inspired layering without excessive OOP-style ceremony.

Auth/storage/database for free alpha deployment:

- Prefer Supabase Free for managed PostgreSQL, Auth, and Storage during alpha.
- Keep auth and storage behind application adapters so the app can later move to Keycloak and S3-compatible storage without rewriting domain code.
- The Go backend remains the business-rule authority.
- Do not let the frontend write directly to business tables.

Payment provider:

- MercadoPago is the only provider for alpha.
- Implement behind a `PaymentProvider` interface so future providers can be added.
- First development task must be a MercadoPago backend spike before full backend implementation.

Deployment target for alpha:

- Frontend: Cloudflare Pages or GitHub Pages.
- Backend: Render Free Web Service, Railway trial/free-credit service, or equivalent GitHub-connected service.
- Database/auth/storage: Supabase Free.
- Redis: Upstash Redis Free or Redis Cloud Free if Redis is required externally.
- Docker Compose must support local development.
- Production deployment must be compatible with a monorepo by setting per-service root directories or build commands.

## Repository structure target

Use this as the starting structure unless there is a strong reason to change it:

```txt
band-manager/
  apps/
    web/
      src/
        app/
        features/
          auth/
          inventory/
          merch-booth/
          financial-reports/
          calendar/
        shared/
        test/
      public/
      package.json
      vite.config.ts
    api/
      cmd/
        api/
      internal/
        domain/
        application/
        infrastructure/
        transport/
        platform/
      migrations/
      openapi/
      test/
      Dockerfile
      go.mod
  packages/
    api-contract/
      openapi.yaml
      generated/
    config/
      eslint/
      prettier/
      tsconfig/
    i18n/
  docs/
    decisions/
    spikes/
    product/
  .github/
    workflows/
  docker-compose.yml
  pnpm-workspace.yaml
  README.md
```

## Architecture rules

Use a dependency direction similar to:

```txt
transport/http -> application/usecase -> domain
infrastructure/postgres -> application/repository interfaces
infrastructure/payment -> application/payment interfaces
infrastructure/auth -> application/auth interfaces
```

Keep domain rules independent from HTTP, PostgreSQL, Redis, Supabase, MercadoPago, and filesystem/object storage.

Avoid unnecessary abstraction. Use small interfaces declared close to consumers.

## Security requirements

- Every mutating request must support idempotency.
- Idempotency keys are scoped by `band_id + operation + idempotency_key`.
- Normal mutation idempotency retention starts at 5–15 minutes.
- Checkout and payment idempotency retention must be longer: 30–60 minutes for checkout, 24 hours for payment creation/finalization, and 7–30 days for webhooks.
- Rate limiting:
  - Unauthenticated routes: IP-based.
  - Authenticated routes: IP + userId.
  - Login: 5 attempts/minute per IP.
  - General internal routes: start with 500 requests/minute.
- CORS must use allowlists only.
- Localhost is allowed only in development.
- Use secure headers middleware.
- Validate all input before business logic, external API calls, or database writes.
- Secrets must never be committed.
- Use `.env.example`, runtime environment variables, and deployment provider secrets.
- Audit every write for debugging:
  - userId
  - bandId
  - action
  - entity type
  - entity ID
  - createdOn
  - updatedOn
  - request ID
  - idempotency key when present
- Audit logs are backend-only and immutable.

## Roles and permissions

Roles:

- owner
- admin
- member
- viewer

Alpha behavior:

- Sign-up creates one owner user.
- Sign-up instructions must ask the user to use a band-related email, for example `really_awesome_band@email.com`.
- Invited users are created as viewers for the alpha.
- Only the owner can create, update, and delete data in alpha.
- Viewers can read only.
- The model must support future owner/admin/member write permissions.

Future intended behavior:

- owner/admin/member can create transactions, edit inventory, and operate merch booth.
- owner/admin can delete.
- viewer can read only.

## Core domain model

Use UUIDs for IDs.

Money:

- Store as integer minor units.
- Include currency code.
- Alpha currency is BRL.
- Do not use floats for money.

Band:

- id
- name
- email
- members
- photo
- defaultCurrency
- timezone

User:

- id
- name
- email
- instrument
- preferences
- photo

Instrument enum:

- Bass
- Guitar
- Drums
- Vocals
- Keys

ProductCategory enum:

- Shirt
- Hat
- Patch
- Button
- Sticker
- Hoodie
- Shorts
- Poster
- Record
- CD
- Tape

ProductSize enum:

- XS
- S
- M
- L
- XL
- XXL
- XXXL

Inventory model:

Use product + variant modeling.

Product-level uniqueness:

- `band_id + category + normalized_name`

Variant-level uniqueness:

- `product_id + size + colour`

A product is the concept shown as `${ProductCategory} ${name}`. A variant is a concrete sellable item with size/colour/price/cost/photo/quantity.

Example variants:

```json
[
  {
    "id": "uuid",
    "name": "Skinless Dude",
    "category": "Shirt",
    "size": "M",
    "colour": "White",
    "priceAmount": 4200,
    "costAmount": 2000,
    "currency": "BRL",
    "photo": "object-storage-key",
    "quantity": 12
  },
  {
    "id": "uuid",
    "name": "Skinless Dude",
    "category": "Shirt",
    "size": "L",
    "colour": "White",
    "priceAmount": 4200,
    "costAmount": 2000,
    "currency": "BRL",
    "photo": "object-storage-key",
    "quantity": 3
  }
]
```

Inventory rules:

- Photo is required.
- Cost is required.
- Price is required.
- Quantity is required.
- Price, cost, quantity, totalIncome, and totalExpense must never be negative in backend storage.
- Size is applicable to Shirt, Hoodie, Shorts, and future size-applicable categories.
- Colour is optional for all items, except items with sizes.
- Sold-out items remain visible in Inventory and Merch Booth.
- Sold-out Merch Booth cards must show SOLD OUT, greyed photo, and disabled Add to Cart.
- Adding a new inventory variant makes it available in Merch Booth.
- A sale updates inventory.
- Track both current quantity and stock movements.
- Inventory movement reasons include purchase, sale, correction, loss, gift, and damaged.

Merch Booth rules:

- Internal band use only in alpha.
- Mobile-first POS flow.
- Multiple items can be added to cart.
- Checkout is disabled for empty carts.
- Sold-out items cannot be added.
- Payments are made in full.
- Mixed payment methods are allowed.
- Cash:
  - User inputs amount given by customer.
  - UI shows change due.
  - Offline cash sales are allowed.
- Pix:
  - Generate MercadoPago Pix QR code.
  - This Pix QR Code should contain the amount due.
  - Confirm payment through webhook/status verification.
- Card:
  - Treat MercadoPago Point/card-reader support as alpha-critical if feasible.
  - The MercadoPago spike must determine the implementation path before production card payment code starts.
  - If MercadoPago Point/card-reader support is blocked for alpha, stop and document the blocker and escalation path before deferring or selecting an alternative.
- A checkout first reserves inventory.
- A sale is finalized only after payment confirmation.
- If payment fails, prompt user to retry payment or cancel purchase.
- Cancelling releases inventory reservation.
- Do not finalize sale for unconfirmed Pix/Card payments.
- For cash sales, local offline completion is allowed and later synced.
- A sale creates one transaction per sold item.
- Historical sale price must be preserved.
- Payment events must be stored separately and linked to transaction IDs.

Financial rules:

- Financial reports are dynamic, not stored snapshots.
- Default report shows the last 3 months.
- User may request reports by selected date range.
- TotalIncome is the sum of income transactions in the period.
- TotalExpense is the sum of expense transactions in the period.
- Balance is income minus expense.
- Store positive amounts with transaction type; render expense as negative formatting at UI level if needed.
- Backend must reject negative amounts.
- Pending/unverified payments are excluded from reports.

Calendar rules for future/minimal later alpha:

- Local calendar initially.
- Google Calendar integration deferred.
- Events support recurrence.
- Events have location/venue.
- Event completion only marks as completed initially.

## TDD and quality rules

All new business behavior must be developed with TDD.

Workflow:

1. Write failing test.
2. Run test and confirm RED.
3. Implement minimal code.
4. Run test and confirm GREEN.
5. Refactor if needed while keeping tests green.

Use unit tests when they add signal for pure domain/application rules, including money validation, inventory reservation decisions, idempotency decisions, payment state transitions, and permission rules.

Do not add unit tests only to inflate coverage or mock implementation details.

Use integration tests for:

- PostgreSQL constraints.
- Inventory reservation.
- Idempotency.
- Payment webhook handling.
- Sale finalization.
- Permission enforcement.
- Supabase and MercadoPago adapters when real calls or provider sandboxes are practical.
- API behavior that crosses transport, validation, persistence, and use cases.

Frontend tests:

- React Testing Library using user-visible behavior.
- MSW for API mocking.
- Cypress for critical flows.
- Prefer integration, E2E, smoke, and real-behavior tests for UI flows, external services, offline sync, and cross-system behavior.

Minimum coverage target is 80%, but meaningful behavior coverage is more important than raw percentage.

Cypress critical flows:

- Add inventory item.
- Update inventory item.
- Remove inventory item.
- Successful sale.
- Failed sale.
- Successful card sale when the MercadoPago spike confirms a viable alpha card path.
- Failed card sale when the MercadoPago spike confirms a viable alpha card path.
- Empty cart has disabled checkout button.
- Remove all items from cart.
- Sold-out items cannot be added to cart.
- Inventory reflects sale.
- Transaction created per item.

CI must block merges on:

- lint errors
- format errors
- type errors
- failing tests
- package audit failures
- backend test failures
- OpenAPI generation drift

## First task

Do not start frontend implementation first.

First task: create a backend spike for MercadoPago integration.

The spike must answer:

1. Can the backend generate a Pix charge and QR code for this app?
2. What fields must be stored to reconcile the payment?
3. How are MercadoPago webhook events verified?
4. How does the backend safely poll payment status if webhook delivery fails?
5. What is the payment status lifecycle?
6. Can MercadoPago Point Pro 2/card reader be integrated directly enough for alpha?
7. What are the local development constraints for webhook testing?
8. Which operations require idempotency?
9. What error cases must the app expose to the frontend?
10. Is MercadoPago Point/card-reader support feasible for alpha, and if not, what blocker and escalation path should the product owner decide on?

Create `docs/spikes/mercadopago.md` with findings and implementation recommendations.

Do not implement production payment code until the spike is complete.
