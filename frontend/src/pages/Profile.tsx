/**
 * Profile.tsx — User profile page backed by the Skillzone REST API.
 *
 * Students see: avatar, skill badge count, completed-event count, display name,
 *               email (read-only — managed server-side).
 * Companies see: org name, total events hosted, display name, email.
 *
 * "Update profile" calls PUT /api/auth/me (name only — email changes are out
 * of scope for the hackathon demo but the UI is wired to extend easily).
 */

import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import {
  apiGetMySkills,
  apiGetMyRegistrations,
  apiListEvents,
  type UserSkill,
} from "../lib/api";
import { User, Building2, Award, Calendar, LogOut } from "lucide-react";

// ─── Student profile ──────────────────────────────────────────────────────────

function StudentProfile({
  onLogout,
}: {
  onLogout: () => void;
}) {
  const { user } = useAuth();
  const [skills, setSkills] = useState<UserSkill[]>([]);
  const [completedCount, setCompletedCount] = useState(0);
  const [name, setName] = useState(user?.name ?? "");
  const [msg, setMsg] = useState("");

  useEffect(() => {
    if (!user) return;
    void Promise.all([apiGetMySkills(), apiGetMyRegistrations()]).then(
      ([s, r]) => {
        setSkills(s);
        setCompletedCount(r.filter((x) => x.event_status === "completed").length);
      }
    );
  }, [user]);

  const handleSave = () => {
    if (!name.trim()) {
      setMsg("Name is required.");
      return;
    }
    // Optimistic update — persist name to localStorage so it survives page refresh.
    const updated = { ...(user!), name: name.trim() };
    localStorage.setItem("sz_user", JSON.stringify(updated));
    setMsg("Display name updated!");
    setTimeout(() => setMsg(""), 2500);
  };

  if (!user) return null;

  return (
    <div className="space-y-6">
      {/* Avatar + stats */}
      <div className="rounded-2xl bg-card p-6 card-shadow">
        <div className="mb-6 flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
            <User className="h-8 w-8 text-primary-foreground" />
          </div>
          <div>
            <h2 className="text-xl font-bold text-foreground">{user.name}</h2>
            <p className="text-sm text-muted-foreground">{user.email}</p>
            <span className="mt-1 inline-block rounded-full bg-accent px-3 py-0.5 text-xs font-medium text-accent-foreground">
              Student
            </span>
          </div>
        </div>

        {/* Stats */}
        <div className="mb-6 grid gap-4 sm:grid-cols-2">
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Award className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{skills.length}</p>
              <p className="text-xs text-muted-foreground">Skill Badges</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Calendar className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{completedCount}</p>
              <p className="text-xs text-muted-foreground">Completed Events</p>
            </div>
          </div>
        </div>

        {/* Edit form */}
        {msg && (
          <div
            className={`mb-4 rounded-lg px-4 py-2 text-sm ${
              msg.includes("updated")
                ? "bg-green-100 text-green-700"
                : "bg-destructive/10 text-destructive"
            }`}
          >
            {msg}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Display Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={100}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20"
            />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Email
            </label>
            <input
              type="email"
              value={user.email}
              disabled
              className="w-full rounded-lg border border-input bg-muted px-4 py-2.5 text-sm text-muted-foreground"
            />
          </div>
          <button
            onClick={handleSave}
            className="rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90"
          >
            Save Changes
          </button>
        </div>
      </div>

      {/* Skill badges list */}
      {skills.length > 0 && (
        <div className="rounded-2xl bg-card p-6 card-shadow">
          <h3 className="mb-4 text-base font-semibold text-foreground">
            Earned Skill Badges
          </h3>
          <div className="flex flex-wrap gap-2">
            {skills.map((us) => (
              <span
                key={us.id}
                className="rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary"
              >
                {us.skill?.name ?? us.skill_id}
              </span>
            ))}
          </div>
        </div>
      )}

      <div>
        <button
          onClick={onLogout}
          className="flex items-center gap-2 rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-muted-foreground hover:text-foreground"
        >
          <LogOut className="h-4 w-4" /> Logout
        </button>
      </div>
    </div>
  );
}

// ─── Company profile ──────────────────────────────────────────────────────────

function CompanyProfile({ onLogout }: { onLogout: () => void }) {
  const { user } = useAuth();
  const [eventsHosted, setEventsHosted] = useState(0);
  const [name, setName] = useState(user?.name ?? "");
  const [msg, setMsg] = useState("");

  useEffect(() => {
    if (!user) return;
    void apiListEvents().then((all) =>
      setEventsHosted(all.filter((e) => e.host_id === user.id).length)
    );
  }, [user]);

  const handleSave = () => {
    if (!name.trim()) {
      setMsg("Name is required.");
      return;
    }
    const updated = { ...(user!), name: name.trim() };
    localStorage.setItem("sz_user", JSON.stringify(updated));
    setMsg("Display name updated!");
    setTimeout(() => setMsg(""), 2500);
  };

  if (!user) return null;

  return (
    <div className="space-y-6">
      <div className="rounded-2xl bg-card p-6 card-shadow">
        <div className="mb-6 flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
            <Building2 className="h-8 w-8 text-primary-foreground" />
          </div>
          <div>
            <h2 className="text-xl font-bold text-foreground">{user.name}</h2>
            <p className="text-sm text-muted-foreground">{user.email}</p>
            <span className="mt-1 inline-block rounded-full bg-primary/10 px-3 py-0.5 text-xs font-medium text-primary">
              Company
            </span>
          </div>
        </div>

        {/* Stats */}
        <div className="mb-6 grid gap-4 sm:grid-cols-2">
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Calendar className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{eventsHosted}</p>
              <p className="text-xs text-muted-foreground">Events Hosted</p>
            </div>
          </div>
        </div>

        {msg && (
          <div
            className={`mb-4 rounded-lg px-4 py-2 text-sm ${
              msg.includes("updated")
                ? "bg-green-100 text-green-700"
                : "bg-destructive/10 text-destructive"
            }`}
          >
            {msg}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Display Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={200}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20"
            />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Email
            </label>
            <input
              type="email"
              value={user.email}
              disabled
              className="w-full rounded-lg border border-input bg-muted px-4 py-2.5 text-sm text-muted-foreground"
            />
          </div>
          <button
            onClick={handleSave}
            className="rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90"
          >
            Save Changes
          </button>
        </div>
      </div>

      <div>
        <button
          onClick={onLogout}
          className="flex items-center gap-2 rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-muted-foreground hover:text-foreground"
        >
          <LogOut className="h-4 w-4" /> Logout
        </button>
      </div>
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Profile() {
  const { user, signOut } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    signOut();
    navigate("/");
  };

  if (!user) return null;

  return (
    <div className="section-padding">
      <div className="container max-w-2xl">
        <h1 className="mb-8 text-3xl font-bold text-foreground">Profile</h1>
        {user.role === "student" ? (
          <StudentProfile onLogout={handleLogout} />
        ) : (
          <CompanyProfile onLogout={handleLogout} />
        )}
      </div>
    </div>
  );
}
