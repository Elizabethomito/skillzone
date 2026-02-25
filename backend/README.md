# Skillzone Backend

Go + SQLite REST API for the Skillzone PWA.

## Stack

| Layer | Technology |
|---|---|
| Language | Go 1.24 |
| Database | SQLite (via `go-sqlite3`) |
| Auth | HS256 JWT (`golang-jwt/jwt/v5`) |
| Passwords | bcrypt (`golang.org/x/crypto`) |
| IDs | UUID v4 (`google/uuid`) |

## Project layout

```
backend/
├── cmd/server/main.go          # Entry point – routes wired here
└── internal/
    ├── models/models.go        # Domain types + DTOs
    ├── db/db.go                # SQLite open + schema migrations
    ├── auth/jwt.go             # Token generation / validation
    ├── middleware/middleware.go # CORS, Authenticate, RequireRole
    └── handlers/
        ├── server.go           # Shared Server struct + helpers
        ├── auth.go             # Register, Login, Me
        ├── events.go           # CRUD events, registration
        ├── skills.go           # CRUD skills
        └── sync.go             # Attendance sync + user skill/registration views
```

## Running locally

```bash
cd backend

# Optional environment variables (defaults shown)
export DATABASE_URL="skillzone.db?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"
export JWT_SECRET="changeme-use-a-real-secret-in-production"
export ADDR=":8080"

go run ./cmd/server/
```

## Running tests

```bash
cd backend
go test ./...
```

All packages use in-memory SQLite so no external services are needed.

---

## API reference

All endpoints return `application/json`. Authenticated endpoints require:

```
Authorization: Bearer <token>
```

### Auth

| Method | Path | Auth | Body / Notes |
|--------|------|------|---|
| POST | `/api/auth/register` | — | `{email, password, name, role}` → `{token, user}` |
| POST | `/api/auth/login` | — | `{email, password}` → `{token, user}` |
| GET  | `/api/auth/me` | ✓ | Returns current user |

`role` must be `"student"` or `"company"`.

### Skills

| Method | Path | Auth | Notes |
|--------|------|------|---|
| POST | `/api/skills` | company | `{name, description}` |
| GET  | `/api/skills` | — | List all skill badges |

### Events

| Method | Path | Auth | Notes |
|--------|------|------|---|
| POST | `/api/events` | company | `{title, description, location, start_time, end_time, skill_ids[]}` |
| GET  | `/api/events` | — | List all events (with linked skills) |
| GET  | `/api/events/{id}` | — | Single event |
| GET  | `/api/events/{id}/checkin-code` | company (host only) | Returns `check_in_code` for QR generation |
| POST | `/api/events/{id}/register` | student | Register intent to attend |

### Sync (local-first core)

| Method | Path | Auth | Notes |
|--------|------|------|---|
| POST | `/api/sync/attendance` | student | Batch-sync offline check-in records |

#### `POST /api/sync/attendance` — request body

```json
{
  "records": [
    {
      "local_id": "client-uuid-for-idempotency",
      "event_id": "...",
      "payload": "{\"event_id\":\"...\",\"host_sig\":\"...\",\"timestamp\":1708812000}"
    }
  ]
}
```

`payload` is the raw JSON string from the host's QR code. The server verifies that `host_sig` matches the event's stored `check_in_code`, and that the timestamp is no older than 24 hours. On success, skill badges are awarded automatically.

#### Response

```json
{
  "results": [
    {
      "local_id": "client-uuid-for-idempotency",
      "status": "verified",
      "message": "attendance verified and skills awarded"
    }
  ]
}
```

`status` is one of: `verified` | `rejected`.

### Student views

| Method | Path | Auth | Notes |
|--------|------|------|---|
| GET | `/api/users/me/skills` | student | All earned skill badges |
| GET | `/api/users/me/registrations` | student | All registered events |

---

## Local-first flow

```
HOST DEVICE (online earlier, now offline)
  1. Opens event in PWA → cached event details loaded from IndexedDB
  2. Taps "Start Check-In"
  3. PWA generates QR code:
       { event_id, host_sig: check_in_code, timestamp }

STUDENT DEVICE (fully offline)
  4. Opens PWA → taps "Scan Check-In"
  5. Scans host's QR → stores ATTENDANCE_PENDING in IndexedDB
  6. UI optimistically shows "Badge will be verified on reconnect"

LATER (back on Wi-Fi)
  7. Service worker fires POST /api/sync/attendance
  8. Server verifies signature → awards skill badges
  9. IndexedDB record updated to ATTENDANCE_VERIFIED
```

No internet is required at the venue. The QR code itself is the proof.

---

## Failure scenarios handled

| Scenario | Behaviour |
|---|---|
| Network drops mid-sync | Client retries; server upserts are idempotent |
| Server returns 500 | Service worker retry with exponential backoff (frontend) |
| Student syncs the same record twice | `ON CONFLICT … DO UPDATE` — safe no-op |
| Wrong / tampered QR code | Signature mismatch → `rejected` result |
| Stale QR code (> 24 h old) | Timestamp check → `rejected` result |
| App closed during write | IndexedDB transaction is atomic; partial writes don't occur |
