/**
 * usePreemptiveCache.ts — Silent background pre-warming of the Dexie cache.
 *
 * Fires once on mount (if online) and again whenever the device comes back
 * online. Fetches events, skills, and (for students) registrations + user
 * skills, then stores them in Dexie so every page has fresh data even after
 * the network disappears.
 *
 * Rules:
 * - No UI loading states.
 * - No error toasts — failures are silently swallowed; stale cache is fine.
 * - Only student-specific endpoints are called when role === "student".
 */

import { useEffect } from "react";
import { apiListEvents, apiListSkills, apiGetMyRegistrations, apiGetMySkills } from "../lib/api";
import { cacheEvents, cacheSkills, cacheRegistrations, cacheUserSkills } from "../db/database";
import type { User } from "../lib/api";

async function warmCache(user: User): Promise<void> {
  if (!navigator.onLine) return;

  // Always fetch events and skills
  const [evts, skills] = await Promise.all([
    apiListEvents().catch(() => null),
    apiListSkills().catch(() => null),
  ]);

  if (evts)   await cacheEvents(evts as Parameters<typeof cacheEvents>[0]).catch(() => {});
  if (skills) await cacheSkills(skills as Parameters<typeof cacheSkills>[0]).catch(() => {});

  // Student-only: registrations + earned badges
  if (user.role === "student") {
    const [regs, userSkills] = await Promise.all([
      apiGetMyRegistrations().catch(() => null),
      apiGetMySkills().catch(() => null),
    ]);
    if (regs)       await cacheRegistrations(regs as Parameters<typeof cacheRegistrations>[0]).catch(() => {});
    if (userSkills) await cacheUserSkills(userSkills as Parameters<typeof cacheUserSkills>[0]).catch(() => {});
  }
}

/**
 * Mount this hook once inside AuthProvider (after the user is known).
 * Pass `null` when the user is not yet logged in — the hook is a no-op.
 */
export function usePreemptiveCache(user: User | null): void {
  useEffect(() => {
    if (!user) return;

    // Fire immediately on mount / user change
    warmCache(user).catch(() => {});

    // Re-fire whenever connectivity returns
    const handleOnline = () => warmCache(user).catch(() => {});
    window.addEventListener("online", handleOnline);
    return () => window.removeEventListener("online", handleOnline);
  }, [user]);
}
