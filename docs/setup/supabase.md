# Supabase setup

This project uses Supabase Auth for browser login/signup. The Go API validates
Supabase access tokens with the Supabase JWT secret and stores application data
in PostgreSQL through `DATABASE_URL`.

## Create the Supabase project

1. Create a Supabase project.
2. Open **Project Settings > API** and copy:
   - Project URL
   - anon public key
   - JWT secret
3. Open **Project Settings > Database** and copy the PostgreSQL connection
   string if you want to use Supabase Postgres instead of local Docker
   Postgres.

Do not put the service role key in frontend environment variables. The current
runtime only needs the frontend anon key and the backend JWT secret.

## Auth settings

In **Authentication > Providers**:

- Enable Email provider.
- Enable email/password signups.
- For local development, use `http://localhost:5173` as the site URL.
- Add `http://localhost:5173/*` to redirect URLs.

Email confirmation may be enabled or disabled for local development. When it is
enabled, owner signup can return the frontend's "check your email" state until
the account is verified and the user logs in again.

## Local frontend environment

Create `apps/web/.env.local`:

```env
VITE_API_BASE_URL=http://localhost:8080
VITE_SUPABASE_URL=https://your-project.supabase.co
VITE_SUPABASE_ANON_KEY=your-anon-key
```

The `VITE_` variables are browser-exposed. Only use the anon public key there.

## Local API environment

Run the API with the same JWT secret Supabase uses to sign access tokens:

```bash
APP_ENV=local \
API_ADDR=:8080 \
API_ALLOWED_ORIGINS=http://localhost:5173 \
DATABASE_URL=postgres://band_manager:band_manager@localhost:5432/band_manager?sslmode=disable \
REDIS_URL=redis://localhost:6379/0 \
SUPABASE_JWT_SECRET=your-supabase-jwt-secret \
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
4. Create an owner account with the same email used in Supabase Auth.
5. Log in and open `/account`.
6. Create a viewer invite and copy the invite link.
7. Log in as the invited email and open the invite link to accept it.

If protected API calls return `401`, verify that `SUPABASE_JWT_SECRET` matches
the Supabase project's JWT secret, not the anon key.
