// filepath: /home/ksilas/Documents/Projects/skillzone/backend/internal/handlers/seed.go
package handlers

// SeedDemo handles POST /api/admin/seed
//
// Idempotent demo seeder. Calls INSERT OR IGNORE so re-seeding a live server
// is harmless. All UUIDs are pre-determined constants.
//
// THREE HOSTING ORGANISATIONS:
//   TechCorp Africa   — technology & software   (host@techcorp.test / demo1234)
//   GreenLeaf Agri    — agriculture & sustainability (filler, no login)
//   MedConnect Health — healthcare & life sciences  (filler, no login)
//
// TWO DEMO-LOGIN STUDENTS:
//   Amara Osei    (amara@student.test  / demo1234) — veteran, 6 TechCorp events
//   Baraka Mwangi (baraka@student.test / demo1234) — newcomer, no history
//
// FILLER STUDENTS (no login):
//   TechCorp track : Chidi, Fatima, Kwame, Aisha, Tobi, Ngozi, Joel, Lila
//   GreenLeaf track: Zara, Emeka, Sade, Kofi
//   MedConnect track: Muna, Dayo, Nia
//
// SKILLS — 37 badges spanning Technology, Agriculture, Healthcare, and Business.

import (
"net/http"
"time"

"github.com/Elizabethomito/skillzone/backend/internal/auth"
"golang.org/x/crypto/bcrypt"
)

// ─────────────────────────────────────────────────────────────────────────────
// Constants — all IDs are pre-determined so the seed is idempotent.
// ─────────────────────────────────────────────────────────────────────────────
const (
// Companies
SeedCompanyID    = "seed-company-00000000-0000-0000-0000-000000000001"
SeedGreenLeafID  = "seed-greenleaf-00000-0000-0000-0000-000000000002"
SeedMedConnectID = "seed-medconnect-0000-0000-0000-0000-000000000003"

// Loginable students
SeedAmaraID  = "seed-amara-000000000-0000-0000-0000-000000000010"
SeedBarakaID = "seed-baraka-00000000-0000-0000-0000-000000000011"

// Filler students — TechCorp track
SeedChidiID  = "seed-chidi-000000000-0000-0000-0000-000000000020"
SeedFatimaID = "seed-fatima-00000000-0000-0000-0000-000000000021"
SeedKwameID  = "seed-kwame-000000000-0000-0000-0000-000000000022"
SeedAishaID  = "seed-aisha-000000000-0000-0000-0000-000000000023"
SeedTobiID   = "seed-tobi-0000000000-0000-0000-0000-000000000024"
SeedNgoziID  = "seed-ngozi-000000000-0000-0000-0000-000000000025"
SeedJoelID   = "seed-joel-0000000000-0000-0000-0000-000000000026"
SeedLilaID   = "seed-lila-0000000000-0000-0000-0000-000000000027"

// Filler students — GreenLeaf track
SeedZaraID  = "seed-zara-0000000000-0000-0000-0000-000000000030"
SeedEmekaID = "seed-emeka-000000000-0000-0000-0000-000000000031"
SeedSadeID  = "seed-sade-0000000000-0000-0000-0000-000000000032"
SeedKofiID  = "seed-kofi-0000000000-0000-0000-0000-000000000033"

// Filler students — MedConnect track
SeedMunaID = "seed-muna-0000000000-0000-0000-0000-000000000040"
SeedDayoID = "seed-dayo-0000000000-0000-0000-0000-000000000041"
SeedNiaID  = "seed-nia-00000000000-0000-0000-0000-000000000042"

// Skills — Technology: Programming
SeedSkillPythonID     = "seed-skill-python--0000-0000-0000-000000000100"
SeedSkillJavaScriptID = "seed-skill-js------0000-0000-0000-000000000101"
SeedSkillTypeScriptID = "seed-skill-ts------0000-0000-0000-000000000102"
SeedSkillGoID         = "seed-skill-go------0000-0000-0000-000000000103"
SeedSkillRustID       = "seed-skill-rust----0000-0000-0000-000000000104"
SeedSkillSQLID        = "seed-skill-sql-----0000-0000-0000-000000000105"

// Skills — Technology: Data & ML
SeedSkillDataSciID = "seed-skill-datasci-0000-0000-0000-000000000110"
SeedSkillMLOpsID   = "seed-skill-mlops---0000-0000-0000-000000000111"
SeedSkillDataEngID = "seed-skill-dataeng-0000-0000-0000-000000000112"

// Skills — Technology: AI
SeedSkillAIDevID          = "seed-skill-aidev--0000-0000-0000-000000000120"
SeedSkillPromptID         = "seed-skill-prompt-0000-0000-0000-000000000121"
SeedSkillAIProdMgmtID     = "seed-skill-aiprod-0000-0000-0000-000000000122"
SeedSkillComputerVisionID = "seed-skill-cv----0000-0000-0000-000000000123"
SeedSkillNLPID            = "seed-skill-nlp----0000-0000-0000-000000000124"

// Skills — Technology: Infrastructure
SeedSkillCloudID      = "seed-skill-cloud--0000-0000-0000-000000000130"
SeedSkillDevOpsID     = "seed-skill-devops-0000-0000-0000-000000000131"
SeedSkillDockerID     = "seed-skill-docker-0000-0000-0000-000000000132"
SeedSkillCyberID      = "seed-skill-cyber--0000-0000-0000-000000000133"
SeedSkillOpenSourceID = "seed-skill-oss----0000-0000-0000-000000000134"

// Skills — Technology: Frontend & Mobile
SeedSkillMobileID    = "seed-skill-mobile-0000-0000-0000-000000000140"
SeedSkillReactID     = "seed-skill-react--0000-0000-0000-000000000141"
SeedSkillUIUXID      = "seed-skill-uiux---0000-0000-0000-000000000142"
SeedSkillAPIDesignID = "seed-skill-api----0000-0000-0000-000000000143"

// Skills — Agriculture & Sustainability
SeedSkillPrecisionAgriID  = "seed-skill-precagri-0000-0000-0000-000000000200"
SeedSkillSoilScienceID    = "seed-skill-soilsci-0000-0000-0000-000000000201"
SeedSkillClimateAdaptID   = "seed-skill-climate-0000-0000-0000-000000000202"
SeedSkillAgroprocessingID = "seed-skill-agropro-0000-0000-0000-000000000203"
SeedSkillFoodSafetyID     = "seed-skill-foodsaf-0000-0000-0000-000000000204"
SeedSkillWaterMgmtID      = "seed-skill-water---0000-0000-0000-000000000205"

// Skills — Healthcare & Life Sciences
SeedSkillHealthDataID       = "seed-skill-healthd-0000-0000-0000-000000000300"
SeedSkillPublicHealthID     = "seed-skill-pubhlth-0000-0000-0000-000000000301"
SeedSkillMedTechID          = "seed-skill-medtech-0000-0000-0000-000000000302"
SeedSkillClinicalResearchID = "seed-skill-clinres-0000-0000-0000-000000000303"
SeedSkillHealthcareAIID     = "seed-skill-hcai---0000-0000-0000-000000000304"

// Skills — Business & Soft Skills
SeedSkillEntrepreneurshipID = "seed-skill-entrepr-0000-0000-0000-000000000400"
SeedSkillProjectMgmtID     = "seed-skill-projmgt-0000-0000-0000-000000000401"

// Events — TechCorp Africa
SeedEventPythonID     = "seed-event-python-0000-0000-0000-000000000500"
SeedEventDataSciID    = "seed-event-datasi-0000-0000-0000-000000000501"
SeedEventHackID       = "seed-event-hack---0000-0000-0000-000000000502"
SeedEventCloudID      = "seed-event-cloud--0000-0000-0000-000000000503"
SeedEventMobileID     = "seed-event-mobile-0000-0000-0000-000000000504"
SeedEventCyberID      = "seed-event-cyber--0000-0000-0000-000000000505"
SeedEventAIWorkshopID = "seed-event-aiwork-0000-0000-0000-000000000506"
SeedEventInternshipID = "seed-event-intern-0000-0000-0000-000000000507"

// Events — GreenLeaf Agri
SeedEventSoilID       = "seed-event-soil---0000-0000-0000-000000000510"
SeedEventClimateID    = "seed-event-climat-0000-0000-0000-000000000511"
SeedEventAgriTechID   = "seed-event-agrtch-0000-0000-0000-000000000512"
SeedEventAgriWorkID   = "seed-event-agrwrk-0000-0000-0000-000000000513"
SeedEventAgriInternID = "seed-event-agrint-0000-0000-0000-000000000514"

// Events — MedConnect Health
SeedEventHealthDataID   = "seed-event-hlthdt-0000-0000-0000-000000000520"
SeedEventPublicHealthID = "seed-event-pubhlt-0000-0000-0000-000000000521"
SeedEventMedTechID      = "seed-event-medtch-0000-0000-0000-000000000522"
SeedEventMedWorkID      = "seed-event-medwrk-0000-0000-0000-000000000523"
SeedEventMedInternID    = "seed-event-medint-0000-0000-0000-000000000524"

// Check-in codes embedded in live QR codes
SeedAIWorkshopCheckInCode = "DEMO-CHECKIN-CODE-AI-WORKSHOP-2026"
SeedAgriWorkCheckInCode   = "DEMO-CHECKIN-CODE-AGRI-WORKSHOP-2026"
SeedMedWorkCheckInCode    = "DEMO-CHECKIN-CODE-MED-WORKSHOP-2026"
)

// SeedDemo handles POST /api/admin/seed
func (s *Server) SeedDemo(w http.ResponseWriter, r *http.Request) {
hash, err := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
if err != nil {
respondError(w, http.StatusInternalServerError, "bcrypt: "+err.Error())
return
}
pw := string(hash)

// Filler accounts use a random-looking password — they are never logged in.
fillerHash, err := bcrypt.GenerateFromPassword([]byte("filler-no-login"), bcrypt.DefaultCost)
if err != nil {
respondError(w, http.StatusInternalServerError, "bcrypt filler: "+err.Error())
return
}
filler := string(fillerHash)

now := time.Now().UTC()

// ── 1. Users ──────────────────────────────────────────────────────────────
type userRow struct{ id, email, name, role, pw string }
for _, u := range []userRow{
// Companies
{SeedCompanyID, "host@techcorp.test", "TechCorp Africa", "company", pw},
{SeedGreenLeafID, "host@greenleaf.test", "GreenLeaf Agri", "company", filler},
{SeedMedConnectID, "host@medconnect.test", "MedConnect Health", "company", filler},
// Demo-login students
{SeedAmaraID, "amara@student.test", "Amara Osei", "student", pw},
{SeedBarakaID, "baraka@student.test", "Baraka Mwangi", "student", pw},
// TechCorp filler
{SeedChidiID, "chidi@student.test", "Chidi Okafor", "student", filler},
{SeedFatimaID, "fatima@student.test", "Fatima Al-Hassan", "student", filler},
{SeedKwameID, "kwame@student.test", "Kwame Asante", "student", filler},
{SeedAishaID, "aisha@student.test", "Aisha Diallo", "student", filler},
{SeedTobiID, "tobi@student.test", "Tobi Adeyemi", "student", filler},
{SeedNgoziID, "ngozi@student.test", "Ngozi Eze", "student", filler},
{SeedJoelID, "joel@student.test", "Joel Mutua", "student", filler},
{SeedLilaID, "lila@student.test", "Lila Nkosi", "student", filler},
// GreenLeaf filler
{SeedZaraID, "zara@student.test", "Zara Mensah", "student", filler},
{SeedEmekaID, "emeka@student.test", "Emeka Chukwu", "student", filler},
{SeedSadeID, "sade@student.test", "Sade Bello", "student", filler},
{SeedKofiID, "kofi@student.test", "Kofi Boateng", "student", filler},
// MedConnect filler
{SeedMunaID, "muna@student.test", "Muna Abdi", "student", filler},
{SeedDayoID, "dayo@student.test", "Dayo Okonkwo", "student", filler},
{SeedNiaID, "nia@student.test", "Nia Kamau", "student", filler},
} {
s.DB.ExecContext(r.Context(),
`INSERT OR IGNORE INTO users (id, email, password_hash, name, role, created_at, updated_at)
 VALUES (?, ?, ?, ?, ?, ?, ?)`,
u.id, u.email, u.pw, u.name, u.role, now, now,
)
}

// ── 2. Skills catalogue ───────────────────────────────────────────────────
type skillRow struct{ id, name, desc string }
for _, sk := range []skillRow{
// Technology — Programming
{SeedSkillPythonID, "Python Programming", "Proficient in Python scripting and data manipulation"},
{SeedSkillJavaScriptID, "JavaScript Development", "Vanilla JS, ES2022+, browser and Node.js environments"},
{SeedSkillTypeScriptID, "TypeScript", "Type-safe JavaScript; interfaces, generics, and strict mode"},
{SeedSkillGoID, "Go Programming", "Concurrent systems programming and REST APIs in Go"},
{SeedSkillRustID, "Rust Systems Programming", "Memory-safe systems code with Rust ownership model"},
{SeedSkillSQLID, "SQL & Relational Databases", "Schema design, query optimisation, and transactions"},
// Technology — Data & ML
{SeedSkillDataSciID, "Data Science", "Applied machine learning and statistical analysis"},
{SeedSkillMLOpsID, "MLOps", "Deploying, monitoring, and retraining ML models in production"},
{SeedSkillDataEngID, "Data Engineering", "ETL pipelines, data warehouses, and streaming architectures"},
// Technology — AI
{SeedSkillAIDevID, "AI Application Development", "Building production AI-powered applications"},
{SeedSkillPromptID, "Prompt Engineering", "Designing effective prompts for large language models"},
{SeedSkillAIProdMgmtID, "AI Product Management", "Roadmapping and shipping AI-first products"},
{SeedSkillComputerVisionID, "Computer Vision", "Image classification, object detection, and segmentation"},
{SeedSkillNLPID, "Natural Language Processing", "Text classification, named entity recognition, and LLMs"},
// Technology — Infrastructure
{SeedSkillCloudID, "Cloud Computing", "AWS / GCP fundamentals and serverless architecture"},
{SeedSkillDevOpsID, "DevOps & CI/CD", "Automated pipelines, infrastructure-as-code, and SRE practices"},
{SeedSkillDockerID, "Docker & Containerisation", "Building, tagging, and running containerised workloads"},
{SeedSkillCyberID, "Cybersecurity Fundamentals", "Threat modelling, OWASP Top-10, secure coding"},
{SeedSkillOpenSourceID, "Open Source Contribution", "Contributed to public repositories during a hackathon"},
// Technology — Frontend & Mobile
{SeedSkillMobileID, "Mobile Development", "Cross-platform mobile apps with React Native"},
{SeedSkillReactID, "React Development", "Component design, hooks, state management, and performance"},
{SeedSkillUIUXID, "UI/UX Design", "User research, wireframing, prototyping, and usability testing"},
{SeedSkillAPIDesignID, "API Design & REST", "RESTful design principles, versioning, and OpenAPI specs"},
// Agriculture & Sustainability
{SeedSkillPrecisionAgriID, "Precision Agriculture", "GPS-guided farming, remote sensing, and yield optimisation"},
{SeedSkillSoilScienceID, "Soil Science & Health", "Soil composition analysis and sustainable nutrient management"},
{SeedSkillClimateAdaptID, "Climate-Smart Farming", "Adapting crop cycles and practices to climate variability"},
{SeedSkillAgroprocessingID, "Agro-processing & Value Chains", "Post-harvest handling, processing, and market linkages"},
{SeedSkillFoodSafetyID, "Food Safety & Quality", "Standards, traceability, and safe food-handling practices"},
{SeedSkillWaterMgmtID, "Water & Irrigation Management", "Efficient water use, drip irrigation, and watershed care"},
// Healthcare & Life Sciences
{SeedSkillHealthDataID, "Health Data Management", "Electronic health records, data standards, and FHIR"},
{SeedSkillPublicHealthID, "Public Health & Epidemiology", "Disease surveillance, outbreak response, and community health"},
{SeedSkillMedTechID, "Medical Technology", "Biomedical devices, point-of-care tools, and health hardware"},
{SeedSkillClinicalResearchID, "Clinical Research Fundamentals", "Trial design, ethics, data collection, and GCP guidelines"},
{SeedSkillHealthcareAIID, "AI in Healthcare", "Diagnostic ML models, clinical decision support systems"},
// Business & Soft Skills
{SeedSkillEntrepreneurshipID, "Entrepreneurship & Innovation", "Lean startup methods, business model canvas, pitching"},
{SeedSkillProjectMgmtID, "Project Management", "Agile/Scrum frameworks, sprint planning, and delivery metrics"},
} {
s.DB.ExecContext(r.Context(),
`INSERT OR IGNORE INTO skills (id, name, description, created_at) VALUES (?, ?, ?, ?)`,
sk.id, sk.name, sk.desc, now,
)
}

// ── 3. Events ─────────────────────────────────────────────────────────────
type eventRow struct {
id, hostID, title, desc, loc string
start, end                   time.Time
status, checkIn              string
capacity, slotsRemaining     interface{}
}

workshopStart := now.Add(-1 * time.Hour)
workshopEnd   := now.Add(7 * time.Hour)
internStart   := now.AddDate(0, 1, 0)
internEnd     := internStart.AddDate(0, 3, 0)

agriWorkStart  := now.Add(-2 * time.Hour)
agriWorkEnd    := now.Add(6 * time.Hour)
agriInternStart := now.AddDate(0, 1, 7)
agriInternEnd   := agriInternStart.AddDate(0, 2, 0)

medWorkStart   := now.Add(-90 * time.Minute)
medWorkEnd     := now.Add(390 * time.Minute)
medInternStart := now.AddDate(0, 1, 14)
medInternEnd   := medInternStart.AddDate(0, 2, 0)

for _, e := range []eventRow{
// TechCorp past events
{SeedEventPythonID, SeedCompanyID, "Intro to Python Workshop",
"Beginner-friendly Python programming workshop", "Zone01 Kisumu Lab A",
now.AddDate(0, -3, 0), now.AddDate(0, -3, 0).Add(4 * time.Hour),
"completed", "past-ci-python", nil, nil},
{SeedEventDataSciID, SeedCompanyID, "Data Science Bootcamp",
"Hands-on data science with pandas and scikit-learn", "Zone01 Kisumu Lab B",
now.AddDate(0, -2, 0), now.AddDate(0, -2, 0).Add(8 * time.Hour),
"completed", "past-ci-datascience", nil, nil},
{SeedEventHackID, SeedCompanyID, "Open Source Hackathon",
"48-hour open source contribution sprint", "Zone01 Kisumu Main Hall",
now.AddDate(0, 0, -42), now.AddDate(0, 0, -40),
"completed", "past-ci-hackathon", nil, nil},
{SeedEventCloudID, SeedCompanyID, "Cloud Computing Seminar",
"AWS and GCP fundamentals for developers", "Online + Zone01 Kisumu Lab A",
now.AddDate(0, -1, 0), now.AddDate(0, -1, 0).Add(3 * time.Hour),
"completed", "past-ci-cloud", nil, nil},
{SeedEventMobileID, SeedCompanyID, "Mobile Dev Internship",
"4-week intensive mobile development programme", "Zone01 Kisumu",
now.AddDate(0, 0, -14), now.AddDate(0, 0, -7),
"completed", "past-ci-mobile", nil, nil},
{SeedEventCyberID, SeedCompanyID, "Cybersecurity Workshop",
"Practical security: OWASP, CTF challenges", "Zone01 Kisumu Lab B",
now.AddDate(0, 0, -7), now.AddDate(0, 0, -7).Add(6 * time.Hour),
"completed", "past-ci-cyber", nil, nil},
// TechCorp active + upcoming
{SeedEventAIWorkshopID, SeedCompanyID, "Building Apps with AI Workshop",
"A full-day hands-on workshop covering LLM APIs, RAG pipelines, and shipping AI-powered PWAs.",
"Zone01 Kisumu Main Hall",
workshopStart, workshopEnd, "active", SeedAIWorkshopCheckInCode, nil, nil},
{SeedEventInternshipID, SeedCompanyID, "AI Product Internship",
"A 3-month paid internship building AI products. Only 1 slot remaining!",
"Zone01 Kisumu + Remote",
internStart, internEnd, "upcoming", "intern-ci-unused", 2, 1},
// GreenLeaf past events
{SeedEventSoilID, SeedGreenLeafID, "Soil Health & Regenerative Farming",
"Practical soil sampling, composting, and regenerative agriculture techniques",
"GreenLeaf Training Centre, Nakuru",
now.AddDate(0, -2, -10), now.AddDate(0, -2, -10).Add(6 * time.Hour),
"completed", "past-ci-soil", nil, nil},
{SeedEventClimateID, SeedGreenLeafID, "Climate-Smart Farming Seminar",
"Adapting crop calendars and irrigation to climate change projections",
"Egerton University Extension, Njoro",
now.AddDate(0, -1, -5), now.AddDate(0, -1, -5).Add(4 * time.Hour),
"completed", "past-ci-climate", nil, nil},
{SeedEventAgriTechID, SeedGreenLeafID, "Precision Agriculture & IoT Sensors",
"Using low-cost IoT sensors and mobile apps for yield optimisation",
"GreenLeaf Demo Farm, Nakuru",
now.AddDate(0, 0, -10), now.AddDate(0, 0, -10).Add(5 * time.Hour),
"completed", "past-ci-agritech", nil, nil},
// GreenLeaf active + upcoming
{SeedEventAgriWorkID, SeedGreenLeafID, "Agro-processing & Market Linkages Workshop",
"Post-harvest value addition, packaging standards, and connecting to urban markets",
"GreenLeaf Training Centre, Nakuru",
agriWorkStart, agriWorkEnd, "active", SeedAgriWorkCheckInCode, nil, nil},
{SeedEventAgriInternID, SeedGreenLeafID, "Sustainable Agriculture Internship",
"8-week placement on a model farm — soil management, crop rotation, and supply chain.",
"GreenLeaf Demo Farm, Nakuru",
agriInternStart, agriInternEnd, "upcoming", "agri-intern-ci-unused", 3, 2},
// MedConnect past events
{SeedEventHealthDataID, SeedMedConnectID, "Health Data Management Bootcamp",
"EHR systems, HL7 FHIR standards, and de-identification best practices",
"MedConnect Hub, Nairobi",
now.AddDate(0, -2, -15), now.AddDate(0, -2, -15).Add(7 * time.Hour),
"completed", "past-ci-healthdata", nil, nil},
{SeedEventPublicHealthID, SeedMedConnectID, "Public Health & Epidemiology Workshop",
"Disease surveillance methods, community health mapping, and outbreak reporting",
"Kenya Medical Training College, Nairobi",
now.AddDate(0, -1, -8), now.AddDate(0, -1, -8).Add(5 * time.Hour),
"completed", "past-ci-pubhealth", nil, nil},
{SeedEventMedTechID, SeedMedConnectID, "Medical Technology Innovation Sprint",
"Rapid prototyping of point-of-care diagnostic tools using open hardware",
"iHub Nairobi",
now.AddDate(0, 0, -12), now.AddDate(0, 0, -12).Add(8 * time.Hour),
"completed", "past-ci-medtech", nil, nil},
// MedConnect active + upcoming
{SeedEventMedWorkID, SeedMedConnectID, "AI in Healthcare: From Data to Diagnosis",
"Building clinical decision-support models with public health datasets",
"MedConnect Hub, Nairobi",
medWorkStart, medWorkEnd, "active", SeedMedWorkCheckInCode, nil, nil},
{SeedEventMedInternID, SeedMedConnectID, "Digital Health Innovation Fellowship",
"6-week fellowship combining clinical research fundamentals with health-tech prototyping.",
"MedConnect Hub, Nairobi + Remote",
medInternStart, medInternEnd, "upcoming", "med-intern-ci-unused", 4, 3},
} {
s.DB.ExecContext(r.Context(),
`INSERT OR IGNORE INTO events
 (id, host_id, title, description, location, start_time, end_time,
  status, check_in_code, capacity, slots_remaining, created_at, updated_at)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
e.id, e.hostID, e.title, e.desc, e.loc,
e.start, e.end, e.status, e.checkIn,
e.capacity, e.slotsRemaining, now, now,
)
}

// ── 4. Event–skill links ──────────────────────────────────────────────────
for _, link := range [][2]string{
// TechCorp
{SeedEventPythonID, SeedSkillPythonID},
{SeedEventDataSciID, SeedSkillDataSciID},
{SeedEventHackID, SeedSkillOpenSourceID},
{SeedEventCloudID, SeedSkillCloudID},
{SeedEventMobileID, SeedSkillMobileID},
{SeedEventCyberID, SeedSkillCyberID},
{SeedEventAIWorkshopID, SeedSkillAIDevID},
{SeedEventAIWorkshopID, SeedSkillPromptID},
{SeedEventInternshipID, SeedSkillAIProdMgmtID},
{SeedEventInternshipID, SeedSkillProjectMgmtID},
// GreenLeaf
{SeedEventSoilID, SeedSkillSoilScienceID},
{SeedEventClimateID, SeedSkillClimateAdaptID},
{SeedEventAgriTechID, SeedSkillPrecisionAgriID},
{SeedEventAgriWorkID, SeedSkillAgroprocessingID},
{SeedEventAgriWorkID, SeedSkillEntrepreneurshipID},
{SeedEventAgriInternID, SeedSkillFoodSafetyID},
{SeedEventAgriInternID, SeedSkillWaterMgmtID},
// MedConnect
{SeedEventHealthDataID, SeedSkillHealthDataID},
{SeedEventPublicHealthID, SeedSkillPublicHealthID},
{SeedEventMedTechID, SeedSkillMedTechID},
{SeedEventMedWorkID, SeedSkillHealthcareAIID},
{SeedEventMedWorkID, SeedSkillHealthDataID},
{SeedEventMedInternID, SeedSkillClinicalResearchID},
{SeedEventMedInternID, SeedSkillProjectMgmtID},
} {
s.DB.ExecContext(r.Context(),
`INSERT OR IGNORE INTO event_skills (event_id, skill_id) VALUES (?, ?)`,
link[0], link[1],
)
}

// ── 5. Attendance helpers ─────────────────────────────────────────────────
type attEntry struct {
studentID, eventID, skillID, ciCode string
when                                 time.Time
}

insertHistory := func(entries []attEntry) {
for i, e := range entries {
s.DB.ExecContext(r.Context(),
`INSERT OR IGNORE INTO registrations (id, event_id, student_id, registered_at, status)
 VALUES (?, ?, ?, ?, 'confirmed')`,
seedID(e.studentID+"-reg", i), e.eventID, e.studentID, e.when.Add(-24*time.Hour),
)
ciToken, _ := auth.GenerateCheckInTokenWithExpiry(
e.eventID, e.ciCode, s.Secret, e.when, e.when.Add(auth.CheckInTokenDuration),
)
s.DB.ExecContext(r.Context(),
`INSERT OR IGNORE INTO attendances (id, event_id, student_id, payload, status, created_at, updated_at)
 VALUES (?, ?, ?, ?, 'verified', ?, ?)`,
seedID(e.studentID+"-att", i), e.eventID, e.studentID,
`{"token":"`+ciToken+`"}`, e.when, e.when,
)
if e.skillID != "" {
s.DB.ExecContext(r.Context(),
`INSERT OR IGNORE INTO user_skills (id, user_id, skill_id, event_id, awarded_at)
 VALUES (?, ?, ?, ?, ?)`,
seedID(e.studentID+"-skl", i), e.studentID, e.skillID, e.eventID, e.when,
)
}
}
}

// ── Amara — 6 past TechCorp events ───────────────────────────────────────
insertHistory([]attEntry{
{SeedAmaraID, SeedEventPythonID, SeedSkillPythonID, "past-ci-python", now.AddDate(0, -3, 0).Add(2 * time.Hour)},
{SeedAmaraID, SeedEventDataSciID, SeedSkillDataSciID, "past-ci-datascience", now.AddDate(0, -2, 0).Add(3 * time.Hour)},
{SeedAmaraID, SeedEventHackID, SeedSkillOpenSourceID, "past-ci-hackathon", now.AddDate(0, 0, -41)},
{SeedAmaraID, SeedEventCloudID, SeedSkillCloudID, "past-ci-cloud", now.AddDate(0, -1, 0).Add(1 * time.Hour)},
{SeedAmaraID, SeedEventMobileID, SeedSkillMobileID, "past-ci-mobile", now.AddDate(0, 0, -13)},
{SeedAmaraID, SeedEventCyberID, SeedSkillCyberID, "past-ci-cyber", now.AddDate(0, 0, -7).Add(2 * time.Hour)},
})
// Amara pre-registered for today's workshop
s.DB.ExecContext(r.Context(),
`INSERT OR IGNORE INTO registrations (id, event_id, student_id, registered_at, status)
 VALUES (?, ?, ?, ?, 'confirmed')`,
"seed-amara-reg-aiworkshop-2026", SeedEventAIWorkshopID, SeedAmaraID, now.Add(-30*time.Minute),
)

// ── TechCorp filler students ──────────────────────────────────────────────
// Chidi — 5 events (near-veteran)
insertHistory([]attEntry{
{SeedChidiID, SeedEventPythonID, SeedSkillPythonID, "past-ci-python", now.AddDate(0, -3, 0).Add(3 * time.Hour)},
{SeedChidiID, SeedEventDataSciID, SeedSkillDataSciID, "past-ci-datascience", now.AddDate(0, -2, 0).Add(4 * time.Hour)},
{SeedChidiID, SeedEventHackID, SeedSkillOpenSourceID, "past-ci-hackathon", now.AddDate(0, 0, -41).Add(1 * time.Hour)},
{SeedChidiID, SeedEventCloudID, SeedSkillCloudID, "past-ci-cloud", now.AddDate(0, -1, 0).Add(2 * time.Hour)},
{SeedChidiID, SeedEventAIWorkshopID, SeedSkillAIDevID, SeedAIWorkshopCheckInCode, now.Add(-30 * time.Minute)},
})
// Fatima — 4 events
insertHistory([]attEntry{
{SeedFatimaID, SeedEventPythonID, SeedSkillPythonID, "past-ci-python", now.AddDate(0, -3, 0).Add(2 * time.Hour)},
{SeedFatimaID, SeedEventHackID, SeedSkillOpenSourceID, "past-ci-hackathon", now.AddDate(0, 0, -41).Add(2 * time.Hour)},
{SeedFatimaID, SeedEventMobileID, SeedSkillMobileID, "past-ci-mobile", now.AddDate(0, 0, -13).Add(1 * time.Hour)},
{SeedFatimaID, SeedEventAIWorkshopID, SeedSkillPromptID, SeedAIWorkshopCheckInCode, now.Add(-25 * time.Minute)},
})
// Kwame — 3 events (intermediate)
insertHistory([]attEntry{
{SeedKwameID, SeedEventDataSciID, SeedSkillDataSciID, "past-ci-datascience", now.AddDate(0, -2, 0).Add(5 * time.Hour)},
{SeedKwameID, SeedEventCloudID, SeedSkillCloudID, "past-ci-cloud", now.AddDate(0, -1, 0).Add(3 * time.Hour)},
{SeedKwameID, SeedEventCyberID, SeedSkillCyberID, "past-ci-cyber", now.AddDate(0, 0, -7).Add(3 * time.Hour)},
})
// Aisha — 3 events (intermediate, different mix)
insertHistory([]attEntry{
{SeedAishaID, SeedEventPythonID, SeedSkillPythonID, "past-ci-python", now.AddDate(0, -3, 0).Add(1 * time.Hour)},
{SeedAishaID, SeedEventMobileID, SeedSkillMobileID, "past-ci-mobile", now.AddDate(0, 0, -13).Add(2 * time.Hour)},
{SeedAishaID, SeedEventAIWorkshopID, SeedSkillAIDevID, SeedAIWorkshopCheckInCode, now.Add(-20 * time.Minute)},
})
// Tobi — 2 events
insertHistory([]attEntry{
{SeedTobiID, SeedEventHackID, SeedSkillOpenSourceID, "past-ci-hackathon", now.AddDate(0, 0, -41).Add(3 * time.Hour)},
{SeedTobiID, SeedEventCyberID, SeedSkillCyberID, "past-ci-cyber", now.AddDate(0, 0, -7).Add(4 * time.Hour)},
})
// Ngozi — 2 events
insertHistory([]attEntry{
{SeedNgoziID, SeedEventDataSciID, SeedSkillDataSciID, "past-ci-datascience", now.AddDate(0, -2, 0).Add(6 * time.Hour)},
{SeedNgoziID, SeedEventAIWorkshopID, SeedSkillPromptID, SeedAIWorkshopCheckInCode, now.Add(-15 * time.Minute)},
})
// Joel — 1 event (newcomer)
insertHistory([]attEntry{
{SeedJoelID, SeedEventAIWorkshopID, SeedSkillAIDevID, SeedAIWorkshopCheckInCode, now.Add(-10 * time.Minute)},
})
// Lila — 1 event (newcomer)
insertHistory([]attEntry{
{SeedLilaID, SeedEventCyberID, SeedSkillCyberID, "past-ci-cyber", now.AddDate(0, 0, -7).Add(5 * time.Hour)},
})

// ── GreenLeaf filler students ─────────────────────────────────────────────
// Zara — 3 events (experienced agri student)
insertHistory([]attEntry{
{SeedZaraID, SeedEventSoilID, SeedSkillSoilScienceID, "past-ci-soil", now.AddDate(0, -2, -10).Add(2 * time.Hour)},
{SeedZaraID, SeedEventClimateID, SeedSkillClimateAdaptID, "past-ci-climate", now.AddDate(0, -1, -5).Add(1 * time.Hour)},
{SeedZaraID, SeedEventAgriTechID, SeedSkillPrecisionAgriID, "past-ci-agritech", now.AddDate(0, 0, -10).Add(2 * time.Hour)},
})
// Emeka — 2 events
insertHistory([]attEntry{
{SeedEmekaID, SeedEventSoilID, SeedSkillSoilScienceID, "past-ci-soil", now.AddDate(0, -2, -10).Add(3 * time.Hour)},
{SeedEmekaID, SeedEventAgriWorkID, SeedSkillAgroprocessingID, SeedAgriWorkCheckInCode, now.Add(-90 * time.Minute)},
})
// Sade — 2 events
insertHistory([]attEntry{
{SeedSadeID, SeedEventClimateID, SeedSkillClimateAdaptID, "past-ci-climate", now.AddDate(0, -1, -5).Add(2 * time.Hour)},
{SeedSadeID, SeedEventAgriWorkID, SeedSkillEntrepreneurshipID, SeedAgriWorkCheckInCode, now.Add(-85 * time.Minute)},
})
// Kofi — 1 event (newcomer)
insertHistory([]attEntry{
{SeedKofiID, SeedEventAgriWorkID, SeedSkillAgroprocessingID, SeedAgriWorkCheckInCode, now.Add(-80 * time.Minute)},
})

// ── MedConnect filler students ────────────────────────────────────────────
// Muna — 3 events (experienced health student)
insertHistory([]attEntry{
{SeedMunaID, SeedEventHealthDataID, SeedSkillHealthDataID, "past-ci-healthdata", now.AddDate(0, -2, -15).Add(3 * time.Hour)},
{SeedMunaID, SeedEventPublicHealthID, SeedSkillPublicHealthID, "past-ci-pubhealth", now.AddDate(0, -1, -8).Add(2 * time.Hour)},
{SeedMunaID, SeedEventMedTechID, SeedSkillMedTechID, "past-ci-medtech", now.AddDate(0, 0, -12).Add(3 * time.Hour)},
})
// Dayo — 2 events
insertHistory([]attEntry{
{SeedDayoID, SeedEventHealthDataID, SeedSkillHealthDataID, "past-ci-healthdata", now.AddDate(0, -2, -15).Add(4 * time.Hour)},
{SeedDayoID, SeedEventMedWorkID, SeedSkillHealthcareAIID, SeedMedWorkCheckInCode, now.Add(-60 * time.Minute)},
})
// Nia — 1 event (newcomer)
insertHistory([]attEntry{
{SeedNiaID, SeedEventMedWorkID, SeedSkillHealthcareAIID, SeedMedWorkCheckInCode, now.Add(-55 * time.Minute)},
})

// ── 6. Response ───────────────────────────────────────────────────────────
respond(w, http.StatusOK, map[string]any{
"seeded": true,
"accounts": []map[string]string{
{"role": "company", "email": "host@techcorp.test", "password": "demo1234", "name": "TechCorp Africa"},
{"role": "student", "email": "amara@student.test", "password": "demo1234", "name": "Amara Osei (veteran)"},
{"role": "student", "email": "baraka@student.test", "password": "demo1234", "name": "Baraka Mwangi (newcomer)"},
},
"companies": []map[string]string{
{"id": SeedCompanyID, "name": "TechCorp Africa", "domain": "Technology"},
{"id": SeedGreenLeafID, "name": "GreenLeaf Agri", "domain": "Agriculture & Sustainability"},
{"id": SeedMedConnectID, "name": "MedConnect Health", "domain": "Healthcare & Life Sciences"},
},
"active_workshop": map[string]string{
"event_id": SeedEventAIWorkshopID, "check_in_code": SeedAIWorkshopCheckInCode,
"title": "Building Apps with AI Workshop", "host": "TechCorp Africa",
},
"active_agri_workshop": map[string]string{
"event_id": SeedEventAgriWorkID, "check_in_code": SeedAgriWorkCheckInCode,
"title": "Agro-processing & Market Linkages Workshop", "host": "GreenLeaf Agri",
},
"active_med_workshop": map[string]string{
"event_id": SeedEventMedWorkID, "check_in_code": SeedMedWorkCheckInCode,
"title": "AI in Healthcare: From Data to Diagnosis", "host": "MedConnect Health",
},
"internship": map[string]any{
"event_id": SeedEventInternshipID, "title": "AI Product Internship", "slots_remaining": 1,
},
})
}

// seedID generates a stable deterministic ID for seeded rows.
func seedID(prefix string, n int) string {
s := prefix
for len(s) < 30 {
s += "-0"
}
return s[:30] + itoa(int64(n)) + "x"
}

// itoa converts an int64 to a decimal string without importing strconv.
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
