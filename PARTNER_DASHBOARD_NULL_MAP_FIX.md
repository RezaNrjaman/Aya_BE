# Fix Report — Partner Dashboard null.map

## Root cause
The Partner Dashboard loads unit usaha from `GET /m2/v1/admin/business-units`. When a Partner has no units (or PostgreSQL returns zero rows), the Go repository returns a nil slice. Gin serializes that slice as `data: null`. The frontend then used `units.map(...)`, causing:
`Cannot read properties of null (reading 'map')`.

## Fixes applied
### Backend
- `internal/modules/m2/handler/unit_handler.go`: `ListAdminUnits` now converts a nil unit slice to an empty `[]domain.BusinessUnit{}` before creating the list response. API returns `data: []` for zero rows.

### Frontend
- `src/features/m2/api.ts`: `fetchAdminUnits` normalizes non-array `data` to `[]`.
- `src/features/m2/api.ts`: `fetchActiveUnits` normalizes non-array `data` to `[]`.
- `src/pages/dashboard/DashboardPage.tsx`: dashboard loader checks the returned list is an array and PartnerDashboardPage uses a defensive `safeUnits` before `.map()`.

## Verification
- TypeScript `npx tsc --noEmit -p tsconfig.app.json`: PASS.
- Go formatting (`gofmt`): PASS.
- Vite production build: NOT COMPLETED; local `vite` executable returned `Permission denied`.
- Go unit/integration tests: NOT RUN; supplied environment uses Go 1.23.2, while project `go.mod` requires a newer Go version.
- Live Partner login against the user's Supabase: NOT RUN; no access to the user's local app or Supabase credentials.

## Deployment
No Supabase migration is required. Replace the FE and BE source with these archives, keep `.env`, restart BE and FE. A Partner with zero business units should see the dashboard with an empty unit filter rather than a React crash.
