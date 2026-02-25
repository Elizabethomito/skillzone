import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { getUserBadges, getUserRegistrations, getOrgStats, getEventsByOrganization } from '../db/database';
import { Award, Calendar, CheckCircle, Clock, BarChart3, Users } from 'lucide-react';

function StatCard({ icon: Icon, label, value, color = 'bg-accent text-accent-foreground' }) {
  return (
    <div className="rounded-2xl bg-card p-6 card-shadow">
      <div className="flex items-center gap-4">
        <div className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-xl ${color}`}>
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

function UserDashboard({ user }) {
  const [badges, setBadges] = useState([]);
  const [regs, setRegs] = useState([]);

  useEffect(() => {
    setBadges(getUserBadges(user.id));
    setRegs(getUserRegistrations(user.id));
  }, [user.id]);

  const completed = regs.filter((r) => r.status === 'completed');
  const ongoing = regs.filter((r) => r.status !== 'completed');

  return (
    <>
      {/* User info */}
      <div className="mb-8 rounded-2xl bg-card p-6 card-shadow">
        <h2 className="mb-1 text-xl font-bold text-foreground">{user.first_name} {user.last_name}</h2>
        <p className="text-sm text-muted-foreground">{user.email}</p>
        <span className="mt-2 inline-block rounded-full bg-accent px-3 py-1 text-xs font-medium text-accent-foreground">Individual</span>
      </div>

      {/* Stats */}
      <div className="mb-8 grid gap-4 sm:grid-cols-3">
        <StatCard icon={Award} label="Total Badges" value={badges.length} color="bg-primary text-primary-foreground" />
        <StatCard icon={CheckCircle} label="Completed Events" value={completed.length} />
        <StatCard icon={Clock} label="Ongoing Events" value={ongoing.length} />
      </div>

      {/* Badges */}
      <div className="mb-8">
        <h3 className="mb-4 text-lg font-semibold text-foreground">Badges Earned</h3>
        {badges.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">No badges yet. Complete events to earn badges!</p>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {badges.map((b) => (
              <div key={b.id} className="rounded-xl bg-card p-5 card-shadow">
                <div className="mb-2 flex items-center gap-2">
                  <Award className="h-5 w-5 text-primary" />
                  <span className="font-semibold text-foreground">{b.skill_name}</span>
                </div>
                <p className="text-xs text-muted-foreground">Event: {b.event_title}</p>
                <p className="text-xs text-muted-foreground">Issued: {new Date(b.issued_at).toLocaleDateString()}</p>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Completed Events */}
      <div className="mb-8">
        <h3 className="mb-4 text-lg font-semibold text-foreground">Completed Events</h3>
        {completed.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">No completed events yet.</p>
        ) : (
          <div className="space-y-3">
            {completed.map((r) => (
              <div key={r.id} className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow">
                <div>
                  <p className="font-medium text-foreground">{r.title}</p>
                  <p className="text-xs text-muted-foreground">{r.organization_name} · {r.skill_category}</p>
                </div>
                <CheckCircle className="h-5 w-5 text-green-600" />
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Ongoing Events */}
      <div>
        <h3 className="mb-4 text-lg font-semibold text-foreground">Ongoing Events</h3>
        {ongoing.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">No ongoing events.</p>
        ) : (
          <div className="space-y-3">
            {ongoing.map((r) => (
              <div key={r.id} className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow">
                <div>
                  <p className="font-medium text-foreground">{r.title}</p>
                  <p className="text-xs text-muted-foreground">{r.organization_name} · {r.skill_category}</p>
                </div>
                <span className="rounded-full bg-accent px-2.5 py-0.5 text-xs font-medium text-accent-foreground">{r.status}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  );
}

function OrgDashboard({ user }) {
  const [stats, setStats] = useState({ totalEvents: 0, activeEvents: 0, totalApplicants: 0 });
  const [events, setEvents] = useState([]);

  useEffect(() => {
    setStats(getOrgStats(user.id));
    setEvents(getEventsByOrganization(user.id));
  }, [user.id]);

  const active = events.filter((e) => e.status !== 'completed');

  return (
    <>
      <div className="mb-8 rounded-2xl bg-card p-6 card-shadow">
        <h2 className="mb-1 text-xl font-bold text-foreground">{user.organization_name}</h2>
        <p className="text-sm text-muted-foreground">{user.email}</p>
        <span className="mt-2 inline-block rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">Organization</span>
      </div>

      <div className="mb-8 grid gap-4 sm:grid-cols-3">
        <StatCard icon={Calendar} label="Total Events" value={stats.totalEvents} color="bg-primary text-primary-foreground" />
        <StatCard icon={BarChart3} label="Active Events" value={stats.activeEvents} />
        <StatCard icon={Users} label="Total Applicants" value={stats.totalApplicants} />
      </div>

      <div>
        <h3 className="mb-4 text-lg font-semibold text-foreground">Active Events</h3>
        {active.length === 0 ? (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">No active events.</p>
        ) : (
          <div className="space-y-3">
            {active.map((ev) => (
              <div key={ev.id} className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow">
                <div>
                  <p className="font-medium text-foreground">{ev.title}</p>
                  <p className="text-xs text-muted-foreground">{ev.skill_category}</p>
                </div>
                <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
                  ev.status === 'ongoing' ? 'bg-amber-100 text-amber-700' : 'bg-accent text-accent-foreground'
                }`}>{ev.status}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </>
  );
}

export default function Dashboard() {
  const { user } = useAuth();

  return (
    <div className="section-padding">
      <div className="container max-w-4xl">
        <h1 className="mb-8 text-3xl font-bold text-foreground">Dashboard</h1>
        {user.role === 'user' ? <UserDashboard user={user} /> : <OrgDashboard user={user} />}
      </div>
    </div>
  );
}
