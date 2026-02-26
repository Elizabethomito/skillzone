import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Routes, Route, Navigate, useParams } from "react-router-dom";
import { Toaster as HotToaster } from "react-hot-toast";
import { AuthProvider, useAuth } from "./context/AuthContext";
import Header from "./components/Header";
import Footer from "./components/Footer";
import OfflineBanner from "./components/OfflineBanner";
import ProtectedRoute from "./components/ProtectedRoute";
import Home from "./pages/Home";
import About from "./pages/About";
import WhoThisIsFor from "./pages/WhoThisIsFor";
import SignIn from "./pages/SignIn";
import SignUp from "./pages/SignUp";
import Events from "./pages/Events";
import Dashboard from "./pages/Dashboard";
import Profile from "./pages/Profile";
import { PublicProfileView } from "./pages/Profile";
import Candidates from "./pages/Candidates";
import NotFound from "./pages/NotFound";
import { setQueryClient } from "./lib/sync";

const queryClient = new QueryClient();
// Phase 4: give sync.ts a reference so it can invalidate caches after sync runs.
setQueryClient(queryClient);

/** Redirect authenticated users away from public marketing pages. */
function PublicOnlyRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  if (loading) return null;
  if (user) return <Navigate to="/dashboard" replace />;
  return <>{children}</>;
}

/** Thin wrapper so useParams works inside BrowserRouter. */
function PublicProfileWrapper() {
  const { id } = useParams<{ id: string }>();
  if (!id) return <Navigate to="/" replace />;
  return <PublicProfileView userId={id} />;
}

const App = () => (
  <QueryClientProvider client={queryClient}>
    <TooltipProvider>
      {/* shadcn toasters */}
      <Toaster />
      <Sonner />
      {/* react-hot-toast for offline/sync notifications */}
      <HotToaster position="bottom-center" />
      <BrowserRouter>
        <AuthProvider>
          <div className="flex min-h-screen flex-col">
            <Header />
            <OfflineBanner />
            <main className="flex-1">
              <Routes>
                {/* Public marketing pages — hidden once logged in */}
                <Route
                  path="/"
                  element={
                    <PublicOnlyRoute>
                      <Home />
                    </PublicOnlyRoute>
                  }
                />
                <Route
                  path="/about"
                  element={
                    <PublicOnlyRoute>
                      <About />
                    </PublicOnlyRoute>
                  }
                />
                <Route
                  path="/who-this-is-for"
                  element={
                    <PublicOnlyRoute>
                      <WhoThisIsFor />
                    </PublicOnlyRoute>
                  }
                />

                {/* Auth pages */}
                <Route path="/signin" element={<SignIn />} />
                <Route path="/signup" element={<SignUp />} />

                {/* Protected app pages */}
                <Route
                  path="/events"
                  element={
                    <ProtectedRoute>
                      <Events />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/dashboard"
                  element={
                    <ProtectedRoute>
                      <Dashboard />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/profile"
                  element={
                    <ProtectedRoute>
                      <Profile />
                    </ProtectedRoute>
                  }
                />
                <Route
                  path="/skills"
                  element={<Navigate to="/events" replace />}
                />
                <Route
                  path="/profile/:id"
                  element={<PublicProfileWrapper />}
                />
                <Route
                  path="/candidates"
                  element={
                    <ProtectedRoute>
                      <Candidates />
                    </ProtectedRoute>
                  }
                />

                <Route path="*" element={<NotFound />} />
              </Routes>
            </main>
            <Footer />
          </div>
        </AuthProvider>
      </BrowserRouter>
    </TooltipProvider>
  </QueryClientProvider>
);

export default App;
