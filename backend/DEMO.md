# Skillzone — Local-Network Demo Runbook

Use this guide to run the full end-to-end demo on a local Wi-Fi network
(e.g. a hotspot from your laptop) with no internet required.

---

## Prerequisites

- Go 1.22+ installed on the demo laptop
- All demo devices on the **same Wi-Fi network** (or phone hotspot)
- A modern browser on each device (Chrome/Firefox/Edge)
- The frontend `dist/` already built (or Vite dev server running)

---

## Step 1 — Find your laptop's local IP address

```bash
# Linux / macOS
ip addr show | grep "inet " | grep -v 127.0.0.1
# or
ifconfig | grep "inet " | grep -v 127.0.0.1

# Windows
ipconfig
```

Look for an address like `192.168.x.x` or `10.x.x.x`.
Write it down — you'll need it throughout this runbook.

---

## Step 2 — Build the backend binary

```bash
cd /home/akihara/hackathons/skillzone/backend
go build -o skillzone ./cmd/server/
```

This produces a single binary `skillzone` with SQLite embedded (pure Go,
no shared libraries needed).

---

## Step 3 — Run the server bound to all interfaces

```bash
ADDR=0.0.0.0:8080 \
JWT_SECRET="hackathon-demo-secret" \
./skillzone
```

> **Why `0.0.0.0`?**  
> The default `:8080` only listens on localhost. `0.0.0.0:8080` listens on
> every network interface, including the Wi-Fi adapter, so other devices can
> reach it.

You should see:
```
2024/xx/xx xx:xx:xx Skillzone API listening on 0.0.0.0:8080
```

---

## Step 4 — Point the frontend at the backend

If running the Vite dev server, set the API base URL:

```bash
cd /home/akihara/hackathons/skillzone/frontend
VITE_API_URL=http://<YOUR_LAPTOP_IP>:8080 npm run dev -- --host
```

The `--host` flag makes Vite listen on `0.0.0.0` so other devices can reach
the frontend too.

**Other devices** open: `http://<YOUR_LAPTOP_IP>:5173`

---

## Step 5 — Seed demo data

Run these curl commands from any terminal (substitute your IP):

```bash
BASE=http://localhost:8080

# 1. Create a company account (event host)
curl -s -X POST $BASE/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"host@demo.com","password":"password1","name":"Zone01 Kisumu","role":"company"}' | jq .

# 2. Log in and save the token
COMPANY_TOKEN=$(curl -s -X POST $BASE/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"host@demo.com","password":"password1"}' | jq -r .token)

# 3. Create a skill badge
SKILL_ID=$(curl -s -X POST $BASE/api/skills \
  -H "Authorization: Bearer $COMPANY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Web Development","description":"Built a local-first PWA"}' | jq -r .id)

echo "Skill ID: $SKILL_ID"

# 4. Create an event (adjust times as needed)
EVENT_ID=$(curl -s -X POST $BASE/api/events \
  -H "Authorization: Bearer $COMPANY_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"title\": \"Hackathon Check-In\",
    \"description\": \"Zone01 LOCAL FIRST hackathon\",
    \"location\": \"Zone01 Kisumu\",
    \"start_time\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"end_time\":   \"$(date -u -d '+8 hours' +%Y-%m-%dT%H:%M:%SZ)\",
    \"skill_ids\": [\"$SKILL_ID\"]
  }" | jq -r .id)

echo "Event ID: $EVENT_ID"

# 5. Get the check-in code (this goes into the QR code)
curl -s $BASE/api/events/$EVENT_ID/checkin-code \
  -H "Authorization: Bearer $COMPANY_TOKEN" | jq .
```

---

## Step 6 — Student registration

On a **second device** (phone or another laptop), open:
`http://<YOUR_LAPTOP_IP>:5173`

1. Register as a student.
2. Browse events and tap **Register** on the hackathon event.

---

## Step 7 — Demo the offline check-in flow

This is the key local-first demo:

### 7a. Host gets the QR code

On the host device (still online):
- Log in as `host@demo.com`
- Open the event → tap **Show Check-In QR**
- The frontend calls `/api/events/{id}/checkin-code` and renders a QR

### 7b. Turn the student device offline

- On the student's phone: **turn off Wi-Fi AND mobile data** (Airplane Mode)
- The PWA should still load from the service worker cache

### 7c. Student scans the QR

- Student opens the PWA → **Scan Check-In**
- Scans the host's QR code
- The PWA stores the payload in IndexedDB with status `PENDING`
- UI shows: *"Checked in ✓ — badge will be awarded when you reconnect"*

### 7d. Reconnect and sync

- Turn Wi-Fi back on
- The service worker background sync fires automatically
  (or the student taps **Sync Now**)
- The PWA sends the pending record to `POST /api/sync/attendance`
- Server verifies the signature → awards the **Web Development** badge
- PWA updates IndexedDB record to `VERIFIED` and shows the badge 🎉

---

## Step 8 — Show the judge what happened

```bash
# List the student's earned badges
STUDENT_TOKEN=$(curl -s -X POST $BASE/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"student@demo.com","password":"password1"}' | jq -r .token)

curl -s $BASE/api/users/me/skills \
  -H "Authorization: Bearer $STUDENT_TOKEN" | jq .
```

---

## Simulating a slow / unreliable network (Chrome DevTools)

1. Open Chrome DevTools on the student's device → **Network** tab
2. Set throttling to **Slow 3G** or **Offline**
3. Show that the PWA still loads and check-in still works
4. Switch back to online → show the sync completing

To show the service worker:
- DevTools → **Application** → **Service Workers**
- You can see the background sync queue under **Background Services → Background Sync**

---

## Failure scenarios to demonstrate

| Scenario | How to trigger | Expected result |
|---|---|---|
| Wrong QR code | Edit the payload JSON before syncing | `rejected: invalid check-in signature` |
| Stale QR (>24 h) | Manually set `timestamp` to yesterday | `rejected: check-in payload has expired` |
| Double-sync (retry) | Call sync twice with same `local_id` | Second call returns `verified` silently (idempotent) |
| Server down during sync | Stop the server, scan QR, restart server, sync | Sync succeeds on reconnect |
| No internet at venue | Turn off Wi-Fi entirely, scan QR | Check-in stored locally; badge awarded on reconnect |

---

## Troubleshooting

**Other devices can't reach the server**
- Confirm you used `ADDR=0.0.0.0:8080` not the default `:8080`
- Check your laptop's firewall: `sudo ufw allow 8080`

**CORS errors in the browser**
- The backend sends `Access-Control-Allow-Origin: *` for all requests
- If you see CORS errors, ensure the frontend is using the correct IP (not `localhost`)

**`date -d` not available (macOS)**
- Use `date -u -v+8H +%Y-%m-%dT%H:%M:%SZ` instead of `date -u -d '+8 hours'`

**Port 8080 already in use**
```bash
lsof -i :8080
# Change the port:
ADDR=0.0.0.0:9090 ./skillzone
```
