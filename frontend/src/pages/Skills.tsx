/**
 * Skills.tsx — Browsable skills catalogue.
 *
 * Lists every skill from GET /api/skills with a live search filter.
 * Available to all authenticated users.
 * Company users see an extra "Find candidates →" link next to each skill.
 */

import { useState, useEffect, useMemo } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { apiListSkills, type Skill } from "../lib/api";
import { Award, Search, Users, WifiOff } from "lucide-react";
import { useOnlineStatus } from "../hooks/useOnlineStatus";

export default function Skills() {
  const { user } = useAuth();
  const [skills, setSkills] = useState<Skill[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const online = useOnlineStatus();

  useEffect(() => {
    void (async () => {
      try {
        const data = await apiListSkills();
        setSkills(data);
      } catch (e: unknown) {
        setError((e as Error).message ?? "Failed to load skills");
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const filtered = useMemo(() => {
    const q = query.toLowerCase();
    return skills.filter(
      (s) =>
        s.name.toLowerCase().includes(q) ||
        s.description.toLowerCase().includes(q)
    );
  }, [skills, query]);

  return (
    <div className="section-padding">
      <div className="container max-w-4xl">
        {/* Page header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-foreground">Skills Catalogue</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Skills awarded to students who complete Skillzone events.
            {user?.role === "company" &&
              " Click any skill to search for qualified candidates."}
          </p>
        </div>

        {/* Search bar */}
        <div className="mb-6 flex items-center gap-3 rounded-xl border border-input bg-background px-4 py-2.5">
          <Search className="h-4 w-4 shrink-0 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search skills…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
          />
          {query && (
            <button
              onClick={() => setQuery("")}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Clear
            </button>
          )}
        </div>

        {/* Offline notice */}
        {!online && (
          <div className="mb-4 flex items-center gap-2 rounded-lg bg-amber-50 px-4 py-3 text-sm text-amber-700">
            <WifiOff className="h-4 w-4" />
            You&apos;re offline — showing cached skill list.
          </div>
        )}

        {/* States */}
        {loading && (
          <div className="flex items-center justify-center py-20">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          </div>
        )}

        {!loading && error && (
          <p className="rounded-xl bg-destructive/10 p-6 text-center text-sm text-destructive">
            {error}
          </p>
        )}

        {!loading && !error && filtered.length === 0 && (
          <p className="rounded-xl bg-card p-6 text-center text-sm text-muted-foreground card-shadow">
            {query ? `No skills match "${query}".` : "No skills found."}
          </p>
        )}

        {/* Skills grid */}
        {!loading && !error && filtered.length > 0 && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filtered.map((skill) => (
              <div
                key={skill.id}
                className="flex flex-col rounded-xl bg-card p-5 card-shadow"
              >
                <div className="mb-3 flex items-start gap-3">
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10">
                    <Award className="h-4 w-4 text-primary" />
                  </div>
                  <div className="min-w-0">
                    <h3 className="font-semibold leading-snug text-foreground">
                      {skill.name}
                    </h3>
                  </div>
                </div>

                <p className="flex-1 text-xs leading-relaxed text-muted-foreground">
                  {skill.description}
                </p>

                {user?.role === "company" && (
                  <Link
                    to={`/candidates?skill_id=${skill.id}`}
                    className="mt-4 flex items-center gap-1.5 text-xs font-medium text-primary hover:underline"
                  >
                    <Users className="h-3.5 w-3.5" />
                    Find candidates with this skill →
                  </Link>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Count footer */}
        {!loading && !error && (
          <p className="mt-6 text-center text-xs text-muted-foreground">
            {filtered.length} of {skills.length} skill
            {skills.length !== 1 ? "s" : ""}
          </p>
        )}
      </div>
    </div>
  );
}
