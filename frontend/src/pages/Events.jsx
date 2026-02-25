import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../context/AuthContext';
import {
  getAllEvents,
  getEventsByOrganization,
  createEvent,
  registerForEvent,
  unregisterFromEvent,
  removeUserFromEvent,
  getUserRegistrations,
  getUserBadges,
  getEventApplicants,
  confirmParticipation,
  completeRegistration,
  updateEventStatus,
  issueBadge,
  getUserBadgeCounts,
  getOrgApplicantsWithProgress,
  createLink,
} from '../db/database';
import { Calendar, Plus, Users, Check, Award, X, Star, Clock, CheckCircle, Briefcase, Eye, Link as LinkIcon, Trash2 } from 'lucide-react';
import { Link } from 'react-router-dom';

/* ─────────────── Shared Modals ─────────────── */

function CreateEventModal({ orgId, onClose, onCreated }) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [skillCategory, setSkillCategory] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!title.trim() || !description.trim() || !skillCategory.trim()) {
      setError('All fields are required.');
      return;
    }
    console.debug('CreateEvent submit', { title, description, skillCategory, orgId });
    try {
      createEvent({ title: title.trim(), description: description.trim(), organizationId: orgId, skillCategory: skillCategory.trim() });
      onCreated();
      onClose();
    } catch (err) {
      console.error('createEvent threw', err, { title, description, skillCategory, orgId });
      setError(err.message || 'Failed to create event');
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 p-4">
      <div className="w-full max-w-lg rounded-2xl bg-background p-6 card-shadow">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-xl font-bold text-foreground">Create Event</h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>
        {error && <div className="mb-3 rounded-lg bg-destructive/10 px-4 py-2 text-sm text-destructive">{error}</div>}
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Title</label>
            <input type="text" value={title} onChange={(e) => setTitle(e.target.value)} maxLength={200}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Description</label>
            <textarea value={description} onChange={(e) => setDescription(e.target.value)} rows={3} maxLength={1000}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20" />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">Skill Category</label>
            <input type="text" value={skillCategory} onChange={(e) => setSkillCategory(e.target.value)} maxLength={100}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none focus:border-primary focus:ring-2 focus:ring-ring/20"
              placeholder="e.g. Web Development, Data Science" />
          </div>
          <button type="submit" className="w-full rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
            Create Event
          </button>
        </form>
      </div>
    </div>
  );
}

function ConfirmRegisterModal({ event, onClose, onConfirm }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 p-4">
      <div className="w-full max-w-sm rounded-2xl bg-background p-6 card-shadow">
        <h2 className="mb-4 text-lg font-bold text-foreground">Confirm Registration</h2>
        <p className="mb-4 text-sm text-muted-foreground">
          Are you sure you want to register for <span className="font-semibold text-foreground">{event.title}</span>?
        </p>
        <p className="mb-6 text-sm text-muted-foreground">{event.description}</p>
        <div className="flex gap-3">
          <button onClick={onClose}
            className="flex-1 rounded-lg border border-input bg-background px-4 py-2.5 text-sm font-medium text-foreground hover:bg-muted">
            Cancel
          </button>
          <button onClick={onConfirm}
            className="flex-1 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
            Confirm
          </button>
        </div>
      </div>
    </div>
  );
}

function ApplicantsModal({ event, onClose, onRefresh }) {
  const [applicants, setApplicants] = useState([]);

  const load = useCallback(() => {
    setApplicants(getEventApplicants(event.id));
  }, [event.id]);

  useEffect(() => { load(); }, [load]);

  const handleConfirm = (regId) => {
    try {
      console.info('handleConfirm called with regId', regId);
      confirmParticipation(regId);
      console.info('handleConfirm succeeded');
    } catch (err) {
      console.error('handleConfirm failed', err);
    } finally {
      load();
      onRefresh?.();
    }
  };

  const handleComplete = (regId, userId) => {
    try {
      console.info('handleComplete called with regId, userId', regId, userId);
      completeRegistration(regId);
      issueBadge(userId, event.id, event.skill_category);
      console.info('handleComplete succeeded - badge issued');
    } catch (err) {
      console.error('handleComplete failed', err);
    } finally {
      load();
      onRefresh?.();
    }
  };

  const handleRemove = (userId) => {
    if (confirm('Are you sure you want to remove this user from the event?')) {
      try {
        console.info('handleRemove called with userId, eventId', userId, event.id);
        removeUserFromEvent(userId, event.id);
        console.info('handleRemove succeeded');
      } catch (err) {
        console.error('handleRemove failed', err);
      } finally {
        load();
        onRefresh?.();
      }
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 p-4">
      <div className="w-full max-w-lg rounded-2xl bg-background p-6 card-shadow max-h-[80vh] overflow-y-auto">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-bold text-foreground">Applicants — {event.title}</h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>
        {applicants.length === 0 ? (
          <p className="text-sm text-muted-foreground">No applicants yet.</p>
        ) : (
          <div className="space-y-3">
            {applicants.map((a) => (
              <div key={a.id} className="flex items-center justify-between rounded-xl border border-border p-4">
                <div>
                  <p className="font-medium text-foreground"><Link to={`/users/${a.user_id}`} className="text-primary hover:underline">{a.first_name} {a.last_name}</Link></p>
                  <p className="text-xs text-muted-foreground">{a.email}</p>
                  <span className={`mt-1 inline-block rounded-full px-2 py-0.5 text-xs font-medium ${
                    a.status === 'completed' ? 'bg-green-100 text-green-700' :
                    a.status === 'confirmed' ? 'bg-accent text-accent-foreground' :
                    'bg-secondary text-secondary-foreground'
                  }`}>{a.status}</span>
                </div>
                <div className="flex gap-2">
                  {a.status === 'applied' && (
                    <>
                      <button onClick={() => handleConfirm(a.id)} className="rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90">
                        <Check className="inline h-3 w-3" /> Confirm
                      </button>
                      <button onClick={() => handleRemove(a.user_id)} className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">
                        <Trash2 className="inline h-3 w-3" />
                      </button>
                    </>
                  )}
                  {a.status === 'confirmed' && (
                    <>
                      <button onClick={() => handleComplete(a.id, a.user_id)} className="rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700">
                        <Award className="inline h-3 w-3" /> Complete & Badge
                      </button>
                      <button onClick={() => handleRemove(a.user_id)} className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">
                        <Trash2 className="inline h-3 w-3" />
                      </button>
                    </>
                  )}
                  {a.status === 'completed' && (
                    <button onClick={() => handleRemove(a.user_id)} className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">
                      <Trash2 className="inline h-3 w-3" />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

/* ─────────────── Talent Progress Modal (for org) ─────────────── */

function TalentProgressModal({ orgId, onClose }) {
  const [talents, setTalents] = useState([]);
  const [selectedUser, setSelectedUser] = useState(null);
  const [badgeCounts, setBadgeCounts] = useState([]);

  useEffect(() => {
    setTalents(getOrgApplicantsWithProgress(orgId));
  }, [orgId]);

  const viewBadges = (userId) => {
    setSelectedUser(userId);
    setBadgeCounts(getUserBadgeCounts(userId));
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 p-4">
      <div className="w-full max-w-2xl rounded-2xl bg-background p-6 card-shadow max-h-[85vh] overflow-y-auto">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-bold text-foreground">Talent Progress</h2>
          <button onClick={onClose} className="rounded-lg p-1 text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        {talents.length === 0 ? (
          <p className="text-sm text-muted-foreground">No applicants yet.</p>
        ) : (
          <div className="space-y-3">
            {talents.map((t) => (
              <div key={t.user_id} className="rounded-xl border border-border p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="font-medium text-foreground"><Link to={`/users/${t.user_id}`} className="text-primary hover:underline">{t.first_name} {t.last_name}</Link></p>
                    <p className="text-xs text-muted-foreground">{t.email}</p>
                  </div>
                  <div className="flex gap-2">
                    <button onClick={() => viewBadges(t.user_id)}
                      className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted">
                      <Eye className="h-3 w-3" /> Badges
                    </button>
                    <LinkToEmployerButton orgId={orgId} userId={t.user_id} onLinked={() => { setTalents(getOrgApplicantsWithProgress(orgId)); }} />
                  </div>
                </div>
                <div className="mt-3 flex flex-wrap gap-3">
                  <span className="flex items-center gap-1 rounded-full bg-secondary px-2.5 py-0.5 text-xs font-medium text-secondary-foreground">
                    <Briefcase className="h-3 w-3" /> {t.total_registrations} registered
                  </span>
                  <span className="flex items-center gap-1 rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-700">
                    <Clock className="h-3 w-3" /> {t.ongoing_events} ongoing
                  </span>
                  <span className="flex items-center gap-1 rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-700">
                    <CheckCircle className="h-3 w-3" /> {t.completed_events} completed
                  </span>
                  <span className="flex items-center gap-1 rounded-full bg-accent px-2.5 py-0.5 text-xs font-medium text-accent-foreground">
                    <Award className="h-3 w-3" /> {t.total_badges} badges
                  </span>
                </div>

                {selectedUser === t.user_id && badgeCounts.length > 0 && (
                  <div className="mt-3 rounded-lg bg-muted p-3">
                    <p className="mb-2 text-xs font-semibold text-foreground">Badge Breakdown</p>
                    <div className="flex flex-wrap gap-2">
                      {badgeCounts.map((bc) => (
                        <span key={bc.skill_name} className="flex items-center gap-1 rounded-full bg-primary/10 px-2.5 py-1 text-xs font-medium text-primary">
                          <Star className="h-3 w-3" />
                          {bc.skill_name}
                          {bc.count > 1 && (
                            <span className="ml-0.5 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-primary-foreground">
                              {bc.count}
                            </span>
                          )}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

/* ─────────────── User Events View ─────────────── */

function UserEventsView({ user }) {
  const [allEvents, setAllEvents] = useState([]);
  const [myRegs, setMyRegs] = useState([]);
  const [myBadges, setMyBadges] = useState([]);
  const [confirmEvent, setConfirmEvent] = useState(null);

  const loadData = useCallback(() => {
    setAllEvents(getAllEvents());
    setMyRegs(getUserRegistrations(user.id));
    setMyBadges(getUserBadges(user.id));
  }, [user.id]);

  useEffect(() => { loadData(); }, [loadData]);

  const handleRegisterClick = (event) => {
    setConfirmEvent(event);
  };

  const handleRegisterConfirm = () => {
    try {
      const result = registerForEvent(user.id, confirmEvent.id);
      console.info('handleRegister result', result);
    } catch (err) {
      console.error('registerForEvent failed', err);
    } finally {
      setConfirmEvent(null);
      loadData();
    }
  };

  const handleUnregister = (eventId) => {
    if (confirm('Are you sure you want to unregister from this event?')) {
      try {
        unregisterFromEvent(user.id, eventId);
        console.info('handleUnregister succeeded', { userId: user.id, eventId });
      } catch (err) {
        console.error('unregisterFromEvent failed', err);
      } finally {
        loadData();
      }
    }
  };

  const registeredIds = new Set(myRegs.map((r) => r.event_id));

  // Upcoming = open events the user has NOT registered for
  const upcoming = allEvents.filter((ev) => ev.status === 'open' && !registeredIds.has(ev.id));

  // Registered/In-progress = user registered and not completed
  const inProgress = myRegs.filter((r) => r.status !== 'completed');

  // Completed/Attended
  const completed = myRegs.filter((r) => r.status === 'completed');

  const statusBadge = (s) => {
    if (s === 'completed') return 'bg-green-100 text-green-700';
    if (s === 'confirmed') return 'bg-accent text-accent-foreground';
    return 'bg-amber-100 text-amber-700';
  };

  return (
    <>
      {confirmEvent && (
        <ConfirmRegisterModal
          event={confirmEvent}
          onClose={() => setConfirmEvent(null)}
          onConfirm={handleRegisterConfirm}
        />
      )}
      <div className="space-y-10">
        {/* Upcoming / Available Events */}
      <section>
        <h2 className="mb-4 flex items-center gap-2 text-xl font-bold text-foreground">
          <Calendar className="h-5 w-5 text-primary" /> Upcoming Events
        </h2>
        {upcoming.length === 0 ? (
          <div className="rounded-2xl bg-card p-8 text-center card-shadow">
            <Calendar className="mx-auto mb-2 h-8 w-8 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">No upcoming events available right now.</p>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {upcoming.map((ev) => (
              <div key={ev.id} className="rounded-2xl bg-card p-6 card-shadow transition hover:card-shadow-hover">
                <div className="mb-2">
                  <h3 className="text-lg font-semibold text-foreground">{ev.title}</h3>
                  <p className="text-xs text-muted-foreground">by {ev.organization_name}</p>
                </div>
                <p className="mb-3 text-sm leading-relaxed text-muted-foreground">{ev.description}</p>
                <span className="mb-4 inline-block rounded-full bg-secondary px-3 py-1 text-xs font-medium text-secondary-foreground">
                  {ev.skill_category}
                </span>
                <div className="mt-3">
                  <button onClick={() => handleRegisterClick(ev)}
                    className="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground hover:bg-primary/90">
                    Apply / Register
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Registered / In-Progress */}
      <section>
        <h2 className="mb-4 flex items-center gap-2 text-xl font-bold text-foreground">
          <Clock className="h-5 w-5 text-amber-600" /> Registered / In Progress
        </h2>
        {inProgress.length === 0 ? (
          <div className="rounded-2xl bg-card p-8 text-center card-shadow">
            <p className="text-sm text-muted-foreground">No registered events yet. Apply to an event above!</p>
          </div>
        ) : (
          <div className="space-y-3">
            {inProgress.map((r) => (
              <div key={r.id} className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow">
                <div>
                  <p className="font-medium text-foreground">{r.title}</p>
                  <p className="text-xs text-muted-foreground">{r.organization_name} · {r.skill_category}</p>
                </div>
                <div className="flex items-center gap-3">
                  <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${statusBadge(r.status)}`}>
                    {r.status === 'applied' ? 'Registered' : r.status}
                  </span>
                  <button onClick={() => handleUnregister(r.event_id)} className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700">
                    <Trash2 className="inline h-3 w-3" /> Unregister
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Completed / Attended */}
      <section>
        <h2 className="mb-4 flex items-center gap-2 text-xl font-bold text-foreground">
          <CheckCircle className="h-5 w-5 text-green-600" /> Completed / Attended
        </h2>
        {completed.length === 0 ? (
          <div className="rounded-2xl bg-card p-8 text-center card-shadow">
            <p className="text-sm text-muted-foreground">No completed events yet.</p>
          </div>
        ) : (
          <div className="space-y-3">
            {completed.map((r) => {
              const badge = myBadges.find((b) => b.event_id === r.event_id);
              return (
                <div key={r.id} className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow">
                  <div>
                    <p className="font-medium text-foreground">{r.title}</p>
                    <p className="text-xs text-muted-foreground">{r.organization_name} · {r.skill_category}</p>
                    {badge && (
                      <div className="mt-2 flex items-center gap-2 text-sm">
                        <Award className="h-4 w-4 text-primary" />
                        <span className="text-xs text-muted-foreground">Badge: {badge.skill_name}</span>
                      </div>
                    )}
                  </div>
                  <CheckCircle className="h-5 w-5 text-green-600" />
                </div>
              );
            })}
          </div>
        )}
      </section>
      </div>
    </>
  );
}

/* ─────────────── Organization Events View ─────────────── */

function OrgEventsView({ user }) {
  const [events, setEvents] = useState([]);
  const [showCreate, setShowCreate] = useState(false);
  const [showApplicants, setShowApplicants] = useState(null);
  const [showTalentProgress, setShowTalentProgress] = useState(false);

  const loadData = useCallback(() => {
    setEvents(getEventsByOrganization(user.id));
  }, [user.id]);

  useEffect(() => { loadData(); }, [loadData]);

  const handleStatusChange = (eventId, status) => {
    updateEventStatus(eventId, status);
    loadData();
  };

  const statusColor = (s) =>
    s === 'completed' ? 'bg-green-100 text-green-700' :
    s === 'ongoing' ? 'bg-amber-100 text-amber-700' :
    'bg-accent text-accent-foreground';

  const activeEvents = events.filter((e) => e.status !== 'completed');
  const completedEvents = events.filter((e) => e.status === 'completed');

  return (
    <div className="space-y-8">
      {/* Action buttons */}
      <div className="flex flex-wrap gap-3">
        <button onClick={() => setShowCreate(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
          <Plus className="h-4 w-4" /> Post Event / Job
        </button>
        <button onClick={() => setShowTalentProgress(true)}
          className="flex items-center gap-2 rounded-lg border border-border bg-card px-5 py-2.5 text-sm font-semibold text-foreground hover:bg-muted">
          <Users className="h-4 w-4" /> View Talent Progress
        </button>
      </div>

      {/* Active Events */}
      <section>
        <h2 className="mb-4 flex items-center gap-2 text-xl font-bold text-foreground">
          <Calendar className="h-5 w-5 text-primary" /> Active Events
        </h2>
        {activeEvents.length === 0 ? (
          <div className="rounded-2xl bg-card p-8 text-center card-shadow">
            <p className="text-sm text-muted-foreground">No active events. Create one to get started!</p>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2">
            {activeEvents.map((ev) => (
              <div key={ev.id} className="rounded-2xl bg-card p-6 card-shadow transition hover:card-shadow-hover">
                <div className="mb-3 flex items-start justify-between">
                  <div>
                    <h3 className="text-lg font-semibold text-foreground">{ev.title}</h3>
                  </div>
                  <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${statusColor(ev.status)}`}>
                    {ev.status}
                  </span>
                </div>
                <p className="mb-3 text-sm leading-relaxed text-muted-foreground">{ev.description}</p>
                <span className="mb-4 inline-block rounded-full bg-secondary px-3 py-1 text-xs font-medium text-secondary-foreground">
                  {ev.skill_category}
                </span>
                <div className="mt-3 flex flex-wrap gap-2">
                  <button onClick={() => setShowApplicants(ev)}
                    className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted">
                    <Users className="h-3 w-3" /> Applicants
                  </button>
                  {ev.status === 'open' && (
                    <button onClick={() => handleStatusChange(ev.id, 'ongoing')}
                      className="rounded-lg bg-amber-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-600">
                      Start
                    </button>
                  )}
                  {ev.status === 'ongoing' && (
                    <button onClick={() => handleStatusChange(ev.id, 'completed')}
                      className="rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700">
                      Complete
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      {/* Completed Events */}
      {completedEvents.length > 0 && (
        <section>
          <h2 className="mb-4 flex items-center gap-2 text-xl font-bold text-foreground">
            <CheckCircle className="h-5 w-5 text-green-600" /> Completed Events
          </h2>
          <div className="space-y-3">
            {completedEvents.map((ev) => (
              <div key={ev.id} className="flex items-center justify-between rounded-xl bg-card p-4 card-shadow">
                <div>
                  <p className="font-medium text-foreground">{ev.title}</p>
                  <p className="text-xs text-muted-foreground">{ev.skill_category}</p>
                </div>
                <div className="flex items-center gap-2">
                  <button onClick={() => setShowApplicants(ev)}
                    className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted">
                    <Users className="h-3 w-3" /> Applicants
                  </button>
                  <CheckCircle className="h-5 w-5 text-green-600" />
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Modals */}
      {showCreate && (
        <CreateEventModal orgId={user.id} onClose={() => setShowCreate(false)} onCreated={loadData} />
      )}
      {showApplicants && (
        <ApplicantsModal event={showApplicants} onClose={() => setShowApplicants(null)} onRefresh={loadData} />
      )}
      {showTalentProgress && (
        <TalentProgressModal orgId={user.id} onClose={() => setShowTalentProgress(false)} />
      )}
    </div>
  );
}

/* ─────────────── Main Events Page ─────────────── */

export default function Events() {
  const { user } = useAuth();

  return (
    <div className="section-padding">
      <div className="container max-w-5xl">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-foreground">Events</h1>
          <p className="mt-1 text-muted-foreground">
            {user.role === 'organization' ? 'Manage your events and talent' : 'Discover and track opportunities'}
          </p>
        </div>

        {user.role === 'user' ? <UserEventsView user={user} /> : <OrgEventsView user={user} />}
      </div>
    </div>
  );
}

/* ─────────────── Link To Employer Button + Modal ─────────────── */
function LinkToEmployerButton({ orgId, userId, onLinked }) {
  const [open, setOpen] = useState(false);
  const [employer, setEmployer] = useState('');
  const [contact, setContact] = useState('');
  const [note, setNote] = useState('');

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!employer.trim()) return;
    createLink(orgId, userId, employer.trim(), contact.trim() || null, note.trim() || null);
    setOpen(false);
    setEmployer(''); setContact(''); setNote('');
    onLinked?.();
  };

  return (
    <>
      <button onClick={() => setOpen(true)} className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted">
        <LinkIcon className="h-3 w-3" /> Link
      </button>
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 p-4">
          <div className="w-full max-w-md rounded-2xl bg-background p-6 card-shadow">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-bold text-foreground">Link Talent to Employer</h2>
              <button onClick={() => setOpen(false)} className="rounded-lg p-1 text-muted-foreground hover:text-foreground"><X className="h-5 w-5" /></button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-foreground">Employer / Company</label>
                <input type="text" value={employer} onChange={(e) => setEmployer(e.target.value)} className="w-full rounded-lg border border-input px-3 py-2" />
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-foreground">Contact (email/phone)</label>
                <input type="text" value={contact} onChange={(e) => setContact(e.target.value)} className="w-full rounded-lg border border-input px-3 py-2" />
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-foreground">Note</label>
                <textarea value={note} onChange={(e) => setNote(e.target.value)} className="w-full rounded-lg border border-input px-3 py-2" rows={3} />
              </div>
              <div className="flex justify-end gap-2">
                <button type="button" onClick={() => setOpen(false)} className="rounded-lg border px-4 py-2">Cancel</button>
                <button type="submit" className="rounded-lg bg-primary px-4 py-2 text-white">Link</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
