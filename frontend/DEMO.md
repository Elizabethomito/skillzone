# Skillzone — Frontend Demo Script

> **Duration:** ~15 minutes end-to-end  
> **Audience:** Hackathon judges, potential users  
> **Setup required:** Backend running on `http://localhost:8080`, browser on Chrome/Chromium

---

## Prerequisites

### 1 — Start the backend

```bash
cd backend
go run ./cmd/server
# Server listening on :8080
```

### 2 — Start the frontend (dev server)

```bash
cd frontend
npm run dev
# → http://localhost:5173
```

> For a full PWA experience (service worker active in dev), the `vite.config.ts`
> already sets `devOptions: { enabled: true }`. The SW is registered automatically.

### 3 — Seed demo data

```bash
curl -X POST http://localhost:8080/api/admin/seed
```

Expected response:

```json
{
  "seeded": true,
  "active_workshop": {
    "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
    "title": "Building Apps with AI Workshop"
  },
  "internship": {
    "event_id": "seed-event-intern-0000-0000-0000-000000000031",
    "title": "AI Product Internship",
    "slots_remaining": 1
  }
}
```

**Demo accounts (all password `demo1234`):**

| Account | Email | Role |
|---------|-------|------|
| TechCorp Africa | `host@techcorp.test` | Company (host) |
| Amara Osei | `amara@student.test` | Student (pre-registered + history) |
| Baraka Mwangi | `baraka@student.test` | Student (fresh account) |

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
2. Show the full event list — 6 completed past events, 1 active workshop, 1 upcoming internship.
3. Highlight the **AI Workshop** card (status badge: `active`).

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

#### 6a — Open a second browser window as Amara

1. Open a new **Incognito window** (or a different browser profile).
2. Navigate to `http://localhost:5173`.
3. Sign In as `amara@student.test` / `demo1234`.
4. Go to **Events**.

#### 6b — Simulate going offline

1. Open **Chrome DevTools** (`F12`) → **Network** tab.
2. Change the throttle dropdown from "No throttle" to **Offline**.
3. The amber **"You're offline"** banner appears at the top of the page.

> **What to say:** "The app detects the network loss and surfaces it immediately.
> The events are still visible from the Dexie IndexedDB cache."

#### 6c — Scan the QR code offline

1. On Amara's browser (offline), click the **Scan QR** button on the AI Workshop card.
2. The QR scanner modal opens using the device camera.
3. Point the camera at the host's QR code displayed in the other window.
4. The scanner decodes the JWT, extracts the `event_id`, and stores the record in IndexedDB.
5. A toast notification appears: **"Check-in queued — will sync when online."**

> **What to say:** "The check-in is captured and stored locally. The student is
> done — they can put their phone away. No network needed."

#### 6d — Come back online and watch the sync fire

1. In Chrome DevTools → Network tab → change back to **No throttle** (online).
2. Within 1–2 seconds, a toast appears: **"✓ Sync complete — 1 check-in verified."**

> **What to say:** "The moment connectivity returns, the background sync engine
> drains the queue automatically. The server verifies the JWT signature and
> awards the skill badge."

#### 6e — Verify the badge in the Dashboard

1. On Amara's session, navigate to **Dashboard**.
2. Under **Skill Badges**, the newly awarded badges ("AI Application Development",
   "Prompt Engineering") appear.

---

### Act 7 — The Slot Conflict (Internship Registration)

> This demo requires **two student sessions** and the AI Product Internship
> (capacity: 2, currently 1 slot remaining → 1 slot left after Amara's earlier
> confirmed registration).

#### 7a — Both students register offline simultaneously

1. Put **both** student sessions offline (DevTools → Offline on each tab).
2. On Amara's tab: click **Register** on the **AI Product Internship** card.
   - Toast: "Registration queued offline."
3. On Baraka's tab: click **Register** on the same card.
   - Toast: "Registration queued offline."

#### 7b — Bring Amara online first

1. On Amara's DevTools → set back to No throttle.
2. Sync fires — Amara gets the last slot (`confirmed`).

#### 7c — Bring Baraka online

1. On Baraka's DevTools → set back to No throttle.
2. Baraka's sync fires — the slot is gone → registration lands as `conflict_pending`.
3. A toast appears: "1 registration needs host review."

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
2. Show the grid of 9 skill badges.
3. Type "AI" in the search bar — the grid filters live to 3 results.
4. As the company account, each skill card shows **"Find candidates →"** link.

---

### Act 9 — Candidate Search (Company Feature)

1. While signed in as `host@techcorp.test`, click **Candidates** in the nav.
2. The page loads all students with at least one badge.
3. Click the skill picker, type "AI Application" and select it.
4. The results narrow to students who hold that badge.
5. Add a second filter: "Prompt Engineering" — results show students with BOTH badges.
6. Hover over Amara's card — it shows her full badge list.

> **What to say:** "Companies can instantly find students with any combination
> of verified skills — no resumes, no guesswork. Every badge is backed by a
> real attendance record on our server."

---

### Act 10 — Profile Page

1. Click **Profile** in the nav (as Amara).
2. Show the stat cards: badge count + completed events count.
3. Show the full badge list at the bottom.
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
