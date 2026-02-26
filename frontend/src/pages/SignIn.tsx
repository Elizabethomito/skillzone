import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { LogIn } from "lucide-react";

export default function SignIn() {
  const { signIn } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!email.trim() || !password) {
      setError("Please fill in all fields.");
      return;
    }
    setSubmitting(true);
    try {
      await signIn(email.trim(), password);
      navigate("/dashboard");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Invalid email or password.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex min-h-[80vh] items-center justify-center px-4">
      <div className="w-full max-w-md rounded-2xl bg-card p-8 card-shadow">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-primary">
            <LogIn className="h-6 w-6 text-primary-foreground" />
          </div>
          <h1 className="text-2xl font-bold text-foreground">Welcome Back</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Sign in to your SkillZone account
          </p>
        </div>

        {error && (
          <div className="mb-4 rounded-lg bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Email
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none transition focus:border-primary focus:ring-2 focus:ring-ring/20"
              placeholder="you@example.com"
            />
          </div>
          <div>
            <label className="mb-1.5 block text-sm font-medium text-foreground">
              Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="w-full rounded-lg border border-input bg-background px-4 py-2.5 text-sm text-foreground outline-none transition focus:border-primary focus:ring-2 focus:ring-ring/20"
              placeholder="••••••••"
            />
          </div>
          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground transition hover:bg-primary/90 disabled:opacity-50"
          >
            {submitting ? "Signing in…" : "Sign In"}
          </button>
        </form>

        <div className="mt-5 rounded-xl border border-border bg-muted/40 px-4 py-3">
          <p className="mb-2 text-center text-xs font-medium text-muted-foreground">
            Demo accounts — click to fill
          </p>
          <div className="flex flex-col gap-1.5">
            {[
              { label: "Amara Osei", sub: "Veteran student · 6 badges", email: "amara@student.test" },
              { label: "Baraka Mwangi", sub: "New student · no history", email: "baraka@student.test" },
              { label: "TechCorp Africa", sub: "Company host", email: "host@techcorp.test" },
            ].map(({ label, sub, email: demoEmail }) => (
              <button
                key={demoEmail}
                type="button"
                onClick={() => { setEmail(demoEmail); setPassword("demo1234"); setError(""); }}
                className="flex items-center justify-between rounded-lg border border-border bg-background px-3 py-2 text-left text-xs transition hover:border-primary/60 hover:bg-primary/5"
              >
                <span>
                  <span className="font-medium text-foreground">{label}</span>
                  <span className="ml-1.5 text-muted-foreground">{sub}</span>
                </span>
                <code className="text-muted-foreground">{demoEmail}</code>
              </button>
            ))}
          </div>
          <p className="mt-2 text-center text-xs text-muted-foreground">
            All passwords: <code className="rounded bg-muted px-1">demo1234</code>
          </p>
        </div>

        <p className="mt-4 text-center text-sm text-muted-foreground">
          Don&apos;t have an account?{" "}
          <Link to="/signup" className="font-medium text-primary hover:underline">
            Sign Up
          </Link>
        </p>
      </div>
    </div>
  );
}
