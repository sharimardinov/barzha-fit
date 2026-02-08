import { useEffect, useState, useCallback } from "react";
import { AppProvider, useAppState } from "./hooks/useAppState";
import { ToastProvider } from "./components/Toast";
import Nav from "./components/Nav";
import GradualBlur from "./components/GradualBlur";
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
  const [authFailed, setAuthFailed] = useState(false);
  const [needsOnboarding, setNeedsOnboarding] = useState(false);

  const checkAuth = useCallback(async () => {
    const token = getAuthToken();
    const initData = getInitData();
    if (!token && !initData) {
      setAuthFailed(true);
      setAuthChecked(true);
      return;
    }
    if (token) {
      try {
        await api("/auth/verify");
      } catch {
        localStorage.removeItem("auth_token");
        setAuthFailed(true);
        setAuthChecked(true);
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

  if (authFailed) {
    return (
      <div style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", minHeight: "100vh", padding: 32, textAlign: "center" }}>
        <h2 style={{ marginBottom: 12 }}>Требуется авторизация</h2>
        <p className="muted" style={{ marginBottom: 24 }}>Откройте приложение через Telegram или войдите</p>
        <a href="/login" className="btn btn-accent" style={{ textDecoration: "none" }}>Войти</a>
      </div>
    );
  }

  if (needsOnboarding) {
    return <OnboardingPage onComplete={handleOnboardingComplete} />;
  }

  const tab = state.activeTab;

  useEffect(() => {
    const handler = (event) => {
      const t = event?.detail?.tab;
      if (t) {
        dispatch({ type: "SET_TAB", payload: t });
      }
    };
    window.addEventListener("nativeTab", handler);
    return () => window.removeEventListener("nativeTab", handler);
  }, [dispatch]);

  useEffect(() => {
    try {
      if (window?.webkit?.messageHandlers?.nativeNav) {
        window.webkit.messageHandlers.nativeNav.postMessage({ tab });
      }
    } catch (_) {}
  }, [tab]);

  return (
    <>
      <header className="app-header" style={{ padding: "8px 16px 12px" }}>
        <div className="brand">
          <img className="brand-logo" src="/app/bott.png" alt="BarzhaFit" />
        </div>
        <div className="screen-label">{
          tab === "today" ? "Сегодня" :
          tab === "workout" ? "Тренировка" :
          tab === "meals" ? "Еда" :
          tab === "profile" ? "Профиль" : ""
        }</div>
      </header>
      <main className="app">
        <div style={{ display: tab === "today" ? "block" : "none" }}><TodayPage /></div>
        <div style={{ display: tab === "workout" ? "block" : "none" }}><WorkoutPage /></div>
        <div style={{ display: tab === "meals" ? "block" : "none" }}><MealsPage /></div>
        <div style={{ display: tab === "profile" ? "block" : "none" }}><ProfilePage /></div>
      </main>
      <GradualBlur position="bottom" height="8rem" strength={1.5} layers={5} />
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
