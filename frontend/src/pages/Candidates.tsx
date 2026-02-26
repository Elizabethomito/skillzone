/**
 * Candidates.tsx — Student discovery page for company accounts.
 *
 * Calls GET /api/users/students?skill_id=<id> to find students who hold the
 * selected skill badge.  Multiple skill filters stack (AND).
 *
 * Route: /candidates  (protected, company-only in practice — but any auth'd
 * user can view; it's read-only data).
 */

import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import {
  apiSearchStudents,
  apiListSkills,
  type StudentWithSkills,
  type Skill,
} from "../lib/api";
import { Search, User, Award, X, WifiOff } from "lucide-react";
import { useOnlineStatus } from "../hooks/useOnlineStatus";

// ─── Skill filter pill ────────────────────────────────────────────────────────

function FilterPill({
  skill,
  onRemove,
}: {
  skill: Skill;
  onRemove: (id: string) => void;
}) {
  return (
    <span className="flex items-center gap-1.5 rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
      {skill.name}
      <button
        onClick={() => onRemove(skill.id)}
        className="rounded-full p-0.5 hover:bg-primary/20"
        aria-label={`Remove ${skill.name} filter`}
      >
        <X className="h-3 w-3" />
      </button>
    </span>
  );
}

// ─── Student card ─────────────────────────────────────────────────────────────

function StudentCard({ student }: { student: StudentWithSkills }) {
  return (
    <div className="rounded-xl bg-card p-5 card-shadow">
      <div className="mb-3 flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-primary/10">
          <User className="h-5 w-5 text-primary" />
        </div>
        <div className="min-w-0">
          <p className="truncate font-semibold text-foreground">{student.name}</p>
          <p className="truncate text-xs text-muted-foreground">{student.email}</p>
        </div>
      </div>

      {student.skills && student.skills.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {student.skills.map((us) => (
            <span
              key={us.id}
              className="flex items-center gap-1 rounded-full bg-accent px-2.5 py-0.5 text-xs text-accent-foreground"
            >
              <Award className="h-3 w-3" />
              {us.skill?.name ?? us.skill_id}
            </span>
          ))}
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">No skill badges yet.</p>
      )}
    </div>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────

export default function Candidates() {
  const [searchParams, setSearchParams] = useSearchParams();
  const online = useOnlineStatus();

  // All available skills for the filter dropdown
  const [allSkills, setAllSkills] = useState<Skill[]>([]);

  // Active skill-id filters (AND logic on the backend)
  const [filterIds, setFilterIds] = useState<string[]>(() => {
    const id = searchParams.get("skill_id");
    return id ? [id] : [];
  });

  // Skill picker state
  const [pickerQuery, setPickerQuery] = useState("");
  const [pickerOpen, setPickerOpen] = useState(false);

  // Results
  const [students, setStudents] = useState<StudentWithSkills[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  // Load skill catalogue once
  useEffect(() => {
    void apiListSkills().then(setAllSkills);
  }, []);

  // Run search whenever filters change
  const runSearch = useCallback(async (ids: string[]) => {
    setLoading(true);
    setError("");
    try {
      // The API supports one skill_id at a time for AND filtering via repeated
      // calls; for simplicity we pass the first filter and do client-side
      // intersection for additional filters.
      const firstId = ids[0];
      const results = await apiSearchStudents(firstId);

      // Client-side AND for additional skill filters
      const filtered =
        ids.length <= 1
          ? results
          : results.filter((s) =>
              ids.every((id) =>
                (s.skills ?? []).some((us) => us.skill_id === id)
              )
            );

      setStudents(filtered);
    } catch (e: unknown) {
      setError((e as Error).message ?? "Search failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void runSearch(filterIds);
    // Sync URL param with first filter
    if (filterIds.length > 0) {
      setSearchParams({ skill_id: filterIds[0] }, { replace: true });
    } else {
      setSearchParams({}, { replace: true });
    }
  }, [filterIds]); // eslint-disable-line react-hooks/exhaustive-deps

  const addFilter = (id: string) => {
    if (!filterIds.includes(id)) setFilterIds((prev) => [...prev, id]);
    setPickerQuery("");
    setPickerOpen(false);
  };

  const removeFilter = (id: string) => {
    setFilterIds((prev) => prev.filter((x) => x !== id));
  };

  const pickerResults = allSkills.filter(
    (s) =>
      !filterIds.includes(s.id) &&
      s.name.toLowerCase().includes(pickerQuery.toLowerCase())
  );

  const filterSkills = allSkills.filter((s) => filterIds.includes(s.id));

  return (
    <div className="section-padding">
      <div className="container max-w-4xl">
        {/* Page header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-foreground">
            Candidate Search
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Find students who have earned specific skill badges through
            Skillzone events.
          </p>
        </div>

        {/* Filter bar */}
        <div className="mb-6 rounded-2xl bg-card p-5 card-shadow space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium text-foreground">
              Filter by skill:
            </span>
            {filterSkills.map((s) => (
              <FilterPill key={s.id} skill={s} onRemove={removeFilter} />
            ))}
            {filterIds.length === 0 && (
              <span className="text-sm text-muted-foreground">
                No filter — showing all students with at least one badge.
              </span>
            )}
          </div>

          {/* Skill picker */}
          <div className="relative">
            <div className="flex items-center gap-3 rounded-xl border border-input bg-background px-4 py-2.5">
              <Search className="h-4 w-4 shrink-0 text-muted-foreground" />
              <input
                type="text"
                placeholder="Add a skill filter…"
                value={pickerQuery}
                onFocus={() => setPickerOpen(true)}
                onChange={(e) => {
                  setPickerQuery(e.target.value);
                  setPickerOpen(true);
                }}
                onBlur={() => setTimeout(() => setPickerOpen(false), 150)}
                className="flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
              />
            </div>

            {pickerOpen && pickerResults.length > 0 && (
              <div className="absolute z-10 mt-1 w-full rounded-xl border border-border bg-popover shadow-lg">
                {pickerResults.slice(0, 8).map((s) => (
                  <button
                    key={s.id}
                    onMouseDown={() => addFilter(s.id)}
                    className="flex w-full items-center gap-2 px-4 py-2.5 text-sm text-foreground hover:bg-accent hover:text-accent-foreground first:rounded-t-xl last:rounded-b-xl"
                  >
                    <Award className="h-3.5 w-3.5 text-primary" />
                    {s.name}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Offline notice */}
        {!online && (
          <div className="mb-4 flex items-center gap-2 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-700">
            <WifiOff className="h-4 w-4" />
            You&apos;re offline — results may be stale.
          </div>
        )}

        {/* Loading */}
        {loading && (
          <div className="flex items-center justify-center py-20">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          </div>
        )}

        {/* Error */}
        {!loading && error && (
          <p className="rounded-xl bg-destructive/10 p-6 text-center text-sm text-destructive">
            {error}
          </p>
        )}

        {/* Empty */}
        {!loading && !error && students.length === 0 && (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">
            {filterIds.length > 0
              ? "No students found with all selected skills."
              : "No students with skill badges found yet."}
          </p>
        )}

        {/* Results grid */}
        {!loading && !error && students.length > 0 && (
          <>
            <p className="mb-4 text-sm text-muted-foreground">
              {students.length} candidate{students.length !== 1 ? "s" : ""}{" "}
              found
            </p>
            <div className="grid gap-4 sm:grid-cols-2">
              {students.map((s) => (
                <StudentCard key={s.id} student={s} />
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
