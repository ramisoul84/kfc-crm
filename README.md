# kfc-crm

Management backend for the KFC Russia platform. Single source of truth
for regions, restaurants, users, devices, and menus.

## What it does

CRM is where **humans** configure the system:

- **Regions** — cities. Only `super_admin` can create/update/delete them.
- **Restaurants** — belong to a region. Managed by `regional_manager` or above.
- **Users** — created with a role, receive a welcome email with an initial
  password, must change it on first login.
- **Devices** — kiosks, POS terminals, kitchen displays. Registered per
  restaurant, paired using short-lived codes.
- **Menus** — categories, items, variations, per-restaurant overrides,
  and promotions. The "effective menu" is computed on demand by merging
  master data with overrides and active promotions.
- **Menu delivery** — exposes the effective menu to `kfc-userapi` over gRPC
  so devices can fetch it without hitting Postgres directly.

## Architecture
┌────────────────────────────┐
CRM web UI ──HTTP▶│ │
│ kfc-crm │
kfc-userapi ─gRPC▶│ │
└──────────┬─────────────────┘
│
┌──────────┴──────────┐
│ │
┌─────▼─────┐ ┌──────▼──────┐
│ Postgres │ │ Redis │
│ (source │ │ (sessions, │
│ of truth)│ │ tokens) │
└───────────┘ └─────────────┘
│
│ gRPC
▼
┌─────────────────────┐
│ kfc-notification │ welcome + pairing emails
└─────────────────────┘
│ gRPC
▼
┌─────────────────────┐
│ kfc-userapi │ pairing code cache
└─────────────────────┘


**Postgres is the source of truth.** Redis is used only for short-lived
session/refresh-token state. Nothing is authoritative in Redis.

## Key flows

### User registration

1. Manager (or above) creates a user with a role and scope (region or
   restaurant).
2. CRM generates an initial password, hashes it, stores the user with
   `status = pending`.
3. CRM calls `kfc-notification` over gRPC to send a welcome email
   containing the initial password.
4. The user logs in with the initial password. The response forces a
   profile-completion + password-change step.
5. After the change, the user is `active`.

### Device pairing

1. Manager clicks "Pair all devices" (or "Pair device") in the CRM UI.
2. CRM generates one 6-digit code per device, in memory only — **CRM does
   not persist pairing codes**.
3. CRM pushes the codes to `kfc-userapi` over gRPC. userapi stores them
   in Redis (bcrypt-hashed).
4. CRM sends a pairing email to the manager via `kfc-notification`.
5. The manager walks to each device and enters its code.
6. The device calls userapi over HTTP, validates the code, and receives a
   JWT. The code is consumed (single-use).

### Menu delivery

1. `kfc-userapi` calls CRM's `MenuService.GetEffectiveMenu` over gRPC on a
   cache miss.
2. CRM loads categories, items, variations, per-restaurant overrides, and
   active promotions, then merges them into an effective menu.
3. The effective menu is returned as a protobuf tree, cached by userapi in
   Redis for 24h.
4. On any menu change, CRM publishes a Kafka event `crm.menu.updated` so
   userapi can invalidate the affected cache entry.

## Roles and permissions

| Role | Scope | Can |
|---|---|---|
| `super_admin` | Everything | Manage regions, restaurants, users, devices, menus |
| `regional_manager` | One region | Manage restaurants, users, devices, menus in their region |
| `restaurant_manager` | One restaurant | Manage users, devices, and menus in their restaurant |
| `shift_manager` | One restaurant | Read devices, menus, users in their restaurant |
| `cashier` | One restaurant | Read-only |

Permissions are enforced by `RBACService`, which combines a **permission
check** (does the role have `device.pair`?) with a **scope check** (does
the user's region/restaurant contain this resource?).

## Configuration

Read from environment variables, loaded via `.env` in development.

| Variable | Default | Purpose |
|---|---|---|
| `APP_ENV` | `development` | `development` / `staging` / `production` |
| `HTTP_PORT` | `8000` | HTTP server port |
| `GRPC_PORT` | `9090` | gRPC server port (menu + device pairing client target) |
| `DB_*` | — | Postgres connection |
| `REDIS_*` | — | Redis connection (sessions, refresh tokens) |
| `JWT_SECRET` | — | HMAC secret for user session tokens |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime |
| `NOTIFICATION_ADDRESS` | `localhost:9101` | kfc-notification gRPC |
| `NOTIFICATION_ENABLED` | `true` | Set `false` to use a no-op client in dev |
| `USERAPI_ADDRESS` | `localhost:9102` | kfc-userapi gRPC |
| `USERAPI_ENABLED` | `true` | Set `false` to use a no-op client in dev |
| `KAFKA_BROKERS` | — | For publishing `crm.menu.updated` |

See `.env.example` for the full list.

## Running

### Local

```bash
cp .env.example .env
make proto-gen
go run ./cmd/crm