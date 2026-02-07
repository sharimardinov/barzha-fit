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
  const [timerDisplay, setTimerDisplay] = useState("00:00");
  const [timerProgress, setTimerProgress] = useState(0);
  const [totalTimeMs, setTotalTimeMs] = useState(0);
  const tickRef = useRef(null);
  const totalTickRef = useRef(null);
  const lastTimerKeyRef = useRef(null);
  const lastTimerTotalRef = useRef(0);
  const lastSetKeyRef = useRef(null);
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

  useEffect(() => { loadData(); }, [loadData]);

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
  }, [toast]);

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

  // Auto-fill weight/reps for new set
  useEffect(() => {
    if (!session || session.phase !== "set" || !plan) return;
    const ex = plan.exercises?.[session.exerciseIndex];
    if (!ex || ex.type === "cardio") return;
    const key = `${session.exerciseIndex}:${session.setIndex}`;
    if (key !== lastSetKeyRef.current) {
      lastSetKeyRef.current = key;
      if (session.setIndex > 0) {
        setWeight(String(ex.weight || ""));
        setReps(String(ex.reps || ""));
      } else {
        setWeight("");
        setReps("");
      }
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
    const w = isWarmup ? 0 : Number(weight) || 0;
    const r = isWarmup ? 0 : Math.round(Number(reps) || 0);
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

  if (loading) return <div className="screen-loader is-loading"><div className="screen-spinner" /></div>;

  const exercises = plan?.exercises || [];
  const ex = exercises[session?.exerciseIndex];
  const phase = session?.phase;
  const status = session?.status;
  const isActive = status === "in_progress";
  const hasValidPlan = exercises.length > 0 && planIssues.length === 0;

  // Set label text
  let setLabelText = "";
  if (ex && ex.type !== "cardio") {
    if (phase === "set") {
      setLabelText = session.setIndex === 0 ? "Разминочный подход" : `Подход ${session.setIndex} из ${ex.sets}`;
    } else if (phase === "rest") {
      if (session.timerKind === "rest") {
        const finished = Math.max(0, session.setIndex - 1);
        setLabelText = finished === 0 ? "Отдых после разминки" : `Отдых после подхода ${finished} из ${ex.sets}`;
      } else {
        setLabelText = "Отдых между упражнениями";
      }
    }
  }

  const showInputs = phase === "set" && session?.setIndex > 0 && ex?.type !== "cardio";

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

            {/* Exercise info — fixed height container */}
            <div style={{ marginBottom: 8, minHeight: 60 }}>
              {ex && (
                <>
                  <div className="workout-exercise-name" style={{ fontWeight: 700 }}>{ex.name}</div>
                  <div className="workout-exercise-target muted" style={{ fontSize: 13 }}>
                    {ex.type === "cardio"
                      ? `Длительность: ${formatMinutes(ex.durationSec)} мин`
                      : `Вес: ${ex.weight || "—"} кг · Повторы: ${ex.reps} · Подходы: ${ex.sets}`}
                  </div>
                  {setLabelText && (
                    <div className="workout-set-label muted" style={{ fontSize: 13 }}>{setLabelText}</div>
                  )}
                </>
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

            {/* Input fields — fixed height */}
            <div style={{ minHeight: showInputs ? 0 : 0, marginBottom: 8 }}>
              {showInputs && (
                <div className="workout-inputs" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
                  <label>Вес (кг)
                    <input type="number" value={weight} onChange={(e) => setWeight(e.target.value)} placeholder={String(ex?.weight || 0)} />
                  </label>
                  <label>Повторения
                    <input type="number" value={reps} onChange={(e) => setReps(e.target.value)} placeholder={String(ex?.reps || 0)} />
                  </label>
                </div>
              )}
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
