# SkillZone

SkillZone is a Progressive Web App that helps youths gain and track various hard and soft skills they acquire throughout the years. 
The app is a credible way for potential employers to verify that employee candidates actually have the set of skills they are looking for. Users can find internships, workshops, seminars and short courses, provided by trustworthy hosts, via the app where they can grow their skills. Completing these “events” earns a user skill badges that can be used as referral points.

 SkillZone allows students to:
Attend events and internships
Check in fully offline
Sync attendance later
Automatically earn verified skill badges
No internet is required at the venue.

# The Problem
Many young people:
Attend internships and skill events
Gain real experience
Have no structured digital proof
Operate in low-connectivity environments

## Traditional systems:
Require constant internet
Break when offline
Lose attendance data
Do not support resilient sync
SkillZone solves this with a Local-First Architecture.

# How It Works (Local-First Model)
Attendance verification works even when:
WiFi is off
Network is unstable
Server is temporarily unavailable
Offline Check-In Flow
```
HOST DEVICE (offline at venue)
  1. Opens event in PWA (cached from IndexedDB)
  2. Generates QR code:
       { event_id, host_sig, timestamp }

STUDENT DEVICE (offline)
  3. Scans QR code
  4. Saves record locally in IndexedDB
  5. UI shows: "Badge will be verified on reconnect"

LATER (online again)
  6. Service Worker triggers POST /api/sync/attendance
  7. Server verifies signature
  8. Skill badge awarded
  9. Local record updated to VERIFIED
```

The QR code itself is the proof.
Internet is only required later for verification.

## Architecture Overview
```
Frontend (React PWA)
  ↓
Service Worker
  ↓
IndexedDB (Local Storage)
  ↓
Sync Engine
  ↓
Go REST API
  ↓
SQLite Database
```

### Golden Rule:
UI never waits for the network.

All actions:
Save locally first
Sync in background

# Backend – SkillZone API
Go + SQLite REST API

## Tech Stack
Layer	Technology
Language	Go 1.24
Database	SQLite (modernc.org/sqlite)
Auth	JWT (HS256)
Passwords	bcrypt
IDs	UUID v4

Pure Go SQLite driver (no CGO required).

## Project Structure
```
backend/
├── cmd/server/main.go
└── internal/
    ├── models/
    ├── db/
    ├── auth/
    ├── middleware/
    └── handlers/
```
▶ Running Backend Locally
```
cd backend

export DATABASE_URL="skillzone.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
export JWT_SECRET="changeme"
export ADDR=":8080"
```

go run ./cmd/server/

Server runs on:
```
http://localhost:8080
```

## Running Tests
```
cd backend
go test ./...
```
Uses in-memory SQLite — no external services required.

## Authentication

All protected endpoints require:
```
Authorization: Bearer <token>
```
Roles:

student

company

## Core API Endpoints
Auth
| Method | Path               | Auth |
| ------ | ------------------ | ---- |
| POST   | /api/auth/register | —    |
| POST   | /api/auth/login    | —    |
| GET    | /api/auth/me       | ✓    |

| Method | Path        | Role    |
| ------ | ----------- | ------- |
| POST   | /api/skills | company |
| GET    | /api/skills | public  |

| Method | Path                          | Role    |
| ------ | ----------------------------- | ------- |
| POST   | /api/events                   | company |
| GET    | /api/events                   | public  |
| GET    | /api/events/{id}              | public  |
| GET    | /api/events/{id}/checkin-code | company |
| POST   | /api/events/{id}/register     | student |


## Local-First Sync Endpoint
POST /api/sync/attendance

Used to batch-sync offline check-ins.
```
{
  "records": [
    {
      "local_id": "client-uuid",
      "event_id": "uuid",
      "payload": "{\"event_id\":\"...\",\"host_sig\":\"...\",\"timestamp\":1708812000}"
    }
  ]
}
```
Response
```
{
  "results": [
    {
      "local_id": "client-uuid",
      "status": "verified",
      "message": "attendance verified and skills awarded"
    }
  ]
}
```

Status values:
verified
rejected

## Failure Handling
| Scenario                | Behaviour                     |
| ----------------------- | ----------------------------- |
| Network drops mid-sync  | Client retries                |
| Server returns 500      | Exponential backoff           |
| Duplicate sync          | Safe idempotent upsert        |
| Tampered QR             | Signature mismatch → rejected |
| QR older than 24h       | Timestamp validation fails    |
| App closed during write | IndexedDB transaction atomic  |

# Frontend – SkillZone PWA

Built with:

Vite
TypeScript
React
shadcn-ui
Tailwind CSS

The frontend:
Uses IndexedDB for local storage
Implements a service worker
Queues offline attendance records
Syncs automatically when online
Uses optimistic UI updates

▶ Running Frontend Locally
```
cd frontend
# Install dependencies
npm install

# Start development server
npm run dev
App runs on:
```
By default, Vite runs at:
```
http://localhost:5173
```
# Connecting Frontend to Backend
Make sure the backend is running:
```
http://localhost:8080
```

# PWA Features
Installable
Offline support
Background sync
IndexedDB persistence
Optimistic UI updates

