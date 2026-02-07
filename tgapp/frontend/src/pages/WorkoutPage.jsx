import { useEffect, useState, useCallback, useRef } from "react";
import { api } from "../services/api";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import { AccordionItem } from "../components/Accordion";
import { postNativeMessage } from "../services/telegram";
import { parsePlan } from "../services/planUtils";

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
  const [days, setDays] = useState(() =>
    dayNames.map((name, i) => {
      const day = weekPlan[i] || {};
      const items = (day.items || []).map((it) => typeof it === "string" ? it : it.name || "").join("\n");
      const lowered = items.toLowerCase();
      const isRest = day.type === "rest" || /(^|\s)(отдых|выходн|rest|off)(\s|$)/i.test(lowered);
      return { dayName: name, focus: day.focus || "", items: isRest ? "" : items, isRest };
    })
  );

  const updateDay = (i, field, value) => {
    setDays((prev) => { const d = [...prev]; d[i] = { ...d[i], [field]: value }; return d; });
  };
  const toggleRest = (i) => {
    setDays((prev) => {
      const d = [...prev];
      const isRest = !d[i].isRest;
      d[i] = { ...d[i], isRest, items: isRest ? "" : d[i].items, focus: isRest ? "" : d[i].focus };
      return d;
    });
  };

  const save = () => {
    const result = days.map((d) => ({
      dayName: d.dayName,
      focus: d.focus,
      items: d.isRest ? ["Отдых"] : d.items.split("\n").map((s) => s.trim()).filter(Boolean),
      type: d.isRest ? "rest" : "train",
    }));
    onSave(result);
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, marginTop: 8 }}>
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
              <input type="text" placeholder="Фокус" value={day.focus} onChange={(e) => updateDay(i, "focus", e.target.value)} style={{ marginBottom: 4, fontSize: 13 }} />
              <textarea placeholder="Упражнения (по строке)" value={day.items} onChange={(e) => updateDay(i, "items", e.target.value)} rows={2} style={{ fontSize: 13 }} />
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
  const [totalTime, setTotalTime] = useState("00:00");
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
      setTimerDisplay("00:00");
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

  // Total workout time
  useEffect(() => {
    if (!session?.startedAt) {
      setTotalTime("00:00");
      if (totalTickRef.current) clearInterval(totalTickRef.current);
      return;
    }
    const update = () => {
      const now = Date.now();
      const startedAt = new Date(session.startedAt).getTime();
      let elapsed = Math.floor((now - startedAt) / 1000) - (session.pausedTotalSec || 0);
      if (session.status === "paused" && session.pausedAt) {
        elapsed -= Math.floor((now - new Date(session.pausedAt).getTime()) / 1000);
      }
      setTotalTime(formatDuration(Math.max(0, elapsed)));
    };
    update();
    totalTickRef.current = setInterval(update, 1000);
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

  return (
    <div className="screen active">
      {/* Session card */}
      <div className="card">
        <div className="card-title">Таймер</div>

        {/* Total workout time */}
        {session && status !== "finished" && status !== "stopped" && status !== "completed" && (
          <div style={{
            display: "flex", alignItems: "center", justifyContent: "center",
            padding: "14px 24px", borderRadius: 16, marginBottom: 12,
            background: "var(--white)", border: "1px solid var(--border)",
          }}>
            <span style={{ fontSize: 32, fontWeight: 700, fontVariantNumeric: "tabular-nums", color: "var(--accent)", letterSpacing: 1 }}>{totalTime}</span>
          </div>
        )}

        {!session || status === "finished" || status === "stopped" || status === "completed" ? (
          <div style={{ textAlign: "center", padding: 16 }}>
            {planIssues.length > 0 ? (
              <div className="muted" style={{ marginBottom: 12 }}>Проверь формат плана в разделе «План»</div>
            ) : !hasValidPlan ? (
              <div className="muted" style={{ marginBottom: 12 }}>Сегодня нет тренировки</div>
            ) : (
              <>
                <div className="muted" style={{ marginBottom: 12 }}>Нет активной тренировки</div>
                <button className="btn btn-accent" onClick={startWorkout} disabled={actionLoading}>
                  Начать тренировку
                </button>
              </>
            )}
          </div>
        ) : (
          <div>
            {/* Phase label */}
            <div className="workout-phase-label" style={{ marginBottom: 8, fontWeight: 600 }}>
              {phase === "warmup" && "Разминка"}
              {phase === "set" && "Подход"}
              {phase === "rest" && (session.timerKind === "between" ? "Отдых между упражнениями" : "Отдых между подходами")}
              {phase === "cardio" && "Кардио"}
              {phase === "finished" && "Готово"}
            </div>

            {/* Exercise info */}
            {ex && (
              <div style={{ marginBottom: 8 }}>
                <div className="workout-exercise-name" style={{ fontWeight: 700 }}>{ex.name}</div>
                <div className="workout-exercise-target muted" style={{ fontSize: 13 }}>
                  {ex.type === "cardio"
                    ? `Длительность: ${formatMinutes(ex.durationSec)} мин`
                    : `Вес: ${ex.weight || "—"} кг · Повторы: ${ex.reps} · Подходы: ${ex.sets}`}
                </div>
                {setLabelText && (
                  <div className="workout-set-label muted" style={{ fontSize: 13 }}>{setLabelText}</div>
                )}
              </div>
            )}

            {/* Timer */}
            {(phase === "rest" || phase === "cardio") && (
              <div className="workout-timer-block" style={{ "--timer-progress": `${timerProgress * 100}%` }}>
                <div className="workout-timer-label">
                  {phase === "cardio" ? "Осталось" : session.timerKind === "between" ? "Отдых между упражнениями" : "Отдых"}
                </div>
                <div className="workout-timer-value">{timerDisplay}</div>
              </div>
            )}

            {/* Input fields */}
            {showInputs && (
              <div className="workout-inputs" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8, marginTop: 8 }}>
                <label>Вес (кг)
                  <input type="number" value={weight} onChange={(e) => setWeight(e.target.value)} placeholder={String(ex?.weight || 0)} />
                </label>
                <label>Повторения
                  <input type="number" value={reps} onChange={(e) => setReps(e.target.value)} placeholder={String(ex?.reps || 0)} />
                </label>
              </div>
            )}

            {/* Actions */}
            <div style={{ display: "flex", gap: 8, marginTop: 12, flexWrap: "wrap" }}>
              {phase === "warmup" && <button className="btn btn-accent" onClick={endWarmup} disabled={actionLoading}>Закончить разминку</button>}
              {phase === "set" && <button className="btn btn-accent" onClick={finishSet} disabled={actionLoading}>Готово</button>}
              {phase === "rest" && isActive && <button className="btn btn-outline" onClick={endRest} disabled={actionLoading}>Закончить отдых</button>}
              {isActive && status !== "paused" && <button className="btn btn-outline" onClick={pause} disabled={actionLoading}>Пауза</button>}
              {status === "paused" && <button className="btn btn-accent" onClick={resume} disabled={actionLoading}>Продолжить</button>}
              {session && <button className="btn btn-outline" onClick={stopWorkout} disabled={actionLoading} style={{ color: "var(--accent)" }}>Стоп</button>}
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
          {editingProgram && <ProgramEditor
            weekPlan={weekPlan}
            onSave={async (newPlan) => {
              try {
                await api("/api/plan/set", { text: JSON.stringify({ week_plan: newPlan }) });
                setWeekPlan(newPlan);
                setEditingProgram(false);
                toast("Программа сохранена");
                loadData();
              } catch (err) { toast(formatApiError(err)); }
            }}
            onCancel={() => setEditingProgram(false)}
          />}
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            {weekPlan.map((day, i) => {
              const items = day.items || [];
              const lowered = items.map(it => typeof it === "string" ? it : it.name || "").join(" ").toLowerCase();
              const isRest = day.type === "rest" || (items.length <= 1 && /(^|\s)(отдых|выходн|rest|off)(\s|$)/i.test(lowered));
              return (
                <AccordionItem key={i} title={
                  <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
                    {day.dayName || `День ${i + 1}`}
                    {day.focus && <span style={{ fontSize: 12, color: "var(--muted)", fontWeight: 400 }}>— {day.focus}</span>}
                    {isRest && <span style={{ fontSize: 11, padding: "1px 8px", borderRadius: 999, background: "rgba(0,0,0,0.05)", color: "var(--muted)" }}>Отдых</span>}
                  </span>
                }>
                  {isRest ? (
                    <div className="muted" style={{ padding: "8px 0", fontSize: 13 }}>День отдыха</div>
                  ) : (
                    <div style={{ display: "flex", flexDirection: "column", gap: 6, padding: "4px 0" }}>
                      {(day.items || []).map((item, j) => (
                        <div key={j} style={{ fontSize: 13, padding: "4px 0", borderBottom: j < day.items.length - 1 ? "1px solid rgba(0,0,0,0.04)" : "none" }}>
                          {typeof item === "string" ? item : `${item.name || "—"}${item.sets ? ` ${item.sets}x${item.reps || ""}` : ""}${item.duration ? ` ${item.duration}` : ""}`}
                        </div>
                      ))}
                    </div>
                  )}
                </AccordionItem>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
