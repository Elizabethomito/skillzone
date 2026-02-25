// Package models defines all the domain types and data-transfer objects (DTOs)
// for the Skillzone API.
//
// LEARNING NOTE — why a separate models package?
// Keeping your types in one place means every other package imports them
// without creating circular dependencies. Handlers, DB, and middleware all
// need these types, so none of them can own the definitions.
package models

import "time"

// UserRole is a "type alias over string" — it lets the compiler catch
// mistakes like passing the wrong string where a role is expected.
type UserRole string

const (
RoleStudent UserRole = "student"
RoleCompany UserRole = "company"
)

// EventStatus represents the lifecycle state of an event.
// Using named constants instead of raw strings prevents typos
// and makes `switch` exhaustiveness obvious.
type EventStatus string

const (
EventStatusUpcoming  EventStatus = "upcoming"
EventStatusActive    EventStatus = "active"
EventStatusCompleted EventStatus = "completed"
)

// AttendanceStatus tracks a pending check-in's sync state on the server.
// The client stores its own "pending" state locally (in IndexedDB / Dexie).
type AttendanceStatus string

const (
AttendancePending  AttendanceStatus = "pending"
AttendanceVerified AttendanceStatus = "verified"
AttendanceRejected AttendanceStatus = "rejected"
)

// RegistrationStatus tracks the slot-allocation outcome for a registration.
// This matters when students register offline and the event is already full.
type RegistrationStatus string

const (
	RegistrationConfirmed       RegistrationStatus = "confirmed"
	RegistrationConflictPending RegistrationStatus = "conflict_pending"
	RegistrationWaitlisted      RegistrationStatus = "waitlisted"
)

// User represents both student and company accounts.
// The json:"-" tag on PasswordHash tells encoding/json to NEVER include it
// in a JSON response — even if you forget to filter it manually.
type User struct {
ID           string    `json:"id"`
Email        string    `json:"email"`
PasswordHash string    `json:"-"` // NEVER serialised to JSON
Name         string    `json:"name"`
Role         UserRole  `json:"role"`
CreatedAt    time.Time `json:"created_at"`
UpdatedAt    time.Time `json:"updated_at"`
}

// Skill is a badge that can be awarded to a student upon event completion.
type Skill struct {
ID          string    `json:"id"`
Name        string    `json:"name"`
Description string    `json:"description"`
CreatedAt   time.Time `json:"created_at"`
}

// Event is a learning opportunity hosted by a company.
type Event struct {
	ID          string      `json:"id"`
	HostID      string      `json:"host_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Location    string      `json:"location"`
	StartTime   time.Time   `json:"start_time"`
	EndTime     time.Time   `json:"end_time"`
	Status      EventStatus `json:"status"`

	// CheckInCode is a short-lived secret embedded in the host's QR code.
	// omitempty means it is omitted from JSON when empty — the list endpoint
	// never populates it; only the host's /checkin-code endpoint does.
	CheckInCode string `json:"check_in_code,omitempty"`

	// Capacity is the maximum number of confirmed registrations allowed.
	// nil / 0 means unlimited. SlotsRemaining is decremented on each
	// confirmed registration and is what the frontend displays.
	Capacity        *int `json:"capacity,omitempty"`
	SlotsRemaining  *int `json:"slots_remaining,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Skills is populated by a JOIN query when reading events.
	// It is omitted from JSON when nil (no skills linked yet).
	Skills []Skill `json:"skills,omitempty"`
}// EventSkill links a skill to an event (many-to-many join table).
type EventSkill struct {
EventID string `json:"event_id"`
SkillID string `json:"skill_id"`
}

// Registration records a student's intent to attend an event.
// Created when a student calls POST /api/events/{id}/register.
type Registration struct {
	ID           string             `json:"id"`
	EventID      string             `json:"event_id"`
	StudentID    string             `json:"student_id"`
	RegisteredAt time.Time          `json:"registered_at"`
	Status       RegistrationStatus `json:"status"`
}// Attendance is the cryptographic check-in proof submitted by a student.
//
// Local-first flow:
//  1. Host's device shows a QR code containing a CheckInPayload JSON.
//  2. Student scans it offline — their device stores it in IndexedDB with
//     status "pending".
//  3. When the student regains connectivity, their PWA calls SyncAttendance.
//  4. The server verifies the payload and writes an Attendance row here.
type Attendance struct {
ID        string `json:"id"`
EventID   string `json:"event_id"`
StudentID string `json:"student_id"`

// Payload is the raw JSON string from the host's QR code — stored verbatim
// for auditability even after verification.
Payload   string           `json:"payload"`
Status    AttendanceStatus `json:"status"`
CreatedAt time.Time        `json:"created_at"`
UpdatedAt time.Time        `json:"updated_at"`
}

// UserSkill records a skill badge awarded to a student.
// A student earns a badge when their attendance at the event is verified.
type UserSkill struct {
ID        string    `json:"id"`
UserID    string    `json:"user_id"`
SkillID   string    `json:"skill_id"`
EventID   string    `json:"event_id"` // which event triggered the award
AwardedAt time.Time `json:"awarded_at"`

// Skill is populated by a JOIN when reading /users/me/skills.
Skill *Skill `json:"skill,omitempty"`
}

// ---- Request / Response DTOs ----
// DTOs (Data Transfer Objects) are structs used only for reading JSON from
// request bodies or writing JSON to response bodies. They are separate from
// the domain types above so that the API surface can evolve independently.

type RegisterRequest struct {
Email    string   `json:"email"`
Password string   `json:"password"`
Name     string   `json:"name"`
Role     UserRole `json:"role"`
}

type LoginRequest struct {
Email    string `json:"email"`
Password string `json:"password"`
}

type LoginResponse struct {
Token string `json:"token"`
User  User   `json:"user"`
}

type CreateEventRequest struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	// SkillIDs is a list of existing skill IDs to attach to this event.
	// After completing the event, students earn all of these as badges.
	SkillIDs []string `json:"skill_ids"`
	// Capacity, if > 0, caps the number of confirmed registrations.
	// Leave 0 or omit for unlimited. Used for internships with limited slots.
	Capacity int `json:"capacity,omitempty"`
}

// UpdateEventStatusRequest is used by PATCH /api/events/{id}/status
type UpdateEventStatusRequest struct {
	Status EventStatus `json:"status"`
}

// ResolveConflictRequest is used by PATCH /api/events/{id}/registrations/{reg_id}
// The host either confirms the registration (taking the last slot) or waitlists it.
type ResolveConflictRequest struct {
	// Action must be "confirm" or "waitlist"
	Action string `json:"action"`
}

type SyncAttendanceRequest struct {
// Records is a batch of locally stored ATTENDANCE_PENDING check-ins.
// Sending them in one request reduces round-trips when coming back online.
Records []AttendanceSyncRecord `json:"records"`
}

// AttendanceSyncRecord is one item in a sync batch.
type AttendanceSyncRecord struct {
// LocalID is a UUID generated by the client device. The server echoes it
// back in SyncResult so the client can match results to its local records.
LocalID string `json:"local_id"`
EventID string `json:"event_id"`
// Payload is the JSON string from the host's QR code, stored verbatim.
Payload string `json:"payload"`
}

type SyncAttendanceResponse struct {
// Results has one entry per input record, in the same order.
Results []SyncResult `json:"results"`
}

// SyncResult tells the client whether each record was accepted or rejected,
// with a human-readable message for any rejection reason.
type SyncResult struct {
LocalID string           `json:"local_id"`
Status  AttendanceStatus `json:"status"`
Message string           `json:"message,omitempty"`
}

// CheckInPayload is the structure embedded in the host's QR code.
//
// Security model:
//   - EventID ties the QR to a specific event.
//   - HostSig is the server-generated check_in_code (shared secret). A student
//     can only produce a valid payload if they physically scanned the host's QR.
//   - Timestamp lets the server reject stale payloads (>24 h old).
type CheckInPayload struct {
EventID   string `json:"event_id"`
HostSig   string `json:"host_sig"`
Timestamp int64  `json:"timestamp"` // Unix seconds (UTC)
}
