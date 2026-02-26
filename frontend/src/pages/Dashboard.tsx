/**
 * Dashboard.tsx — Role-aware dashboard backed by the Skillzone REST API.
 *
 * Student view  : skill badges earned, event registrations (upcoming + past).
 * Company view  : events hosted, registration counts, quick link to Candidates.
 *
 * Offline behaviour: data is loaded from the API when online; the last-fetched
 * values remain visible while offline (React state survives navigation).
 */

import { useEffect } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { db, cacheUserSkills, cacheRegistrations } from "../db/database";
import {
  apiGetMySkills,
  apiGetMyRegistrations,
  apiListEvents,
  type UserSkill,
  type RegistrationWithEvent,
  type ApiEvent,
} from "../lib/api";
import {
  Award,
  Calendar,
  CheckCircle,
  Clock,
  BarChart3,
  Users,
  Wifi,
  WifiOff,
  ExternalLink,
} from "lucide-react";
import { useOnlineStatus } from "../hooks/useOnlineStatus";
import { useQuery, useQueryClient } from "@tanstack/react-query";

// ─── Shared stat card ─────────────────────────────────────────────────────────

function StatCard({
  icon: Icon,
  label,
  value,
  color = "bg-accent text-accent-foreground",
}: {
  icon: React.ElementType;
  label: string;
  value: number | string;
  color?: string;
}) {
  return (
    <div className="rounded-2xl bg-card p-6 card-shadow">
      <div className="flex items-center gap-4">
        <div
          className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-xl ${color}`}
        >
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <p className="text-2xl font-bold text-foreground">{value}</p>
          <p className="text-sm text-muted-foreground">{label}</p>
        </div>
      </div>
    </div>
  );
}

// ─── Student dashboard ────────────────────────────────────────────────────────

function StudentDashboard({ userId }: { userId: string }) {
  const online = useOnlineStatus();
  const queryClient = useQueryClient();

  // ── User skills (SWR: network-first, Dexie as placeholder) ───────────────
  const { data: skills = [], isLoading: skillsLoading } = useQuery<UserSkill[]>({
    queryKey: ["my-skills", userId],
    networkMode: "offlineFirst",
    staleTime: 30_000,
    queryFn: async () => {
      const s = await apiGetMySkills();
      cacheUserSkills(s as any).catch(() => {});
      return s;
    },
    placeholderData: () => {
      // Will be replaced by the real async seed below if query is still loading
      return [] as UserSkill[];
    },
  });

  // ── My registrations (SWR: network-first, Dexie as placeholder) ──────────
  const { data: regs = [], isLoading: regsLoading } = useQuery<RegistrationWithEvent[]>({
    queryKey: ["my-registrations"],
    networkMode: "offlineFirst",
    staleTime: 30_000,
    queryFn: async () => {
      const r = await apiGetMyRegistrations();
      cacheRegistrations(r as any).catch(() => {});
      return r;
    },
    placeholderData: () => [] as RegistrationWithEvent[],
  });

  const loading = skillsLoading || regsLoading;

  // Seed from Dexie while offline and query has no data yet
  useEffect(() => {
    if (online || (skills.length > 0 && regs.length > 0)) return;
    db.user_skills.toArray().then((cached) => {
      if (cached.length > 0) {
        queryClient.setQueryData(["my-skills", userId], cached as unknown as UserSkill[]);
      }
    }).catch(() => {});
    db.registrations.toArray().then((cached) => {
      if (cached.length > 0) {
        queryClient.setQueryData(["my-registrations"], cached as unknown as RegistrationWithEvent[]);
      }
    }).catch(() => {});
  }, [online, skills.length, regs.length, userId, queryClient]);

  const upcoming = regs.filter(
    (r) => r.event_status === "upcoming" || r.event_status === "active"
  );
  const past = regs.filter((r) => r.event_status === "completed");

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <>
      {/* Stats row */}
      <div className="mb-8 grid gap-4 sm:grid-cols-3">
        <StatCard
          icon={Award}
          label="Skills Earned"
          value={skills.length}
          color="bg-primary text-primary-foreground"
        />
        <StatCard icon={CheckCircle} label="Events Completed" value={past.length} />
        <StatCard icon={Clock} label="Upcoming / Active" value={upcoming.length} />
      </div>

      {/* Skill badges */}
      <div className="mb-8">
        <h3 className="mb-4 text-lg font-semibold text-foreground">
          Skill Badges
        </h3>
        {skills.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">
            No badges yet — attend events to earn skill badges!
          </p>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {skills.map((us) => (
              <div key={us.id} className="rounded-xl bg-card p-5 card-shadow">
                <div className="mb-2 flex items-center gap-2">
                  <Award className="h-5 w-5 text-primary" />
                  <span className="font-semibold text-foreground">
                    {us.skill?.name ?? us.skill_id}
                  </span>
                </div>
                <p className="text-xs text-muted-foreground">
                  Awarded {new Date(us.awarded_at).toLocaleDateString()}
                </p>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Upcoming / active registrations */}
      <div className="mb-8">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-foreground">
            Registered Events
          </h3>
          <Link
            to="/events"
            className="flex items-center gap-1 text-sm text-primary hover:underline"
          >
            Browse all <ExternalLink className="h-3 w-3" />
          </Link>
        </div>
        {upcoming.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">
            No upcoming events. Register from the{" "}
            <Link to="/events" className="text-primary hover:underline">
              Events page
            </Link>
            .
          </p>
        ) : (
          <div className="space-y-3">
            {upcoming.map((r) => (
              <div
                key={r.id}
                className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow"
              >
                <div>
                  <p className="font-medium text-foreground">{r.event_title}</p>
                  <p className="text-xs text-muted-foreground">
                    {new Date(r.start_time).toLocaleDateString()} ·{" "}
                    {r.location}
                  </p>
                </div>
                <span
                  className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
                    r.event_status === "active"
                      ? "bg-green-100 text-green-700"
                      : "bg-accent text-accent-foreground"
                  }`}
                >
                  {r.status}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Past registrations */}
      <div>
        <h3 className="mb-4 text-lg font-semibold text-foreground">
          Completed Events
        </h3>
        {past.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">
            No completed events yet.
          </p>
        ) : (
          <div className="space-y-3">
            {past.map((r) => (
              <div
                key={r.id}
                className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow"
              >
                <div>
                  <p className="font-medium text-foreground">{r.event_title}</p>
                  <p className="text-xs text-muted-foreground">
                    {new Date(r.start_time).toLocaleDateString()} ·{" "}
                    {r.location}
                  </p>
                </div>
                <CheckCircle className="h-5 w-5 text-green-600" />
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Offline indicator */}
      {!online && (
        <p className="mt-6 flex items-center gap-2 text-sm text-amber-600">
          <WifiOff className="h-4 w-4" /> Showing cached data — reconnect to
          refresh.
        </p>
      )}
    </>
  );
}

// ─── Company dashboard ────────────────────────────────────────────────────────

function CompanyDashboard({ userId }: { userId: string }) {
  const online = useOnlineStatus();

  // Re-uses the same ["events"] query key as Events.tsx — data is already
  // in the React Query cache if the user visited Events first.
  const { data: allEvents = [], isLoading: loading } = useQuery<ApiEvent[]>({
    queryKey: ["events"],
    networkMode: "offlineFirst",
    staleTime: 30_000,
    queryFn: async () => {
      const evts = await apiListEvents();
      return evts;
    },
    placeholderData: () => [] as ApiEvent[],
  });

  const events   = allEvents.filter((e) => e.host_id === userId);
  const active   = events.filter((e) => e.status === "active");
  const upcoming = events.filter((e) => e.status === "upcoming");
  const completed = events.filter((e) => e.status === "completed");

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <>
      {/* Stats row */}
      <div className="mb-8 grid gap-4 sm:grid-cols-3">
        <StatCard
          icon={Calendar}
          label="Total Events"
          value={events.length}
          color="bg-primary text-primary-foreground"
        />
        <StatCard
          icon={BarChart3}
          label="Active Now"
          value={active.length}
          color="bg-green-100 text-green-700"
        />
        <StatCard icon={Users} label="Upcoming" value={upcoming.length} />
      </div>

      {/* Quick links */}
      <div className="mb-8 flex flex-wrap gap-3">
        <Link
          to="/events"
          className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Calendar className="h-4 w-4" /> Manage Events
        </Link>
        <Link
          to="/candidates"
          className="inline-flex items-center gap-2 rounded-xl border border-border bg-card px-4 py-2 text-sm font-medium text-foreground hover:bg-muted"
        >
          <Users className="h-4 w-4" /> Search Candidates
        </Link>
      </div>

      {/* Active events */}
      {active.length > 0 && (
        <div className="mb-8">
          <h3 className="mb-4 text-lg font-semibold text-foreground">
            🔴 Live Events
          </h3>
          <div className="space-y-3">
            {active.map((ev) => (
              <EventRow key={ev.id} event={ev} />
            ))}
          </div>
        </div>
      )}

      {/* Upcoming events */}
      <div className="mb-8">
        <h3 className="mb-4 text-lg font-semibold text-foreground">
          Upcoming Events
        </h3>
        {upcoming.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">
            No upcoming events.{" "}
            <Link to="/events" className="text-primary hover:underline">
              Create one
            </Link>
            .
          </p>
        ) : (
          <div className="space-y-3">
            {upcoming.map((ev) => (
              <EventRow key={ev.id} event={ev} />
            ))}
          </div>
        )}
      </div>

      {/* Completed events */}
      <div>
        <h3 className="mb-4 text-lg font-semibold text-foreground">
          Past Events
        </h3>
        {completed.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">
            No completed events yet.
          </p>
        ) : (
          <div className="space-y-3">
            {completed.map((ev) => (
              <EventRow key={ev.id} event={ev} />
            ))}
          </div>
        )}
      </div>

      {!online && (
        <p className="mt-6 flex items-center gap-2 text-sm text-amber-600">
          <WifiOff className="h-4 w-4" /> Showing cached data — reconnect to
          refresh.
        </p>
      )}
    </>
  );
}

function EventRow({ event }: { event: ApiEvent }) {
  return (
    <div className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow">
      <div>
        <p className="font-medium text-foreground">{event.title}</p>
        <p className="text-xs text-muted-foreground">
          {new Date(event.start_time).toLocaleDateString()} · {event.location}
        </p>
      </div>
      <div className="flex items-center gap-3">
        {event.capacity != null && (
          <span className="text-xs text-muted-foreground">
            {event.slots_remaining ?? "?"}/{event.capacity} slots
          </span>
        )}
        <span
          className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
            event.status === "active"
              ? "bg-green-100 text-green-700"
              : event.status === "upcoming"
              ? "bg-accent text-accent-foreground"
              : "bg-muted text-muted-foreground"
          }`}
        >
          {event.status}
        </span>
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Dashboard() {
  const { user } = useAuth();
  const online = useOnlineStatus();

  if (!user) return null;

  return (
    <div className="section-padding">
      <div className="container max-w-4xl">
        {/* Header row */}
        <div className="mb-8 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-3xl font-bold text-foreground">Dashboard</h1>
            <p className="mt-1 text-sm text-muted-foreground">{user.email}</p>
          </div>
          <div className="flex items-center gap-2">
            {online ? (
              <span className="flex items-center gap-1.5 rounded-full bg-green-100 px-3 py-1 text-xs font-medium text-green-700">
                <Wifi className="h-3 w-3" /> Online
              </span>
            ) : (
              <span className="flex items-center gap-1.5 rounded-full bg-amber-100 px-3 py-1 text-xs font-medium text-amber-700">
                <WifiOff className="h-3 w-3" /> Offline
              </span>
            )}
            <span
              className={`rounded-full px-3 py-1 text-xs font-medium ${
                user.role === "company"
                  ? "bg-primary/10 text-primary"
                  : "bg-accent text-accent-foreground"
              }`}
            >
              {user.role === "company" ? "Company" : "Student"}
            </span>
          </div>
        </div>

        {/* Profile card */}
        <div className="mb-8 rounded-2xl bg-card p-6 card-shadow">
          <h2 className="text-xl font-bold text-foreground">{user.name}</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Member since{" "}
            {user.created_at
              ? new Date(user.created_at).toLocaleDateString()
              : "—"}
          </p>
        </div>

        {/* Role-specific content */}
        {user.role === "student" ? (
          <StudentDashboard userId={user.id} />
        ) : (
          <CompanyDashboard userId={user.id} />
        )}
      </div>
    </div>
  );
}
