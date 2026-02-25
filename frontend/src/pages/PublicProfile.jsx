import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { getUserById, getUserBadges, getUserRegistrations, getLinksForUser } from '../db/database';
import { Award, Calendar } from 'lucide-react';

export default function PublicProfile() {
  const { id } = useParams();
  const [user, setUser] = useState(null);
  const [badges, setBadges] = useState([]);
  const [completed, setCompleted] = useState([]);
  const [links, setLinks] = useState([]);

  useEffect(() => {
    const u = getUserById(Number(id));
    setUser(u);
    if (u) {
      setBadges(getUserBadges(u.id));
      const regs = getUserRegistrations(u.id);
      setCompleted(regs.filter((r) => r.status === 'completed'));
      setLinks(getLinksForUser(u.id));
    }
  }, [id]);

  if (!user) return (
    <div className="section-padding">
      <div className="container max-w-2xl">
        <p className="text-sm text-muted-foreground">User not found.</p>
      </div>
    </div>
  );

  return (
    <div className="section-padding">
      <div className="container max-w-3xl">
        <div className="mb-6 rounded-2xl bg-card p-6 card-shadow">
          <h1 className="text-2xl font-bold text-foreground">{user.role === 'user' ? `${user.first_name} ${user.last_name}` : user.organization_name}</h1>
          <p className="text-sm text-muted-foreground">{user.email}</p>
        </div>

        <div className="mb-6 grid gap-4 sm:grid-cols-2">
          <div className="rounded-2xl bg-card p-6 card-shadow">
            <h3 className="mb-2 text-sm font-semibold text-foreground">Badges</h3>
            {badges.length === 0 ? <p className="text-sm text-muted-foreground">No badges yet.</p> : (
              <div className="flex flex-wrap gap-3">
                {badges.map((b) => (
                  <div key={b.id} className="rounded-lg bg-muted px-3 py-2 text-sm">
                    <Award className="inline h-4 w-4 mr-2 text-primary" />{b.skill_name}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="rounded-2xl bg-card p-6 card-shadow">
            <h3 className="mb-2 text-sm font-semibold text-foreground">Completed Events</h3>
            {completed.length === 0 ? <p className="text-sm text-muted-foreground">No completed events.</p> : (
              <ul className="space-y-2">
                {completed.map((c) => (
                  <li key={c.id} className="text-sm text-foreground">{c.title} — <span className="text-xs text-muted-foreground">{c.organization_name}</span></li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div className="rounded-2xl bg-card p-6 card-shadow">
          <h3 className="mb-3 text-sm font-semibold text-foreground">Linked Opportunities</h3>
          {links.length === 0 ? <p className="text-sm text-muted-foreground">No links.</p> : (
            <ul className="space-y-2">
              {links.map((l) => (
                <li key={l.id} className="text-sm">
                  <strong className="text-foreground">{l.employer_name}</strong> — <span className="text-xs text-muted-foreground">{l.employer_contact}</span>
                  <div className="text-xs text-muted-foreground">Linked by: <Link to="/profile" className="text-primary">{l.org_name || 'Organization'}</Link></div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
