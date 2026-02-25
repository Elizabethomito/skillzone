# Skillzone — Local-Network Demo Runbook

Use this guide to run the full end-to-end demo on a local Wi-Fi network
(e.g. a hotspot from your laptop) with no internet required.

---

## Cast of characters

| Person | Role | Email | Password |
|--------|------|-------|----------|
| **TechCorp Africa** | Event host (company) | `host@techcorp.test` | `demo1234` |
| **Amara Osei** | Veteran student | `amara@student.test` | `demo1234` |
| **Baraka Mwangi** | Newcomer student | `baraka@student.test` | `demo1234` |

**Amara** has been on the platform for months: 6 completed events, 6 skill
badges (Python, Data Science, Open Source, Cloud, Mobile Dev, Cybersecurity),
and is already registered for today's workshop.

**Baraka** just signed up. No history, no pre-registration — they arrived at
the venue and will scan the QR like everyone else.

---

## Step 1 — Find your laptop's local IP

```bash
# Linux / macOS
ip addr show | grep "inet " | grep -v 127.0.0.1
# Windows
ipconfig
```

Note the address — something like `192.168.x.x`. You'll use it everywhere below.

---

## Step 2 — Build and start the server

```bash
cd /home/akihara/hackathons/skillzone/backend
go build -o skillzone ./cmd/server/

ADDR=0.0.0.0:8080 JWT_SECRET=hackathon-demo ./skillzone
```

> **Why `0.0.0.0`?**  The default `:8080` only listens on localhost.
> `0.0.0.0:8080` listens on all interfaces, so phones and other laptops on
> the same Wi-Fi can reach it.

---

## Step 3 — Point the frontend at the backend

```bash
cd /home/akihara/hackathons/skillzone/frontend
VITE_API_URL=http://<YOUR_LAPTOP_IP>:8080 npm run dev -- --host
```

Other devices open: `http://<YOUR_LAPTOP_IP>:5173`

---

## Step 4 — Seed all demo data (one command)

```bash
curl -s -X POST http://localhost:8080/api/admin/seed | jq .
```

This creates the three accounts, 9 skill badges, 8 events (6 past/completed,
1 active workshop, 1 upcoming internship), Amara's full attendance history,
her skill badges, and her pre-registration for today's workshop.

**Safe to call multiple times** — every INSERT uses `OR IGNORE` so re-seeding
a running server is harmless.

The response includes the event IDs you'll need below:

```json
{
  "active_workshop": {
    "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
    "check_in_code": "DEMO-CHECKIN-CODE-AI-WORKSHOP-2026",
    "title": "Building Apps with AI Workshop"
  },
  "internship": {
    "event_id": "seed-event-intern-0000-0000-0000-000000000031",
    "title": "AI Product Internship",
    "slots_remaining": 1
  }
}
```

Save the IDs in environment variables for the curl snippets below:

```bash
BASE=http://localhost:8080
WORKSHOP_ID=seed-event-aiwork-0000-0000-0000-000000000030
INTERN_ID=seed-event-intern-0000-0000-0000-000000000031
CHECKIN_CODE=DEMO-CHECKIN-CODE-AI-WORKSHOP-2026

COMPANY_TOKEN=$(curl -s -X POST $BASE/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"host@techcorp.test","password":"demo1234"}' | jq -r .token)

AMARA_TOKEN=$(curl -s -X POST $BASE/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"amara@student.test","password":"demo1234"}' | jq -r .token)

BARAKA_TOKEN=$(curl -s -X POST $BASE/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"baraka@student.test","password":"demo1234"}' | jq -r .token)
```

---

## Demo script

### Scene 1 — Show Amara's history ("this is a real user")

```bash
# Her 6 skill badges
curl -s $BASE/api/users/me/skills \
  -H "Authorization: Bearer $AMARA_TOKEN" | jq '[.[] | .skill.name]'

# Her registered events (workshop is in there as confirmed)
curl -s $BASE/api/users/me/registrations \
  -H "Authorization: Bearer $AMARA_TOKEN" | jq '[.[] | {title: .event_title, status: .event_status}]'
```

### Scene 2 — Workshop is live: host activates it

The workshop is seeded as `active`, but you can demonstrate the control:

```bash
curl -s -X PATCH $BASE/api/events/$WORKSHOP_ID/status \
  -H "Authorization: Bearer $COMPANY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"active"}' | jq .
```

### Scene 3 — Host generates the check-in QR

```bash
curl -s $BASE/api/events/$WORKSHOP_ID/checkin-code \
  -H "Authorization: Bearer $COMPANY_TOKEN" | jq .
```

The frontend renders this as a QR code containing:

```json
{
  "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
  "host_sig": "DEMO-CHECKIN-CODE-AI-WORKSHOP-2026",
  "timestamp": <unix_now>
}
```

Display this QR on the projector screen.

### Scene 4 — Network drops

> "The venue is packed. The cell tower is overwhelmed — network is effectively
> down. But our app keeps working."

Chrome DevTools → **Network** tab → set throttle to **Offline**.

### Scene 5 — Amara and Baraka scan the QR offline

Both scan with their PWAs. Each app:
1. Validates the payload locally (no server call needed).
2. Stores `ATTENDANCE_PENDING` in IndexedDB.
3. Shows: *"Checked in ✓ — badges will appear when you reconnect."*

**Baraka was NOT pre-registered.** The QR scan still works; the server will
auto-register her when the sync fires.

### Scene 6 — Both apply for the internship while offline

While still offline, both open the internship event and tap **Apply**. The
PWA queues the registration locally with status `PENDING`. Only 1 slot remains.

### Scene 7 — Network returns, sync fires

Re-enable the network. The service worker's Background Sync fires automatically.

To demo manually with curl:

```bash
NOW=$(date +%s)

# ── Amara syncs her workshop check-in ──────────────────────────────────────
curl -s -X POST $BASE/api/sync/attendance \
  -H "Authorization: Bearer $AMARA_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"records\":[{
    \"local_id\":\"amara-local-001\",
    \"event_id\":\"$WORKSHOP_ID\",
    \"payload\":\"{\\\"event_id\\\":\\\"$WORKSHOP_ID\\\",\\\"host_sig\\\":\\\"$CHECKIN_CODE\\\",\\\"timestamp\\\":$NOW}\"
  }]}" | jq .

# Amara applies for the internship (gets the last slot)
curl -s -X POST $BASE/api/events/$INTERN_ID/register \
  -H "Authorization: Bearer $AMARA_TOKEN" | jq '{id, status}'

# ── Baraka syncs her workshop check-in (also auto-registers her) ────────────
curl -s -X POST $BASE/api/sync/attendance \
  -H "Authorization: Bearer $BARAKA_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"records\":[{
    \"local_id\":\"baraka-local-001\",
    \"event_id\":\"$WORKSHOP_ID\",
    \"payload\":\"{\\\"event_id\\\":\\\"$WORKSHOP_ID\\\",\\\"host_sig\\\":\\\"$CHECKIN_CODE\\\",\\\"timestamp\\\":$NOW}\"
  }]}" | jq .

# Baraka applies for the internship (slot gone → conflict_pending)
curl -s -X POST $BASE/api/events/$INTERN_ID/register \
  -H "Authorization: Bearer $BARAKA_TOKEN" | jq '{id, status}'
```

**Expected registration statuses:**
- Amara → `"confirmed"` (first applicant, slot was available)
- Baraka → `"conflict_pending"` (slot exhausted)

### Scene 8 — Badges awarded automatically

```bash
# Amara now has 8 badges (6 old + AI Application Development + Prompt Engineering)
curl -s $BASE/api/users/me/skills \
  -H "Authorization: Bearer $AMARA_TOKEN" | jq '[.[] | .skill.name]'

# Baraka has her first badge
curl -s $BASE/api/users/me/skills \
  -H "Authorization: Bearer $BARAKA_TOKEN" | jq '[.[] | .skill.name]'
```

### Scene 9 — Host sees the conflict on the dashboard

```bash
curl -s $BASE/api/events/$INTERN_ID/registrations \
  -H "Authorization: Bearer $COMPANY_TOKEN" \
  | jq '[.[] | {name: .student_name, status}]'
```

Output:
```json
[
  { "name": "Amara Osei",    "status": "confirmed" },
  { "name": "Baraka Mwangi", "status": "conflict_pending" }
]
```

### Scene 10 — Host resolves the conflict

```bash
# Get Baraka's registration ID
BARAKA_REG_ID=$(curl -s $BASE/api/events/$INTERN_ID/registrations \
  -H "Authorization: Bearer $COMPANY_TOKEN" | \
  jq -r '.[] | select(.student_name=="Baraka Mwangi") | .id')

# Confirm Baraka (host expands to 2 slots)
curl -s -X PATCH $BASE/api/events/$INTERN_ID/registrations/$BARAKA_REG_ID \
  -H "Authorization: Bearer $COMPANY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"confirm"}' | jq .

# Or waitlist instead:
# -d '{"action":"waitlist"}'
```

### Scene 11 — End the workshop early (EOD)

```bash
curl -s -X PATCH $BASE/api/events/$WORKSHOP_ID/status \
  -H "Authorization: Bearer $COMPANY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"completed"}' | jq .
```

> Attendance syncs within 24 hours of the QR timestamp are still accepted
> after the event is completed — students who left before reconnecting can
> still sync from home.

---

## Showing offline mode in Chrome DevTools

1. **Application** → **Service Workers** → tick "Offline"
2. Show the PWA loading from cache
3. **Application** → **Storage** → **IndexedDB** → `skillzone`
   — pending attendance records visible here
4. Untick "Offline" → watch **Background Services → Background Sync** fire

Slow network simulation: **Network** tab → **Slow 3G** preset.

---

## Failure scenarios to demonstrate

| Scenario | How to trigger | Expected result |
|---|---|---|
| Wrong QR code | Change `host_sig` in payload | `rejected: invalid check-in signature` |
| Stale QR (>24 h) | Set `timestamp` to yesterday | `rejected: check-in payload has expired` |
| Double-sync retry | Call sync twice, same `local_id` | Second call returns `verified` — idempotent |
| Two offline internship applicants | Run Scenes 6 + 7 | First = confirmed, second = conflict_pending |
| Host waitlists applicant | Use `"action":"waitlist"` | Status becomes `waitlisted` |
| Server down during sync | Stop server, scan QR, restart, re-sync | Sync succeeds on reconnect |

---

## New API endpoints added for this demo

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `POST` | `/api/admin/seed` | — | Load all demo fixture data (idempotent) |
| `PATCH` | `/api/events/{id}/status` | company (host only) | Transition event: `upcoming` → `active` → `completed` |
| `GET` | `/api/events/{id}/registrations` | company (host only) | Attendee list with name, email, registration status |
| `PATCH` | `/api/events/{id}/registrations/{reg_id}` | company (host only) | Resolve conflict: `{"action":"confirm"}` or `{"action":"waitlist"}` |

### How the internship slot model works

Events have optional `capacity` and `slots_remaining` fields (both `null` = unlimited).

```
capacity: 2, slots_remaining: 1
```

When a registration arrives and `slots_remaining == 0`:
1. The registration is recorded as `status: "conflict_pending"`.
2. `slots_remaining` is **not** decremented (the host decides whether to expand).
3. The host sees it in `GET /api/events/{id}/registrations`.
4. The host calls `PATCH` to confirm or waitlist.

The attendance sync (`POST /api/sync/attendance`) also runs this same slot
logic — scanning a QR code auto-registers the student, applying the same
capacity checks.

---

## Troubleshooting

**Other devices can't connect**
```bash
sudo ufw allow 8080
```

**Port in use**
```bash
lsof -i :8080
ADDR=0.0.0.0:9090 ./skillzone
```

**macOS `date -d` unavailable**
```bash
date -u +%s   # current unix timestamp
```

**Re-seed without restarting**
```bash
curl -s -X POST http://localhost:8080/api/admin/seed | jq .seeded
```
