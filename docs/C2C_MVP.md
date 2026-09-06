# TokenC2C MVP

TokenC2C extends Sub2API with an owner-scoped marketplace for user-contributed
official provider accounts. New accounts enter the shared scheduler only after
an upstream verification succeeds.

## Accounting model

- A normal Sub2API usage record remains the source of truth for the buyer charge.
- A C2C trade is created in the same database transaction as usage billing.
- The seller receives 90% in a pending wallet and the platform receives 10%.
- Pending seller earnings mature into the available balance after 24 hours.
- `(request_id, api_key_id)` and ledger idempotency keys prevent duplicate charges.
- Buyer and seller request/token/currency totals are derived from immutable trades.
- A user may consume their own provider account; the double-entry ledger still balances.

## Safety boundaries

- Provider records are owner-scoped for listing, verification, pause/resume, and deletion.
- Credentials are accepted only for supported official platforms and are never returned by
  the C2C API.
- User-supplied custom upstream URLs and generic relay account types are rejected.
- Accounts start unschedulable and join the pool only after verification.
- Optional daily contribution limits automatically pause exhausted accounts.
- Deleting an account removes its active pool metadata while historical trades remain.

The MVP reuses Sub2API's existing account credential storage. Before exposing a NAS
deployment to the public Internet, add application-level encryption at rest for provider
credentials, a formal dispute/refund workflow, abuse monitoring, and legal/compliance review.

## User API

All endpoints below are under `/api/v1/c2c` and require a logged-in user:

- `GET/POST /provider-accounts`
- `POST /provider-accounts/:id/verify`
- `PUT /provider-accounts/:id/pool-state`
- `DELETE /provider-accounts/:id`
- `GET /wallet`
- `GET /stats`
- `GET /ledger?limit=50`

## Mac Docker deployment

Create `deploy/.env.c2c` from `deploy/.env.c2c.example`, replace every placeholder,
then run:

```bash
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml up -d --build --wait --wait-timeout 180
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml restart app
docker compose --env-file deploy/.env.c2c -f deploy/docker-compose.c2c.yml start --wait --wait-timeout 180 app
curl -fsS http://127.0.0.1:18080/health
```

The default bind is loopback-only (`127.0.0.1:18080`). The one-shot bootstrap service
enables registration, requires invitation codes, and disables email verification for this
local preview. Invitation codes can then be created from the administrator UI.
The one-time app restart reloads the bootstrapped branding and registration settings.

## Verification commands

```bash
cd backend
go test -tags=unit ./...
go test -tags=integration ./internal/repository -run '^TestC2CBilling' -count=1

cd ../frontend
pnpm typecheck
pnpm test:run
pnpm build
```

For a later NAS migration, copy the repository and environment file separately, preserve
the PostgreSQL/Redis/application volumes with verified backups, and change the port bind only
after the NAS reverse proxy and TLS route are ready.

## QNAP bridge deployment

`deploy/docker-compose.c2c.nas.yml` starts the stack on the unpublished
`subc2c-nas-internal` Docker bridge. After the application becomes healthy,
`deploy/deploy-c2c-nas.sh` attaches only the application container to the external
`qnet-static-eth2-b0591c` network, exposing `192.168.31.225:8080` for inbound LAN
traffic while retaining the private bridge for service discovery. PostgreSQL and Redis
never join qnet and publish no host ports. Change
`SUBC2C_BRIDGE_IP` only after checking the live NAS container inventory and LAN address
use. The application, PostgreSQL, and Redis use `unless-stopped` so they return after a
NAS reboot. The one-shot bootstrap container remains non-restarting by design.
