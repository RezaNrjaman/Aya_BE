# ARAYA Backend — Vercel deployment

This package adds a Vercel Go Function entry point while preserving the existing Gin API routes.

## Vercel project setup

1. Import this backend folder as a separate Vercel project.
2. Set **Root Directory** to `BE` if the Git repository contains the backend inside a `BE/` subfolder. If you upload this ZIP and extract it, select the extracted `BE` folder as the project root.
3. Do not configure a custom Build Command or Output Directory. Vercel should detect the Go function at `api/index.go` using `go.mod`.
4. Deploy after setting all environment variables below.

## Required Vercel Environment Variables

Set these under Project → Settings → Environment Variables for Production (and Preview if needed):

- `APP_ENV=production`
- `DATABASE_URL` = Supabase PostgreSQL connection string (use the Supabase connection pooler URI when appropriate)
- `JWT_SECRET` = unique random secret, at least 32 characters
- `JWT_EXPIRATION_HOURS=8`
- `SERVICE_KEY` = unique random service-to-service secret, at least 32 characters
- `ALLOWED_ORIGINS` = exact HTTPS origin of the deployed frontend, e.g. `https://your-frontend.vercel.app` (no trailing slash; add more origins comma-separated if required)
- `SANDBOX_CALLBACK_SECRET` = random callback secret of at least 32 characters, matching the sandbox callback verification configuration

Do not paste secret values into source files or commit `.env`.

## Smoke tests after deployment

Open these URLs using the deployed backend domain:

- `/` — should return service status JSON
- `/health` — basic process health
- `/api/v1/health` — API and database status; database should report `connected`

Then test the existing API routes, for example:

- `/api/m1/v1/auth/login`
- `/api/m2/v1/admin/business-units`

The API routes are still registered with their existing `/api/...` prefix. `vercel.json` forwards the original path to the Go Function.

## Important

- No Supabase migration is introduced by this deployment adapter.
- Keep the existing database and data. Do not reset Supabase.
- The function initializes the Gin router once per warm Vercel instance and retains its PostgreSQL pool for reuse.
- If `/api/v1/health` returns `SERVICE_UNAVAILABLE`, inspect Vercel Function Logs. The response deliberately avoids exposing database credentials or secret values.
- Ensure the Vercel Project Root Directory points to this backend module (the folder containing `go.mod`, `vercel.json`, and `api/`).
