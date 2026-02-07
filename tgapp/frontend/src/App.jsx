import { useEffect, useState, useCallback } from "react";
import { AppProvider, useAppState } from "./hooks/useAppState";
import { ToastProvider } from "./components/Toast";
import Nav from "./components/Nav";
import TodayPage from "./pages/TodayPage";
import WorkoutPage from "./pages/WorkoutPage";
import MealsPage from "./pages/MealsPage";
import ProfilePage from "./pages/ProfilePage";
import OnboardingPage from "./pages/OnboardingPage";
import { api, getAuthToken, getInitData } from "./services/api";
import { expandTg, isTelegram } from "./services/telegram";
import "./styles/app.css";

function AppContent() {
  const { state, dispatch } = useAppState();
  const [authChecked, setAuthChecked] = useState(false);
  const [needsOnboarding, setNeedsOnboarding] = useState(false);

  const checkAuth = useCallback(async () => {
    const token = getAuthToken();
    const initData = getInitData();
    if (!token && !initData) {
      window.location.href = "/login";
      return;
    }
    if (token) {
      try {
        await api("/auth/verify");
      } catch {
        localStorage.removeItem("auth_token");
        window.location.href = "/login";
        return;
      }
    }
    // Check if onboarding is needed
    try {
      const profile = await api("/api/profile/get");
      if (!profile || !profile.sex || !profile.age) {
        setNeedsOnboarding(true);
      }
    } catch (err) {
      if (err.message === "profile_not_found") {
        setNeedsOnboarding(true);
      }
    }
    setAuthChecked(true);
  }, []);

  useEffect(() => {
    if (!isTelegram()) {
      document.documentElement.classList.add("no-telegram");
    }
    expandTg();
    checkAuth();
  }, [checkAuth]);

  const handleOnboardingComplete = useCallback(() => {
    setNeedsOnboarding(false);
    dispatch({ type: "SET_TAB", payload: "today" });
  }, [dispatch]);

  if (!authChecked) {
    return <div className="screen-loader is-loading"><div className="screen-spinner" /></div>;
  }

  if (needsOnboarding) {
    return <OnboardingPage onComplete={handleOnboardingComplete} />;
  }

  const renderPage = () => {
    switch (state.activeTab) {
      case "today": return <TodayPage />;
      case "workout": return <WorkoutPage />;
      case "meals": return <MealsPage />;
      case "profile": return <ProfilePage />;
      default: return <TodayPage />;
    }
  };

  return (
    <>
      <header className="app-header">
        <div className="brand-logo">barzhafit</div>
      </header>
      <main className="app">
        {renderPage()}
      </main>
      <Nav />
    </>
  );
}

export default function App() {
  return (
    <AppProvider>
      <ToastProvider>
        <AppContent />
      </ToastProvider>
    </AppProvider>
  );
}
