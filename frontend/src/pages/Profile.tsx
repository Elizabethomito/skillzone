/**
 * Profile.tsx — User profile page backed by the Skillzone REST API.
 *
 * Students see: avatar, skill badge count, completed-event count, display name,
 *               grouped skill badges (with repeat counts), clickable badge
 *               detail modal showing every event + host that awarded the badge,
 *               and a shareable public-profile link for recruiters.
 * Companies see: org name, total events hosted, display name, email.
 */

import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import {
  apiGetMySkills,
  apiGetMyRegistrations,
  apiListEvents,
  apiGetPublicProfile,
  type UserSkill,
  type PublicBadge,
  type PublicProfile,
} from "../lib/api";
import {
  User, Building2, Award, Calendar, LogOut, X, MapPin, Clock, Mail, Copy, Check,
} from "lucide-react";

// ─── Badge detail modal ───────────────────────────────────────────────────────

interface BadgeDetailModalProps {
  skillName: string;
  skillDescription: string;
  instances: PublicBadge[];
  onClose: () => void;
}

function BadgeDetailModal({ skillName, skillDescription, instances, onClose }: BadgeDetailModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg max-h-[85vh] overflow-y-auto rounded-2xl bg-background p-6 shadow-xl">
        <div className="mb-4 flex items-start justify-between gap-2">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <Award className="h-5 w-5 text-primary" />
              <h2 className="text-lg font-bold text-foreground">{skillName}</h2>
              {instances.length > 1 && (
                <span className="rounded-full bg-primary px-2 py-0.5 text-xs font-bold text-primary-foreground">
                  ×{instances.length}
                </span>
              )}
            </div>
            <p className="text-sm text-muted-foreground">{skillDescription}</p>
          </div>
          <button onClick={onClose} className="shrink-0 rounded p-1 hover:bg-muted">
            <X className="h-5 w-5" />
          </button>
        </div>

        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          Earned at {instances.length} event{instances.length !== 1 ? "s" : ""}
        </p>

        <div className="space-y-3">
          {instances.map((b, i) => (
            <div key={b.id} className="rounded-xl border border-border bg-card p-4">
              <div className="mb-2 flex items-start justify-between gap-2">
                <p className="font-semibold text-foreground text-sm">{b.event_title}</p>
                <span className="shrink-0 rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700">
                  #{i + 1}
                </span>
              </div>
              <div className="space-y-1 text-xs text-muted-foreground">
                <span className="flex items-center gap-1.5">
                  <Clock className="h-3 w-3" />
                  {new Date(b.event_start).toLocaleDateString(undefined, { dateStyle: "medium" })}
                </span>
                <span className="flex items-center gap-1.5">
                  <MapPin className="h-3 w-3" />
                  {b.event_location || "TBD"}
                </span>
              </div>
              <div className="mt-3 border-t border-border pt-3">
                <p className="text-xs font-medium text-foreground mb-0.5">Hosted by</p>
                <p className="text-sm font-semibold text-foreground">{b.host_name}</p>
                <a href={`mailto:${b.host_email}`}
                  className="flex items-center gap-1 text-xs text-primary hover:underline mt-0.5">
                  <Mail className="h-3 w-3" /> {b.host_email}
                </a>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ─── Student profile ──────────────────────────────────────────────────────────

function StudentProfile({ onLogout }: { onLogout: () => void }) {
  const { user } = useAuth();
  const [profile, setProfile] = useState<PublicProfile | null>(null);
  const [completedCount, setCompletedCount] = useState(0);
  const [name, setName] = useState(user?.name ?? "");
  const [msg, setMsg] = useState("");
  const [activeBadge, setActiveBadge] = useState<{ name: string; desc: string; instances: PublicBadge[] } | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!user) return;
    void Promise.all([
      apiGetPublicProfile(user.id),
      apiGetMyRegistrations(),
    ]).then(([pub, regs]) => {
      setProfile(pub);
      setCompletedCount(regs.filter((x) => x.event_status === "completed").length);
    });
  }, [user]);

  const handleSave = () => {
    if (!name.trim()) { setMsg("Name is required."); return; }
    const updated = { ...(user!), name: name.trim() };
    localStorage.setItem("sz_user", JSON.stringify(updated));
    setMsg("Display name updated!");
    setTimeout(() => setMsg(""), 2500);
  };

  const copyProfileLink = () => {
    const url = `${window.location.origin}/profile/${user!.id}`;
    navigator.clipboard.writeText(url).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  };

  if (!user) return null;

  // Group badges by skill_id, keeping all instances for the modal
  const badgeGroups: Map<string, { name: string; desc: string; instances: PublicBadge[] }> = new Map();
  (profile?.badges ?? []).forEach((b) => {
    if (!badgeGroups.has(b.skill_id)) {
      badgeGroups.set(b.skill_id, { name: b.skill_name, desc: b.skill_description, instances: [] });
    }
    badgeGroups.get(b.skill_id)!.instances.push(b);
  });
  const uniqueSkillCount = badgeGroups.size;
  const totalBadgeCount  = profile?.badges.length ?? 0;

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
        <div className="mb-6 grid gap-4 sm:grid-cols-3">
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Award className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{uniqueSkillCount}</p>
              <p className="text-xs text-muted-foreground">Unique Skills</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Award className="h-5 w-5 text-amber-500" />
            <div>
              <p className="text-lg font-bold text-foreground">{totalBadgeCount}</p>
              <p className="text-xs text-muted-foreground">Total Badges</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Calendar className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{completedCount}</p>
              <p className="text-xs text-muted-foreground">Events Attended</p>
            </div>
          </div>
        </div>

        {/* Edit form */}
        {msg && (
          <div className={`mb-4 rounded-lg px-4 py-2 text-sm ${
            msg.includes("updated") ? "bg-green-100 text-green-700" : "bg-destructive/10 text-destructive"
          }`}>{msg}</div>
        )}
        <div className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Display Name</label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)} maxLength={100}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Email</label>
            <input type="email" value={user.email} disabled
              className="w-full rounded-lg border border-input bg-muted px-4 py-2.5 text-sm text-muted-foreground" />
          </div>
          <button onClick={handleSave}
            className="rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
            Save Changes
          </button>
        </div>
      </div>

      {/* Skill badges — grouped, clickable, with repeat count */}
      {badgeGroups.size > 0 && (
        <div className="rounded-2xl bg-card p-6 card-shadow">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-base font-semibold text-foreground">Earned Skill Badges</h3>
            <p className="text-xs text-muted-foreground">Click a badge to see verification details</p>
          </div>
          <div className="flex flex-wrap gap-2">
            {Array.from(badgeGroups.entries()).map(([skillId, group]) => (
              <button
                key={skillId}
                onClick={() => setActiveBadge(group)}
                className="group flex items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1.5 text-xs font-medium text-primary transition hover:bg-primary hover:text-primary-foreground"
              >
                <Award className="h-3 w-3" />
                {group.name}
                {group.instances.length > 1 && (
                  <span className="ml-0.5 rounded-full bg-primary px-1.5 py-0.5 text-[10px] font-bold text-primary-foreground group-hover:bg-primary-foreground group-hover:text-primary">
                    ×{group.instances.length}
                  </span>
                )}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Public profile / recruiter share link */}
      <div className="rounded-2xl border border-border bg-card p-5">
        <h3 className="mb-1 text-sm font-semibold text-foreground">Public Profile Link</h3>
        <p className="mb-3 text-xs text-muted-foreground">
          Share this link with recruiters so they can verify your skills and see the events where you earned each badge.
        </p>
        <div className="flex items-center gap-2">
          <code className="flex-1 truncate rounded-lg bg-muted px-3 py-2 text-xs text-foreground">
            {window.location.origin}/profile/{user.id}
          </code>
          <button onClick={copyProfileLink}
            className="flex items-center gap-1 rounded-lg border border-input px-3 py-2 text-xs font-medium transition hover:bg-muted">
            {copied ? <Check className="h-3.5 w-3.5 text-green-600" /> : <Copy className="h-3.5 w-3.5" />}
            {copied ? "Copied!" : "Copy"}
          </button>
        </div>
      </div>

      <div>
        <button onClick={onLogout}
          className="flex items-center gap-2 rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-muted-foreground hover:text-foreground">
          <LogOut className="h-4 w-4" /> Logout
        </button>
      </div>

      {/* Badge detail modal */}
      {activeBadge && (
        <BadgeDetailModal
          skillName={activeBadge.name}
          skillDescription={activeBadge.desc}
          instances={activeBadge.instances}
          onClose={() => setActiveBadge(null)}
        />
      )}
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
    if (!name.trim()) { setMsg("Name is required."); return; }
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
          <div className={`mb-4 rounded-lg px-4 py-2 text-sm ${
            msg.includes("updated") ? "bg-green-100 text-green-700" : "bg-destructive/10 text-destructive"
          }`}>{msg}</div>
        )}

        <div className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Display Name</label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)} maxLength={200}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Email</label>
            <input type="email" value={user.email} disabled
              className="w-full rounded-lg border border-input bg-muted px-4 py-2.5 text-sm text-muted-foreground" />
          </div>
          <button onClick={handleSave}
            className="rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
            Save Changes
          </button>
        </div>
      </div>

      <div>
        <button onClick={onLogout}
          className="flex items-center gap-2 rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-muted-foreground hover:text-foreground">
          <LogOut className="h-4 w-4" /> Logout
        </button>
      </div>
    </div>
  );
}

// ─── Public recruiter view ─────────────────────────────────────────────────────

export function PublicProfileView({ userId }: { userId: string }) {
  const [profile, setProfile] = useState<PublicProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeBadge, setActiveBadge] = useState<{ name: string; desc: string; instances: PublicBadge[] } | null>(null);

  useEffect(() => {
    apiGetPublicProfile(userId)
      .then(setProfile)
      .finally(() => setLoading(false));
  }, [userId]);

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="section-padding">
        <div className="container max-w-2xl text-center py-16">
          <p className="text-muted-foreground">Profile not found.</p>
        </div>
      </div>
    );
  }

  const badgeGroups: Map<string, { name: string; desc: string; instances: PublicBadge[] }> = new Map();
  profile.badges.forEach((b) => {
    if (!badgeGroups.has(b.skill_id)) {
      badgeGroups.set(b.skill_id, { name: b.skill_name, desc: b.skill_description, instances: [] });
    }
    badgeGroups.get(b.skill_id)!.instances.push(b);
  });

  return (
    <div className="section-padding">
      <div className="container max-w-2xl space-y-6">
        {/* Header */}
        <div className="rounded-2xl bg-card p-6 card-shadow">
          <div className="flex items-center gap-4 mb-4">
            <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
              <User className="h-8 w-8 text-primary-foreground" />
            </div>
            <div>
              <h1 className="text-2xl font-bold text-foreground">{profile.name}</h1>
              <p className="text-sm text-muted-foreground">{profile.email}</p>
              <span className="mt-1 inline-block rounded-full bg-accent px-3 py-0.5 text-xs font-medium text-accent-foreground">
                Student · SkillZone Verified
              </span>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="flex items-center gap-3 rounded-xl bg-muted p-3">
              <Award className="h-4 w-4 text-primary" />
              <div>
                <p className="text-base font-bold text-foreground">{badgeGroups.size}</p>
                <p className="text-xs text-muted-foreground">Unique Skills</p>
              </div>
            </div>
            <div className="flex items-center gap-3 rounded-xl bg-muted p-3">
              <Award className="h-4 w-4 text-amber-500" />
              <div>
                <p className="text-base font-bold text-foreground">{profile.badges.length}</p>
                <p className="text-xs text-muted-foreground">Total Badges</p>
              </div>
            </div>
          </div>
        </div>

        {/* Badges */}
        {badgeGroups.size > 0 ? (
          <div className="rounded-2xl bg-card p-6 card-shadow">
            <h2 className="mb-1 text-base font-semibold text-foreground">Verified Skill Badges</h2>
            <p className="mb-4 text-xs text-muted-foreground">Click a badge to see the event and host details.</p>
            <div className="flex flex-wrap gap-2">
              {Array.from(badgeGroups.entries()).map(([skillId, group]) => (
                <button key={skillId} onClick={() => setActiveBadge(group)}
                  className="group flex items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1.5 text-xs font-medium text-primary transition hover:bg-primary hover:text-primary-foreground">
                  <Award className="h-3 w-3" />
                  {group.name}
                  {group.instances.length > 1 && (
                    <span className="ml-0.5 rounded-full bg-primary px-1.5 py-0.5 text-[10px] font-bold text-primary-foreground group-hover:bg-primary-foreground group-hover:text-primary">
                      ×{group.instances.length}
                    </span>
                  )}
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div className="rounded-2xl bg-card p-8 text-center">
            <Award className="mx-auto mb-3 h-10 w-10 text-muted-foreground/40" />
            <p className="text-sm text-muted-foreground">No badges earned yet.</p>
          </div>
        )}

        {activeBadge && (
          <BadgeDetailModal
            skillName={activeBadge.name}
            skillDescription={activeBadge.desc}
            instances={activeBadge.instances}
            onClose={() => setActiveBadge(null)}
          />
        )}
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
