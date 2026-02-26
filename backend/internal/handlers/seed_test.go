package handlers

// seed_test.go — integration tests that call SeedDemo via HTTP and then
// verify the resulting database state with targeted API calls.
//
// Every test uses newTestServer (in-memory SQLite, no shared state) and
// calls srv.SeedDemo directly, so the tests are hermetic and fast.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// runSeed fires SeedDemo and asserts it returns 200.
func runSeed(t *testing.T, srv *Server) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/seed", nil)
	rec := httptest.NewRecorder()
	srv.SeedDemo(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("seed: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("seed: decode response: %v", err)
	}
	return out
}

// dbInt is a helper to read a single integer from the database.
func dbInt(t *testing.T, srv *Server, query string, args ...any) int {
	t.Helper()
	var n int
	if err := srv.DB.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("dbInt(%q): %v", query, err)
	}
	return n
}

// ─────────────────────────────────────────────────────────────────────────────
// Seed response shape
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_ResponseShape(t *testing.T) {
	srv := newTestServer(t)
	out := runSeed(t, srv)

	if out["seeded"] != true {
		t.Errorf("expected seeded=true, got %v", out["seeded"])
	}
	for _, key := range []string{
		"accounts", "companies", "active_workshop",
		"active_agri_workshop", "active_med_workshop", "internship",
	} {
		if out[key] == nil {
			t.Errorf("response missing key %q", key)
		}
	}
}

func TestSeedDemo_Idempotent(t *testing.T) {
	srv := newTestServer(t)
	// Calling seed twice must not error or produce duplicates.
	runSeed(t, srv)
	runSeed(t, srv) // second call

	n := dbInt(t, srv, `SELECT COUNT(*) FROM users WHERE email='host@techcorp.test'`)
	if n != 1 {
		t.Errorf("expected exactly 1 TechCorp user, got %d", n)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Users
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_UserCount(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	total := dbInt(t, srv, `SELECT COUNT(*) FROM users`)
	if total < 20 {
		t.Errorf("expected ≥20 users, got %d", total)
	}

	companies := dbInt(t, srv, `SELECT COUNT(*) FROM users WHERE role='company'`)
	if companies != 3 {
		t.Errorf("expected 3 company users, got %d", companies)
	}

	students := dbInt(t, srv, `SELECT COUNT(*) FROM users WHERE role='student'`)
	if students < 17 {
		t.Errorf("expected ≥17 student users, got %d", students)
	}
}

func TestSeedDemo_DemoAccountsExist(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	for _, email := range []string{
		"host@techcorp.test",
		"amara@student.test",
		"baraka@student.test",
	} {
		n := dbInt(t, srv, `SELECT COUNT(*) FROM users WHERE email=?`, email)
		if n != 1 {
			t.Errorf("demo account %q not seeded", email)
		}
	}
}

func TestSeedDemo_FillerStudentsExist(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	fillerEmails := []string{
		// TechCorp track
		"chidi@student.test", "fatima@student.test", "kwame@student.test",
		"aisha@student.test", "tobi@student.test", "ngozi@student.test",
		"joel@student.test", "lila@student.test",
		// GreenLeaf track
		"zara@student.test", "emeka@student.test", "sade@student.test", "kofi@student.test",
		// MedConnect track
		"muna@student.test", "dayo@student.test", "nia@student.test",
	}
	for _, email := range fillerEmails {
		n := dbInt(t, srv, `SELECT COUNT(*) FROM users WHERE email=?`, email)
		if n != 1 {
			t.Errorf("filler student %q not seeded", email)
		}
	}
}

func TestSeedDemo_CompanyAccountsExist(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	for _, email := range []string{
		"host@techcorp.test",
		"host@greenleaf.test",
		"host@medconnect.test",
	} {
		n := dbInt(t, srv, `SELECT COUNT(*) FROM users WHERE email=? AND role='company'`, email)
		if n != 1 {
			t.Errorf("company account %q not seeded", email)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Skills catalogue
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_SkillCount(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	n := dbInt(t, srv, `SELECT COUNT(*) FROM skills`)
	if n < 37 {
		t.Errorf("expected ≥37 skills (37 seeded), got %d", n)
	}
}

func TestSeedDemo_SkillDomainCoverage(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Spot-check one badge from each domain.
	for _, name := range []string{
		"Python Programming",            // Technology: Programming
		"Data Science",                  // Technology: Data & ML
		"AI Application Development",    // Technology: AI
		"Cloud Computing",               // Technology: Infrastructure
		"Mobile Development",            // Technology: Frontend & Mobile
		"Soil Science & Health",         // Agriculture
		"Climate-Smart Farming",         // Agriculture
		"Health Data Management",        // Healthcare
		"Public Health & Epidemiology",  // Healthcare
		"Entrepreneurship & Innovation", // Business
		"Project Management",            // Business
	} {
		n := dbInt(t, srv, `SELECT COUNT(*) FROM skills WHERE name=?`, name)
		if n != 1 {
			t.Errorf("skill %q not found in catalogue", name)
		}
	}
}

func TestSeedDemo_SkillsListAPI(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	rec := httptest.NewRecorder()
	srv.ListSkills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ListSkills: expected 200, got %d", rec.Code)
	}
	var skills []map[string]any
	json.NewDecoder(rec.Body).Decode(&skills)
	if len(skills) < 37 {
		t.Errorf("ListSkills: expected ≥37, got %d", len(skills))
	}
	if len(skills) > 0 {
		first := skills[0]["name"].(string)
		if first[0] != 'A' && first[0] != 'a' {
			t.Errorf("skills not sorted alphabetically; first=%q", first)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Events
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_EventCount(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	total := dbInt(t, srv, `SELECT COUNT(*) FROM events`)
	if total != 20 {
		t.Errorf("expected 20 events (10 TechCorp + 5 GreenLeaf + 5 MedConnect), got %d", total)
	}
}

func TestSeedDemo_EventsPerCompany(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	cases := []struct {
		companyID string
		name      string
		want      int
	}{
		{SeedCompanyID, "TechCorp Africa", 10},
		{SeedGreenLeafID, "GreenLeaf Agri", 5},
		{SeedMedConnectID, "MedConnect Health", 5},
	}
	for _, c := range cases {
		n := dbInt(t, srv, `SELECT COUNT(*) FROM events WHERE host_id=?`, c.companyID)
		if n != c.want {
			t.Errorf("%s: expected %d events, got %d", c.name, c.want, n)
		}
	}
}

func TestSeedDemo_ActiveEventsPerCompany(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Each company should have exactly 1 active event today.
	for _, companyID := range []string{SeedCompanyID, SeedGreenLeafID, SeedMedConnectID} {
		n := dbInt(t, srv,
			`SELECT COUNT(*) FROM events WHERE host_id=? AND status='active'`, companyID)
		if n != 1 {
			t.Errorf("company %s: expected 1 active event, got %d", companyID, n)
		}
	}
}

func TestSeedDemo_UpcomingEventsHaveSlots(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Every upcoming event with a capacity set must have slots_remaining > 0.
	rows, err := srv.DB.Query(
		`SELECT id, slots_remaining FROM events WHERE status='upcoming' AND capacity IS NOT NULL`)
	if err != nil {
		t.Fatalf("query upcoming events: %v", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id string
		var slots int
		rows.Scan(&id, &slots)
		if slots <= 0 {
			t.Errorf("upcoming event %s has slots_remaining=%d (want >0)", id, slots)
		}
		count++
	}
	if count == 0 {
		t.Error("no upcoming events with capacity found")
	}
}

func TestSeedDemo_EventSkillLinks(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	n := dbInt(t, srv, `SELECT COUNT(*) FROM event_skills`)
	if n < 24 {
		t.Errorf("expected ≥24 event–skill links, got %d", n)
	}

	// Active TechCorp workshop awards 2 skills.
	n = dbInt(t, srv,
		`SELECT COUNT(*) FROM event_skills WHERE event_id=?`, SeedEventAIWorkshopID)
	if n != 2 {
		t.Errorf("AI workshop: expected 2 skill links, got %d", n)
	}

	// Active MedConnect workshop also awards 2 skills.
	n = dbInt(t, srv,
		`SELECT COUNT(*) FROM event_skills WHERE event_id=?`, SeedEventMedWorkID)
	if n != 2 {
		t.Errorf("MedConnect workshop: expected 2 skill links, got %d", n)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Registrations
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_AmaraRegistrations(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Amara has 6 past event registrations + 1 pre-registration for today's workshop.
	n := dbInt(t, srv,
		`SELECT COUNT(*) FROM registrations WHERE student_id=? AND status='confirmed'`, SeedAmaraID)
	if n < 7 {
		t.Errorf("Amara: expected ≥7 confirmed registrations, got %d", n)
	}
}

func TestSeedDemo_TechCorpEventsHaveMultipleAttendees(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Every completed TechCorp event should have at least 2 attendees (not just Amara).
	pastEventIDs := []string{
		SeedEventPythonID, SeedEventDataSciID, SeedEventHackID,
		SeedEventCloudID, SeedEventMobileID, SeedEventCyberID,
	}
	for _, eventID := range pastEventIDs {
		n := dbInt(t, srv,
			`SELECT COUNT(DISTINCT student_id) FROM registrations WHERE event_id=?`, eventID)
		if n < 2 {
			t.Errorf("TechCorp past event %s: expected ≥2 attendees, got %d", eventID, n)
		}
	}
}

func TestSeedDemo_GreenLeafEventsHaveAttendees(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	for _, eventID := range []string{SeedEventSoilID, SeedEventClimateID, SeedEventAgriTechID} {
		n := dbInt(t, srv,
			`SELECT COUNT(DISTINCT student_id) FROM registrations WHERE event_id=?`, eventID)
		if n < 1 {
			t.Errorf("GreenLeaf past event %s: expected ≥1 attendee, got %d", eventID, n)
		}
	}
}

func TestSeedDemo_MedConnectEventsHaveAttendees(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	for _, eventID := range []string{SeedEventHealthDataID, SeedEventPublicHealthID, SeedEventMedTechID} {
		n := dbInt(t, srv,
			`SELECT COUNT(DISTINCT student_id) FROM registrations WHERE event_id=?`, eventID)
		if n < 1 {
			t.Errorf("MedConnect past event %s: expected ≥1 attendee, got %d", eventID, n)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Skill badges (user_skills)
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_AmaraBadges(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	n := dbInt(t, srv, `SELECT COUNT(*) FROM user_skills WHERE user_id=?`, SeedAmaraID)
	if n < 6 {
		t.Errorf("Amara: expected ≥6 badges, got %d", n)
	}
}

func TestSeedDemo_FillerStudentsBadges(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Every filler student should have at least 1 badge.
	fillerIDs := []string{
		SeedChidiID, SeedFatimaID, SeedKwameID, SeedAishaID,
		SeedTobiID, SeedNgoziID, SeedJoelID, SeedLilaID,
		SeedZaraID, SeedEmekaID, SeedSadeID, SeedKofiID,
		SeedMunaID, SeedDayoID, SeedNiaID,
	}
	for _, id := range fillerIDs {
		n := dbInt(t, srv, `SELECT COUNT(*) FROM user_skills WHERE user_id=?`, id)
		if n < 1 {
			t.Errorf("filler student %s: expected ≥1 badge, got %d", id, n)
		}
	}
}

func TestSeedDemo_VaryingExperienceLevels(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Veteran: Chidi ≥ 4 badges (near-Amara level)
	n := dbInt(t, srv, `SELECT COUNT(*) FROM user_skills WHERE user_id=?`, SeedChidiID)
	if n < 4 {
		t.Errorf("Chidi (veteran): expected ≥4 badges, got %d", n)
	}

	// Newcomer: Joel should have exactly 1 badge
	n = dbInt(t, srv, `SELECT COUNT(*) FROM user_skills WHERE user_id=?`, SeedJoelID)
	if n != 1 {
		t.Errorf("Joel (newcomer): expected exactly 1 badge, got %d", n)
	}

	// Newcomer: Nia should have exactly 1 badge
	n = dbInt(t, srv, `SELECT COUNT(*) FROM user_skills WHERE user_id=?`, SeedNiaID)
	if n != 1 {
		t.Errorf("Nia (newcomer): expected exactly 1 badge, got %d", n)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Attendance records
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_AttendanceRecordsVerified(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	unverified := dbInt(t, srv,
		`SELECT COUNT(*) FROM attendances WHERE status != 'verified'`)
	if unverified != 0 {
		t.Errorf("expected all seeded attendances to be verified, got %d unverified", unverified)
	}
}

func TestSeedDemo_AttendanceTokensAreSigned(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Every attendance payload should contain a non-empty "token" field.
	rows, err := srv.DB.Query(`SELECT id, payload FROM attendances`)
	if err != nil {
		t.Fatalf("query attendances: %v", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id, payload string
		rows.Scan(&id, &payload)
		var p map[string]any
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			t.Errorf("attendance %s: invalid JSON payload: %v", id, err)
			continue
		}
		tok, ok := p["token"].(string)
		if !ok || tok == "" {
			t.Errorf("attendance %s: missing or empty token in payload", id)
		}
		count++
	}
	if count == 0 {
		t.Error("no attendance records found after seeding")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Candidate search — cross-company & cross-domain
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_SearchStudents_AllStudents(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/users/students", nil)
	req = ctxWithUser(req, SeedCompanyID, "company")
	rec := httptest.NewRecorder()
	srv.SearchStudents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("SearchStudents: expected 200, got %d", rec.Code)
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	if len(result) < 17 {
		t.Errorf("SearchStudents (no filter): expected ≥17 students, got %d", len(result))
	}
}

func TestSeedDemo_SearchStudents_ByPythonSkill(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	req := httptest.NewRequest(http.MethodGet,
		"/api/users/students?skill_id="+SeedSkillPythonID, nil)
	req = ctxWithUser(req, SeedCompanyID, "company")
	rec := httptest.NewRecorder()
	srv.SearchStudents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	// Amara, Chidi, Fatima, Aisha all attended Python events.
	if len(result) < 4 {
		t.Errorf("Python filter: expected ≥4 students, got %d", len(result))
	}
}

func TestSeedDemo_SearchStudents_AgriSkill(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	req := httptest.NewRequest(http.MethodGet,
		"/api/users/students?skill_id="+SeedSkillSoilScienceID, nil)
	req = ctxWithUser(req, SeedGreenLeafID, "company")
	rec := httptest.NewRecorder()
	srv.SearchStudents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	// Zara and Emeka both attended the Soil Health event.
	if len(result) < 2 {
		t.Errorf("SoilScience filter: expected ≥2 students, got %d", len(result))
	}
}

func TestSeedDemo_SearchStudents_HealthcareAISkill(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	req := httptest.NewRequest(http.MethodGet,
		"/api/users/students?skill_id="+SeedSkillHealthcareAIID, nil)
	req = ctxWithUser(req, SeedMedConnectID, "company")
	rec := httptest.NewRecorder()
	srv.SearchStudents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	// Dayo and Nia both attended the MedConnect workshop today.
	if len(result) < 2 {
		t.Errorf("HealthcareAI filter: expected ≥2 students, got %d", len(result))
	}
}

func TestSeedDemo_SearchStudents_MultiSkillAND(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Amara has both Python AND Data Science badges — she should appear.
	// Baraka has no badges — she should NOT appear.
	url := "/api/users/students?skill_id=" + SeedSkillPythonID +
		"&skill_id=" + SeedSkillDataSciID
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req = ctxWithUser(req, SeedCompanyID, "company")
	rec := httptest.NewRecorder()
	srv.SearchStudents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	if len(result) < 1 {
		t.Errorf("Python+DataScience filter: expected ≥1 result, got 0")
	}
	// Verify Baraka is not in the result.
	for _, s := range result {
		if id, ok := s["id"].(string); ok && id == SeedBarakaID {
			t.Error("Baraka (no badges) should not appear in multi-skill filter")
		}
	}
}

func TestSeedDemo_SearchStudents_NoMatchReturnsEmpty(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	// Rust badge has never been awarded by any seeded event.
	req := httptest.NewRequest(http.MethodGet,
		"/api/users/students?skill_id="+SeedSkillRustID, nil)
	req = ctxWithUser(req, SeedCompanyID, "company")
	rec := httptest.NewRecorder()
	srv.SearchStudents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var result []map[string]any
	json.NewDecoder(rec.Body).Decode(&result)
	if len(result) != 0 {
		t.Errorf("Rust filter: expected 0 results, got %d", len(result))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Event listing — company isolation
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_ListEvents_ContainsAllCompanies(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()
	srv.ListEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ListEvents: expected 200, got %d", rec.Code)
	}
	var events []map[string]any
	json.NewDecoder(rec.Body).Decode(&events)
	if len(events) < 18 {
		t.Errorf("ListEvents: expected ≥18 events, got %d", len(events))
	}
}

func TestSeedDemo_GetEvent_GreenLeafWorkshop(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/events/"+SeedEventAgriWorkID, nil)
	req.SetPathValue("id", SeedEventAgriWorkID)
	rec := httptest.NewRecorder()
	srv.GetEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GetEvent (GreenLeaf): expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var event map[string]any
	json.NewDecoder(rec.Body).Decode(&event)
	if event["status"] != "active" {
		t.Errorf("GreenLeaf workshop should be active, got %v", event["status"])
	}
	skills, ok := event["skills"].([]any)
	if !ok || len(skills) < 2 {
		t.Errorf("GreenLeaf workshop: expected ≥2 skill links, got %v", event["skills"])
	}
}

func TestSeedDemo_GetEvent_MedConnectInternship(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/api/events/"+SeedEventMedInternID, nil)
	req.SetPathValue("id", SeedEventMedInternID)
	rec := httptest.NewRecorder()
	srv.GetEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GetEvent (MedConnect intern): expected 200, got %d", rec.Code)
	}
	var event map[string]any
	json.NewDecoder(rec.Body).Decode(&event)
	if event["status"] != "upcoming" {
		t.Errorf("MedConnect fellowship should be upcoming, got %v", event["status"])
	}
	// capacity=4, slots_remaining=3
	slots, _ := event["slots_remaining"].(float64)
	if slots != 3 {
		t.Errorf("MedConnect fellowship: expected 3 slots_remaining, got %v", slots)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Baraka starts fresh (no badges, not registered for anything)
// ─────────────────────────────────────────────────────────────────────────────

func TestSeedDemo_BarakaStartsFresh(t *testing.T) {
	srv := newTestServer(t)
	runSeed(t, srv)

	badges := dbInt(t, srv, `SELECT COUNT(*) FROM user_skills WHERE user_id=?`, SeedBarakaID)
	if badges != 0 {
		t.Errorf("Baraka should start with 0 badges, got %d", badges)
	}

	regs := dbInt(t, srv, `SELECT COUNT(*) FROM registrations WHERE student_id=?`, SeedBarakaID)
	if regs != 0 {
		t.Errorf("Baraka should have 0 registrations, got %d", regs)
	}
}
