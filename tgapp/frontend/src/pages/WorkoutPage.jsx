import { useEffect, useState, useCallback, useRef } from "react";
import { api } from "../services/api";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import { AccordionItem } from "../components/Accordion";
import { postNativeMessage } from "../services/telegram";
import { parsePlan } from "../services/planUtils";

/* Heart rate from HealthKit via native bridge */
function useHeartRate() {
  const [bpm, setBpm] = useState(null);
  useEffect(() => {
    window.onHeartRateUpdate = (value) => {
      if (typeof value === "number" && value > 0) setBpm(value);
    };
    // Request heart rate from native app
    postNativeMessage("requestHeartRate", {});
    const interval = setInterval(() => postNativeMessage("requestHeartRate", {}), 5000);
    return () => {
      clearInterval(interval);
      delete window.onHeartRateUpdate;
    };
  }, []);
  return bpm;
}

/* Heart icon SVG */
function HeartIcon({ size = 16, color = "#ff033e" }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill={color} stroke="none">
      <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
    </svg>
  );
}

function formatDuration(sec) {
  if (!Number.isFinite(sec) || sec < 0) return "00:00";
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

function formatMinutes(seconds) {
  const total = Math.max(0, Math.round(seconds || 0));
  return Math.max(1, Math.round(total / 60));
}

function formatRest(seconds) {
  const total = Math.max(0, Math.round(seconds || 0));
  if (total % 60 === 0) return `${total / 60} мин`;
  return `${total} сек`;
}

function formatNum(value, digits = 1) {
  if (!Number.isFinite(Number(value))) return "—";
  return Number(value).toFixed(digits).replace(/\.0$/, "");
}

function formatStatDate(value) {
  const dt = new Date(value);
  if (Number.isNaN(dt.getTime())) return "—";
  const d = String(dt.getDate()).padStart(2, "0");
  const m = String(dt.getMonth() + 1).padStart(2, "0");
  return `${d}.${m}`;
}

function Sparkline({ points, color, valueKey }) {
  if (!Array.isArray(points) || points.length < 2) {
    return <div className="insights-empty-chart">Недостаточно данных</div>;
  }
  const width = 320;
  const height = 88;
  const pad = 8;
  const values = points.map((point) => Number(point[valueKey]) || 0);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = max - min || 1;
  const step = (width - pad * 2) / Math.max(points.length - 1, 1);

  const path = points.map((point, i) => {
    const x = pad + i * step;
    const value = Number(point[valueKey]) || 0;
    const y = pad + (height - pad * 2) * (1 - (value - min) / span);
    return `${x},${y}`;
  }).join(" ");

  const last = points[points.length - 1];
  const lastY = pad + (height - pad * 2) * (1 - ((Number(last[valueKey]) || 0) - min) / span);

  return (
    <svg className="insights-sparkline" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none">
      <polyline points={path} fill="none" stroke={color} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
      <line x1={pad} y1={lastY} x2={width - pad} y2={lastY} stroke={color} strokeOpacity="0.16" strokeDasharray="3 4" />
      <circle cx={width - pad} cy={lastY} r="3.5" fill={color} />
    </svg>
  );
}

function ProgramEditor({ weekPlan, onSave, onCancel }) {
  const dayNames = ["Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"];
  const [mode, setMode] = useState("text"); // "text" or "structured"

  // Helper: check if items are only rest words
  const isRestItems = (items) => {
    if (!items || items.length === 0) return true;
    const joined = items.join(" ").toLowerCase();
    return /^[\s]*(отдых|выходн|rest|off|—|-)[\s]*$/i.test(joined);
  };

  const [days, setDays] = useState(() =>
    dayNames.map((name, i) => {
      const day = weekPlan[i] || {};
      const rawItems = (day.items || []).map((it) => typeof it === "string" ? it : it.name || "");
      const isRest = isRestItems(rawItems);
      return {
        dayName: name,
        focus: day.focus || "",
        textItems: isRest ? "" : rawItems.join("\n"),
        structuredItems: isRest ? [] : rawItems.map((line) => {
          // Parse "Name | 3x10 | 60 | 120" format
          const parts = line.split("|").map((s) => s.trim());
          const exName = parts[0] || "";
          let sets = "", reps = "", rest = "";
          if (parts[1]) {
            const sxr = parts[1].match(/^(\d+)\s*[xх×]\s*(\d+)$/i);
            if (sxr) { sets = sxr[1]; reps = sxr[2]; }
            else { sets = parts[1]; }
          }
          if (parts[2]) rest = parts[2];
          return { name: exName, sets, reps, rest };
        }),
        isRest,
      };
    })
  );

  const updateDay = (i, field, value) => {
    setDays((prev) => { const d = [...prev]; d[i] = { ...d[i], [field]: value }; return d; });
  };

  const toggleRest = (i) => {
    setDays((prev) => {
      const d = [...prev];
      const isRest = !d[i].isRest;
      d[i] = { ...d[i], isRest, textItems: isRest ? "" : d[i].textItems, focus: isRest ? "" : d[i].focus };
      if (isRest) d[i].structuredItems = [];
      return d;
    });
  };

  const updateStructuredItem = (dayIdx, itemIdx, field, value) => {
    setDays((prev) => {
      const d = [...prev];
      const items = [...d[dayIdx].structuredItems];
      items[itemIdx] = { ...items[itemIdx], [field]: value };
      d[dayIdx] = { ...d[dayIdx], structuredItems: items };
      return d;
    });
  };

  const addStructuredItem = (dayIdx) => {
    setDays((prev) => {
      const d = [...prev];
      d[dayIdx] = { ...d[dayIdx], structuredItems: [...d[dayIdx].structuredItems, { name: "", sets: "", reps: "", rest: "" }] };
      return d;
    });
  };

  const removeStructuredItem = (dayIdx, itemIdx) => {
    setDays((prev) => {
      const d = [...prev];
      d[dayIdx] = { ...d[dayIdx], structuredItems: d[dayIdx].structuredItems.filter((_, j) => j !== itemIdx) };
      return d;
    });
  };

  const save = () => {
    const result = days.map((d, i) => {
      let items;
      if (d.isRest) {
        items = ["Отдых"];
      } else if (mode === "structured") {
        items = d.structuredItems
          .filter((ex) => ex.name.trim())
          .map((ex) => {
            let line = ex.name.trim();
            if (ex.sets && ex.reps) line += ` | ${ex.sets}x${ex.reps}`;
            else if (ex.sets) line += ` | ${ex.sets}`;
            if (ex.rest) line += ` | ${ex.rest}`;
            return line;
          });
        if (items.length === 0) items = ["Отдых"];
      } else {
        items = d.textItems.split("\n").map((s) => s.trim()).filter(Boolean);
        if (items.length === 0) items = ["Отдых"];
      }
      return {
        day: i + 1,
        name: `День ${i + 1}`,
        focus: d.focus,
        items,
        type: d.isRest ? "rest" : "train",
      };
    });
    onSave(result);
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, marginTop: 8 }}>
      {/* Mode toggle */}
      <div style={{ display: "flex", gap: 4, background: "rgba(0,0,0,0.04)", borderRadius: 10, padding: 3 }}>
        {[
          { id: "text", label: "Текстом" },
          { id: "structured", label: "По полям" },
        ].map((m) => (
          <button key={m.id} onClick={() => setMode(m.id)} style={{
            flex: 1, padding: "6px 8px", borderRadius: 8, border: "none", cursor: "pointer",
            fontSize: 12, fontWeight: 600,
            background: mode === m.id ? "var(--white)" : "transparent",
            color: mode === m.id ? "var(--black)" : "var(--muted)",
            boxShadow: mode === m.id ? "0 1px 3px rgba(0,0,0,0.08)" : "none",
            transition: "all 0.2s ease",
          }}>{m.label}</button>
        ))}
      </div>

      {mode === "text" && (
        <div style={{ fontSize: 11, color: "var(--muted)", padding: "0 2px" }}>
          Формат: Название | 3x10 | 60кг | 120сек (по одному на строку)
        </div>
      )}

      {days.map((day, i) => (
        <div key={i} style={{
          border: `1px solid ${day.isRest ? "rgba(0,0,0,0.06)" : "var(--border)"}`,
          borderRadius: 12, padding: 10, background: day.isRest ? "rgba(0,0,0,0.02)" : "var(--white)",
        }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: day.isRest ? 0 : 6 }}>
            <span style={{ fontWeight: 600, fontSize: 13 }}>{day.dayName}</span>
            <button onClick={() => toggleRest(i)} style={{
              padding: "3px 10px", borderRadius: 999, fontSize: 11, fontWeight: 600, border: "none", cursor: "pointer",
              background: day.isRest ? "var(--accent)" : "rgba(0,0,0,0.06)", color: day.isRest ? "var(--white)" : "var(--muted)",
            }}>{day.isRest ? "Отдых ✓" : "Отдых"}</button>
          </div>

          {!day.isRest && (
            <>
              <input type="text" placeholder="Фокус (напр. Грудь / Спина)" value={day.focus} onChange={(e) => updateDay(i, "focus", e.target.value)} style={{ marginBottom: 4, fontSize: 13 }} />

              {mode === "text" ? (
                <textarea
                  placeholder="Упражнения (по одному на строку)"
                  value={day.textItems}
                  onChange={(e) => updateDay(i, "textItems", e.target.value)}
                  rows={3}
                  style={{ fontSize: 13 }}
                />
              ) : (
                <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                  {day.structuredItems.map((ex, j) => (
                    <div key={j} style={{ display: "flex", gap: 4, alignItems: "center" }}>
                      <input
                        type="text" placeholder="Упражнение" value={ex.name}
                        onChange={(e) => updateStructuredItem(i, j, "name", e.target.value)}
                        style={{ flex: 3, fontSize: 12, padding: "6px 8px" }}
                      />
                      <input
                        type="text" placeholder="Подх" value={ex.sets}
                        onChange={(e) => updateStructuredItem(i, j, "sets", e.target.value)}
                        style={{ flex: 0.7, fontSize: 12, padding: "6px 4px", textAlign: "center" }}
                      />
                      <span style={{ fontSize: 12, color: "var(--muted)" }}>×</span>
                      <input
                        type="text" placeholder="Повт" value={ex.reps}
                        onChange={(e) => updateStructuredItem(i, j, "reps", e.target.value)}
                        style={{ flex: 0.7, fontSize: 12, padding: "6px 4px", textAlign: "center" }}
                      />
                      <input
                        type="text" placeholder="Отдых" value={ex.rest}
                        onChange={(e) => updateStructuredItem(i, j, "rest", e.target.value)}
                        style={{ flex: 0.8, fontSize: 12, padding: "6px 4px", textAlign: "center" }}
                      />
                      <button onClick={() => removeStructuredItem(i, j)} style={{
                        background: "none", border: "none", cursor: "pointer", color: "var(--muted)",
                        fontSize: 14, padding: "2px 4px", flexShrink: 0,
                      }}>✕</button>
                    </div>
                  ))}
                  <button onClick={() => addStructuredItem(i)} style={{
                    padding: "4px 10px", borderRadius: 8, border: "1px dashed var(--border)",
                    background: "none", cursor: "pointer", fontSize: 11, color: "var(--muted)",
                  }}>+ упражнение</button>
                </div>
              )}
            </>
          )}
        </div>
      ))}
      <div style={{ display: "flex", gap: 8 }}>
        <button className="btn btn-outline" onClick={onCancel} style={{ flex: 1 }}>Отмена</button>
        <button className="btn btn-accent" onClick={save} style={{ flex: 1 }}>Сохранить</button>
      </div>
    </div>
  );
}

export default function WorkoutPage() {
  const toast = useToast();
  const heartRate = useHeartRate();
  const [plan, setPlan] = useState(null);
  const [planIssues, setPlanIssues] = useState([]);
  const [weekPlan, setWeekPlan] = useState([]);
  const [session, setSession] = useState(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [editingProgram, setEditingProgram] = useState(false);
  const [editDays, setEditDays] = useState([]);
  const [weight, setWeight] = useState("");
  const [reps, setReps] = useState("");
  const [setsMetric, setSetsMetric] = useState("");
  const [timerDisplay, setTimerDisplay] = useState("00:00");
  const [timerProgress, setTimerProgress] = useState(0);
  const [totalTimeMs, setTotalTimeMs] = useState(0);
  const [insights, setInsights] = useState(null);
  const [insightsLoading, setInsightsLoading] = useState(true);
  const [insightsError, setInsightsError] = useState("");
  const [statsExercise, setStatsExercise] = useState("");
  const [statsPeriodDays, setStatsPeriodDays] = useState(90);
  const [insightsAILoading, setInsightsAILoading] = useState(false);
  const tickRef = useRef(null);
  const totalTickRef = useRef(null);
  const lastTimerKeyRef = useRef(null);
  const lastTimerTotalRef = useRef(0);
  const lastSetKeyRef = useRef(null);
  const metricTouchRef = useRef({});
  const wakeLockRef = useRef(null);

  // Wake lock
  const requestWakeLock = useCallback(async () => {
    if (!("wakeLock" in navigator) || wakeLockRef.current) return;
    try {
      wakeLockRef.current = await navigator.wakeLock.request("screen");
      wakeLockRef.current.addEventListener("release", () => { wakeLockRef.current = null; });
    } catch { wakeLockRef.current = null; }
  }, []);

  const releaseWakeLock = useCallback(async () => {
    if (!wakeLockRef.current) return;
    try { await wakeLockRef.current.release(); } catch {}
    wakeLockRef.current = null;
  }, []);

  // Load data
  const loadData = useCallback(async () => {
    try {
      // Fetch plan
      let planData = null;
      let issues = [];
      try {
        const pResp = await api("/api/workout/plan/get");
        planData = pResp?.plan || null;
      } catch (err) {
        if (err.message === "workout_plan_invalid") {
          issues = err.data?.issues || [];
        } else if (err.message === "workout_plan_not_found") {
          issues = ["План тренировок не найден"];
        }
      }

      // Fetch session
      let sessionData = null;
      try {
        const sResp = await api("/api/workout/session/get");
        sessionData = sResp?.session || null;
        if (sResp?.plan) {
          planData = sResp.plan;
          issues = [];
        }
      } catch (err) {
        if (err.message !== "workout_session_not_found") {
          // ignore not found
        }
      }

      setPlan(planData);
      setPlanIssues(issues);
      setSession(sessionData);

      // Fetch full week plan for training program
      try {
        const planResp = await api("/api/plan/get");
        const parsed = parsePlan(planResp?.text || "");
        if (parsed.items?.length > 0) setWeekPlan(parsed.items);
      } catch { /* ignore */ }
    } finally {
      setLoading(false);
    }
  }, []);

  const loadInsights = useCallback(async ({ includeAI = false, exerciseName, periodDays } = {}) => {
    const selectedExercise = exerciseName !== undefined ? exerciseName : statsExercise;
    const selectedPeriod = periodDays || statsPeriodDays;
    if (includeAI) {
      setInsightsAILoading(true);
    } else {
      setInsightsLoading(true);
    }
    try {
      const data = await api("/api/workout/stats/get", {
        exerciseName: selectedExercise || "",
        periodDays: selectedPeriod,
        includeAI,
      });
      setInsights(data || null);
      setInsightsError("");
      if (data?.selectedExercise && !statsExercise) {
        setStatsExercise(data.selectedExercise);
      }
    } catch (err) {
      setInsightsError(formatApiError(err));
    } finally {
      if (includeAI) {
        setInsightsAILoading(false);
      } else {
        setInsightsLoading(false);
      }
    }
  }, [statsExercise, statsPeriodDays]);

  useEffect(() => { loadData(); }, [loadData]);
  useEffect(() => { loadInsights(); }, [loadInsights]);

  // Apply session from action response
  const applySession = useCallback((data) => {
    if (data?.plan) setPlan(data.plan);
    if (data?.session?.status === "completed") {
      setSession(null);
      toast("Тренировка завершена");
      postNativeMessage("workoutTimer", { action: "stop" });
    } else {
      const s = data?.session || null;
      setSession(s);
      postNativeMessage("workoutTimer", { action: s ? "start" : "stop" });
    }
    loadInsights();
  }, [toast, loadInsights]);

  // Timer tick for rest/cardio
  useEffect(() => {
    if (!session || (session.phase !== "rest" && session.phase !== "cardio")) {
      setTimerDisplay("--:--");
      setTimerProgress(0);
      if (tickRef.current) clearInterval(tickRef.current);
      return;
    }

    const tick = () => {
      if (session.status === "paused") {
        const remaining = Math.max(0, session.timerDurationSec || 0);
        setTimerDisplay(formatDuration(remaining));
        setTimerProgress(0);
        return;
      }
      if (!session.timerStartedAt || !session.timerDurationSec) return;
      const end = new Date(session.timerStartedAt).getTime() + session.timerDurationSec * 1000;
      const remainingMs = Math.max(0, end - Date.now());
      const remaining = Math.ceil(remainingMs / 1000);
      setTimerDisplay(formatDuration(remaining));

      // Resolve total for this timer key
      const key = [session.phase, session.timerKind, session.exerciseIndex, session.setIndex].join("|");
      if (key !== lastTimerKeyRef.current) {
        lastTimerKeyRef.current = key;
        lastTimerTotalRef.current = session.timerDurationSec || 0;
      }
      const total = Math.max(lastTimerTotalRef.current, remaining);
      lastTimerTotalRef.current = total;
      const totalMs = total * 1000;
      const progress = totalMs > 0 ? Math.min(1, Math.max(0, 1 - remainingMs / totalMs)) : 0;
      setTimerProgress(progress);

      if (remaining <= 0) {
        clearInterval(tickRef.current);
        loadData();
      }
    };

    tick();
    tickRef.current = setInterval(tick, 120);
    return () => clearInterval(tickRef.current);
  }, [session, loadData]);

  // Total workout time (ms precision for animated digits)
  useEffect(() => {
    if (!session?.startedAt) {
      setTotalTimeMs(0);
      if (totalTickRef.current) clearInterval(totalTickRef.current);
      return;
    }
    const update = () => {
      const now = Date.now();
      const startedAt = new Date(session.startedAt).getTime();
      let elapsedMs = (now - startedAt) - (session.pausedTotalSec || 0) * 1000;
      if (session.status === "paused" && session.pausedAt) {
        elapsedMs -= (now - new Date(session.pausedAt).getTime());
      }
      setTotalTimeMs(Math.max(0, elapsedMs));
    };
    update();
    totalTickRef.current = setInterval(update, 500);
    return () => clearInterval(totalTickRef.current);
  }, [session]);

  // Wake lock sync
  useEffect(() => {
    const shouldLock = session && session.status === "in_progress" && (session.phase === "rest" || session.phase === "cardio");
    if (shouldLock) requestWakeLock();
    else releaseWakeLock();
    return () => releaseWakeLock();
  }, [session, requestWakeLock, releaseWakeLock]);

  // Keep page scroll locked while user adjusts metric circles.
  useEffect(() => {
    const blockIfMetricTarget = (event) => {
      const target = event.target;
      if (!(target instanceof Element)) return;
      if (!target.closest(".workout-focus-circle.is-adjustable")) return;
      event.preventDefault();
    };
    window.addEventListener("wheel", blockIfMetricTarget, { passive: false, capture: true });
    window.addEventListener("touchmove", blockIfMetricTarget, { passive: false, capture: true });
    return () => {
      window.removeEventListener("wheel", blockIfMetricTarget, true);
      window.removeEventListener("touchmove", blockIfMetricTarget, true);
    };
  }, []);

  // Auto-fill metrics for current exercise/set
  useEffect(() => {
    if (!session || session.phase !== "set" || !plan) return;
    const ex = plan.exercises?.[session.exerciseIndex];
    if (!ex || ex.type === "cardio") return;
    const key = `${session.exerciseIndex}:${session.setIndex}`;
    if (key !== lastSetKeyRef.current) {
      lastSetKeyRef.current = key;
      setWeight(String(ex.weight ?? ""));
      setReps(String(ex.reps ?? ""));
      setSetsMetric(String(ex.sets ?? ""));
    }
  }, [session, plan]);

  const doAction = async (path, body) => {
    setActionLoading(true);
    try {
      const data = await api(path, body);
      applySession(data);
    } catch (err) {
      toast(formatApiError(err));
    } finally {
      setActionLoading(false);
    }
  };

  const startWorkout = () => doAction("/api/workout/session/start");
  const endWarmup = () => doAction("/api/workout/session/warmup/end");
  const finishSet = () => {
    const isWarmup = session?.setIndex === 0;
    const plannedWeight = Number(ex?.weight) || 0;
    const plannedReps = Number(ex?.reps) || 0;
    const w = isWarmup ? 0 : Number(weight === "" ? plannedWeight : weight) || 0;
    const r = isWarmup ? 0 : Math.round(Number(reps === "" ? plannedReps : reps) || 0);
    doAction("/api/workout/session/set/finish", {
      exerciseIndex: session.exerciseIndex,
      setIndex: session.setIndex,
      actualWeight: w,
      actualReps: r,
    });
  };
  const endRest = () => doAction("/api/workout/session/rest/end");
  const pause = () => doAction("/api/workout/session/pause");
  const resume = () => doAction("/api/workout/session/resume");
  const stopWorkout = () => doAction("/api/workout/session/stop");
  const refreshInsights = () => loadInsights();
  const refreshAIAdvice = () => loadInsights({ includeAI: true });

  if (loading) return <div className="screen-loader is-loading"><div className="screen-spinner" /></div>;

  const exercises = plan?.exercises || [];
  const ex = exercises[session?.exerciseIndex];
  const phase = session?.phase;
  const status = session?.status;
  const isActive = status === "in_progress";
  const hasValidPlan = exercises.length > 0 && planIssues.length === 0;
  const canAdjustMetrics = phase === "set" && session?.setIndex > 0 && ex?.type !== "cardio";
  const metricWeight = ex?.type === "cardio" ? "—" : (weight === "" ? (ex?.weight ?? "—") : weight);
  const metricReps = ex?.type === "cardio" ? "—" : (reps === "" ? (ex?.reps ?? "—") : reps);
  const metricSets = ex?.type === "cardio" ? "—" : (setsMetric === "" ? (ex?.sets ?? "—") : setsMetric);
  const effectiveSets = ex?.type === "cardio"
    ? 0
    : Math.max(1, Math.round(Number(setsMetric === "" ? ex?.sets : setsMetric) || 1));
  const statsPoints = insights?.points || [];
  const statsMetrics = insights?.metrics || {};
  const statsTrends = insights?.trends || {};
  const statsDirection = statsTrends.direction === "up"
    ? "Прогресс"
    : statsTrends.direction === "down"
      ? "Спад"
      : "Плато";
  const statsDirectionClass = statsTrends.direction === "up"
    ? "is-up"
    : statsTrends.direction === "down"
      ? "is-down"
      : "is-flat";

  const adjustMetric = (kind, direction, multiplier = 1) => {
    if (!canAdjustMetrics || !ex || direction === 0) return;
    const step = kind === "weight" ? 2.5 : 1;
    const delta = direction * step * multiplier;

    if (kind === "weight") {
      setWeight((prev) => {
        const fallback = Number(ex.weight) || 0;
        const base = prev === "" ? fallback : Number(prev);
        const safeBase = Number.isFinite(base) ? base : fallback;
        const next = Math.max(0, Math.round((safeBase + delta) * 10) / 10);
        return String(next);
      });
      return;
    }
    if (kind === "reps") {
      setReps((prev) => {
        const fallback = Number(ex.reps) || 0;
        const base = prev === "" ? fallback : Number(prev);
        const safeBase = Number.isFinite(base) ? base : fallback;
        return String(Math.max(0, Math.round(safeBase + delta)));
      });
      return;
    }
    if (kind === "sets") {
      setSetsMetric((prev) => {
        const fallback = Number(ex.sets) || 1;
        const base = prev === "" ? fallback : Number(prev);
        const safeBase = Number.isFinite(base) ? base : fallback;
        return String(Math.max(1, Math.round(safeBase + delta)));
      });
    }
  };

  const handleMetricWheel = (event, kind) => {
    if (!canAdjustMetrics) return;
    event.preventDefault();
    event.stopPropagation();
    const direction = event.deltaY < 0 ? 1 : -1;
    adjustMetric(kind, direction, event.shiftKey ? 5 : 1);
  };

  const handleMetricTouchStart = (event, kind) => {
    if (!canAdjustMetrics) return;
    const point = event.touches?.[0];
    if (!point) return;
    metricTouchRef.current[kind] = point.clientY;
  };

  const handleMetricTouchMove = (event, kind) => {
    if (!canAdjustMetrics) return;
    event.preventDefault();
    event.stopPropagation();
    const point = event.touches?.[0];
    if (!point) return;
    const prevY = metricTouchRef.current[kind];
    if (typeof prevY !== "number") {
      metricTouchRef.current[kind] = point.clientY;
      return;
    }
    const deltaY = prevY - point.clientY;
    if (Math.abs(deltaY) < 14) return;
    adjustMetric(kind, deltaY > 0 ? 1 : -1);
    metricTouchRef.current[kind] = point.clientY;
  };

  const handleMetricTouchEnd = (kind) => {
    delete metricTouchRef.current[kind];
  };

  // Set label text
  let setLabelText = "";
  if (ex && ex.type !== "cardio") {
    if (phase === "set") {
      setLabelText = session.setIndex === 0 ? "Разминочный подход" : `Подход ${session.setIndex} из ${effectiveSets}`;
    } else if (phase === "rest") {
      if (session.timerKind === "rest") {
        const finished = Math.max(0, session.setIndex - 1);
        setLabelText = finished === 0 ? "Отдых после разминки" : `Отдых после подхода ${finished} из ${effectiveSets}`;
      } else {
        setLabelText = "Отдых между упражнениями";
      }
    }
  }

  // Format total time as HH:MM:SS
  const formatTotal = (ms) => {
    const totalSec = Math.max(0, Math.floor(ms / 1000));
    const h = Math.floor(totalSec / 3600);
    const m = Math.floor((totalSec % 3600) / 60);
    const s = totalSec % 60;
    return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
  };

  // Determine action button
  let actionLabel = "";
  let actionFn = null;
  if (session && status !== "finished" && status !== "stopped" && status !== "completed") {
    if (status === "paused") { actionLabel = "Продолжить"; actionFn = resume; }
    else if (phase === "warmup") { actionLabel = "Закончить разминку"; actionFn = endWarmup; }
    else if (phase === "set") { actionLabel = "Готово"; actionFn = finishSet; }
    else if (phase === "rest") { actionLabel = "Закончить отдых"; actionFn = endRest; }
    else if (phase === "cardio") { actionLabel = "Пропустить"; actionFn = endRest; }
  }

  const sessionActive = session && status !== "finished" && status !== "stopped" && status !== "completed";

  return (
    <div className="screen active">
      {/* Session card */}
      <div className="card">
        <div className="card-title">Таймер</div>

        {/* No session — big START button */}
        {!sessionActive ? (
          <div>
            {planIssues.length > 0 ? (
              <div className="muted" style={{ textAlign: "center", padding: 16 }}>Проверь формат плана в разделе «План»</div>
            ) : !hasValidPlan ? (
              <div className="muted" style={{ textAlign: "center", padding: 16 }}>Сегодня нет тренировки</div>
            ) : (
              <button
                onClick={startWorkout}
                disabled={actionLoading}
                style={{
                  width: "100%", padding: "20px 0", border: "none", borderRadius: 999,
                  background: "var(--accent)", color: "#fff", fontSize: 18, fontWeight: 700,
                  letterSpacing: 2, cursor: "pointer", textTransform: "uppercase",
                  boxShadow: "0 4px 16px rgba(255,3,62,0.25)",
                  transition: "all 0.2s ease",
                  opacity: actionLoading ? 0.7 : 1,
                }}
              >НАЧАТЬ</button>
            )}
          </div>
        ) : (
          <div style={{ minHeight: 320, display: "flex", flexDirection: "column" }}>
            {/* Total time block — same style as rest timer */}
            <div style={{
              border: "1px solid var(--border)", borderRadius: 16, padding: "12px 16px",
              display: "flex", alignItems: "center", justifyContent: "space-between",
              background: "var(--white)", marginBottom: 12,
            }}>
              <div>
                <div style={{ fontSize: 11, color: "var(--muted)", marginBottom: 2 }}>Общее время</div>
                <div style={{ fontSize: 28, fontWeight: 700, fontVariantNumeric: "tabular-nums", letterSpacing: 1 }}>
                  {formatTotal(totalTimeMs)}
                </div>
              </div>
              {/* Heart rate */}
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <HeartIcon size={20} color={heartRate ? "#ff033e" : "rgba(0,0,0,0.15)" } />
                <span style={{
                  fontSize: 22, fontWeight: 700, fontVariantNumeric: "tabular-nums",
                  color: heartRate ? "#ff033e" : "var(--muted)",
                }}>
                  {heartRate || "—"}
                </span>
              </div>
            </div>

            {/* Phase label */}
            <div className="workout-phase-label" style={{ marginBottom: 8, fontWeight: 600, minHeight: 20 }}>
              {phase === "warmup" && "Разминка"}
              {phase === "set" && "Подход"}
              {phase === "rest" && (session.timerKind === "between" ? "Отдых между упражнениями" : "Отдых между подходами")}
              {phase === "cardio" && "Кардио"}
              {phase === "finished" && "Готово"}
            </div>

            {/* Exercise info */}
            <div style={{ marginBottom: 8, minHeight: 170 }}>
              {ex && (
                <div className="workout-focus-block">
                  <div className="workout-focus-name">{ex.name || "—"}</div>
                  <div className="workout-focus-metrics">
                    <div
                      className={`workout-focus-circle${canAdjustMetrics ? " is-adjustable" : ""}`}
                      onWheelCapture={(event) => handleMetricWheel(event, "weight")}
                      onTouchStart={(event) => handleMetricTouchStart(event, "weight")}
                      onTouchMove={(event) => handleMetricTouchMove(event, "weight")}
                      onTouchEnd={() => handleMetricTouchEnd("weight")}
                      onTouchCancel={() => handleMetricTouchEnd("weight")}
                      title={canAdjustMetrics ? "Прокрутите, чтобы изменить" : ""}
                    >
                      <div className="workout-focus-value">{metricWeight}</div>
                      <div className="workout-focus-label">кг</div>
                    </div>
                    <div
                      className={`workout-focus-circle${canAdjustMetrics ? " is-adjustable" : ""}`}
                      onWheelCapture={(event) => handleMetricWheel(event, "reps")}
                      onTouchStart={(event) => handleMetricTouchStart(event, "reps")}
                      onTouchMove={(event) => handleMetricTouchMove(event, "reps")}
                      onTouchEnd={() => handleMetricTouchEnd("reps")}
                      onTouchCancel={() => handleMetricTouchEnd("reps")}
                      title={canAdjustMetrics ? "Прокрутите, чтобы изменить" : ""}
                    >
                      <div className="workout-focus-value">{metricReps}</div>
                      <div className="workout-focus-label">повт</div>
                    </div>
                    <div
                      className={`workout-focus-circle${canAdjustMetrics ? " is-adjustable" : ""}`}
                      onWheelCapture={(event) => handleMetricWheel(event, "sets")}
                      onTouchStart={(event) => handleMetricTouchStart(event, "sets")}
                      onTouchMove={(event) => handleMetricTouchMove(event, "sets")}
                      onTouchEnd={() => handleMetricTouchEnd("sets")}
                      onTouchCancel={() => handleMetricTouchEnd("sets")}
                      title={canAdjustMetrics ? "Прокрутите, чтобы изменить" : ""}
                    >
                      <div className="workout-focus-value">{metricSets}</div>
                      <div className="workout-focus-label">подх</div>
                    </div>
                  </div>
                  {ex.type === "cardio" && (
                    <div className="workout-set-label muted" style={{ fontSize: 13 }}>
                      Длительность: {formatMinutes(ex.durationSec)} мин
                    </div>
                  )}
                  {setLabelText && (
                    <div className="workout-set-label muted" style={{ fontSize: 13 }}>
                      {setLabelText}
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Rest/Cardio timer — always visible */}
            <div style={{ minHeight: 0, marginBottom: 8 }}>
              <div className="workout-timer-block" style={{ "--timer-progress": `${timerProgress * 100}%` }}>
                <div className="workout-timer-label">
                  {phase === "cardio"
                    ? "Осталось"
                    : phase === "rest"
                      ? (session.timerKind === "between" ? "Отдых между упражнениями" : "Отдых")
                      : "Отдых"}
                </div>
                <div className="workout-timer-value">
                  {phase === "rest" || phase === "cardio" ? timerDisplay : "--:--"}
                </div>
              </div>
            </div>

            {/* Actions — fixed at bottom */}
            <div style={{ display: "flex", gap: 8, marginTop: "auto", alignItems: "stretch" }}>
              {/* Left group: pause + stop */}
              <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
                {isActive && status !== "paused" && (
                  <button className="btn btn-outline" onClick={pause} disabled={actionLoading}
                    style={{ padding: "10px 14px", fontSize: 13, whiteSpace: "nowrap" }}>
                    Пауза
                  </button>
                )}
                <button className="btn btn-outline" onClick={stopWorkout} disabled={actionLoading}
                  style={{ padding: "10px 14px", fontSize: 13, color: "var(--accent)", whiteSpace: "nowrap" }}>
                  Стоп
                </button>
              </div>
              {/* Right: action button fills remaining space */}
              {actionFn && (
                <button className="btn btn-accent" onClick={actionFn} disabled={actionLoading}
                  style={{ flex: 1, fontSize: 14 }}>
                  {actionLabel}
                </button>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Plan issues */}
      {planIssues.length > 0 && (
        <div className="card" style={{ color: "var(--accent)" }}>
          {planIssues.map((issue, i) => <div key={i}>{issue}</div>)}
        </div>
      )}

      {/* Today's plan - flat, no accordion */}
      {exercises.length > 0 && (
        <div className="card">
          <div className="card-title">План тренировки</div>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {exercises.map((ex, i) => (
              <div key={i} className="list-item" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <div>
                  <div style={{ fontWeight: 600 }}>{ex.name || "—"}</div>
                  <div className="meta">
                    {ex.type === "cardio"
                      ? `Длительность: ${formatMinutes(ex.durationSec)} мин`
                      : `${ex.sets}x${ex.reps} · ${ex.weight || "—"} кг · Отдых: ${formatRest(ex.restSec || 120)}`}
                  </div>
                </div>
                <div style={{
                  padding: "2px 8px", borderRadius: 999, fontSize: 11, fontWeight: 600,
                  background: ex.type === "cardio" ? "rgba(34,197,94,0.1)" : "rgba(255,3,62,0.08)",
                  color: ex.type === "cardio" ? "#22c55e" : "var(--accent)",
                }}>
                  {ex.type === "cardio" ? "Кардио" : "Силовое"}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Exercise analytics */}
      <div className="card">
        <div className="card-title">Аналитика упражнения</div>
        <div className="insights-controls">
          <select
            value={statsExercise || insights?.selectedExercise || ""}
            onChange={(event) => setStatsExercise(event.target.value)}
            disabled={insightsLoading || (insights?.exercises || []).length === 0}
          >
            {(insights?.exercises || []).map((name) => (
              <option key={name} value={name}>{name}</option>
            ))}
          </select>
          <div className="insights-periods">
            {[30, 90, 180].map((days) => (
              <button
                key={days}
                className={`btn btn-outline${statsPeriodDays === days ? " active" : ""}`}
                onClick={() => setStatsPeriodDays(days)}
                disabled={insightsLoading}
                style={{ padding: "6px 10px", fontSize: 12 }}
              >
                {days}д
              </button>
            ))}
          </div>
          <button
            className="btn btn-outline"
            onClick={refreshInsights}
            disabled={insightsLoading}
            style={{ padding: "6px 10px", fontSize: 12 }}
          >
            Обновить
          </button>
          <button
            className="btn btn-accent"
            onClick={refreshAIAdvice}
            disabled={insightsAILoading || insightsLoading || !(insights?.aiAvailable)}
            style={{ padding: "6px 10px", fontSize: 12, whiteSpace: "nowrap" }}
          >
            {insightsAILoading ? "AI..." : "AI совет"}
          </button>
        </div>

        {insightsLoading ? (
          <div className="muted" style={{ padding: "10px 0" }}>Загрузка аналитики...</div>
        ) : insightsError ? (
          <div style={{ color: "var(--accent)", fontSize: 13 }}>{insightsError}</div>
        ) : (
          <>
            <div className="insights-status-row">
              <span className={`insights-direction ${statsDirectionClass}`}>{statsDirection}</span>
              <span className="insights-meta">
                e1RM: {formatNum(statsTrends.e1rmDeltaPct)}% · Объем: {formatNum(statsTrends.volumeDeltaPct)}%
              </span>
            </div>

            <div className="stats-overview">
              <div className="stat-tile">
                <div className="stat-label">Текущий e1RM</div>
                <div className="stat-value">{formatNum(statsMetrics.currentE1RM)} кг</div>
              </div>
              <div className="stat-tile">
                <div className="stat-label">Лучший e1RM</div>
                <div className="stat-value">{formatNum(statsMetrics.bestE1RM)} кг</div>
              </div>
              <div className="stat-tile">
                <div className="stat-label">Макс. вес</div>
                <div className="stat-value">{formatNum(statsMetrics.maxWeight)} кг</div>
              </div>
              <div className="stat-tile">
                <div className="stat-label">Объем</div>
                <div className="stat-value">{formatNum(statsMetrics.totalVolume, 0)} кг</div>
              </div>
              <div className="stat-tile">
                <div className="stat-label">Подходы / Сессии</div>
                <div className="stat-value">{statsMetrics.sets || 0} / {statsMetrics.sessions || 0}</div>
              </div>
            </div>

            <div className="insights-charts">
              <div className="insights-chart-card">
                <div className="insights-chart-title">e1RM</div>
                <Sparkline points={statsPoints} valueKey="e1rm" color="#ff033e" />
              </div>
              <div className="insights-chart-card">
                <div className="insights-chart-title">Объем подхода</div>
                <Sparkline points={statsPoints} valueKey="volume" color="#111" />
              </div>
            </div>

            {statsPoints.length > 0 && (
              <div className="insights-range">
                {formatStatDate(statsPoints[0]?.completedAt)} - {formatStatDate(statsPoints[statsPoints.length - 1]?.completedAt)}
              </div>
            )}

            <div className="insights-advice-block">
              <div style={{ fontWeight: 700, marginBottom: 6 }}>Рекомендация</div>
              <div style={{ marginBottom: 8 }}>{insights?.recommendation || "—"}</div>
              <div className="insights-actions">
                {(insights?.actions || []).map((item, i) => (
                  <div key={i} className="insights-action-item">• {item}</div>
                ))}
              </div>
            </div>

            {insights?.aiAdvice && (
              <div className="insights-ai-block">
                <div className="insights-ai-title">
                  AI-разбор <span className="muted">({insights.aiAdvice.confidence || 0}%)</span>
                </div>
                <div className="insights-ai-text">{insights.aiAdvice.summary}</div>
                <div className="insights-ai-text"><strong>Нагрузка:</strong> {insights.aiAdvice.loadAdvice}</div>
                <div className="insights-ai-text"><strong>Восстановление:</strong> {insights.aiAdvice.recovery}</div>
                <div className="insights-ai-text"><strong>Следующая:</strong> {insights.aiAdvice.nextSession}</div>
              </div>
            )}
            {insights?.aiError && (
              <div className="muted" style={{ marginTop: 8, fontSize: 12 }}>
                AI временно недоступен: {insights.aiError}
              </div>
            )}
          </>
        )}
      </div>

      {/* Full training program - week view */}
      {weekPlan.length > 0 && (
        <div className="card">
          <div className="card-title" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <span>Тренировочная программа</span>
            <button
              className="btn btn-outline"
              style={{ fontSize: 12, padding: "4px 12px" }}
              onClick={() => setEditingProgram(true)}
            >Редактировать</button>
          </div>
          {editingProgram ? (
            <ProgramEditor
              weekPlan={weekPlan}
              onSave={async (newPlan) => {
                try {
                  await api("/api/plan/set", { text: JSON.stringify({ week_plan: newPlan }) });
                  setEditingProgram(false);
                  toast("Программа сохранена");
                  loadData();
                } catch (err) { toast(formatApiError(err)); }
              }}
              onCancel={() => setEditingProgram(false)}
            />
          ) : (
            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
              {weekPlan.map((day, i) => {
                const items = day.items || [];
                const itemStrings = items.map(it => typeof it === "string" ? it : it.name || "");
                // Only consider "rest" if items are truly rest-like words (not exercises)
                const onlyRestWords = itemStrings.every((s) => /^[\s]*(отдых|выходн|rest|off|—|-|)[\s]*$/i.test(s));
                const isRest = onlyRestWords && (day.type === "rest" || items.length === 0);
                const dayLabel = day.dayName || day.name || `День ${day.day || i + 1}`;
                return (
                  <AccordionItem key={i} title={
                    <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
                      {dayLabel}
                      {day.focus && <span style={{ fontSize: 12, color: "var(--muted)", fontWeight: 400 }}>— {day.focus}</span>}
                      {isRest && <span style={{ fontSize: 11, padding: "1px 8px", borderRadius: 999, background: "rgba(0,0,0,0.05)", color: "var(--muted)" }}>Отдых</span>}
                    </span>
                  }>
                    {isRest ? (
                      <div className="muted" style={{ padding: "8px 0", fontSize: 13 }}>День отдыха</div>
                    ) : (
                      <div style={{ display: "flex", flexDirection: "column", gap: 6, padding: "4px 0" }}>
                        {itemStrings.map((item, j) => (
                          <div key={j} style={{ fontSize: 13, padding: "4px 0", borderBottom: j < items.length - 1 ? "1px solid rgba(0,0,0,0.04)" : "none" }}>
                            {item || "—"}
                          </div>
                        ))}
                      </div>
                    )}
                  </AccordionItem>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
