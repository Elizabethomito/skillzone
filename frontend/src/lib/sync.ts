/**
 * sync.ts — Background sync engine.
 *
 * Drains the Dexie sync_queue for the currently logged-in user by
 * calling the appropriate backend endpoints. Safe to call multiple times;
 * already-processed items are skipped.
 *
 * Phase 4: after a successful sync run, invalidates React Query caches for
 * "events", "my-registrations", and "my-skills" so every page refetches
 * fresh data automatically.
 */

import {
  db,
  getPendingItems,
  updateSyncStatus,
  type SyncQueueItem,
} from "../db/database";
import {
  apiSyncAttendance,
  apiRegisterForEvent,
  apiUnregisterFromEvent,
} from "./api";
import type { QueryClient } from "@tanstack/react-query";

// Module-level QueryClient reference — set once from App.tsx / AuthProvider.
let _queryClient: QueryClient | null = null;

/** Call this once after the QueryClient is created (e.g. in App.tsx). */
export function setQueryClient(qc: QueryClient): void {
  _queryClient = qc;
}

export type SyncEventName =
  | "sync:start"
  | "sync:item:ok"
  | "sync:item:fail"
  | "sync:done";

// A tiny event bus so UI components can react to sync progress
const listeners = new Map<SyncEventName, Set<(data?: unknown) => void>>();

export function onSync(
  event: SyncEventName,
  cb: (data?: unknown) => void
): () => void {
  if (!listeners.has(event)) listeners.set(event, new Set());
  listeners.get(event)!.add(cb);
  return () => listeners.get(event)!.delete(cb);
}

function emit(event: SyncEventName, data?: unknown) {
  listeners.get(event)?.forEach((cb) => cb(data));
}

let syncing = false;

/**
 * Run a full sync for the given user.
 * Attendance check-ins are batched into one request; register/unregister
 * are sent individually (each needs its own REST endpoint call).
 */
export async function runSync(userId: string): Promise<void> {
  if (syncing) return;
  syncing = true;
  emit("sync:start");

  try {
    const pending = await getPendingItems(userId);
    if (pending.length === 0) {
      emit("sync:done");
      return;
    }

    // Mark all as SYNCING so they don't get picked up again mid-flight
    await Promise.all(
      pending.map((item) => updateSyncStatus(item.local_id, "SYNCING"))
    );

    // ── Batch attendance records ──────────────────────────────────────────
    const checkIns = pending.filter((i) => i.action === "ATTENDANCE_CHECK_IN");
    if (checkIns.length > 0) {
      try {
        const response = await apiSyncAttendance(
          checkIns.map((i) => ({
            local_id: i.local_id,
            event_id: i.event_id,
            payload: i.payload,
          }))
        );
        for (const result of response.results) {
          if (result.status === "verified") {
            await updateSyncStatus(result.local_id, "VERIFIED");
            emit("sync:item:ok", result);
          } else {
            await updateSyncStatus(result.local_id, "REJECTED", result.message);
            emit("sync:item:fail", result);
          }
        }
      } catch (err) {
        // Network failure — revert to PENDING for retry
        for (const item of checkIns) {
          await updateSyncStatus(item.local_id, "PENDING");
        }
        throw err; // rethrow so the caller knows sync failed
      }
    }

    // ── Individual register actions ───────────────────────────────────────
    const registers = pending.filter((i) => i.action === "REGISTER");
    for (const item of registers) {
      try {
        await apiRegisterForEvent(item.event_id);
        await updateSyncStatus(item.local_id, "VERIFIED");
        emit("sync:item:ok", item);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "unknown error";
        // 409 = already registered — treat as success
        if (msg.includes("409") || msg.toLowerCase().includes("already")) {
          await updateSyncStatus(item.local_id, "VERIFIED");
          emit("sync:item:ok", item);
        } else {
          await updateSyncStatus(item.local_id, "REJECTED", msg);
          emit("sync:item:fail", { ...item, message: msg });
        }
      }
    }

    // ── Individual unregister actions ─────────────────────────────────────
    const unregisters = pending.filter((i) => i.action === "UNREGISTER");
    for (const item of unregisters) {
      try {
        await apiUnregisterFromEvent(item.event_id);
        await updateSyncStatus(item.local_id, "VERIFIED");
        emit("sync:item:ok", item);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "unknown error";
        // 404 = already unregistered — treat as success
        if (msg.includes("404") || msg.includes("not found")) {
          await updateSyncStatus(item.local_id, "VERIFIED");
          emit("sync:item:ok", item);
        } else {
          await updateSyncStatus(item.local_id, "REJECTED", msg);
          emit("sync:item:fail", { ...item, message: msg });
        }
      }
    }
  } finally {
    syncing = false;
    emit("sync:done");

    // Phase 4 — Invalidate React Query caches so all pages see fresh data
    // after a sync run (regardless of whether items succeeded or failed).
    if (_queryClient) {
      _queryClient.invalidateQueries({ queryKey: ["events"] });
      _queryClient.invalidateQueries({ queryKey: ["my-registrations"] });
      _queryClient.invalidateQueries({ queryKey: ["my-skills"] });
    }
  }
}

/** Wire up online/offline events to trigger auto-sync.
 *  Returns a cleanup function — call it to remove the listener.
 *  (Prevents duplicate listeners when AuthContext re-renders.) */
export function initSyncListener(getUserId: () => string | null): () => void {
  const handler = () => {
    const uid = getUserId();
    if (uid) runSync(uid).catch(console.warn);
  };
  window.addEventListener("online", handler);
  return () => window.removeEventListener("online", handler);
}
