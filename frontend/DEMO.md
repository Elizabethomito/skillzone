# Skillzone — Frontend Demo Script

> **Duration:** ~15 minutes end-to-end  
> **Audience:** Hackathon judges, potential users  
> **People needed:** 1 presenter (server laptop) + 2 helpers#### 6b — Simulate going offline (both helpers)

1. On **each helper's device**, open **Chrome DevTools** (`F12`) → **Network** tab.
2. Change the throttle dropdown from "No throttle" to **Offline**.
3. The amber **"You're offline"** banner appears at the top of the page.

> **What to say:** "The app detects the network loss and surfaces it immediately.
> Events are still visible from the Dexie IndexedDB cache — no server call needed."

#### 6c — Scan the QR code offline (both helpers)Baraka)

---

## Who does what

| Person | Device | Role |
|--------|--------|------|
| **Presenter** | Server laptop | Runs backend + Vite, drives the host dashboard (TechCorp Africa), narrates |
| **Helper A — Amara** | Phone or second laptop | Signs in as `amara@student.test` — veteran student, 6 badges |
| **Helper B — Baraka** | Phone or second laptop | Signs in as `baraka@student.test` — newcomer, zero history |

> **Solo mode:** Open the host session in a normal Chrome window and open two
> Incognito windows for Amara and Baraka. Incognito windows have isolated
> storage so the JWTs don't bleed across sessions.

---

## Prerequisites

### 1 — Start the backend (server laptop)

```bash
cd backend
ADDR=0.0.0.0:8080 JWT_SECRET=hackathon-demo go run ./cmd/server
# Listening on 0.0.0.0:8080
```

> `0.0.0.0` makes the API reachable from every device on the same Wi-Fi.
> Find the server's local IP with `ip addr show | grep "inet "` (Linux/macOS)
> or `ipconfig` (Windows). Share `http://<SERVER_IP>:8080` with your helpers
> if you want them to hit the raw API — usually the frontend URL below is enough.

### 2 — Start the frontend (server laptop)

```bash
cd frontend
npm run dev -- --host
# → http://<SERVER_IP>:5173  (accessible from all devices on the network)
```

No `VITE_API_URL` needed — the app derives the backend address from the
browser's own hostname at runtime (`window.location.hostname + :8080`), so
every device that opens the page automatically talks to the right server.

**Helper A and Helper B** open `http://<SERVER_IP>:5173` on their devices.
The sign-in page has **click-to-fill buttons** — helpers tap their name,
then tap **Sign In**. No typing needed.

> For a full PWA experience (service worker active in dev), the `vite.config.ts`
> already sets `devOptions: { enabled: true }`. Chrome will show an **Install**
> prompt (⊕ in the address bar) — accept it for the standalone app experience.

### 3 — Seed demo data (server laptop, one time)

```bash
curl -X POST http://localhost:8080/api/admin/seed
```

Expected response (abbreviated):

```json
{
  "seeded": true,
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

**Demo accounts (all password `demo1234`):**

| Account | Email | Role |
|---------|-------|------|
| TechCorp Africa | `host@techcorp.test` | Company (host) |
| Amara Osei | `amara@student.test` | Student (pre-registered + 6 skill history) |
| Baraka Mwangi | `baraka@student.test` | Student (fresh account) |

**Filler students (15 total, not loginable):**
TechCorp: Chidi, Fatima, Kwame, Aisha, Tobi, Ngozi, Joel, Lila ·
GreenLeaf: Zara, Emeka, Sade, Kofi ·
MedConnect: Muna, Dayo, Nia

---

## Demo Flow

### Act 1 — The PWA is Installable

1. Open `http://localhost:5173` in Chrome.
2. Look for the **install icon** (⊕) in the address bar — click it.
3. The app installs to the desktop as a standalone PWA.
4. Open it from the desktop shortcut — notice it opens without browser chrome.

> **What to say:** "Skillzone ships as a Progressive Web App — no app store, no
> download, install directly from the browser. Works on any device."

---

### Act 2 — Sign In as the Event Host

1. Navigate to `http://localhost:5173` (or click the app from the desktop).
2. Click **Sign In**.
3. Enter `host@techcorp.test` / `demo1234` → **Sign In**.
4. You are redirected to `/dashboard`.

**What to show on the Dashboard:**

- The host's name "TechCorp Africa" and "Company" badge in the header.
- Active/Upcoming event counts from the API.
- Two quick-action buttons: **Manage Events** and **Search Candidates**.

---

### Act 3 — Events Page (Host View)

1. Click **Events** in the navigation.
2. Show the full event list — 18 events across 3 companies:
   - **TechCorp Africa**: 6 completed, 1 active (AI Workshop), 1 upcoming (AI Product Internship)
   - **GreenLeaf Agri**: 3 completed, 1 active (Agro-processing Workshop), 1 upcoming (Sustainable Agriculture Internship)
   - **MedConnect Health**: 3 completed, 1 active (AI in Healthcare), 1 upcoming (Digital Health Innovation Fellowship)
3. As the TechCorp host, only TechCorp's events show host controls. Highlight the **Building Apps with AI Workshop** card (status badge: `active`).

**Host controls on the AI Workshop card:**

| Button | What it does |
|--------|-------------|
| **Edit** | Opens `CreateEditEventModal` pre-filled — change the title, then save |
| **QR Code** | Fetches a signed token from the server, renders it as a canvas QR |
| **Guests** | Opens `ManageGuestsModal` — shows Amara's pre-confirmed registration |

> **What to say:** "Hosts get full CRUD over their events, live guest management,
> and one-click QR generation — all in the same UI."

---

### Act 4 — Edit an Event Live

1. On the AI Workshop card, click **Edit** (pencil icon).
2. The `CreateEditEventModal` opens pre-filled with the current values.
3. Change **Title** to "Building Apps with AI Workshop — LIVE DEMO".
4. Click **Save Changes**.
5. The card title updates instantly.

---

### Act 5 — Display the Host QR Code

1. On the AI Workshop card, click **QR** (QR icon).
2. A modal appears with a canvas QR code.

> **What to say:** "The QR encodes a signed JWT issued by our server. It's
> valid for 6 hours — enough to cover the entire event. Students scan it on
> their phones."

Leave this modal open on the host's screen (or a second monitor).

---

### Act 6 — Student Checks In Offline (The Core Demo)

#### 6a — Helper A (Amara) and Helper B (Baraka) open the app

Both helpers should already be signed in from the setup step.  If not:

1. Open `http://<SERVER_IP>:5173` on each device.
2. Tap the **Amara Osei** or **Baraka Mwangi** quick-fill button on the sign-in page.
3. Tap **Sign In**.

Have both helpers navigate to **Events** — they can both see the
"Building Apps with AI Workshop" card.

#### 6b — Simulate going offline (both helpers)

1. Open **Chrome DevTools** (`F12`) → **Network** tab.
2. Change the throttle dropdown from "No throttle" to **Offline**.
3. The amber **"You're offline"** banner appears at the top of the page.

> **What to say:** "The app detects the network loss and surfaces it immediately.
> The events are still visible from the Dexie IndexedDB cache."

#### 6c — Scan the QR code offline (both helpers)

1. On **Amara's device** (offline), click **Scan QR** on the AI Workshop card.
2. Point the camera at the host's QR code on the projector/laptop screen.
3. Toast: **"Check-in queued — will sync when online."**
4. Repeat on **Baraka's device** — same QR, same result.

> **What to say:** "Both students captured their attendance locally.
> Baraka was NOT pre-registered — the server will auto-register them
> when the sync fires. No network needed to check in."

#### 6d — Come back online and watch the sync fire (both helpers)

1. On **each helper's** DevTools → Network tab → change back to **No throttle**.
2. Within 1–2 seconds, a toast appears: **"✓ Sync complete — 1 check-in verified."**

> **What to say:** "The moment connectivity returns, background sync drains
> the queue automatically. The server verifies the JWT signature and awards
> the skill badge — on both devices simultaneously."

#### 6e — Verify the badges

1. **Amara's device** → navigate to **Dashboard**.
   Under **Skill Badges**: "AI Application Development" and "Prompt Engineering" appear.
   She now has **8 badges** total.
2. **Baraka's device** → navigate to **Dashboard**.
   Under **Skill Badges**: "AI Application Development" — his **first ever badge**.

> **What to say:** "This is the contrast we wanted to show. Amara's rich
> history was already there. Baraka walked in with nothing — one QR scan
> and one sync later, he has his first verified credential."

---

### Act 7 — The Slot Conflict (Internship Registration)

> This demo requires **two student sessions** and the AI Product Internship
> (capacity: 2, 1 slot already confirmed for Amara → 1 slot remaining).

#### 7a — Both helpers register offline simultaneously

1. Put **both helper devices** offline (DevTools → Offline on each).
2. **Amara's device**: click **Register** on the **AI Product Internship** card.
   - Toast: "Registration queued offline."
3. **Baraka's device**: click **Register** on the same card.
   - Toast: "Registration queued offline."

#### 7b — Bring Amara online first

1. On **Amara's** DevTools → set back to No throttle.
2. Sync fires — Amara gets the last slot (`confirmed`).

#### 7c — Bring Baraka online

1. On **Baraka's** DevTools → set back to No throttle.
2. Baraka's sync fires — the slot is gone → `conflict_pending`.
3. Toast: "1 registration needs host review."

#### 7d — Host resolves the conflict

1. Switch to the **Host session**.
2. On the AI Product Internship card, click **Guests**.
3. The `ManageGuestsModal` shows both registrations:
   - Amara: `confirmed` (green badge)
   - Baraka: `conflict_pending` (amber badge)
4. Click **Confirm** or **Waitlist** next to Baraka's row.
5. The status updates live.

> **What to say:** "The system handles concurrent offline registrations gracefully.
> First sync wins the slot. The host gets a conflict dashboard to resolve the
> rest fairly."

---

### Act 8 — Skills Catalogue

1. Click **Skills** in the navigation.
2. Show the grid of **37 skill badges** across 5 domains:
   - **Tech — Programming** (6): Python, JavaScript, TypeScript, Go, Rust, SQL
   - **Tech — Data & AI** (8): Data Science, MLOps, Data Engineering, AI Application Development, Prompt Engineering, AI Product Management, Computer Vision, NLP
   - **Tech — Infrastructure** (5): Cloud Computing, DevOps, Docker & Containers, Cybersecurity Fundamentals, Open Source
   - **Tech — Frontend** (4): Mobile Development, React & Web Apps, UI/UX Design, API Design
   - **Agriculture** (6): Precision Agriculture, Soil Science & Health, Climate-Smart Farming, Agro-processing, Food Safety, Water Management
   - **Healthcare** (5): Health Data Management, Public Health & Epidemiology, Medical Technology, Clinical Research, AI in Healthcare
   - **Business** (3): Entrepreneurship & Innovation, Project Management, System Design
3. Type "AI" in the search bar — the grid live-filters to 5 results.
4. Type "Health" — filters to 3 healthcare badges.
5. As the company account, each skill card shows **"Find candidates →"** link.

---

### Act 9 — Candidate Search (Company Feature)

1. While signed in as `host@techcorp.test`, click **Candidates** in the nav.
2. The page loads **≥17 students** with at least one verified badge.
3. Click the skill picker, type "AI Application" and select it.
4. The results narrow to students who hold that badge (Amara + several TechCorp filler students).
5. Add a second filter: "Prompt Engineering" — results show students with BOTH badges.
6. Clear filters, type "Soil Science" — shows GreenLeaf filler students.
7. Type "AI in Healthcare" — shows MedConnect filler students.
8. Hover over Amara's card — it shows her full badge list.

> **What to say:** "The search works across all three companies and all 37 skill
> domains — tech, agriculture, and healthcare. Every badge is backed by a real
> attendance record. No self-reported skills, no resumes."

---

### Act 10 — Profile Page

1. Click **Profile** in the nav (as Amara).
2. Show the stat cards: **8 badges** (after the AI workshop sync) + 7 completed events.
3. Show the full badge list at the bottom — 6 history badges + 2 from today's workshop.
4. Demonstrate editing the display name and clicking Save.

---

## Offline Architecture Explained (1-slide summary)

```
┌──────────────────────────────────────────────────────────────────┐
│                         PWA (Vite + React)                       │
│                                                                  │
│  Events page  ──reads──►  Dexie.js (IndexedDB)  ◄──writes──     │
│  QR Scanner   ──stores──► sync_queue table        Sync Engine    │
│  Register btn ──stores──► sync_queue table        (lib/sync.ts)  │
│                                                        │         │
│                                           online event │         │
│                                                        ▼         │
│                                        POST /api/sync/attendance │
│                                        POST /api/events/:id/register │
│                                        DELETE /api/events/:id/register │
└──────────────────────────────────────────────────────────────────┘
         ▲                                         │
         │ JWT in Authorization header             │ 200 OK / results
         │                                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Go Backend (SQLite)                          │
│                                                                  │
│  POST /api/sync/attendance  — verify JWT, award badges          │
│  PUT  /api/events/:id       — partial patch with capacity guard │
│  DELETE /api/events/:id/register   — unregister student         │
│  DELETE /api/events/:id/registrations/:reg_id  — kick guest     │
│  GET  /api/users/students?skill_id=  — candidate search         │
└─────────────────────────────────────────────────────────────────┘
```

### Key design decisions

| Decision | Rationale |
|----------|-----------|
| Dexie.js (IndexedDB) instead of sql.js | Native browser DB, no WASM, async, works in service worker |
| JWT signed token in QR | No network call at scan time; server verifies signature at sync |
| `user_id` tag on sync_queue rows | Multi-user device safety — Baraka's queued items never mix with Amara's |
| NetworkFirst caching for `/api/*` | Fresh data when online; cached fallback when offline |
| `generateSW` mode (Workbox) | Zero-config service worker; precaches all built assets automatically |

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| QR scanner modal shows blank | Browser camera permission not granted — click Allow |
| "Sync complete: 0 verified" | Backend not running — start `go run ./cmd/server` |
| Events page shows empty | Hit `POST /api/admin/seed` first |
| JWT decode error in scanner | QR was generated from a different backend instance — re-seed |
| Build fails on `vite-plugin-pwa` | Run `npm install` in `frontend/` |
| Service worker not updating | Chrome DevTools → Application → Service Workers → click "Skip waiting" |
