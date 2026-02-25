import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { updateUser, getUserBadges, getUserRegistrations, getOrgStats } from '../db/database';
import { User, Building2, Award, Calendar, LogOut, Trash2 } from 'lucide-react';

function UserProfile({ user, onRefresh, onLogout, onDelete }) {
  const [firstName, setFirstName] = useState(user.first_name || '');
  const [lastName, setLastName] = useState(user.last_name || '');
  const [email, setEmail] = useState(user.email || '');
  const [msg, setMsg] = useState('');
  const [badges, setBadges] = useState([]);
  const [completed, setCompleted] = useState(0);

  useEffect(() => {
    setBadges(getUserBadges(user.id));
    const regs = getUserRegistrations(user.id);
    setCompleted(regs.filter((r) => r.status === 'completed').length);
  }, [user.id]);

  const handleSave = () => {
    if (!firstName.trim() || !lastName.trim() || !email.trim()) {
      setMsg('All fields are required.');
      return;
    }
    updateUser(user.id, { first_name: firstName.trim(), last_name: lastName.trim(), email: email.trim() });
    onRefresh();
    setMsg('Profile updated!');
    setTimeout(() => setMsg(''), 2000);
  };

  return (
    <div className="space-y-6">
      <div className="rounded-2xl bg-card p-6 card-shadow">
        <div className="mb-6 flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
            <User className="h-8 w-8 text-primary-foreground" />
          </div>
          <div>
            <h2 className="text-xl font-bold text-foreground">{user.first_name} {user.last_name}</h2>
            <p className="text-sm text-muted-foreground">Individual Account</p>
          </div>
        </div>

        <div className="mb-6 grid gap-4 sm:grid-cols-2">
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Award className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{badges.length}</p>
              <p className="text-xs text-muted-foreground">Badges</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Calendar className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{completed}</p>
              <p className="text-xs text-muted-foreground">Completed Events</p>
            </div>
          </div>
        </div>

        {msg && (
          <div className={`mb-4 rounded-lg px-4 py-2 text-sm ${msg.includes('updated') ? 'bg-green-100 text-green-700' : 'bg-destructive/10 text-destructive'}`}>
            {msg}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">First Name</label>
            <input type="text" value={firstName} onChange={(e) => setFirstName(e.target.value)} maxLength={100}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Last Name</label>
            <input type="text" value={lastName} onChange={(e) => setLastName(e.target.value)} maxLength={100}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Email</label>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} maxLength={255}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <button onClick={handleSave} className="rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
            Save Changes
          </button>
        </div>
      </div>

      <div className="flex gap-3">
        <button onClick={onLogout} className="flex items-center gap-2 rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-muted-foreground hover:text-foreground">
          <LogOut className="h-4 w-4" /> Logout
        </button>
        <button onClick={onDelete} className="flex items-center gap-2 rounded-lg border border-destructive/30 px-5 py-2.5 text-sm font-medium text-destructive hover:bg-destructive/10">
          <Trash2 className="h-4 w-4" /> Delete Account
        </button>
      </div>
    </div>
  );
}

function OrgProfile({ user, onRefresh, onLogout, onDelete }) {
  const [orgName, setOrgName] = useState(user.organization_name || '');
  const [email, setEmail] = useState(user.email || '');
  const [msg, setMsg] = useState('');
  const [stats, setStats] = useState({ totalEvents: 0, totalApplicants: 0 });

  useEffect(() => {
    setStats(getOrgStats(user.id));
  }, [user.id]);

  const handleSave = () => {
    if (!orgName.trim() || !email.trim()) {
      setMsg('All fields are required.');
      return;
    }
    updateUser(user.id, { organization_name: orgName.trim(), email: email.trim() });
    onRefresh();
    setMsg('Profile updated!');
    setTimeout(() => setMsg(''), 2000);
  };

  return (
    <div className="space-y-6">
      <div className="rounded-2xl bg-card p-6 card-shadow">
        <div className="mb-6 flex items-center gap-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
            <Building2 className="h-8 w-8 text-primary-foreground" />
          </div>
          <div>
            <h2 className="text-xl font-bold text-foreground">{user.organization_name}</h2>
            <p className="text-sm text-muted-foreground">Organization Account</p>
          </div>
        </div>

        <div className="mb-6 grid gap-4 sm:grid-cols-2">
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <Calendar className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{stats.totalEvents}</p>
              <p className="text-xs text-muted-foreground">Events Posted</p>
            </div>
          </div>
          <div className="flex items-center gap-3 rounded-xl bg-muted p-4">
            <User className="h-5 w-5 text-primary" />
            <div>
              <p className="text-lg font-bold text-foreground">{stats.totalApplicants}</p>
              <p className="text-xs text-muted-foreground">Total Applicants</p>
            </div>
          </div>
        </div>

        {msg && (
          <div className={`mb-4 rounded-lg px-4 py-2 text-sm ${msg.includes('updated') ? 'bg-green-100 text-green-700' : 'bg-destructive/10 text-destructive'}`}>
            {msg}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Organization Name</label>
            <input type="text" value={orgName} onChange={(e) => setOrgName(e.target.value)} maxLength={200}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Email</label>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} maxLength={255}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <button onClick={handleSave} className="rounded-lg bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
            Save Changes
          </button>
        </div>
      </div>

      <div className="flex gap-3">
        <button onClick={onLogout} className="flex items-center gap-2 rounded-lg border border-border px-5 py-2.5 text-sm font-medium text-muted-foreground hover:text-foreground">
          <LogOut className="h-4 w-4" /> Logout
        </button>
        <button onClick={onDelete} className="flex items-center gap-2 rounded-lg border border-destructive/30 px-5 py-2.5 text-sm font-medium text-destructive hover:bg-destructive/10">
          <Trash2 className="h-4 w-4" /> Delete Account
        </button>
      </div>
    </div>
  );
}

export default function Profile() {
  const { user, signOut, refreshUser, deleteAccount } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    signOut();
    navigate('/');
  };

  const handleDelete = () => {
    if (window.confirm('Are you sure you want to delete your account? This cannot be undone.')) {
      deleteAccount();
      navigate('/');
    }
  };

  return (
    <div className="section-padding">
      <div className="container max-w-2xl">
        <h1 className="mb-8 text-3xl font-bold text-foreground">Profile</h1>
        {user.role === 'user' ? (
          <UserProfile user={user} onRefresh={refreshUser} onLogout={handleLogout} onDelete={handleDelete} />
        ) : (
          <OrgProfile user={user} onRefresh={refreshUser} onLogout={handleLogout} onDelete={handleDelete} />
        )}
      </div>
    </div>
  );
}
