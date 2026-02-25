package models

import "time"

// UserRole defines the type of user account.
type UserRole string

const (
	RoleStudent UserRole = "student"
	RoleCompany UserRole = "company"
)

// EventStatus represents the lifecycle state of an event.
type EventStatus string

const (
	EventStatusUpcoming  EventStatus = "upcoming"
	EventStatusActive    EventStatus = "active"
	EventStatusCompleted EventStatus = "completed"
)

// AttendanceStatus tracks a pending check-in's sync state.
type AttendanceStatus string

const (
	AttendancePending  AttendanceStatus = "pending"
	AttendanceVerified AttendanceStatus = "verified"
	AttendanceRejected AttendanceStatus = "rejected"
)

// User represents both student and company accounts.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
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
	CheckInCode string    `json:"check_in_code,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Populated on read
	Skills []Skill `json:"skills,omitempty"`
}

// EventSkill links a skill to an event (many-to-many).
type EventSkill struct {
	EventID string `json:"event_id"`
	SkillID string `json:"skill_id"`
}

// Registration records a student's intent to attend an event.
type Registration struct {
	ID         string    `json:"id"`
	EventID    string    `json:"event_id"`
	StudentID  string    `json:"student_id"`
	RegisteredAt time.Time `json:"registered_at"`
}

// Attendance is the cryptographic check-in proof submitted by a student.
// It is created offline and synced to the server later.
type Attendance struct {
	ID        string           `json:"id"`
	EventID   string           `json:"event_id"`
	StudentID string           `json:"student_id"`
	// Payload is the raw JSON from the host's QR code.
	Payload   string           `json:"payload"`
	Status    AttendanceStatus `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// UserSkill records a skill badge awarded to a student.
type UserSkill struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	SkillID      string    `json:"skill_id"`
	EventID      string    `json:"event_id"` // which event awarded it
	AwardedAt    time.Time `json:"awarded_at"`

	// Populated on read
	Skill *Skill `json:"skill,omitempty"`
}

// ---- Request / Response DTOs ----

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
	SkillIDs    []string  `json:"skill_ids"`
}

type SyncAttendanceRequest struct {
	// Each item is a locally stored ATTENDANCE_PENDING record.
	Records []AttendanceSyncRecord `json:"records"`
}

type AttendanceSyncRecord struct {
	LocalID   string `json:"local_id"` // client-side UUID for idempotency
	EventID   string `json:"event_id"`
	// Payload is the JSON string from the host's QR code.
	Payload   string `json:"payload"`
}

type SyncAttendanceResponse struct {
	Results []SyncResult `json:"results"`
}

type SyncResult struct {
	LocalID  string           `json:"local_id"`
	Status   AttendanceStatus `json:"status"`
	Message  string           `json:"message,omitempty"`
}

// CheckInPayload is decoded from the host's QR code.
type CheckInPayload struct {
	EventID   string `json:"event_id"`
	HostSig   string `json:"host_sig"`
	Timestamp int64  `json:"timestamp"`
}
