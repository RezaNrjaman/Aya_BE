# Test Report — Araya M4/M5 Cart, Product Ownership, Service Request, Partner Login

## Changes covered
- Empty cart API responses are normalized to `items: []` in backend and frontend.
- Cart page safely handles legacy/null cart payloads.
- Cart add/update validates active product availability and stock.
- One cart is restricted to one selling Partner because M4 creates one tenant-owned order/invoice per checkout.
- Product creation derives `products.partner_id` from the selected active `business_units` row; client-supplied partner IDs are ignored.
- Product management dropdown shows searchable business-unit names and owning Partner names.
- Public catalog product cards and filters show business unit and Partner.
- Service-request form requires a real public unit, shows Partner + Unit destination, and backend resolves Partner from the unit ID (never trusts a client-supplied Partner ID).
- Partner request inbox displays Partner and Unit names.
- Partner-owner role assignment is deferred until `users.partner_id` has been linked, preventing the registration/verification transaction from failing before linkage.
- Existing partner-role permissions and partially linked verified owner accounts are repaired by migration 000013.

## Database
Run `000013_partner_portal_permissions.up.sql` after `000012_product_type.up.sql`.
No database reset is required. Back up Supabase before applying the migration.
Migration 000013 also reconciles existing `products.partner_id` values from `business_units.partner_id`.

## Verification performed in this environment
- Go formatting (`gofmt -w` / `gofmt -l`): PASS.
- Frontend TypeScript (`tsc --noEmit -p tsconfig.app.json`): PASS.
- Static source checks for cart normalization, product ownership derivation, service-request unit validation, partner role deferral, and migration presence: PASS.
- FE production bundle: NOT RUN successfully. The supplied `node_modules` archive is missing the platform-specific Rolldown native binding; Vite cannot start.
- FE Vitest/Oxlint: NOT RUN successfully for the same missing native optional dependencies.
- Go unit/integration tests: NOT RUN. The project requires Go 1.26.2; this environment has Go 1.23.2, and dependencies cannot be downloaded because network access to proxy.golang.org is unavailable.
- Live Supabase integration tests: NOT RUN; this environment has no access to the user's Supabase credentials/database.

## Important operational note
If a verified Partner owner account still has an empty `password_hash`, that account must complete its existing activation flow before login. Migration 000013 repairs owner linkage and role assignment; it does not invent or reset passwords.
