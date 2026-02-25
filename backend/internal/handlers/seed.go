package handlers

// SeedDemo handles POST /api/admin/seed
//
// This endpoint is ONLY for hackathon demos. It inserts a fixed set of users,
// skills, events, registrations, and skill badges so the demo can start from
// a known, richly-populated state without running external scripts.
//
// The endpoint is idempotent — calling it twice is safe because every INSERT
// uses INSERT OR IGNORE on the email/name unique constraints, and UUIDs are
// pre-determined (hard-coded) so the same rows are produced every time.
//
// DEMO SCENARIO
// ─────────────────────────────────────────────────────────────────────────
// Company  : "TechCorp Africa"    (host@techcorp.test / demo1234)
// Veteran  : "Amara Osei"         (amara@student.test  / demo1234)
//              → rich history: 6 past events, 5 skill badges
//              → registered for the AI Workshop (waiting to check in at venue)
//              → will apply for the internship during the demo
// Newcomer : "Baraka Mwangi"      (baraka@student.test / demo1234)
//              → no history, fresh account
//              → did NOT pre-register; will scan QR when network drops
//              → will also apply for the internship offline
//
// Events (all hosted by TechCorp Africa):
//   PAST (completed)  — provides historical context for Amara's profile
//     1. Intro to Python Workshop        (3 months ago)
//     2. Data Science Bootcamp           (2 months ago)
//     3. Open Source Hackathon           (6 weeks ago)
//     4. Cloud Computing Seminar         (1 month ago)
//     5. Mobile Dev Internship           (2 weeks ago — Amara completed)
//     6. Cybersecurity Workshop          (1 week ago)
//
//   ACTIVE (ongoing today) — used for the live QR check-in demo
//     7. Building Apps with AI Workshop  (TODAY, started 1h ago, ends in 7h)
//        Skills awarded: "AI Application Development", "Prompt Engineering"
//        → Amara is pre-registered (confirmed); she just needs to scan the QR
//        → Baraka is NOT registered; she will scan the QR offline
//
//   UPCOMING (future) — used for the offline internship registration conflict
//     8. AI Product Internship           (next month, capacity = 1 slot remaining)
//        Skills awarded: "AI Product Management"
//        → During the demo both Amara and Baraka apply offline
//        → On reconnect, first sync wins the slot; second becomes conflict_pending
//        → Host resolves via PATCH /api/events/{id}/registrations/{reg_id}

import (
	"net/http"
	"time"

	"github.com/Elizabethomito/skillzone/backend/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

// Pre-determined UUIDs keep the seed idempotent across restarts.
const (
	SeedCompanyID = "seed-company-00000000-0000-0000-0000-000000000001"
	SeedAmaraID   = "seed-amara-000000000-0000-0000-0000-000000000002"
	SeedBarakaID  = "seed-baraka-00000000-0000-0000-0000-000000000003"

	// Skills
	SeedSkillPythonID     = "seed-skill-python--0000-0000-0000-000000000010"
	SeedSkillDataSciID    = "seed-skill-datasci-0000-0000-0000-000000000011"
	SeedSkillOpenSourceID = "seed-skill-oss----0000-0000-0000-000000000012"
	SeedSkillCloudID      = "seed-skill-cloud--0000-0000-0000-000000000013"
	SeedSkillMobileID     = "seed-skill-mobile-0000-0000-0000-000000000014"
	SeedSkillCyberID      = "seed-skill-cyber--0000-0000-0000-000000000015"
	SeedSkillAIDevID      = "seed-skill-aidev--0000-0000-0000-000000000016"
	SeedSkillPromptID     = "seed-skill-prompt-0000-0000-0000-000000000017"
	SeedSkillAIProdMgmtID = "seed-skill-aiprod-0000-0000-0000-000000000018"

	// Past events
	SeedEventPythonID  = "seed-event-python-0000-0000-0000-000000000020"
	SeedEventDataSciID = "seed-event-datasi-0000-0000-0000-000000000021"
	SeedEventHackID    = "seed-event-hack---0000-0000-0000-000000000022"
	SeedEventCloudID   = "seed-event-cloud--0000-0000-0000-000000000023"
	SeedEventMobileID  = "seed-event-mobile-0000-0000-0000-000000000024"
	SeedEventCyberID   = "seed-event-cyber--0000-0000-0000-000000000025"

	// Active + upcoming
	SeedEventAIWorkshopID = "seed-event-aiwork-0000-0000-0000-000000000030"
	SeedEventInternshipID = "seed-event-intern-0000-0000-0000-000000000031"

	// Check-in code for the active workshop — the host will embed this in the QR.
	SeedAIWorkshopCheckInCode = "DEMO-CHECKIN-CODE-AI-WORKSHOP-2026"
)

// SeedDemo handles POST /api/admin/seed
func (s *Server) SeedDemo(w http.ResponseWriter, r *http.Request) {
	hash, err := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "bcrypt: "+err.Error())
		return
	}
	pw := string(hash)

	now := time.Now().UTC()

	// ── Users ────────────────────────────────────────────────────────────
	users := []struct{ id, email, name, role string }{
		{SeedCompanyID, "host@techcorp.test", "TechCorp Africa", "company"},
		{SeedAmaraID, "amara@student.test", "Amara Osei", "student"},
		{SeedBarakaID, "baraka@student.test", "Baraka Mwangi", "student"},
	}
	for _, u := range users {
		s.DB.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO users (id, email, password_hash, name, role, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			u.id, u.email, pw, u.name, u.role, now, now,
		)
	}

	// ── Skills ───────────────────────────────────────────────────────────
	skills := []struct{ id, name, desc string }{
		{SeedSkillPythonID, "Python Programming", "Proficient in Python scripting and data manipulation"},
		{SeedSkillDataSciID, "Data Science", "Applied machine learning and statistical analysis"},
		{SeedSkillOpenSourceID, "Open Source Contribution", "Contributed to public repositories during a hackathon"},
		{SeedSkillCloudID, "Cloud Computing", "AWS / GCP fundamentals and serverless architecture"},
		{SeedSkillMobileID, "Mobile Development", "Cross-platform mobile apps with React Native"},
		{SeedSkillCyberID, "Cybersecurity Fundamentals", "Threat modelling, OWASP Top-10, secure coding"},
		{SeedSkillAIDevID, "AI Application Development", "Building production AI-powered applications"},
		{SeedSkillPromptID, "Prompt Engineering", "Designing effective prompts for large language models"},
		{SeedSkillAIProdMgmtID, "AI Product Management", "Roadmapping and shipping AI-first products"},
	}
	for _, sk := range skills {
		s.DB.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO skills (id, name, description, created_at) VALUES (?, ?, ?, ?)`,
			sk.id, sk.name, sk.desc, now,
		)
	}

	// ── Past events ───────────────────────────────────────────────────────
	type eventRow struct {
		id, title, desc, loc string
		start, end           time.Time
		status               string
		checkIn              string
		capacity             interface{} // nil = unlimited
		slotsRemaining       interface{}
	}
	pastEvents := []eventRow{
		{
			SeedEventPythonID, "Intro to Python Workshop", "Beginner-friendly Python programming workshop",
			"Zone01 Kisumu Lab A",
			now.AddDate(0, -3, 0), now.AddDate(0, -3, 0).Add(4 * time.Hour),
			"completed", "past-ci-python", nil, nil,
		},
		{
			SeedEventDataSciID, "Data Science Bootcamp", "Hands-on data science with pandas and scikit-learn",
			"Zone01 Kisumu Lab B",
			now.AddDate(0, -2, 0), now.AddDate(0, -2, 0).Add(8 * time.Hour),
			"completed", "past-ci-datascience", nil, nil,
		},
		{
			SeedEventHackID, "Open Source Hackathon", "48-hour open source contribution sprint",
			"Zone01 Kisumu Main Hall",
			now.AddDate(0, 0, -42), now.AddDate(0, 0, -40),
			"completed", "past-ci-hackathon", nil, nil,
		},
		{
			SeedEventCloudID, "Cloud Computing Seminar", "AWS and GCP fundamentals for developers",
			"Online + Zone01 Kisumu Lab A",
			now.AddDate(0, -1, 0), now.AddDate(0, -1, 0).Add(3 * time.Hour),
			"completed", "past-ci-cloud", nil, nil,
		},
		{
			SeedEventMobileID, "Mobile Dev Internship", "4-week intensive mobile development programme",
			"Zone01 Kisumu",
			now.AddDate(0, 0, -14), now.AddDate(0, 0, -7),
			"completed", "past-ci-mobile", nil, nil,
		},
		{
			SeedEventCyberID, "Cybersecurity Workshop", "Practical security: OWASP, CTF challenges",
			"Zone01 Kisumu Lab B",
			now.AddDate(0, 0, -7), now.AddDate(0, 0, -7).Add(6 * time.Hour),
			"completed", "past-ci-cyber", nil, nil,
		},
	}
	for _, e := range pastEvents {
		s.DB.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO events
			 (id, host_id, title, description, location, start_time, end_time,
			  status, check_in_code, capacity, slots_remaining, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.id, SeedCompanyID, e.title, e.desc, e.loc,
			e.start, e.end, e.status, e.checkIn,
			e.capacity, e.slotsRemaining, now, now,
		)
	}

	// ── Active workshop (TODAY) ───────────────────────────────────────────
	workshopStart := now.Add(-1 * time.Hour)
	workshopEnd := now.Add(7 * time.Hour)
	s.DB.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO events
		 (id, host_id, title, description, location, start_time, end_time,
		  status, check_in_code, capacity, slots_remaining, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?, NULL, NULL, ?, ?)`,
		SeedEventAIWorkshopID, SeedCompanyID,
		"Building Apps with AI Workshop",
		"A full-day hands-on workshop covering LLM APIs, RAG pipelines, and shipping AI-powered PWAs.",
		"Zone01 Kisumu Main Hall",
		workshopStart, workshopEnd,
		SeedAIWorkshopCheckInCode, now, now,
	)

	// ── Upcoming internship (next month, 1 slot remaining) ───────────────
	internStart := now.AddDate(0, 1, 0)
	internEnd := internStart.AddDate(0, 3, 0)
	one := 1
	_ = one
	s.DB.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO events
		 (id, host_id, title, description, location, start_time, end_time,
		  status, check_in_code, capacity, slots_remaining, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'upcoming', ?, 2, 1, ?, ?)`,
		SeedEventInternshipID, SeedCompanyID,
		"AI Product Internship",
		"A 3-month paid internship building AI products. Only 1 slot remaining — apply before it closes!",
		"Zone01 Kisumu + Remote",
		internStart, internEnd,
		"intern-ci-code-not-used", now, now,
	)

	// ── Event–skill links ─────────────────────────────────────────────────
	eventSkillLinks := [][2]string{
		{SeedEventPythonID, SeedSkillPythonID},
		{SeedEventDataSciID, SeedSkillDataSciID},
		{SeedEventHackID, SeedSkillOpenSourceID},
		{SeedEventCloudID, SeedSkillCloudID},
		{SeedEventMobileID, SeedSkillMobileID},
		{SeedEventCyberID, SeedSkillCyberID},
		{SeedEventAIWorkshopID, SeedSkillAIDevID},
		{SeedEventAIWorkshopID, SeedSkillPromptID},
		{SeedEventInternshipID, SeedSkillAIProdMgmtID},
	}
	for _, link := range eventSkillLinks {
		s.DB.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO event_skills (event_id, skill_id) VALUES (?, ?)`,
			link[0], link[1],
		)
	}

	// ── Amara's attendance history (all past events) ──────────────────────
	amaraPastEvents := []struct {
		eventID, skillID, ciCode string
		when                     time.Time
	}{
		{SeedEventPythonID, SeedSkillPythonID, "past-ci-python", now.AddDate(0, -3, 0).Add(2 * time.Hour)},
		{SeedEventDataSciID, SeedSkillDataSciID, "past-ci-datascience", now.AddDate(0, -2, 0).Add(3 * time.Hour)},
		{SeedEventHackID, SeedSkillOpenSourceID, "past-ci-hackathon", now.AddDate(0, 0, -41)},
		{SeedEventCloudID, SeedSkillCloudID, "past-ci-cloud", now.AddDate(0, -1, 0).Add(1 * time.Hour)},
		{SeedEventMobileID, SeedSkillMobileID, "past-ci-mobile", now.AddDate(0, 0, -13)},
		{SeedEventCyberID, SeedSkillCyberID, "past-ci-cyber", now.AddDate(0, 0, -7).Add(2 * time.Hour)},
	}

	for i, pe := range amaraPastEvents {
		// Registration
		s.DB.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO registrations (id, event_id, student_id, registered_at, status)
			 VALUES (?, ?, ?, ?, 'confirmed')`,
			seedID("amara-reg", i), pe.eventID, SeedAmaraID, pe.when.Add(-24*time.Hour),
		)
		// Attendance — build a proper signed check-in token for auditability.
		// The token's exp is set to 6 h after the historical event time so
		// the payload accurately reflects what the QR would have looked like.
		ciToken, _ := auth.GenerateCheckInTokenWithExpiry(
			pe.eventID, pe.ciCode, s.Secret,
			pe.when,
			pe.when.Add(auth.CheckInTokenDuration),
		)
		payloadJSON := `{"token":"` + ciToken + `"}`
		s.DB.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO attendances (id, event_id, student_id, payload, status, created_at, updated_at)
			 VALUES (?, ?, ?, ?, 'verified', ?, ?)`,
			seedID("amara-att", i), pe.eventID, SeedAmaraID, payloadJSON, pe.when, pe.when,
		)
		// Skill badge
		s.DB.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO user_skills (id, user_id, skill_id, event_id, awarded_at)
			 VALUES (?, ?, ?, ?, ?)`,
			seedID("amara-skill", i), SeedAmaraID, pe.skillID, pe.eventID, pe.when,
		)
	}

	// ── Amara is pre-registered for today's AI Workshop ───────────────────
	s.DB.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO registrations (id, event_id, student_id, registered_at, status)
		 VALUES (?, ?, ?, ?, 'confirmed')`,
		"seed-amara-reg-workshop-000000000030", SeedEventAIWorkshopID, SeedAmaraID,
		now.Add(-30*time.Minute),
	)

	respond(w, http.StatusOK, map[string]any{
		"seeded": true,
		"accounts": []map[string]string{
			{"role": "company", "email": "host@techcorp.test", "password": "demo1234", "name": "TechCorp Africa"},
			{"role": "student", "email": "amara@student.test", "password": "demo1234", "name": "Amara Osei (veteran)"},
			{"role": "student", "email": "baraka@student.test", "password": "demo1234", "name": "Baraka Mwangi (newcomer)"},
		},
		"active_workshop": map[string]string{
			"event_id":      SeedEventAIWorkshopID,
			"check_in_code": SeedAIWorkshopCheckInCode,
			"title":         "Building Apps with AI Workshop",
		},
		"internship": map[string]any{
			"event_id":        SeedEventInternshipID,
			"title":           "AI Product Internship",
			"slots_remaining": 1,
		},
	})
}

// seedID generates a stable fake UUID for seeded rows so the seed is idempotent.
func seedID(prefix string, n int) string {
	s := prefix
	for len(s) < 30 {
		s += "-0"
	}
	return s[:30] + itoa(int64(n)) + "x"
}

// itoa is a small helper to avoid importing strconv in this file.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
