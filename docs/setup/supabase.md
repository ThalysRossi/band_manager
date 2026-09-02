# Supabase setup

This project uses Supabase Auth for browser login/signup. The Go API validates
Supabase access tokens against the project's JWKS signing keys and stores
application data in PostgreSQL through `DATABASE_URL`.

## Create the Supabase project

1. Create a Supabase project.
2. Open **Project Settings > API** and copy:
   - Project URL
   - publishable key
   - secret key for the API server
3. Open **Project Settings > Database** and copy the PostgreSQL connection
   string if you want to use Supabase Postgres instead of local Docker
   Postgres.

Do not copy the legacy JWT secret or service role key into this application.
The frontend and backend use the publishable key, while the backend verifies
access tokens using `${SUPABASE_URL}/auth/v1/.well-known/jwks.json`.
Before onboarding or accepting an invite, the backend also checks
`${SUPABASE_URL}/auth/v1/user` and refuses to persist application records until
the email is verified.
The backend also uses a server-only secret key to create Supabase Storage signed
upload URLs and verify uploaded object metadata.

## Auth settings

In **Authentication > Providers**:

- Enable Email provider.
- Enable email/password signups.
- Require email confirmation in every environment.
- For local development, use `http://localhost:5173` as the site URL.
- Add `http://localhost:5173/auth/callback` to redirect URLs.
- Add the deployed `/auth/callback` URL after the hosting provider and domain
  are selected.

Google OAuth is deferred. Do not create a Google OAuth client yet.

Create `apps/web/.env.local`:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_SUPABASE_URL=https://your-project.supabase.co
VITE_SUPABASE_PUBLISHABLE_KEY=your-publishable-key
```

The `VITE_` variables are browser-exposed. Use the publishable key there; do not
use a secret key there.

## Local API environment

Run the API with the Supabase project URL, publishable key, secret key, and
storage bucket:

```bash
APP_ENV=local \
API_ADDR=:8080 \
API_ALLOWED_ORIGINS=http://localhost:5173 \
DATABASE_URL=postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable \
REDIS_URL=redis://localhost:6379/0 \
SUPABASE_URL=https://your-project.supabase.co \
SUPABASE_PUBLISHABLE_KEY=your-publishable-key \
SUPABASE_SECRET_KEY=your-secret-key \
SUPABASE_STORAGE_BUCKET=inventory-photos \
MERCADOPAGO_ACCESS_TOKEN=replace-me \
MERCADOPAGO_WEBHOOK_SECRET=replace-me \
MERCADOPAGO_POINT_TERMINAL_ID=replace-me \
pnpm dev:api
```

For Supabase Postgres, replace `DATABASE_URL` with the Supabase database
connection string and apply the SQL migrations in `apps/api/migrations` in
numeric order before using the app.

## Smoke test

1. Start local dependencies and the API.
2. Start the web app.
3. Visit `http://localhost:5173/signup`.
4. Create an account and confirm the email.
5. Follow the callback, complete band onboarding, and open `/account`.
6. Create a viewer invite and copy the invite link.
7. Log in as the invited email and open the invite link to accept it.

If protected API calls return `401`, verify `SUPABASE_URL`, confirm the project
uses asymmetric signing keys exposed by JWKS, and inspect the API logs for
issuer, audience, expiry, or key-ID validation failures.
