import { useState } from "react";
import { Link, useLocation } from "react-router-dom";
import {
  Menu,
  X,
  User,
  LogOut,
  Calendar,
  LayoutDashboard,
  Award,
  Users,
} from "lucide-react";
import { useAuth } from "../context/AuthContext";

interface NavItem {
  to: string;
  label: string;
  icon: React.ElementType;
}

export default function Header() {
  const [mobileOpen, setMobileOpen] = useState(false);
  const { user, signOut } = useAuth();
  const location = useLocation();

  const isActive = (path: string) =>
    `text-sm font-medium transition-colors hover:text-primary ${
      location.pathname === path ? "text-primary" : "text-foreground/70"
    }`;

  const publicLinks = [
    { to: "/", label: "Home" },
    { to: "/about", label: "About" },
    { to: "/who-this-is-for", label: "Who This Is For" },
  ];

  const studentLinks: NavItem[] = [
    { to: "/events", label: "Events", icon: Calendar },
    { to: "/skills", label: "Skills", icon: Award },
    { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
    { to: "/profile", label: "Profile", icon: User },
  ];

  const companyLinks: NavItem[] = [
    { to: "/events", label: "Events", icon: Calendar },
    { to: "/candidates", label: "Candidates", icon: Users },
    { to: "/skills", label: "Skills", icon: Award },
    { to: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
    { to: "/profile", label: "Profile", icon: User },
  ];

  const authLinks = user?.role === "company" ? companyLinks : studentLinks;

  return (
    <header className="sticky top-0 z-50 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-16 items-center justify-between">
        {/* Logo */}
        <div className="flex items-center gap-6">
          <Link
            to={user ? "/dashboard" : "/"}
            className="flex items-center gap-2"
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
              <span className="text-sm font-bold text-primary-foreground">
                SZ
              </span>
            </div>
            <span className="text-lg font-bold text-foreground">SkillZone</span>
          </Link>

          {/* Authenticated nav */}
          {user && (
            <nav className="hidden items-center gap-4 md:flex">
              {authLinks.map((link) => (
                <Link key={link.to} to={link.to} className={isActive(link.to)}>
                  {link.label}
                </Link>
              ))}
            </nav>
          )}
        </div>

        {/* Public nav — only shown when logged out */}
        {!user && (
          <nav className="hidden items-center gap-6 lg:flex">
            {publicLinks.map((link) => (
              <Link key={link.to} to={link.to} className={isActive(link.to)}>
                {link.label}
              </Link>
            ))}
          </nav>
        )}

        {/* Right side */}
        <div className="hidden items-center gap-3 md:flex">
          {!user ? (
            <>
              <Link
                to="/signin"
                className="rounded-lg px-4 py-2 text-sm font-medium text-foreground/70 transition-colors hover:text-primary"
              >
                Sign In
              </Link>
              <Link
                to="/signup"
                className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
              >
                Sign Up
              </Link>
            </>
          ) : (
            <div className="flex items-center gap-3">
              <span className="text-sm text-muted-foreground">{user.name}</span>
              <span className="rounded-full bg-accent px-2 py-0.5 text-xs font-medium text-accent-foreground capitalize">
                {user.role}
              </span>
              <button
                onClick={signOut}
                className="flex items-center gap-1 rounded-lg px-3 py-2 text-sm text-muted-foreground transition-colors hover:text-destructive"
                title="Sign out"
              >
                <LogOut className="h-4 w-4" />
              </button>
            </div>
          )}
        </div>

        {/* Mobile hamburger */}
        <button
          onClick={() => setMobileOpen(!mobileOpen)}
          className="flex items-center justify-center rounded-lg p-2 md:hidden"
        >
          {mobileOpen ? (
            <X className="h-5 w-5" />
          ) : (
            <Menu className="h-5 w-5" />
          )}
        </button>
      </div>

      {/* Mobile menu */}
      {mobileOpen && (
        <div className="border-t border-border bg-background p-4 md:hidden">
          <nav className="flex flex-col gap-3">
            {!user &&
              publicLinks.map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  onClick={() => setMobileOpen(false)}
                  className={isActive(link.to)}
                >
                  {link.label}
                </Link>
              ))}
            {user &&
              authLinks.map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  onClick={() => setMobileOpen(false)}
                  className={isActive(link.to)}
                >
                  {link.label}
                </Link>
              ))}
            <div className="mt-2 border-t border-border pt-3">
              {!user ? (
                <div className="flex flex-col gap-2">
                  <Link
                    to="/signin"
                    onClick={() => setMobileOpen(false)}
                    className="rounded-lg border border-border px-4 py-2 text-center text-sm font-medium"
                  >
                    Sign In
                  </Link>
                  <Link
                    to="/signup"
                    onClick={() => setMobileOpen(false)}
                    className="rounded-lg bg-primary px-4 py-2 text-center text-sm font-medium text-primary-foreground"
                  >
                    Sign Up
                  </Link>
                </div>
              ) : (
                <div className="flex flex-col gap-2">
                  <p className="text-sm font-medium text-foreground">
                    {user.name}
                  </p>
                  <button
                    onClick={() => {
                      signOut();
                      setMobileOpen(false);
                    }}
                    className="flex items-center gap-2 rounded-lg px-4 py-2 text-sm text-destructive"
                  >
                    <LogOut className="h-4 w-4" /> Sign Out
                  </button>
                </div>
              )}
            </div>
          </nav>
        </div>
      )}
    </header>
  );
}
