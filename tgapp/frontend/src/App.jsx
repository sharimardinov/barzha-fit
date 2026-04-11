import { useEffect, useMemo, useState } from "react";
import { ToastProvider } from "./components/Toast";
import WorkoutPage from "./pages/WorkoutPage";
import { api, getAuthToken, getInitData } from "./services/api";
import { expandTg, isTelegram } from "./services/telegram";
import "./styles/app.css";

function AppContent() {
  const authBootstrap = useMemo(() => {
    const token = getAuthToken();
    const initData = getInitData();
    return { token, hasAuthSource: Boolean(token || initData) };
  }, []);
  const [authChecked, setAuthChecked] = useState(!authBootstrap.hasAuthSource);
  const [authFailed, setAuthFailed] = useState(!authBootstrap.hasAuthSource);

  useEffect(() => {
    if (!isTelegram()) {
      document.documentElement.classList.add("no-telegram");
    }
    expandTg();
    if (!authBootstrap.hasAuthSource) return;

    let cancelled = false;
    const run = async () => {
      if (authBootstrap.token) {
        try {
          await api("/auth/verify");
        } catch {
          if (cancelled) return;
          localStorage.removeItem("auth_token");
          setAuthFailed(true);
          setAuthChecked(true);
          return;
        }
      }
      if (!cancelled) {
        setAuthChecked(true);
      }
    };

    void run();
    return () => {
      cancelled = true;
    };
  }, [authBootstrap]);

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

  return (
    <>
      <header className="app-header" style={{ padding: "8px 16px 12px" }}>
        <div className="brand">
          <img className="brand-logo" src="/app/bott.png" alt="BarzhaFit" />
        </div>
        <div className="screen-label">Тренировка</div>
      </header>
      <main className="app">
        <WorkoutPage />
      </main>
    </>
  );
}

export default function App() {
  return (
    <ToastProvider>
      <AppContent />
    </ToastProvider>
  );
}
