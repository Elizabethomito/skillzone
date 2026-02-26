/**
 * Events.tsx — Main event listing page.
 *
 * Company view:  Create / Edit events, Start / End events,
 *               Generate & display QR check-in code, Manage guests.
 * Student view:  Browse events, Register / Unregister (offline-safe),
 *               Scan QR check-in (offline-safe via Dexie queue).
 */

import { useState, useEffect, useCallback, useRef } from "react";
import {
  Calendar, Plus, QrCode, Scan, Users, Play, Square, ChevronRight,
  Clock, MapPin, Award, Wifi, WifiOff, Edit, Trash2, X, Check, AlertTriangle,
} from "lucide-react";
import toast from "react-hot-toast";
import QRCode from "qrcode";
import { Html5QrcodeScanner } from "html5-qrcode";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { useAuth } from "../context/AuthContext";
import { useOnlineStatus } from "../hooks/useOnlineStatus";
import {
  apiListEvents, apiCreateEvent, apiUpdateEvent, apiUpdateEventStatus,
  apiGetCheckinCode, apiRegisterForEvent, apiUnregisterFromEvent,
  apiGetEventRegistrations, apiListSkills, apiResolveConflict, apiKickRegistration,
  apiGetMyRegistrations,
  type ApiEvent, type Skill, type RegistrationWithStudent,
} from "../lib/api";
import {
  db, cacheEvents, cacheSkills, enqueueCheckIn, enqueueRegister, enqueueUnregister,
  type CachedEvent,
} from "../db/database";
import { runSync } from "../lib/sync";

// ─── Helpers ──────────────────────────────────────────────────────────────────

function statusBadge(status: string) {
  const map: Record<string, string> = {
    upcoming: "bg-blue-100 text-blue-700",
    active:   "bg-green-100 text-green-700",
    completed: "bg-gray-100 text-gray-500",
  };
  return `rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${map[status] ?? "bg-accent text-accent-foreground"}`;
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    dateStyle: "medium", timeStyle: "short",
  });
}

// ─── Modals ───────────────────────────────────────────────────────────────────

interface CreateEditEventModalProps {
  skills: Skill[];
  existing?: ApiEvent | null;
  onClose: () => void;
  onSaved: () => void;
}

function CreateEditEventModal({ skills, existing, onClose, onSaved }: CreateEditEventModalProps) {
  const isEdit = !!existing;
  const [title, setTitle]       = useState(existing?.title ?? "");
  const [desc, setDesc]         = useState(existing?.description ?? "");
  const [location, setLocation] = useState(existing?.location ?? "");
  const [startTime, setStart]   = useState(existing?.start_time?.slice(0, 16) ?? "");
  const [endTime, setEnd]       = useState(existing?.end_time?.slice(0, 16) ?? "");
  const [capacity, setCapacity] = useState(String(existing?.capacity ?? ""));
  const [selectedSkills, setSelectedSkills] = useState<string[]>(
    existing?.skills?.map((s) => s.id) ?? []
  );
  const [error, setError]   = useState("");
  const [saving, setSaving] = useState(false);

  const toggleSkill = (id: string) =>
    setSelectedSkills((prev) => prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!title.trim() || !startTime || !endTime) { setError("Title, start and end time are required."); return; }
    setSaving(true);
    try {
      const payload = {
        title: title.trim(),
        description: desc,
        location,
        start_time: new Date(startTime).toISOString(),
        end_time:   new Date(endTime).toISOString(),
        skill_ids:  selectedSkills,
        capacity:   capacity ? parseInt(capacity) : undefined,
      };
      if (isEdit && existing) {
        await apiUpdateEvent(existing.id, payload);
        toast.success("Event updated");
      } else {
        await apiCreateEvent(payload);
        toast.success("Event created");
      }
      onSaved();
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to save event");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-2xl bg-background p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-xl font-bold">{isEdit ? "Edit Event" : "Create Event"}</h2>
          <button onClick={onClose} className="rounded p-1 hover:bg-muted"><X className="h-5 w-5" /></button>
        </div>
        {error && <div className="mb-3 rounded-lg bg-destructive/10 px-4 py-2 text-sm text-destructive">{error}</div>}
        <form onSubmit={handleSubmit} className="space-y-4">
          {(["Title", "Location"] as const).map((label) => (
            <div key={label}>
              <label className="mb-1 block text-sm font-medium">{label}</label>
              <input value={label === "Title" ? title : location}
                onChange={(e) => label === "Title" ? setTitle(e.target.value) : setLocation(e.target.value)}
                className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none focus:border-primary" />
            </div>
          ))}
          <div>
            <label className="mb-1 block text-sm font-medium">Description</label>
            <textarea value={desc} onChange={(e) => setDesc(e.target.value)} rows={3}
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none focus:border-primary" />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="mb-1 block text-sm font-medium">Start Time</label>
              <input type="datetime-local" value={startTime} onChange={(e) => setStart(e.target.value)}
                className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none focus:border-primary" />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium">End Time</label>
              <input type="datetime-local" value={endTime} onChange={(e) => setEnd(e.target.value)}
                className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none focus:border-primary" />
            </div>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium">Capacity (leave blank = unlimited)</label>
            <input type="number" min="1" value={capacity} onChange={(e) => setCapacity(e.target.value)}
              placeholder="e.g. 30"
              className="w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none focus:border-primary" />
          </div>
          <div>
            <label className="mb-2 block text-sm font-medium">Skill Badges Awarded</label>
            <div className="flex flex-wrap gap-2">
              {skills.map((s) => (
                <button key={s.id} type="button" onClick={() => toggleSkill(s.id)}
                  className={`rounded-full border px-3 py-1 text-xs font-medium transition ${
                    selectedSkills.includes(s.id)
                      ? "border-primary bg-primary text-primary-foreground"
                      : "border-border text-muted-foreground hover:border-primary"
                  }`}>
                  {s.name}
                </button>
              ))}
            </div>
          </div>
          <button type="submit" disabled={saving}
            className="w-full rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50">
            {saving ? "Saving…" : isEdit ? "Save Changes" : "Create Event"}
          </button>
        </form>
      </div>
    </div>
  );
}

// ─── Registration QR Modal (Host) ────────────────────────────────────────────
// Encodes { type:"register", eventId } — generated fully client-side.
// Students scan this with the Scan QR button to register for the event.

function RegistrationQRModal({ event, onClose }: { event: ApiEvent; onClose: () => void }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const payload = JSON.stringify({ type: "register", eventId: event.id });
    if (canvasRef.current) {
      QRCode.toCanvas(canvasRef.current, payload, {
        width: 280,
        margin: 2,
        color: { dark: "#000000", light: "#ffffff" },
      }).catch(() => toast.error("Could not generate QR code"));
    }
  }, [event.id]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-sm rounded-2xl bg-background p-6 shadow-xl text-center">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-bold">Registration QR Code</h2>
          <button onClick={onClose} className="rounded p-1 hover:bg-muted"><X className="h-5 w-5" /></button>
        </div>
        <p className="mb-1 text-sm font-medium text-foreground truncate">{event.title}</p>
        <p className="mb-4 text-xs text-muted-foreground">Students scan this to register</p>
        <canvas ref={canvasRef} className="mx-auto rounded-xl border border-border shadow-sm" />
        <p className="mt-4 text-xs text-muted-foreground">
          <Users className="mr-1 inline h-3 w-3" />
          Show this to attendees — works offline ✓
        </p>
      </div>
    </div>
  );
}

// ─── Check-in QR Modal (Host, active events only) ─────────────────────────────
// Encodes a signed JWT token from the backend. Students scan this during
// the live event to record their physical attendance.

function CheckInQRModal({ event, onClose }: { event: ApiEvent; onClose: () => void }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [expiresIn, setExpiresIn] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    apiGetCheckinCode(event.id)
      .then(async (res) => {
        setExpiresIn(res.expires_in_seconds);
        const payload = JSON.stringify({ token: res.token });
        if (canvasRef.current) {
          await QRCode.toCanvas(canvasRef.current, payload, {
            width: 280,
            margin: 2,
            color: { dark: "#000000", light: "#ffffff" },
          });
        }
      })
      .catch(() => toast.error("Could not generate check-in QR code"))
      .finally(() => setLoading(false));
  }, [event.id]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-sm rounded-2xl bg-background p-6 shadow-xl text-center">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-bold">Check-in QR Code</h2>
          <button onClick={onClose} className="rounded p-1 hover:bg-muted"><X className="h-5 w-5" /></button>
        </div>
        <p className="mb-1 text-sm font-medium text-foreground truncate">{event.title}</p>
        <p className="mb-4 text-xs text-muted-foreground">Students scan to record attendance</p>
        {loading ? (
          <div className="flex h-[280px] items-center justify-center">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          </div>
        ) : (
          <canvas ref={canvasRef} className="mx-auto rounded-xl border border-border shadow-sm" />
        )}
        {expiresIn > 0 && (
          <p className="mt-3 text-xs text-muted-foreground">
            <Clock className="mr-1 inline h-3 w-3" />
            Valid for {Math.floor(expiresIn / 3600)} hour{Math.floor(expiresIn / 3600) !== 1 ? "s" : ""} — display on projector
          </p>
        )}
        <p className="mt-2 text-xs text-green-600 font-medium">
          Students scan while offline — badges sync later ✓
        </p>
      </div>
    </div>
  );
}

// ─── QR Scanner (Student) ─────────────────────────────────────────────────────
// Handles two QR payload types:
//   { type: "register", eventId }  → enqueue a registration
//   { token: "<jwt>" }             → enqueue an attendance check-in

function QRScannerModal({ userId, onClose }: { userId: string; onClose: () => void }) {
  const scannerRef = useRef<Html5QrcodeScanner | null>(null);
  const [scanned, setScanned] = useState(false);
  const [result, setResult] = useState<string | null>(null);

  useEffect(() => {
    const scanner = new Html5QrcodeScanner(
      "qr-reader",
      { fps: 10, qrbox: { width: 250, height: 250 } },
      false
    );

    scanner.render(
      async (decodedText) => {
        if (scanned) return;
        setScanned(true);
        scanner.clear().catch(() => {});

        try {
          const payload = JSON.parse(decodedText) as
            | { type: "register"; eventId: string }
            | { token: string };

          // ── Registration QR ─────────────────────────────────────────────
          if ("type" in payload && payload.type === "register") {
            if (!payload.eventId) throw new Error("missing eventId");
            await enqueueRegister(userId, payload.eventId);
            setResult("✓ Registration queued! Will confirm when you reconnect.");
            toast.success("Registered via QR!");
            if (navigator.onLine) runSync(userId).catch(console.warn);
            return;
          }

          // ── Check-in QR (signed JWT) ────────────────────────────────────
          if ("token" in payload) {
            if (!payload.token) throw new Error("missing token");
            const parts  = payload.token.split(".");
            const claims = JSON.parse(atob(parts[1]));
            const eventId: string = claims.event_id;
            await enqueueCheckIn(userId, eventId, payload.token);
            setResult("✓ Check-in queued! Badges will appear when you reconnect.");
            toast.success("QR scanned! Check-in queued.");
            if (navigator.onLine) runSync(userId).catch(console.warn);
            return;
          }

          throw new Error("unrecognised QR format");
        } catch (err) {
          setResult(`✗ Invalid QR code: ${err instanceof Error ? err.message : "unknown error"}`);
          toast.error("Invalid QR code");
        }
      },
      () => { /* partial scan frames — normal */ }
    );

    scannerRef.current = scanner;
    return () => { scanner.clear().catch(() => {}); };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-sm rounded-2xl bg-background p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-bold">Scan Check-in QR</h2>
          <button onClick={onClose} className="rounded p-1 hover:bg-muted"><X className="h-5 w-5" /></button>
        </div>
        {!scanned ? (
          <>
            <p className="mb-3 text-sm text-muted-foreground">
              Point your camera at the QR code displayed by the event host.
            </p>
            <div id="qr-reader" className="rounded-lg overflow-hidden" />
          </>
        ) : (
          <div className={`rounded-lg p-4 text-sm font-medium ${
            result?.startsWith("✓") ? "bg-green-50 text-green-700" : "bg-red-50 text-red-700"
          }`}>
            {result}
          </div>
        )}
        <button onClick={onClose} className="mt-4 w-full rounded-lg border border-input py-2 text-sm hover:bg-muted">
          {scanned ? "Done" : "Cancel"}
        </button>
      </div>
    </div>
  );
}

// ─── Manage Guests Modal (Host) ───────────────────────────────────────────────

function ManageGuestsModal({ event, onClose }: { event: ApiEvent; onClose: () => void }) {
  const [regs, setRegs] = useState<RegistrationWithStudent[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    apiGetEventRegistrations(event.id)
      .then(setRegs)
      .catch(() => toast.error("Could not load registrations"))
      .finally(() => setLoading(false));
  }, [event.id]);

  useEffect(() => { load(); }, [load]);

  const resolve = async (regId: string, action: "confirm" | "waitlist") => {
    try {
      await apiResolveConflict(event.id, regId, action);
      toast.success(action === "confirm" ? "Confirmed!" : "Waitlisted");
      load();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed");
    }
  };

  const kick = async (regId: string, name: string) => {
    if (!confirm(`Remove ${name} from this event?`)) return;
    try {
      await apiKickRegistration(event.id, regId);
      toast.success(`${name} removed`);
      load();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed");
    }
  };

  const statusColor: Record<string, string> = {
    confirmed:        "bg-green-100 text-green-700",
    conflict_pending: "bg-amber-100 text-amber-700",
    waitlisted:       "bg-blue-100 text-blue-700",
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-2xl max-h-[90vh] overflow-y-auto rounded-2xl bg-background p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold">Manage Guests</h2>
            <p className="text-sm text-muted-foreground">{event.title}</p>
          </div>
          <button onClick={onClose} className="rounded p-1 hover:bg-muted"><X className="h-5 w-5" /></button>
        </div>
        {loading ? (
          <div className="flex justify-center py-8">
            <div className="h-6 w-6 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          </div>
        ) : regs.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">No registrations yet.</p>
        ) : (
          <div className="space-y-2">
            {regs.map((reg) => (
              <div key={reg.id}
                className="flex items-center justify-between rounded-xl border border-border bg-card p-4">
                <div>
                  <p className="font-medium text-foreground">{reg.student_name}</p>
                  <p className="text-xs text-muted-foreground">{reg.student_email}</p>
                </div>
                <div className="flex items-center gap-2">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusColor[reg.status] ?? ""}`}>
                    {reg.status.replace("_", " ")}
                  </span>
                  {reg.status === "conflict_pending" && (
                    <>
                      <button onClick={() => resolve(reg.id, "confirm")}
                        className="rounded-lg bg-green-600 px-2 py-1 text-xs text-white hover:bg-green-700">
                        <Check className="h-3 w-3" />
                      </button>
                      <button onClick={() => resolve(reg.id, "waitlist")}
                        className="rounded-lg bg-blue-500 px-2 py-1 text-xs text-white hover:bg-blue-600">
                        Waitlist
                      </button>
                    </>
                  )}
                  <button onClick={() => kick(reg.id, reg.student_name)}
                    className="rounded-lg p-1.5 text-destructive hover:bg-destructive/10">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Event Card ───────────────────────────────────────────────────────────────

interface EventCardProps {
  event: ApiEvent;
  isHost: boolean;
  isStudent: boolean;
  myRegistrationStatus?: string;
  onRefresh: () => void;
  userId: string;
  online: boolean;
}

function EventCard({ event, isHost, isStudent, myRegistrationStatus, onRefresh, userId, online }: EventCardProps) {
  const [showRegQR, setShowRegQR]       = useState(false);
  const [showCheckInQR, setShowCheckInQR] = useState(false);
  const [showScanner, setShowScanner]   = useState(false);
  const [showGuests, setShowGuests]     = useState(false);
  const [showEdit, setShowEdit]         = useState(false);
  const [skills, setSkills]             = useState<Skill[]>([]);
  const [busy, setBusy]                 = useState(false);

  const isRegistered = !!myRegistrationStatus;

  const changeStatus = async (status: "upcoming" | "active" | "completed") => {
    setBusy(true);
    try {
      await apiUpdateEventStatus(event.id, status);
      toast.success(`Event ${status}`);
      onRefresh();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : "Failed");
    } finally {
      setBusy(false);
    }
  };

  const handleRegister = async () => {
    setBusy(true);
    try {
      // Always try the network first — even if we think we're offline,
      // the request might succeed (flaky connection).
      await apiRegisterForEvent(event.id);
      toast.success("Registered!");
      onRefresh();
    } catch {
      // Network or server error — queue for later sync silently.
      try {
        await enqueueRegister(userId, event.id);
        toast("Saved offline — will register when reconnected", { icon: "📶" });
        onRefresh();
      } catch (qErr) {
        toast.error("Could not queue registration — please try again");
      }
    } finally {
      setBusy(false);
    }
  };

  const handleUnregister = async () => {
    if (!confirm("Unregister from this event?")) return;
    setBusy(true);
    try {
      // Always try the network first.
      await apiUnregisterFromEvent(event.id);
      toast.success("Unregistered");
      onRefresh();
    } catch {
      // Queue for later sync.
      try {
        await enqueueUnregister(userId, event.id);
        toast("Saved offline — will unregister when reconnected", { icon: "📶" });
        onRefresh();
      } catch (qErr) {
        toast.error("Could not queue unregistration — please try again");
      }
    } finally {
      setBusy(false);
    }
  };

  // Load skills for the edit modal lazily
  const openEdit = async () => {
    const sk = await apiListSkills().catch(() => []);
    setSkills(sk);
    setShowEdit(true);
  };

  return (
    <div className="rounded-2xl bg-card border border-border p-5 shadow-sm flex flex-col gap-3">
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <h3 className="font-bold text-foreground truncate">{event.title}</h3>
          <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">{event.description}</p>
        </div>
        <span className={statusBadge(event.status)}>{event.status}</span>
      </div>

      {/* Meta */}
      <div className="flex flex-col gap-1 text-xs text-muted-foreground">
        <span className="flex items-center gap-1"><MapPin className="h-3 w-3" />{event.location || "TBD"}</span>
        <span className="flex items-center gap-1"><Clock className="h-3 w-3" />{formatDate(event.start_time)}</span>
        {event.slots_remaining !== undefined && (
          <span className="flex items-center gap-1">
            <Users className="h-3 w-3" />
            {event.slots_remaining === 0
              ? <span className="text-destructive font-medium">Full</span>
              : `${event.slots_remaining} slot${event.slots_remaining !== 1 ? "s" : ""} left`}
          </span>
        )}
      </div>

      {/* Skill badges */}
      {event.skills && event.skills.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {event.skills.map((s) => (
            <span key={s.id} className="flex items-center gap-1 rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary font-medium">
              <Award className="h-3 w-3" />{s.name}
            </span>
          ))}
        </div>
      )}

      {/* Actions */}
      <div className="flex flex-wrap gap-2 pt-1">
        {/* Host actions */}
        {isHost && (
          <>
            {event.status === "upcoming" && (
              <>
                <button onClick={() => changeStatus("active")} disabled={busy}
                  className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700 disabled:opacity-50">
                  <Play className="h-3 w-3" /> Start
                </button>
                {/* Registration QR available on upcoming events so host can share before going live */}
                <button onClick={() => setShowRegQR(true)}
                  className="flex items-center gap-1 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90">
                  <QrCode className="h-3 w-3" /> Reg. QR
                </button>
              </>
            )}
            {event.status === "active" && (
              <>
                <button onClick={() => setShowRegQR(true)}
                  className="flex items-center gap-1 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90">
                  <QrCode className="h-3 w-3" /> Reg. QR
                </button>
                <button onClick={() => setShowCheckInQR(true)}
                  className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700">
                  <Scan className="h-3 w-3" /> Check-in QR
                </button>
                <button onClick={() => changeStatus("completed")} disabled={busy}
                  className="flex items-center gap-1 rounded-lg bg-amber-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-amber-600 disabled:opacity-50">
                  <Square className="h-3 w-3" /> End Early
                </button>
              </>
            )}
            <button onClick={() => setShowGuests(true)}
              className="flex items-center gap-1 rounded-lg border border-input px-3 py-1.5 text-xs font-medium hover:bg-muted">
              <Users className="h-3 w-3" /> Guests
            </button>
            <button onClick={openEdit}
              className="flex items-center gap-1 rounded-lg border border-input px-3 py-1.5 text-xs font-medium hover:bg-muted">
              <Edit className="h-3 w-3" /> Edit
            </button>
          </>
        )}

        {/* Student actions */}
        {isStudent && event.status !== "completed" && (
          <>
            {!isRegistered ? (
              <button onClick={handleRegister} disabled={busy}
                className="flex items-center gap-1 rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50">
                {online ? <ChevronRight className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
                {online ? "Register" : "Queue Register"}
              </button>
            ) : (
              <button onClick={handleUnregister} disabled={busy}
                className="flex items-center gap-1 rounded-lg bg-destructive/10 px-3 py-1.5 text-xs font-medium text-destructive hover:bg-destructive/20 disabled:opacity-50">
                Unregister
              </button>
            )}
            {event.status === "active" && (
              <button onClick={() => setShowScanner(true)}
                className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700">
                <Scan className="h-3 w-3" /> Scan QR
              </button>
            )}
          </>
        )}

        {isRegistered && (
          <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700">
            ✓ {myRegistrationStatus}
          </span>
        )}
      </div>

      {/* Sub-modals */}
      {showRegQR && (
        <RegistrationQRModal event={event} onClose={() => setShowRegQR(false)} />
      )}
      {showCheckInQR && (
        <CheckInQRModal event={event} onClose={() => setShowCheckInQR(false)} />
      )}
      {showScanner && <QRScannerModal userId={userId} onClose={() => setShowScanner(false)} />}
      {showGuests && <ManageGuestsModal event={event} onClose={() => setShowGuests(false)} />}
      {showEdit && (
        <CreateEditEventModal
          skills={skills}
          existing={event}
          onClose={() => setShowEdit(false)}
          onSaved={onRefresh}
        />
      )}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function Events() {
  const { user } = useAuth();
  const online = useOnlineStatus();
  const queryClient = useQueryClient();

  const [showCreate, setShowCreate]     = useState(false);
  const [showScanner, setShowScanner]   = useState(false);
  const [filterStatus, setFilterStatus] = useState<string>("all");
  const [search, setSearch]             = useState("");

  const isCompany = user?.role === "company";
  const isStudent = user?.role === "student";

  // ── Events query (SWR with Dexie initialData) ────────────────────────────
  const { data: events = [], isLoading: eventsLoading, refetch: refetchEvents } = useQuery<ApiEvent[]>({
    queryKey: ["events"],
    networkMode: "offlineFirst",
    staleTime: 30_000,
    queryFn: async () => {
      const evts = await apiListEvents();
      cacheEvents(evts as any).catch(() => {});
      return evts;
    },
    initialData: () => {
      // Synchronously seed from Dexie while the network request is in-flight.
      // React Query will replace this with fresh data as soon as the fetch resolves.
      return undefined; // populated async below via placeholderData
    },
    placeholderData: () => [] as ApiEvent[],
  });

  // ── Skills query ─────────────────────────────────────────────────────────
  const { data: skills = [] } = useQuery<Skill[]>({
    queryKey: ["skills"],
    networkMode: "offlineFirst",
    staleTime: 5 * 60_000,
    queryFn: async () => {
      const sk = await apiListSkills();
      cacheSkills(sk as any).catch(() => {});
      return sk;
    },
    placeholderData: () => [] as Skill[],
  });

  // ── My registrations query (students only) ───────────────────────────────
  const { data: myRegsRaw = [] } = useQuery({
    queryKey: ["my-registrations"],
    enabled: isStudent,
    networkMode: "offlineFirst",
    staleTime: 30_000,
    queryFn: () => apiGetMyRegistrations(),
    placeholderData: [],
  });

  // Map event_id → status for quick lookup
  const myRegs: Record<string, string> = {};
  myRegsRaw.forEach((r) => { myRegs[r.event_id] = r.status; });

  const loading = eventsLoading;

  /** Invalidate and refetch all events-related queries after a mutation. */
  const loadEvents = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ["events"] });
    queryClient.invalidateQueries({ queryKey: ["my-registrations"] });
  }, [queryClient]);

  // Seed events from Dexie if query hasn't returned yet and we're offline
  useEffect(() => {
    if (!online && events.length === 0) {
      db.events.orderBy("start_time").toArray().then((cached) => {
        if (cached.length > 0) {
          queryClient.setQueryData(["events"], cached as unknown as ApiEvent[]);
        }
      }).catch(() => {});
    }
  }, [online, events.length, queryClient]);

  // Open create modal — skills already loaded by query above
  const openCreate = async () => {
    setShowCreate(true);
  };

  const filtered = events.filter((e) => {
    const matchStatus = filterStatus === "all" || e.status === filterStatus;
    const matchSearch = !search || e.title.toLowerCase().includes(search.toLowerCase()) ||
      e.location?.toLowerCase().includes(search.toLowerCase());
    const matchHost = !isCompany || e.host_id === user?.id;
    return matchStatus && matchSearch && (isCompany ? matchHost : true);
  });

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="section-padding">
      <div className="container max-w-5xl">
        {/* Page header */}
        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-3xl font-bold text-foreground">Events</h1>
            <p className="text-sm text-muted-foreground">
              {isCompany ? "Manage your hosted events" : "Browse and register for events"}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {!online && (
              <span className="flex items-center gap-1 rounded-full bg-amber-100 px-3 py-1 text-xs font-medium text-amber-700">
                <WifiOff className="h-3 w-3" /> Offline
              </span>
            )}
            {isStudent && (
              <button onClick={() => setShowScanner(true)}
                className="flex items-center gap-2 rounded-lg border border-input px-4 py-2 text-sm font-medium hover:bg-muted">
                <Scan className="h-4 w-4" /> Scan QR
              </button>
            )}
            {isCompany && (
              <button onClick={openCreate}
                className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
                <Plus className="h-4 w-4" /> New Event
              </button>
            )}
          </div>
        </div>

        {/* Filters */}
        <div className="mb-6 flex flex-col gap-3 sm:flex-row">
          <input value={search} onChange={(e) => setSearch(e.target.value)}
            placeholder="Search events or locations…"
            className="flex-1 rounded-lg border border-input bg-background px-4 py-2 text-sm outline-none focus:border-primary" />
          <div className="flex gap-2">
            {["all", "upcoming", "active", "completed"].map((s) => (
              <button key={s} onClick={() => setFilterStatus(s)}
                className={`rounded-lg px-3 py-2 text-xs font-medium capitalize transition ${
                  filterStatus === s ? "bg-primary text-primary-foreground" : "border border-input text-muted-foreground hover:bg-muted"
                }`}>
                {s}
              </button>
            ))}
          </div>
        </div>

        {/* Event grid */}
        {filtered.length === 0 ? (
          <div className="rounded-2xl bg-card p-12 text-center shadow-sm">
            <Calendar className="mx-auto mb-4 h-12 w-12 text-muted-foreground/50" />
            <p className="text-muted-foreground">
              {search ? "No events match your search." : "No events yet."}
            </p>
            {isCompany && !search && (
              <button onClick={openCreate} className="mt-4 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground hover:bg-primary/90">
                Create your first event
              </button>
            )}
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((event) => (
              <EventCard
                key={event.id}
                event={event}
                isHost={isCompany && event.host_id === user?.id}
                isStudent={isStudent}
                myRegistrationStatus={myRegs[event.id]}
                onRefresh={loadEvents}
                userId={user?.id ?? ""}
                online={online}
              />
            ))}
          </div>
        )}
      </div>

      {/* Global modals */}
      {showCreate && (
        <CreateEditEventModal
          skills={skills}
          onClose={() => setShowCreate(false)}
          onSaved={loadEvents}
        />
      )}
      {showScanner && isStudent && (
        <QRScannerModal userId={user?.id ?? ""} onClose={() => setShowScanner(false)} />
      )}
    </div>
  );
}
