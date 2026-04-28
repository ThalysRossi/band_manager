# MercadoPago backend spike

Status: completed from documentation research on 2026-04-27.

## Recommendation

Use Mercado Pago's Orders API as the backend integration boundary for both Pix and Point/card payments.

- Pix is feasible for alpha.
- Webhook verification is feasible and must be implemented before sale finalization.
- Polling is feasible as a backend status re-check, not as the primary confirmation mechanism.
- Mercado Pago Point/card support is feasible for alpha if the band has a supported Point terminal, can configure it in PDV mode, and can use a production Mercado Pago seller account for real card payments.
- Do not move card support to roadmap yet. Escalate only if the product owner cannot provide a supported terminal/account, cannot enable PDV mode, or Mercado Pago blocks the account/device from integrated Point processing.

No proof-of-concept code was added because the repository does not yet have the API skeleton and real provider calls require Mercado Pago credentials.

## Sources

- Pix via Orders API: https://www.mercadopago.com.br/developers/en/docs/checkout-api-orders/payment-integration/pix
- Checkout Transparente notifications: https://www.mercadopago.com.br/developers/en/docs/checkout-api-orders/notifications
- Pix integration test: https://www.mercadopago.com.br/developers/en/docs/checkout-api-orders/integration-test/pix
- Orders transaction statuses: https://www.mercadopago.com.br/developers/en/docs/checkout-api-orders/payment-management/status/transaction-status
- Orders order statuses: https://www.mercadopago.com.br/developers/en/docs/checkout-api-orders/payment-management/status/order-status
- Orders errors: https://www.mercadopago.com.br/developers/en/docs/checkout-api-orders/payment-management/integration-errors
- Point overview: https://www.mercadopago.com.br/developers/en/docs/mp-point/overview
- Point terminal configuration: https://www.mercadopago.com.br/developers/en/docs/mp-point/configure-terminal
- Point payment processing: https://www.mercadopago.com.br/developers/en/docs/mp-point/payment-processing
- Point notifications: https://www.mercadopago.com.br/developers/en/docs/mp-point/notifications
- Point integration test: https://www.mercadopago.com.br/developers/en/docs/mp-point/integration-test
- Point statuses: https://www.mercadopago.com.br/developers/en/docs/mp-point/resources/status-order-transaction
- Idempotency key notice: https://www.mercadopago.com.br/developers/en/news/2023/01/04/Idempotency-key-usage-will-be-mandatory

## Pix flow

Create Pix payments from the backend with `POST https://api.mercadopago.com/v1/orders`.

Request shape:

- `type`: `online`
- `total_amount`: decimal string in BRL, derived from integer minor units at the boundary
- `external_reference`: local checkout/payment reference, unique, no PII
- `processing_mode`: `automatic`
- `transactions.payments[].amount`: decimal string
- `transactions.payments[].payment_method.id`: `pix`
- `transactions.payments[].payment_method.type`: `bank_transfer`
- `transactions.payments[].expiration_time`: ISO 8601 duration, minimum 30 minutes and maximum 30 days if explicitly set
- `payer.email`: required by Mercado Pago

Required headers:

- `Authorization: Bearer <access_token>`
- `Content-Type: application/json`
- `X-Idempotency-Key: <uuid-or-local-idempotency-key>`

Expected initial response:

- order `id`
- order `status` such as `action_required`
- order `status_detail` such as `waiting_transfer`
- payment `id`
- payment `reference_id`
- payment `status`
- payment `status_detail`
- Pix `ticket_url`
- Pix `qr_code`
- Pix `qr_code_base64`

The frontend should display `qr_code_base64` as the scannable QR image and `qr_code` as the copy-and-paste Pix code. `ticket_url` can be stored for support/debugging, but the primary alpha UX should be in-app QR rendering.

## Point/card flow

Mercado Pago Point uses the same Orders API with `type: point`.

Prerequisites:

- Mercado Pago account.
- Supported Point Smart or Point Pro terminal. Mercado Pago lists Point Pro 2 as an available terminal for Point integration.
- Mercado Pago mobile app to associate the terminal.
- Store and point of sale configured in Mercado Pago.
- One terminal in PDV mode per point of sale.
- Terminal ID from `GET https://api.mercadopago.com/terminals/v1/list`.

Create a card order with `POST https://api.mercadopago.com/v1/orders`.

Request shape:

- `type`: `point`
- `external_reference`: local checkout/payment reference, unique, no PII, max 64 chars
- `expiration_time`: use a short value such as `PT16M`; if omitted, Mercado Pago expires the order after 15 minutes
- `transactions.payments[].amount`: decimal string
- `config.point.terminal_id`: selected terminal ID
- `config.point.print_on_terminal`: start with `no_ticket`
- `config.payment_method.default_type`: `credit_card` or `debit_card`
- `config.payment_method.default_installments`: use `1` for alpha unless explicitly changed later
- `config.payment_method.installments_cost`: `seller`

The Point terminal automatically receives the order. Finalization must wait for a verified webhook or a backend GET status check returning a successful terminal/order state.

Card support should stay in alpha scope if the product owner can provide:

- a production Mercado Pago account that can receive payments,
- a supported terminal such as Point Pro 2,
- terminal access to enable PDV mode,
- permission to do at least one low-value real transaction before alpha release.

Escalate to the product owner before deferring card support if any of those are unavailable.

## Webhook verification

Configure Mercado Pago Webhooks for the Order event.

The backend endpoint should be HTTPS in deployed environments. For local development, expose the local API through a temporary HTTPS tunnel such as ngrok or Cloudflare Tunnel and configure that URL in Mercado Pago's integration panel.

Verification algorithm:

1. Read `x-signature`.
2. Read `x-request-id`.
3. Read `data.id` from the query string, not from the JSON body.
4. For order notifications where Mercado Pago sends an uppercase alphanumeric `data.id`, lowercase it before signature validation.
5. Parse `x-signature` into `ts` and `v1`.
6. Build the manifest string: `id:<data.id>;request-id:<x-request-id>;ts:<ts>;`.
7. Compute HMAC-SHA256 hex using the Mercado Pago webhook secret.
8. Compare the computed value to `v1` with constant-time comparison.
9. Enforce a timestamp tolerance and reject stale notifications.
10. Store the raw payload, headers, verification result, and processing result.

After a valid webhook, fetch the order from Mercado Pago before finalizing local state. Do not finalize a sale from webhook body data alone.

Webhook actions to handle:

- `order.processed`: confirm payment, finalize sale, decrement reserved inventory, create item-level transactions.
- `order.canceled`: release reservation, mark payment canceled.
- `order.refunded`: mark payment refunded; refund behavior beyond alpha remains roadmap unless explicitly requested.
- `order.action_required`: keep payment pending and ask operator to check the terminal or retry verification.
- `order.failed`: keep sale unfinalized and ask operator to retry payment or cancel.
- `order.expired`: release reservation and mark payment expired.

## Polling and retry verification

Use `GET https://api.mercadopago.com/v1/orders/{order_id}` for backend status checks.

Polling rules:

- Poll only from backend code.
- Use polling for operator-driven "retry verification" and for a short scheduled recovery window when a webhook is delayed.
- Use bounded retries with structured warnings for transient network, 429, and 5xx responses, then raise the last error.
- Do not retry 4xx validation/authentication errors.
- Do not finalize unless Mercado Pago returns a terminal status that maps to local confirmed payment.
- Stop polling at payment expiration or when the checkout is canceled.

## Local payment state machine

Use local statuses that are provider-neutral and map Mercado Pago statuses at the boundary.

Suggested local statuses:

- `created`: local checkout/payment row exists.
- `provider_pending`: Mercado Pago order exists but is not paid.
- `action_required`: payer/operator action is required.
- `processing`: provider is still processing.
- `confirmed`: provider confirms successful payment.
- `failed`: provider rejected or failed the payment.
- `canceled`: checkout/payment was canceled.
- `expired`: provider order expired.
- `refunded`: provider reports refund.

Suggested mappings:

| Mercado Pago status | Mercado Pago detail | Local status | Inventory effect |
|---|---|---|---|
| `created` | `created` | `provider_pending` | keep reservation |
| `action_required` | `waiting_payment`, `waiting_transfer`, `check_on_terminal`, `action_required` | `action_required` | keep reservation until expiration/cancel |
| `at_terminal` | `at_terminal` | `processing` | keep reservation |
| `processing` | `in_process`, `in_review`, `pending_review_manual` | `processing` | keep reservation |
| `processed` | `accredited`, `processed` | `confirmed` | finalize sale |
| `failed` | any provider failure detail | `failed` | allow retry or cancel |
| `canceled` | `canceled`, `canceled_by_api`, `canceled_on_terminal` | `canceled` | release reservation |
| `expired` | `expired` | `expired` | release reservation |
| `refunded` | `refunded` | `refunded` | no alpha refund automation unless later specified |
| `charged_back` | any | `failed` or future `chargeback` | escalate behavior before production |

For Point, `action_required` can become a terminal-only confirmation state and may not change automatically. The frontend must tell the operator to check the terminal and then retry verification or cancel.

## Required stored fields

Store payment data separately from sales and transactions.

Suggested `payments` fields:

- `id`
- `band_id`
- `checkout_id`
- `provider`
- `method`
- `amount_minor`
- `currency`
- `local_status`
- `provider_order_id`
- `provider_payment_id`
- `provider_reference_id`
- `external_reference`
- `provider_status`
- `provider_status_detail`
- `provider_created_at`
- `provider_updated_at`
- `expires_at`
- `idempotency_key`
- `pix_qr_code`
- `pix_qr_code_base64`
- `pix_ticket_url`
- `point_terminal_id`
- `point_store_id`
- `point_pos_id`
- `raw_provider_response`
- `created_at`
- `updated_at`

Suggested `payment_events` fields:

- `id`
- `payment_id`
- `provider`
- `provider_order_id`
- `provider_event_action`
- `provider_event_type`
- `provider_request_id`
- `signature_timestamp`
- `signature_verified`
- `raw_headers`
- `raw_query`
- `raw_body`
- `processing_status`
- `processing_error`
- `received_at`
- `processed_at`

Do not store Mercado Pago access tokens, webhook secrets, or card data in database rows or logs.

## Idempotency requirements

External Mercado Pago operations requiring `X-Idempotency-Key`:

- Pix order creation.
- Point/card order creation.
- Order cancellation.
- Future refunds.

Internal operations requiring local idempotency:

- checkout creation,
- inventory reservation,
- payment creation,
- webhook processing,
- manual retry verification result processing,
- sale finalization,
- transaction generation,
- cancellation and reservation release.

Use local idempotency scoped by `band_id + operation + idempotency_key`. For provider calls, reuse the same provider idempotency key when retrying the same operation after timeout or transient failure. Never reuse a provider idempotency key for a different body.

## Error cases for frontend

The payment API should return explicit typed errors the UI can translate through i18n.

Required frontend states:

- payment provider unavailable,
- invalid Mercado Pago credentials/configuration,
- Pix QR creation failed,
- Pix QR expired,
- payment still pending,
- payment requires terminal confirmation,
- payment rejected by issuer,
- insufficient funds,
- card disabled,
- invalid installments,
- high risk rejection,
- maximum attempts exceeded,
- payment expired,
- payment canceled,
- cancellation failed because the provider order is already at the terminal,
- webhook verification failed,
- status verification failed,
- duplicate idempotency key with mismatched request,
- terminal not configured,
- terminal not in PDV mode,
- terminal unavailable.

## Testing strategy

Backend unit tests:

- provider status to local status mapping,
- money minor-unit to Mercado Pago decimal string conversion,
- webhook signature manifest construction,
- idempotency decision rules,
- payment state transitions.

Backend integration/smoke tests:

- create Pix order with Mercado Pago test credentials,
- poll Pix order by ID,
- handle Pix webhook payload after signature validation,
- create Point order with Mercado Pago test credentials,
- simulate Point `processed` event through `POST /v1/orders/{order_id}/events`,
- simulate Point `failed` event,
- verify duplicate webhook does not double-finalize a sale.

Manual alpha validation:

- one low-value real Pix payment,
- one low-value real card payment on the configured Point terminal,
- cancel a pending Pix checkout,
- cancel a pending Point checkout before terminal capture,
- handle Point order already at terminal,
- verify inventory reservation release after failed/canceled/expired payment.

## Implementation notes

Keep the Mercado Pago adapter in `apps/api/internal/infrastructure/mercadopago` behind an application-level payment interface. Domain and use case code should only see provider-neutral payment commands and statuses.

Recommended application methods:

- `CreatePixPayment`
- `CreatePointPayment`
- `GetPaymentStatus`
- `CancelPayment`
- `VerifyWebhookSignature`
- `RecordWebhookEvent`
- `ApplyPaymentStatus`

Use structured logs with fields such as `provider`, `operation`, `checkout_id`, `payment_id`, `provider_order_id`, `status_code`, `response_body`, and `mercadopago_request_id`. Do not interpolate secrets or log QR payloads unless explicitly needed for a short-lived debug session.

## Open decisions

- Confirm whether alpha will use one configured terminal or support selecting among multiple terminals.
- Confirm the default Point card type: credit only, debit only, or operator choice.
- Confirm whether installments are disabled for alpha by forcing one installment.
- Confirm the exact expiration time for Pix and Point reservations.
- Confirm who owns the Mercado Pago account and whether credentials are app-level or OAuth-based.
