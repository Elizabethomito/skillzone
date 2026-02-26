# Skillzone — Local-Network Demo Runbook

Use this guide to run the full end-to-end demo on a local Wi-Fi network
(e.g. a hotspot from your laptop) with **no internet required**.

---

## Who does what during the demo

You need **one laptop** (the server) and **two helpers** (or two browser
windows on separate devices) to show the contrast between a veteran user and
a newcomer.

| Person | Device | What they do |
|--------|--------|--------------|
| **Presenter / operator** | Server laptop | Runs the Go backend + Vite dev server, seeds data, shows the host dashboard (TechCorp Africa), drives curl commands for the audience |
| **Helper A — Amara** | Phone or second laptop | Opens `http://<SERVER_IP>:5173` in Chrome, signs in as `amara@student.test` — plays the veteran student with 6 badges |
| **Helper B — Baraka** | Phone or second laptop | Opens `http://<SERVER_IP>:5173` in Chrome, signs in as `baraka@student.test` — plays the newcomer with zero history |

> **Tip:** If you only have one laptop, open three Chrome profiles (or two
> Incognito windows) and use them side-by-side.  The host session stays in
> the main window; Amara and Baraka each get their own Incognito window so
> their JWT cookies don't collide.

---

## Cast of characters

### Loginable accounts (all password `demo1234`)

| Person | Role | Email |
|--------|------|-------|
| **TechCorp Africa** | Event host (company) | `host@techcorp.test` |
| **Amara Osei** | Veteran student | `amara@student.test` |
| **Baraka Mwangi** | Newcomer student | `baraka@student.test` |

### Background companies (filler data only, no login needed)

| Company | Domain |
|---------|--------|
| **GreenLeaf Agri** | Agriculture & Sustainability |
| **MedConnect Health** | Healthcare & Life Sciences |

**Amara** has been on the platform for months: 6 completed events, 6 skill
badges (Python, Data Science, Open Source, Cloud Computing, Mobile Development,
Cybersecurity Fundamentals), and is already registered for today's AI workshop.

**Baraka** just signed up. No history, no pre-registration — they arrived at
the venue and will scan the QR like everyone else.

**15 filler students** are pre-seeded across all three companies to populate
the Candidates search and give realistic badge/attendance counts:
- **TechCorp** (8): Chidi, Fatima, Kwame, Aisha, Tobi, Ngozi, Joel, Lila
- **GreenLeaf** (4): Zara, Emeka, Sade, Kofi
- **MedConnect** (3): Muna, Dayo, Nia

---

## Step 1 — Find the server laptop's local IP

Run this on the **server laptop**:

```bash
# Linux / macOS
ip addr show | grep "inet " | grep -v 127.0.0.1
# Windows
ipconfig
```

You'll see something like `192.168.x.x`. **Share this address with both helpers**
so they can point their browsers at it.

---

## Step 2 — Build and start the server (server laptop only)

```bash
cd /home/akihara/hackathons/skillzone/backend
go build -o skillzone ./cmd/server/

ADDR=0.0.0.0:8080 JWT_SECRET=hackathon-demo ./skillzone
```

> **Why `0.0.0.0`?**  The default `:8080` only listens on localhost.
> `0.0.0.0:8080` listens on all interfaces so Helper A and Helper B can
> reach the API over Wi-Fi.

---

## Step 3 — Point the frontend at the backend (server laptop only)

```bash
cd /home/akihara/hackathons/skillzone/frontend
npm run dev -- --host
```

The `--host` flag makes Vite listen on `0.0.0.0:5173` (not just localhost).
No environment variable needed — the app detects the server's IP from the
browser's own address bar at runtime, so it works on every device automatically.

**Helper A and Helper B** open this URL on their devices:
```
http://<SERVER_IP>:5173
```

They will see the Skillzone PWA. Chrome will offer an **Install** prompt
(⊕ in the address bar) — accept it for the full standalone experience.

**Sign-in credentials for the helpers:**

| Helper | Email | Password |
|--------|-------|----------|
| Helper A (Amara) | `amara@student.test` | `demo1234` |
| Helper B (Baraka) | `baraka@student.test` | `demo1234` |

> The sign-in page has **click-to-fill** buttons for all three demo accounts —
> helpers just tap their name and hit Sign In.

---

## Step 4 — Seed all demo data (one command)

```bash
curl -s -X POST http://localhost:8080/api/admin/seed | jq .
```

This creates **20 users** (3 companies, 2 demo students, 15 filler students),
**37 skill badges** across 5 domains, **18 events** (3 past + 1 active + 1 upcoming
per company), all attendance histories, and Amara's pre-registration for today's
AI workshop.

**Safe to call multiple times** — every INSERT uses `OR IGNORE` so re-seeding
a running server is harmless.

The response includes all three active workshop IDs and the TechCorp internship:

```json
{
  "seeded": true,
  "accounts": [...],
  "companies": [
    {"id": "...", "name": "TechCorp Africa",    "domain": "Technology"},
    {"id": "...", "name": "GreenLeaf Agri",     "domain": "Agriculture & Sustainability"},
    {"id": "...", "name": "MedConnect Health",  "domain": "Healthcare & Life Sciences"}
  ],
  "active_workshop": {
    "event_id": "seed-event-aiwork-0000-0000-0000-000000000506",
    "title": "Building Apps with AI Workshop",
    "host": "TechCorp Africa"
  },
  "active_agri_workshop": {
    "event_id": "seed-event-agrwrk-0000-0000-0000-000000000513",
    "title": "Agro-processing & Market Linkages Workshop",
    "host": "GreenLeaf Agri"
  },
  "active_med_workshop": {
    "event_id": "seed-event-medwrk-0000-0000-0000-000000000523",
    "title": "AI in Healthcare: From Data to Diagnosis",
    "host": "MedConnect Health"
  },
  "internship": {
    "event_id": "seed-event-intern-0000-0000-0000-000000000507",
    "title": "AI Product Internship",
    "slots_remaining": 1
  }
}
```

Save the TechCorp IDs in environment variables for the curl snippets below:

```bash
BASE=http://localhost:8080
WORKSHOP_ID=seed-event-aiwork-0000-0000-0000-000000000506
INTERN_ID=seed-event-intern-0000-0000-0000-000000000507

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

Expected badges: `["Python","Data Science","Open Source","Cloud Computing","Mobile Development","Cybersecurity Fundamentals"]`

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

Response:

```json
{
  "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in_seconds": 21600
}
```

The `token` is a signed JWT that is valid for **6 hours** — that's the
**scan window**. The frontend renders the JSON `{"token":"<jwt>"}` as a QR
code on the projector screen. Students who scan during this window hold a
token they can sync at any time in the future (see Scene 7).

Save the token so you can use it in the manual sync curl:

```bash
CHECKIN_TOKEN=$(curl -s $BASE/api/events/$WORKSHOP_ID/checkin-code \
  -H "Authorization: Bearer $COMPANY_TOKEN" | jq -r .token)
```

Display the QR on the projector screen.

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

> **Key point to explain:** Amara and Baraka scanned the QR while it was live.
> Their tokens were signed by the server at that moment. Even if they sync
> hours or days later, the server accepts the tokens — it verifies the
> **signature**, not the expiry.

To demo manually with curl:

```bash
# ── Amara syncs her workshop check-in ──────────────────────────────────────
curl -s -X POST $BASE/api/sync/attendance \
  -H "Authorization: Bearer $AMARA_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"records\":[{
    \"local_id\":\"amara-local-001\",
    \"event_id\":\"$WORKSHOP_ID\",
    \"payload\":\"{\\\"token\\\":\\\"$CHECKIN_TOKEN\\\"}\"
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
    \"payload\":\"{\\\"token\\\":\\\"$CHECKIN_TOKEN\\\"}\"
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
# Amara now has 8 badges (6 history + AI Application Development + Prompt Engineering)
curl -s $BASE/api/users/me/skills \
  -H "Authorization: Bearer $AMARA_TOKEN" | jq '[.[] | .skill.name]'

# Baraka has her first badge (AI Application Development)
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

### Scene 11 — Cross-domain candidate search

With 3 companies and 37 skill badges across 5 domains, judges can see the
platform used for real hiring.

```bash
# All students with at least one badge (≥17 results)
curl -s "$BASE/api/users/students" \
  -H "Authorization: Bearer $COMPANY_TOKEN" | jq 'length'

# Students with Python (TechCorp filler + Amara = ≥4)
PYTHON_ID=seed-skill-python--0000-0000-0000-000000000100
curl -s "$BASE/api/users/students?skill_id=$PYTHON_ID" \
  -H "Authorization: Bearer $COMPANY_TOKEN" | jq '[.[] | .name]'

# Students with Soil Science & Health (GreenLeaf filler = ≥2)
SOIL_ID=seed-skill-soilsci-0000-0000-0000-000000000201
curl -s "$BASE/api/users/students?skill_id=$SOIL_ID" \
  -H "Authorization: Bearer $COMPANY_TOKEN" | jq '[.[] | .name]'

# Students with AI in Healthcare (MedConnect filler = ≥2)
HCAI_ID=seed-skill-hcai---0000-0000-0000-000000000304
curl -s "$BASE/api/users/students?skill_id=$HCAI_ID" \
  -H "Authorization: Bearer $COMPANY_TOKEN" | jq '[.[] | .name]'

# AND filter — students with BOTH Python AND AI Application Development
AIDEV_ID=seed-skill-aidev--0000-0000-0000-000000000120
curl -s "$BASE/api/users/students?skill_id=$PYTHON_ID&skill_id=$AIDEV_ID" \
  -H "Authorization: Bearer $COMPANY_TOKEN" | jq '[.[] | .name]'
```

> **What to say:** "The same API powers all three companies. A health-tech
> recruiter finds nurses who know FHIR. An agri-startup finds farmers with
> precision-ag certifications. Badges are earned by showing up — not self-reported."

### Scene 12 — End the workshop early (EOD)

```bash
curl -s -X PATCH $BASE/api/events/$WORKSHOP_ID/status \
  -H "Authorization: Bearer $COMPANY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"completed"}' | jq .
```

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
| Wrong QR token | Use a token signed with a different secret | `rejected: invalid check-in token: ...` |
| Tampered token | Edit any character in the token string | `rejected: invalid check-in token: ...` |
| Token from wrong event | Use `$CHECKIN_TOKEN` with a different `event_id` | `rejected: token event_id does not match record event_id` |
| Missing token | Send `payload: "{}"` | `rejected: payload missing token` |
| Stale QR scanned late | Token exp is in the past — server still accepts | `verified` (sync window is unlimited) |
| Double-sync retry | Call sync twice, same `local_id` | Second call returns `verified` — idempotent |
| Two offline internship applicants | Run Scenes 6 + 7 | First = confirmed, second = conflict_pending |
| Host waitlists applicant | Use `"action":"waitlist"` | Status becomes `waitlisted` |
| Server down during sync | Stop server, scan QR, restart, re-sync | Sync succeeds on reconnect |

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
