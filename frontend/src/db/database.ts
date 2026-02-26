/**
 * database.ts — Local-first offline data layer using Dexie.js (IndexedDB)
 *
 * Replaces the old sql.js / localStorage implementation with a proper
 * IndexedDB schema so data survives service worker cache clears and is
 * not limited by the ~5 MB localStorage quota.
 *
 * Multi-user safety: every sync_queue row is tagged with user_id so that
 * if User A queues actions then logs out and User B logs in on the same
 * device, B's sync only sends B's queued rows.
 */

import Dexie, { type Table } from "dexie";

// ─── Domain types stored locally ─────────────────────────────────────────────

export interface CachedEvent {
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
  skills?: CachedSkill[];
  /** Epoch ms when this row was last fetched from the server. */
  _cached_at: number;
}

export interface CachedSkill {
  id: string;
  name: string;
  description: string;
  created_at: string;
  _cached_at: number;
}

export interface CachedRegistration {
  id: string;
  event_id: string;
  student_id: string;
  registered_at: string;
  status: "confirmed" | "conflict_pending" | "waitlisted";
  /** Denormalised event fields for display without a join. */
  event_title?: string;
  start_time?: string;
  end_time?: string;
  event_status?: string;
  location?: string;
  _cached_at: number;
}

export interface CachedUserSkill {
  id: string;
  user_id: string;
  skill_id: string;
  event_id: string;
  awarded_at: string;
  skill?: CachedSkill;
  _cached_at: number;
}

// ─── Sync queue ───────────────────────────────────────────────────────────────

export type SyncActionType =
  | "ATTENDANCE_CHECK_IN"
  | "REGISTER"
  | "UNREGISTER";

export type SyncStatus = "PENDING" | "SYNCING" | "VERIFIED" | "REJECTED";

export interface SyncQueueItem {
  /** Client-generated UUID — echoed back by the server so we can match results. */
  local_id: string;
  /** Tagged with the logged-in user so multi-user devices don't mix queues. */
  user_id: string;
  action: SyncActionType;
  event_id: string;
  /**
   * For ATTENDANCE_CHECK_IN: JSON.stringify({ token: "<jwt>" })
   * For REGISTER / UNREGISTER: empty string (no payload needed)
   */
  payload: string;
  status: SyncStatus;
  /** Date.now() when the action was queued. */
  queued_at: number;
  /** Human-readable rejection reason from the server, if any. */
  error_message?: string;
}

// ─── Dexie database class ─────────────────────────────────────────────────────

class SkillzoneDB extends Dexie {
  events!: Table<CachedEvent>;
  skills!: Table<CachedSkill>;
  registrations!: Table<CachedRegistration>;
  user_skills!: Table<CachedUserSkill>;
  sync_queue!: Table<SyncQueueItem>;

  constructor() {
    super("skillzone");

    this.version(1).stores({
      // Primary key first, then indexed fields
      events: "id, host_id, status, _cached_at",
      skills: "id, name, _cached_at",
      registrations: "id, event_id, student_id, status, _cached_at",
      user_skills: "id, user_id, skill_id, event_id, _cached_at",
      // local_id is the PK; compound index on [user_id+status] for efficient
      // "give me all PENDING items for this user" queries
      sync_queue: "local_id, user_id, [user_id+status], action, status, queued_at",
    });
  }
}

export const db = new SkillzoneDB();

// ─── Cache helpers ────────────────────────────────────────────────────────────

/** Replace the entire events cache with a fresh server response. */
export async function cacheEvents(events: CachedEvent[]): Promise<void> {
  const now = Date.now();
  const rows = events.map((e) => ({ ...e, _cached_at: now }));
  await db.transaction("rw", db.events, async () => {
    await db.events.clear();
    await db.events.bulkPut(rows);
  });
}

export async function cacheSkills(skills: CachedSkill[]): Promise<void> {
  const now = Date.now();
  const rows = skills.map((s) => ({ ...s, _cached_at: now }));
  await db.transaction("rw", db.skills, async () => {
    await db.skills.clear();
    await db.skills.bulkPut(rows);
  });
}

export async function cacheRegistrations(
  regs: CachedRegistration[]
): Promise<void> {
  const now = Date.now();
  const rows = regs.map((r) => ({ ...r, _cached_at: now }));
  await db.transaction("rw", db.registrations, async () => {
    await db.registrations.clear();
    await db.registrations.bulkPut(rows);
  });
}

export async function cacheUserSkills(
  skills: CachedUserSkill[]
): Promise<void> {
  const now = Date.now();
  const rows = skills.map((s) => ({ ...s, _cached_at: now }));
  await db.transaction("rw", db.user_skills, async () => {
    await db.user_skills.clear();
    await db.user_skills.bulkPut(rows);
  });
}

// ─── Sync queue helpers ───────────────────────────────────────────────────────

/** Enqueue an attendance check-in captured from a QR scan. */
export async function enqueueCheckIn(
  userId: string,
  eventId: string,
  token: string
): Promise<string> {
  const localId = crypto.randomUUID();
  await db.sync_queue.add({
    local_id: localId,
    user_id: userId,
    action: "ATTENDANCE_CHECK_IN",
    event_id: eventId,
    payload: JSON.stringify({ token }),
    status: "PENDING",
    queued_at: Date.now(),
  });
  return localId;
}

/** Enqueue an offline event registration. */
export async function enqueueRegister(
  userId: string,
  eventId: string
): Promise<string> {
  const localId = crypto.randomUUID();
  // Prevent duplicate queue entries for the same user+event+action
  const existing = await db.sync_queue
    .where("[user_id+status]")
    .equals([userId, "PENDING"])
    .filter((r) => r.action === "REGISTER" && r.event_id === eventId)
    .first();
  if (existing) return existing.local_id;

  await db.sync_queue.add({
    local_id: localId,
    user_id: userId,
    action: "REGISTER",
    event_id: eventId,
    payload: "",
    status: "PENDING",
    queued_at: Date.now(),
  });
  return localId;
}

/** Enqueue an offline event un-registration. */
export async function enqueueUnregister(
  userId: string,
  eventId: string
): Promise<string> {
  const localId = crypto.randomUUID();
  const existing = await db.sync_queue
    .where("[user_id+status]")
    .equals([userId, "PENDING"])
    .filter((r) => r.action === "UNREGISTER" && r.event_id === eventId)
    .first();
  if (existing) return existing.local_id;

  await db.sync_queue.add({
    local_id: localId,
    user_id: userId,
    action: "UNREGISTER",
    event_id: eventId,
    payload: "",
    status: "PENDING",
    queued_at: Date.now(),
  });
  return localId;
}

/** Return all PENDING sync items for a specific user (ordered oldest first). */
export async function getPendingItems(userId: string): Promise<SyncQueueItem[]> {
  return db.sync_queue
    .where("[user_id+status]")
    .equals([userId, "PENDING"])
    .sortBy("queued_at");
}

/** Mark a sync item's status and optionally record an error message. */
export async function updateSyncStatus(
  localId: string,
  status: SyncStatus,
  errorMessage?: string
): Promise<void> {
  await db.sync_queue.update(localId, { status, error_message: errorMessage });
}

/** Count pending items for a user — used to show the badge on the sync button. */
export async function countPending(userId: string): Promise<number> {
  return db.sync_queue
    .where("[user_id+status]")
    .equals([userId, "PENDING"])
    .count();
}
