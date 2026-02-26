/**
 * api.ts — Typed HTTP client for the Skillzone backend.
 *
 * All functions that modify state (register, sync, etc.) are designed to
 * be called whether the device is online or offline: they attempt the
 * network request and, if it fails, queue the action in Dexie for later sync.
 */

// If an explicit override is set (e.g. for production), use it.
// Otherwise derive the backend URL from the page's own origin so that
// external devices on the same LAN automatically hit the right host
// without needing VITE_API_URL to be set.
// const BASE: string = (() => {
//   if (import.meta.env.VITE_API_URL) return import.meta.env.VITE_API_URL as string;
//   // In dev the frontend runs on :5173 and the backend on :8080 — same host.
//   // In production the backend serves the built frontend, so same origin.
//   const { protocol, hostname } = window.location;
//   const backendPort = import.meta.env.VITE_BACKEND_PORT ?? "8080";
//   return `${protocol}//${hostname}:${backendPort}`;
// })();
// If VITE_API_URL is set (e.g. in Production), use it. 
// Otherwise, default to an empty string to use relative paths so the Vite proxy catches it.
const BASE: string = import.meta.env.VITE_API_URL || "";

// ─── Token storage ────────────────────────────────────────────────────────────

let _token: string | null = localStorage.getItem("sz_token");

export function setToken(token: string | null) {
  _token = token;
  if (token) localStorage.setItem("sz_token", token);
  else localStorage.removeItem("sz_token");
}

export function getToken(): string | null {
  return _token;
}

// ─── Core fetch wrapper ───────────────────────────────────────────────────────

async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (_token) headers["Authorization"] = `Bearer ${_token}`;

  const res = await fetch(`${BASE}${path}`, { ...options, headers });

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body?.error) msg = body.error;
    } catch {}
    throw new Error(msg);
  }

  // 204 No Content
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

export interface User {
  id: string;
  email: string;
  name: string;
  role: "student" | "company";
  created_at: string;
  updated_at: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export async function apiRegister(
  email: string,
  password: string,
  name: string,
  role: "student" | "company"
): Promise<LoginResponse> {
  return apiFetch("/api/auth/register", {
    method: "POST",
    body: JSON.stringify({ email, password, name, role }),
  });
}

export async function apiLogin(
  email: string,
  password: string
): Promise<LoginResponse> {
  return apiFetch("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export async function apiMe(): Promise<User> {
  return apiFetch("/api/auth/me");
}

// ─── Events ───────────────────────────────────────────────────────────────────

export interface Skill {
  id: string;
  name: string;
  description: string;
  created_at: string;
}

export interface ApiEvent {
  id: string;
  host_id: string;
  title: string;
  description: string;
  location: string;
  start_time: string;
  end_time: string;
  status: "upcoming" | "active" | "completed";
  capacity?: number;
  slots_remaining?: number;
  created_at: string;
  updated_at: string;
  skills?: Skill[];
}

export async function apiListEvents(): Promise<ApiEvent[]> {
  return apiFetch("/api/events");
}

export async function apiGetEvent(id: string): Promise<ApiEvent> {
  return apiFetch(`/api/events/${id}`);
}

export interface CreateEventPayload {
  title: string;
  description: string;
  location: string;
  start_time: string;
  end_time: string;
  skill_ids: string[];
  capacity?: number;
}

export async function apiCreateEvent(
  payload: CreateEventPayload
): Promise<ApiEvent> {
  return apiFetch("/api/events", { method: "POST", body: JSON.stringify(payload) });
}

export async function apiUpdateEvent(
  id: string,
  payload: Partial<CreateEventPayload>
): Promise<ApiEvent> {
  return apiFetch(`/api/events/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export async function apiUpdateEventStatus(
  id: string,
  status: "upcoming" | "active" | "completed"
): Promise<{ event_id: string; status: string }> {
  return apiFetch(`/api/events/${id}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });
}

export interface CheckinCodeResponse {
  event_id: string;
  token: string;
  expires_in_seconds: number;
}

export async function apiGetCheckinCode(
  eventId: string
): Promise<CheckinCodeResponse> {
  return apiFetch(`/api/events/${eventId}/checkin-code`);
}

// ─── Registrations ────────────────────────────────────────────────────────────

export interface Registration {
  id: string;
  event_id: string;
  student_id: string;
  registered_at: string;
  status: "confirmed" | "conflict_pending" | "waitlisted" | "rejected";
}

export interface RegistrationWithEvent extends Registration {
  event_title: string;
  start_time: string;
  end_time: string;
  event_status: string;
  location: string;
}

export interface RegistrationWithStudent extends Registration {
  student_name: string;
  student_email: string;
}

export async function apiRegisterForEvent(
  eventId: string
): Promise<Registration> {
  return apiFetch(`/api/events/${eventId}/register`, { method: "POST" });
}

export async function apiUnregisterFromEvent(eventId: string): Promise<void> {
  return apiFetch(`/api/events/${eventId}/register`, { method: "DELETE" });
}

export async function apiGetEventRegistrations(
  eventId: string
): Promise<RegistrationWithStudent[]> {
  return apiFetch(`/api/events/${eventId}/registrations`);
}

export async function apiResolveConflict(
  eventId: string,
  regId: string,
  action: "confirm" | "waitlist"
): Promise<{ registration_id: string; status: string }> {
  return apiFetch(`/api/events/${eventId}/registrations/${regId}`, {
    method: "PATCH",
    body: JSON.stringify({ action }),
  });
}

export async function apiKickRegistration(
  eventId: string,
  regId: string
): Promise<void> {
  return apiFetch(`/api/events/${eventId}/registrations/${regId}`, {
    method: "DELETE",
  });
}

export async function apiReAddRegistration(
  eventId: string,
  regId: string
): Promise<{ registration_id: string; status: string }> {
  return apiFetch(`/api/events/${eventId}/registrations/${regId}/readd`, {
    method: "POST",
  });
}

export async function apiGetMyRegistrations(): Promise<
  RegistrationWithEvent[]
> {
  return apiFetch("/api/users/me/registrations");
}

// ─── Skills ───────────────────────────────────────────────────────────────────

export async function apiListSkills(): Promise<Skill[]> {
  return apiFetch("/api/skills");
}

export async function apiCreateSkill(
  name: string,
  description: string
): Promise<Skill> {
  return apiFetch("/api/skills", {
    method: "POST",
    body: JSON.stringify({ name, description }),
  });
}

// ─── User skills ──────────────────────────────────────────────────────────────

export interface UserSkill {
  id: string;
  user_id: string;
  skill_id: string;
  event_id: string;
  awarded_at: string;
  skill?: Skill;
}

export async function apiGetMySkills(): Promise<UserSkill[]> {
  return apiFetch("/api/users/me/skills");
}

// ─── Public profile (recruiter-facing) ───────────────────────────────────────

export interface PublicBadge {
  id: string;
  skill_id: string;
  skill_name: string;
  skill_description: string;
  event_id: string;
  event_title: string;
  event_start: string;
  event_end: string;
  event_location: string;
  host_id: string;
  host_name: string;
  host_email: string;
  awarded_at: string;
}

export interface PublicProfile {
  id: string;
  name: string;
  email: string;
  role: string;
  badges: PublicBadge[];
}

export async function apiGetPublicProfile(userId: string): Promise<PublicProfile> {
  return apiFetch(`/api/users/${userId}/profile`);
}

// ─── Candidate search (company) ───────────────────────────────────────────────

export interface StudentWithSkills extends User {
  skills: UserSkill[];
}

export async function apiSearchStudents(
  skillId?: string
): Promise<StudentWithSkills[]> {
  const qs = skillId ? `?skill_id=${encodeURIComponent(skillId)}` : "";
  return apiFetch(`/api/users/students${qs}`);
}

// ─── Attendance sync ──────────────────────────────────────────────────────────

export interface SyncAttendanceRecord {
  local_id: string;
  event_id: string;
  payload: string;
}

export interface SyncResult {
  local_id: string;
  status: "verified" | "rejected" | "pending";
  message?: string;
}

export interface SyncAttendanceResponse {
  results: SyncResult[];
}

export async function apiSyncAttendance(
  records: SyncAttendanceRecord[]
): Promise<SyncAttendanceResponse> {
  return apiFetch("/api/sync/attendance", {
    method: "POST",
    body: JSON.stringify({ records }),
  });
}
