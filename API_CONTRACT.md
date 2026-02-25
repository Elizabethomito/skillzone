# Skillzone API Contract

> **For the frontend team.** This document is the single source of truth for
> every HTTP endpoint exposed by the Skillzone backend. Use it to build the
> Vite/React PWA without waiting for a deployed server.

---

## Table of Contents

1. [Connection & Authentication](#1-connection--authentication)
2. [Core Data Models (TypeScript)](#2-core-data-models-typescript)
3. [Auth Endpoints](#3-auth-endpoints)
4. [Event Endpoints](#4-event-endpoints)
5. [Skill Endpoints](#5-skill-endpoints)
6. [Student Endpoints](#6-student-endpoints)
7. [Offline Sync — Deep Dive](#7-offline-sync--deep-dive)
8. [Admin Endpoints](#8-admin-endpoints)
9. [Error Format](#9-error-format)
10. [HTTP Status Code Reference](#10-http-status-code-reference)

---

## 1. Connection & Authentication

### Base URL

| Environment | Base URL |
|-------------|----------|
| Local dev   | `http://localhost:8080` |
| LAN demo    | `http://<laptop-ip>:8080` |

All endpoint paths below are relative to the base URL (e.g. `POST /api/auth/register` → `http://localhost:8080/api/auth/register`).

### Authentication

All protected endpoints require a **Bearer JWT** in the `Authorization` header:

```
Authorization: Bearer <token>
```

Tokens are obtained from `POST /api/auth/register` or `POST /api/auth/login`.
They are **HS256-signed**, **72 hours** valid, and carry two claims used by the
server for access control:

| JWT Claim | Type | Example |
|-----------|------|---------|
| `sub` | `string` | `"a1b2c3d4-..."` — the user's UUID |
| `role` | `string` | `"student"` or `"company"` |

Endpoints marked **Auth: Yes (student)** reject requests from company tokens
with `403 Forbidden`, and vice-versa.

---

## 2. Core Data Models (TypeScript)

These interfaces map 1-to-1 to the Go structs in `internal/models/models.go`.
Copy them into a `src/types/api.ts` file in the frontend.

```typescript
// ─── Enumerations ────────────────────────────────────────────────────────────

export type UserRole = "student" | "company";

export type EventStatus = "upcoming" | "active" | "completed";

/** Server-side attendance state for a single check-in record. */
export type AttendanceStatus = "pending" | "verified" | "rejected";

/**
 * Slot-allocation outcome for a registration.
 * A student gets conflict_pending when they register (online or via QR sync)
 * and no slots remain. The host resolves it via the dashboard.
 */
export type RegistrationStatus =
  | "confirmed"
  | "conflict_pending"
  | "waitlisted";

// ─── Domain Models ───────────────────────────────────────────────────────────

export interface User {
  id: string;           // UUID v4
  email: string;
  name: string;
  role: UserRole;
  created_at: string;   // ISO 8601 — use new Date(user.created_at)
  updated_at: string;
  // password_hash is NEVER present in any API response
}

export interface Skill {
  id: string;           // UUID v4
  name: string;
  description: string;
  created_at: string;
}

/**
 * capacity and slots_remaining are absent (undefined) when the event has
 * no slot limit. A value of 0 for slots_remaining means the event is full.
 */
export interface Event {
  id: string;
  host_id: string;
  title: string;
  description: string;
  location: string;
  start_time: string;        // ISO 8601
  end_time: string;
  status: EventStatus;
  check_in_code?: string;    // only present in GET /api/events/{id}/checkin-code
  capacity?: number;         // absent = unlimited
  slots_remaining?: number;  // absent = unlimited; 0 = full
  created_at: string;
  updated_at: string;
  skills?: Skill[];          // linked badge definitions; omitted if none
}

export interface Registration {
  id: string;
  event_id: string;
  student_id: string;
  registered_at: string;    // ISO 8601
  status: RegistrationStatus;
}

export interface Attendance {
  id: string;
  event_id: string;
  student_id: string;
  payload: string;           // raw QR JSON string (stored verbatim)
  status: AttendanceStatus;
  created_at: string;
  updated_at: string;
}

/**
 * A skill badge awarded to a student.
 * skill is populated when reading GET /api/users/me/skills.
 */
export interface UserSkill {
  id: string;
  user_id: string;
  skill_id: string;
  event_id: string;          // which event triggered the award
  awarded_at: string;
  skill?: Skill;
}

// ─── Request / Response DTOs ─────────────────────────────────────────────────

export interface RegisterRequest {
  email: string;
  password: string;          // minimum 8 characters
  name: string;
  role: UserRole;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface CreateEventRequest {
  title: string;
  description: string;
  location: string;
  start_time: string;        // ISO 8601
  end_time: string;
  skill_ids: string[];       // existing Skill UUIDs to attach as badges
  capacity?: number;         // omit or 0 for unlimited
}

export interface UpdateEventStatusRequest {
  status: EventStatus;
}

export interface ResolveConflictRequest {
  action: "confirm" | "waitlist";
}

/**
 * One pending check-in record from IndexedDB, ready to sync.
 * payload is the raw JSON string captured from the host's QR code.
 */
export interface AttendanceSyncRecord {
  local_id: string;          // UUID generated by the client (used to match results)
  event_id: string;
  payload: string;           // stringified CheckInPayload — see Section 7
}

export interface SyncAttendanceRequest {
  records: AttendanceSyncRecord[];
}

export interface SyncResult {
  local_id: string;
  status: AttendanceStatus;
  message?: string;          // human-readable rejection reason, absent on success
}

export interface SyncAttendanceResponse {
  results: SyncResult[];
}

/**
 * The JSON structure embedded in the host's QR code.
 * The PWA stores this as a JSON string inside AttendanceSyncRecord.payload.
 */
export interface CheckInPayload {
  event_id: string;
  host_sig: string;          // the event's check_in_code — proves physical presence
  timestamp: number;         // Unix seconds (UTC) — rejected if older than 24 h
}

// ─── Extended response shapes (from JOIN queries) ────────────────────────────

/** Returned by GET /api/events/{id}/registrations (host dashboard). */
export interface RegistrationWithStudent extends Registration {
  student_name: string;
  student_email: string;
}

/** Returned by GET /api/users/me/registrations (student dashboard). */
export interface RegistrationWithEvent extends Registration {
  event_title: string;
  start_time: string;
  end_time: string;
  event_status: EventStatus;
  location: string;
}
```

---

## 3. Auth Endpoints

### `POST /api/auth/register`

Create a new user account and receive a JWT immediately.

- **Auth required:** No
- **Request body:** `RegisterRequest`

```json
{
  "email": "amara@student.test",
  "password": "demo1234",
  "name": "Amara Osei",
  "role": "student"
}
```

- **Success:** `201 Created` → `LoginResponse`

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "a1b2c3d4-e5f6-...",
    "email": "amara@student.test",
    "name": "Amara Osei",
    "role": "student",
    "created_at": "2026-02-25T10:00:00Z",
    "updated_at": "2026-02-25T10:00:00Z"
  }
}
```

| Status | Meaning |
|--------|---------|
| `400 Bad Request` | Missing field, password < 8 chars, or invalid role |
| `409 Conflict` | Email already registered |

---

### `POST /api/auth/login`

Authenticate an existing user and receive a fresh JWT.

- **Auth required:** No
- **Request body:** `LoginRequest`

```json
{
  "email": "amara@student.test",
  "password": "demo1234"
}
```

- **Success:** `200 OK` → `LoginResponse` (same shape as register)

| Status | Meaning |
|--------|---------|
| `400 Bad Request` | Malformed JSON |
| `401 Unauthorized` | Email not found or wrong password |

> **Note:** The server deliberately returns `401` (not `404`) for unknown emails
> to avoid leaking whether an address is registered.

---

### `GET /api/auth/me`

Return the currently authenticated user's profile.

- **Auth required:** Yes (any role)
- **Request body:** None

- **Success:** `200 OK` → `User`

```json
{
  "id": "a1b2c3d4-e5f6-...",
  "email": "amara@student.test",
  "name": "Amara Osei",
  "role": "student",
  "created_at": "2026-02-25T10:00:00Z",
  "updated_at": "2026-02-25T10:00:00Z"
}
```

| Status | Meaning |
|--------|---------|
| `401 Unauthorized` | Missing or invalid token |
| `404 Not Found` | User deleted after token was issued |

---

## 4. Event Endpoints

### `GET /api/events`

List all events in chronological order, each with their linked skill badges.

- **Auth required:** No
- **Request body:** None

- **Success:** `200 OK` → `Event[]`

```json
[
  {
    "id": "seed-event-aiwork-0000-0000-0000-000000000030",
    "host_id": "seed-user-company-000-0000-0000-000000000001",
    "title": "Building Apps with AI Workshop",
    "description": "...",
    "location": "TechCorp HQ — Room 3B",
    "start_time": "2026-02-25T09:00:00Z",
    "end_time": "2026-02-25T17:00:00Z",
    "status": "active",
    "created_at": "2026-02-01T00:00:00Z",
    "updated_at": "2026-02-25T09:00:00Z",
    "skills": [
      { "id": "...", "name": "AI Application Development", "description": "...", "created_at": "..." }
    ]
  }
]
```

> `capacity` and `slots_remaining` are omitted from events that have no limit.

---

### `GET /api/events/{id}`

Fetch a single event by UUID.

- **Auth required:** No
- **Path parameter:** `id` — event UUID

- **Success:** `200 OK` → `Event` (same shape as one item from the list)

| Status | Meaning |
|--------|---------|
| `404 Not Found` | No event with that UUID |

---

### `POST /api/events`

Create a new event. A `check_in_code` UUID is generated automatically by the server.

- **Auth required:** Yes (company)
- **Request body:** `CreateEventRequest`

```json
{
  "title": "AI Product Internship",
  "description": "Hands-on AI product management programme.",
  "location": "TechCorp HQ — Floor 4",
  "start_time": "2026-03-20T09:00:00Z",
  "end_time": "2026-03-21T17:00:00Z",
  "skill_ids": ["<skill-uuid-1>", "<skill-uuid-2>"],
  "capacity": 2
}
```

- **Success:** `201 Created` → `Event` (includes the generated `check_in_code`)

| Status | Meaning |
|--------|---------|
| `400 Bad Request` | Missing title, missing/invalid times, end before start |
| `401 Unauthorized` | Missing or invalid token |
| `403 Forbidden` | Token belongs to a student account |

---

### `GET /api/events/{id}/checkin-code`

Retrieve the event's check-in code to render as a QR code on the projector.
Only the host of the event can call this.

- **Auth required:** Yes (company — must be the event host)
- **Path parameter:** `id` — event UUID

- **Success:** `200 OK`

```json
{
  "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
  "check_in_code": "DEMO-CHECKIN-CODE-AI-WORKSHOP-2026"
}
```

Use these two values to build the `CheckInPayload` JSON that gets encoded
into the QR code (see [Section 7](#7-offline-sync--deep-dive)).

| Status | Meaning |
|--------|---------|
| `401 Unauthorized` | Missing or invalid token |
| `403 Forbidden` | Token is not the host of this event |
| `404 Not Found` | No event with that UUID |

---

### `PATCH /api/events/{id}/status`

Transition an event through its lifecycle. Only the event's host can call this.

- **Auth required:** Yes (company — must be the event host)
- **Path parameter:** `id` — event UUID
- **Request body:** `UpdateEventStatusRequest`

```json
{ "status": "active" }
```

Valid transitions (the server does not enforce ordering — any value is accepted):

| Value | Meaning |
|-------|---------|
| `"upcoming"` | Not yet started; check-in QR not live |
| `"active"` | Event is live; students can scan the QR |
| `"completed"` | Event is over; attendance sync still accepted for 24 h |

- **Success:** `200 OK`

```json
{
  "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
  "status": "active"
}
```

| Status | Meaning |
|--------|---------|
| `400 Bad Request` | `status` is not one of the three valid values |
| `403 Forbidden` | Token is not the host of this event |
| `404 Not Found` | No event with that UUID |

---

### `GET /api/events/{id}/registrations`

Return the full attendee list for an event, including each student's name,
email, and registration status. The host uses this to spot `conflict_pending`
entries that need resolution.

- **Auth required:** Yes (company — must be the event host)
- **Path parameter:** `id` — event UUID

- **Success:** `200 OK` → `RegistrationWithStudent[]`

```json
[
  {
    "id": "reg-uuid-1",
    "event_id": "seed-event-intern-0000-0000-0000-000000000031",
    "student_id": "seed-user-amara-000-0000-0000-000000000002",
    "registered_at": "2026-02-25T11:00:00Z",
    "status": "confirmed",
    "student_name": "Amara Osei",
    "student_email": "amara@student.test"
  },
  {
    "id": "reg-uuid-2",
    "event_id": "seed-event-intern-0000-0000-0000-000000000031",
    "student_id": "seed-user-baraka-00-0000-0000-000000000003",
    "registered_at": "2026-02-25T11:01:00Z",
    "status": "conflict_pending",
    "student_name": "Baraka Mwangi",
    "student_email": "baraka@student.test"
  }
]
```

| Status | Meaning |
|--------|---------|
| `403 Forbidden` | Token is not the host of this event |
| `404 Not Found` | No event with that UUID |

---

### `PATCH /api/events/{id}/registrations/{reg_id}`

Resolve a `conflict_pending` registration. The host decides whether the student
gets the slot or is placed on the waitlist.

- **Auth required:** Yes (company — must be the event host)
- **Path parameters:**
  - `id` — event UUID
  - `reg_id` — registration UUID (obtained from the list above)
- **Request body:** `ResolveConflictRequest`

```json
{ "action": "confirm" }
```

| `action` | Resulting `status` |
|----------|--------------------|
| `"confirm"` | `"confirmed"` |
| `"waitlist"` | `"waitlisted"` |

- **Success:** `200 OK`

```json
{
  "registration_id": "reg-uuid-2",
  "status": "confirmed"
}
```

| Status | Meaning |
|--------|---------|
| `400 Bad Request` | `action` is not `"confirm"` or `"waitlist"` |
| `403 Forbidden` | Token is not the host of this event |
| `404 Not Found` | Event or registration not found |
| `409 Conflict` | Registration is not in `conflict_pending` state |

---

### `POST /api/events/{id}/register`

Register the authenticated student for an event.

- **Auth required:** Yes (student)
- **Path parameter:** `id` — event UUID
- **Request body:** None

- **Success:** `201 Created` → `Registration`

```json
{
  "id": "reg-uuid-new",
  "event_id": "seed-event-intern-0000-0000-0000-000000000031",
  "student_id": "seed-user-amara-000-0000-0000-000000000002",
  "registered_at": "2026-02-25T11:00:00Z",
  "status": "confirmed"
}
```

> If the event has a capacity limit and `slots_remaining == 0`, the returned
> `status` will be `"conflict_pending"` instead of `"confirmed"`.
> Calling this endpoint twice for the same student + event is **idempotent** —
> the second call returns the same registration without creating a duplicate.

| Status | Meaning |
|--------|---------|
| `401 Unauthorized` | Missing or invalid token |
| `403 Forbidden` | Token belongs to a company account |
| `404 Not Found` | No event with that UUID |

---

## 5. Skill Endpoints

### `GET /api/skills`

Return all skill badges defined on the platform, sorted alphabetically.

- **Auth required:** No
- **Request body:** None

- **Success:** `200 OK` → `Skill[]`

```json
[
  {
    "id": "skill-uuid-1",
    "name": "AI Application Development",
    "description": "Build and deploy AI-powered applications.",
    "created_at": "2026-01-01T00:00:00Z"
  }
]
```

---

### `POST /api/skills`

Create a new skill badge. Skills are global — any event can link to them.

- **Auth required:** Yes (company)
- **Request body:**

```json
{
  "name": "Machine Learning",
  "description": "Supervised and unsupervised ML fundamentals."
}
```

- **Success:** `201 Created` → `Skill`

```json
{
  "id": "new-skill-uuid",
  "name": "Machine Learning",
  "description": "Supervised and unsupervised ML fundamentals.",
  "created_at": "2026-02-25T10:00:00Z"
}
```

| Status | Meaning |
|--------|---------|
| `400 Bad Request` | `name` is empty |
| `401 Unauthorized` | Missing or invalid token |
| `403 Forbidden` | Token belongs to a student account |
| `409 Conflict` | A skill with that name already exists |

---

## 6. Student Endpoints

### `GET /api/users/me/skills`

Return all skill badges earned by the authenticated student, newest first.

- **Auth required:** Yes (student)
- **Request body:** None

- **Success:** `200 OK` → `UserSkill[]`

```json
[
  {
    "id": "us-uuid-1",
    "user_id": "seed-user-amara-000-0000-0000-000000000002",
    "skill_id": "skill-uuid-ai",
    "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
    "awarded_at": "2026-02-25T12:00:00Z",
    "skill": {
      "id": "skill-uuid-ai",
      "name": "AI Application Development",
      "description": "Build and deploy AI-powered applications.",
      "created_at": "2026-01-01T00:00:00Z"
    }
  }
]
```

| Status | Meaning |
|--------|---------|
| `401 Unauthorized` | Missing or invalid token |
| `403 Forbidden` | Token belongs to a company account |

---

### `GET /api/users/me/registrations`

Return all events the student is registered for, with event metadata embedded.

- **Auth required:** Yes (student)
- **Request body:** None

- **Success:** `200 OK` → `RegistrationWithEvent[]`

```json
[
  {
    "id": "reg-uuid-1",
    "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
    "student_id": "seed-user-amara-000-0000-0000-000000000002",
    "registered_at": "2026-02-01T00:00:00Z",
    "status": "confirmed",
    "event_title": "Building Apps with AI Workshop",
    "start_time": "2026-02-25T09:00:00Z",
    "end_time": "2026-02-25T17:00:00Z",
    "event_status": "active",
    "location": "TechCorp HQ — Room 3B"
  }
]
```

| Status | Meaning |
|--------|---------|
| `401 Unauthorized` | Missing or invalid token |
| `403 Forbidden` | Token belongs to a company account |

---

## 7. Offline Sync — Deep Dive

> This section explains the exact data shapes needed to implement the
> **local-first QR check-in flow** in the PWA.

### How the flow works

```
HOST (online)                         STUDENT (can be offline)
─────────────────────────────────────────────────────────────────────
GET /api/events/{id}/checkin-code
  → { event_id, check_in_code }

Build CheckInPayload JSON:
  {
    "event_id":  "<id>",
    "host_sig":  "<check_in_code>",
    "timestamp": <unix_now>
  }

Display as QR on screen  ──────────►  PWA scans QR
                                       Stores in IndexedDB (Dexie):
                                       {
                                         local_id: crypto.randomUUID(),
                                         event_id: payload.event_id,
                                         payload:  JSON.stringify(scanned)
                                       }

                         ◄── Network returns ──
                                       Background Sync fires:
                                       POST /api/sync/attendance
                                       → SyncAttendanceRequest
```

### `POST /api/sync/attendance`

Submit a batch of locally stored check-in records to the server for verification.

- **Auth required:** Yes (student)
- **Request body:** `SyncAttendanceRequest`

```json
{
  "records": [
    {
      "local_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "event_id": "seed-event-aiwork-0000-0000-0000-000000000030",
      "payload": "{\"event_id\":\"seed-event-aiwork-0000-0000-0000-000000000030\",\"host_sig\":\"DEMO-CHECKIN-CODE-AI-WORKSHOP-2026\",\"timestamp\":1740477600}"
    }
  ]
}
```

> **Important:** `payload` is a **JSON-encoded string** (i.e. the `CheckInPayload`
> object serialised with `JSON.stringify()`), not a nested object. The server
> stores it verbatim for auditability.

- **Success:** `200 OK` → `SyncAttendanceResponse`

```json
{
  "results": [
    {
      "local_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
      "status": "verified",
      "message": "attendance verified and skills awarded"
    }
  ]
}
```

The `results` array has **one entry per input record**, in the same order.
Use `local_id` to match each result back to its IndexedDB record and update
its status to `VERIFIED` or `REJECTED`.

### Per-record `status` values

| `status` | Meaning | PWA action |
|----------|---------|------------|
| `"verified"` | Accepted; badges awarded | Update IndexedDB → `VERIFIED`; refresh skill badges UI |
| `"rejected"` | Invalid; see `message` | Update IndexedDB → `REJECTED`; surface error to user |

### Rejection `message` values

| Message | Cause |
|---------|-------|
| `"invalid payload JSON"` | `payload` is not valid JSON |
| `"payload missing event_id or host_sig"` | QR payload was incomplete |
| `"payload event_id mismatch"` | Outer `event_id` ≠ `payload.event_id` |
| `"event not found"` | Unknown event UUID |
| `"invalid check-in signature"` | `host_sig` doesn't match server's `check_in_code` |
| `"check-in payload has expired"` | `timestamp` is more than 24 hours in the past |
| `"database error recording attendance"` | Server-side error |
| `"could not record registration: ..."` | Auto-registration failed |
| `"could not award skills: ..."` | Badge award failed |

### Auto-registration on QR scan

When the server successfully verifies a check-in, it **automatically creates
a registration** for the student if one does not exist. This handles the case
where Baraka scans the QR without having pre-registered.

The same capacity rules apply:
- Slots available → `status: "confirmed"` registration created.
- No slots remaining → `status: "conflict_pending"` registration created.

The sync itself still returns `"verified"` in both cases — the student is
checked in either way. The host resolves the slot conflict separately.

### Building the `CheckInPayload` (frontend code)

```typescript
import type { CheckInPayload, AttendanceSyncRecord } from "@/types/api";

/** Call after the student scans the QR code. */
function buildSyncRecord(scannedJson: string): AttendanceSyncRecord {
  const payload: CheckInPayload = JSON.parse(scannedJson);

  return {
    local_id: crypto.randomUUID(),
    event_id: payload.event_id,
    // Store the raw string — the server expects payload as a JSON string,
    // NOT a nested object.
    payload: scannedJson,
  };
}
```

### IndexedDB schema suggestion (Dexie.js)

```typescript
interface PendingCheckIn {
  local_id: string;          // primary key
  event_id: string;
  payload: string;           // raw QR JSON string
  status: "PENDING" | "VERIFIED" | "REJECTED";
  scanned_at: number;        // Date.now() — for display
}
```

---

## 8. Admin Endpoints

### `POST /api/admin/seed`

Load all demo fixture data into the database. Safe to call multiple times
(fully idempotent — uses `INSERT OR IGNORE` throughout).

- **Auth required:** No
- **Request body:** None

- **Success:** `200 OK`

```json
{
  "seeded": true,
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

**What gets created:**

| Type | Details |
|------|---------|
| Users | TechCorp Africa (company), Amara Osei (student), Baraka Mwangi (student) — all password `demo1234` |
| Skills | 9 badges: Python, Data Science, Open Source, Cloud, Mobile Dev, Cybersecurity, AI Application Development, Prompt Engineering, AI Product Management |
| Events | 6 completed past events, 1 active workshop (today), 1 upcoming internship (next month, capacity=2, slots_remaining=1) |
| Amara's history | 6 confirmed registrations, 6 attendance records, 6 skill badges |
| Amara's pre-reg | Confirmed registration for today's workshop |

> **⚠ Remove or gate behind an environment variable before any public deployment.**

---

## 9. Error Format

All error responses use a consistent JSON envelope:

```json
{ "error": "human-readable description" }
```

Example:

```json
{ "error": "email already registered" }
```

The frontend should always check for an `error` key in non-2xx responses.

---

## 10. HTTP Status Code Reference

| Code | Used when |
|------|-----------|
| `200 OK` | Successful read or update |
| `201 Created` | New resource created (register, create event, create skill, register for event) |
| `400 Bad Request` | Malformed JSON, missing required field, or invalid field value |
| `401 Unauthorized` | Missing `Authorization` header, expired token, or invalid signature |
| `403 Forbidden` | Valid token but wrong role, or not the event host |
| `404 Not Found` | Resource with that UUID does not exist |
| `409 Conflict` | Duplicate unique field (email, skill name) or invalid state transition |
| `500 Internal Server Error` | Unexpected server-side error — report to backend team |
