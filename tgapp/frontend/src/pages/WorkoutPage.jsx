import { useEffect, useState, useCallback, useRef } from "react";
import { api } from "../services/api";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import { postNativeMessage } from "../services/telegram";
import { formatWeekPlanForEditor, parsePastedWeekPlan, parsePlan } from "../services/planUtils";

const workoutDayStorageKey = "workout.selected_day";

function useHeartRate() {
  const [bpm, setBpm] = useState(null);

  useEffect(() => {
    window.onHeartRateUpdate = (value) => {
      if (typeof value === "number" && value > 0) {
        setBpm(value);
        return;
      }
      setBpm(null);
    };

    postNativeMessage("requestHeartRate", {});
    const interval = setInterval(() => postNativeMessage("requestHeartRate", {}), 5000);

    return () => {
      clearInterval(interval);
      delete window.onHeartRateUpdate;
    };
  }, []);

  return bpm;
}

function HeartIcon({ size = 16, color = "#ff033e" }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill={color} stroke="none">
      <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z" />
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

function formatTotal(ms) {
  const totalSec = Math.max(0, Math.floor(ms / 1000));
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

function loadStoredWorkoutDay() {
  const raw = Number(localStorage.getItem(workoutDayStorageKey) || "");
  if (Number.isInteger(raw) && raw > 0) {
    return raw;
  }
  return 1;
}

function extractPlanDays(parsedPlan, rawPlanText) {
  if (parsedPlan?.structured && Array.isArray(parsedPlan.items) && parsedPlan.items.length > 0) {
    return parsedPlan.items
      .map((item, index) => {
        if (typeof item === "string") return index + 1;
        return Number(item?.day) || index + 1;
      })
      .filter((day, index, array) => day > 0 && array.indexOf(day) === index)
      .sort((a, b) => a - b);
  }

  const pasted = parsePastedWeekPlan(rawPlanText);
  return pasted
    .map((item) => Number(item.day) || 0)
    .filter((day, index, array) => day > 0 && array.indexOf(day) === index)
    .sort((a, b) => a - b);
}

export default function WorkoutPage() {
  const toast = useToast();
  const heartRate = useHeartRate();
  const [plan, setPlan] = useState(null);
  const [planIssues, setPlanIssues] = useState([]);
  const [session, setSession] = useState(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [availablePlanDays, setAvailablePlanDays] = useState([]);
  const [selectedDay, setSelectedDay] = useState(loadStoredWorkoutDay);
  const [planEditorText, setPlanEditorText] = useState("");
  const [planSaving, setPlanSaving] = useState(false);
  const [weight, setWeight] = useState("");
  const [reps, setReps] = useState("");
  const [setsMetric, setSetsMetric] = useState("");
  const [timerDisplay, setTimerDisplay] = useState("00:00");
  const [timerProgress, setTimerProgress] = useState(0);
  const [totalTimeMs, setTotalTimeMs] = useState(0);
  const tickRef = useRef(null);
  const totalTickRef = useRef(null);
  const lastTimerKeyRef = useRef(null);
  const lastTimerTotalRef = useRef(0);
  const lastSetKeyRef = useRef(null);
  const metricTouchRef = useRef({});
  const wakeLockRef = useRef(null);
  const heartRateStatsRef = useRef({});
  const reportedSessionsRef = useRef(new Set());

  const requestWakeLock = useCallback(async () => {
    if (!("wakeLock" in navigator) || wakeLockRef.current) return;
    try {
      wakeLockRef.current = await navigator.wakeLock.request("screen");
      wakeLockRef.current.addEventListener("release", () => {
        wakeLockRef.current = null;
      });
    } catch {
      wakeLockRef.current = null;
    }
  }, []);

  const releaseWakeLock = useCallback(async () => {
    if (!wakeLockRef.current) return;
    try {
      await wakeLockRef.current.release();
    } catch (err) {
      console.warn("Wake lock release failed", err);
    }
    wakeLockRef.current = null;
  }, []);

  const loadPlanEditor = useCallback(async () => {
    try {
      const fullPlanResp = await api("/api/plan/get");
      const rawPlanText = String(fullPlanResp?.text || "");
      const parsedPlan = parsePlan(rawPlanText);
      const days = extractPlanDays(parsedPlan, rawPlanText);
      setAvailablePlanDays(days);
      setSelectedDay((current) => {
        if (days.length === 0) return current;
        if (days.includes(current)) return current;
        const stored = loadStoredWorkoutDay();
        if (days.includes(stored)) return stored;
        return days[0];
      });
      if (parsedPlan.structured && Array.isArray(parsedPlan.items) && parsedPlan.items.length > 0) {
        setPlanEditorText(formatWeekPlanForEditor(parsedPlan.items));
      } else {
        setPlanEditorText(rawPlanText);
      }
    } catch {
      setAvailablePlanDays([]);
      setPlanEditorText("");
    }
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      let planData = null;
      let issues = [];

      try {
        const planResp = await api("/api/workout/plan/get", { day: selectedDay });
        planData = planResp?.plan || null;
      } catch (err) {
        if (err.message === "workout_plan_invalid") {
          issues = err.data?.issues || [];
        } else if (err.message === "workout_plan_not_found") {
          issues = ["План тренировки не найден"];
        }
      }

      let sessionData = null;
      try {
        const sessionResp = await api("/api/workout/session/get");
        sessionData = sessionResp?.session || null;
        if (sessionResp?.plan) {
          planData = sessionResp.plan;
          issues = [];
        }
        if (sessionData?.workoutDay > 0) {
          setSelectedDay(sessionData.workoutDay);
        }
        if (sessionData?.status === "completed") {
          void sendWorkoutReport(sessionData);
        }
      } catch (err) {
        if (err.message !== "workout_session_not_found") {
          toast(formatApiError(err));
        }
      }

      setPlan(planData);
      setPlanIssues(issues);
      setSession(sessionData);
    } finally {
      setLoading(false);
    }
  }, [selectedDay, toast]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  useEffect(() => {
    void loadPlanEditor();
  }, [loadPlanEditor]);

  useEffect(() => {
    localStorage.setItem(workoutDayStorageKey, String(selectedDay));
  }, [selectedDay]);

  useEffect(() => {
    const status = session?.status;
    const isLiveSession = Boolean(session && status !== "finished" && status !== "stopped" && status !== "completed");
    postNativeMessage("workoutTimer", { action: isLiveSession ? "start" : "stop" });
  }, [session]);

  const sendWorkoutReport = useCallback(async (sessionData) => {
    if (!sessionData?.id || reportedSessionsRef.current.has(sessionData.id)) {
      return;
    }

    const stats = heartRateStatsRef.current[sessionData.id] || {};
    reportedSessionsRef.current.add(sessionData.id);

    try {
      await api("/api/workout/session/report", {
        sessionID: sessionData.id,
        day: sessionData.workoutDay || selectedDay,
        heartRate: {
          avgBpm: stats.samples ? Math.round((stats.sum || 0) / stats.samples) : 0,
          maxBpm: stats.max || 0,
          minBpm: stats.min || 0,
          lastBpm: stats.last || 0,
          samples: stats.samples || 0,
        },
      });
    } catch (err) {
      reportedSessionsRef.current.delete(sessionData.id);
      toast(formatApiError(err));
    }
  }, [selectedDay, toast]);

  const applySession = useCallback((data) => {
    if (data?.plan) setPlan(data.plan);

    const nextSession = data?.session || null;
    if (nextSession?.status === "completed") {
      void sendWorkoutReport(nextSession);
      setSession(nextSession);
      toast("Тренировка завершена");
      return;
    }

    setSession(nextSession);
  }, [sendWorkoutReport, toast]);

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
        void loadData();
      }
    };

    tick();
    tickRef.current = setInterval(tick, 120);
    return () => clearInterval(tickRef.current);
  }, [session, loadData]);

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
        elapsedMs -= now - new Date(session.pausedAt).getTime();
      }
      setTotalTimeMs(Math.max(0, elapsedMs));
    };

    update();
    totalTickRef.current = setInterval(update, 500);
    return () => clearInterval(totalTickRef.current);
  }, [session]);

  useEffect(() => {
    const shouldLock = session && session.status === "in_progress" && (session.phase === "rest" || session.phase === "cardio");
    if (shouldLock) requestWakeLock();
    else void releaseWakeLock();
    return () => {
      void releaseWakeLock();
    };
  }, [session, requestWakeLock, releaseWakeLock]);

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

  useEffect(() => {
    if (!session || session.phase !== "set" || !plan) return;
    const exercise = plan.exercises?.[session.exerciseIndex];
    if (!exercise || exercise.type === "cardio") return;
    const key = `${session.exerciseIndex}:${session.setIndex}`;
    if (key !== lastSetKeyRef.current) {
      lastSetKeyRef.current = key;
      setWeight(String(exercise.weight ?? ""));
      setReps(String(exercise.reps ?? ""));
      setSetsMetric(String(exercise.sets ?? ""));
    }
  }, [session, plan]);

  useEffect(() => {
    const sessionID = session?.id;
    if (!sessionID) return;
    if (!heartRateStatsRef.current[sessionID]) {
      heartRateStatsRef.current[sessionID] = {
        sum: 0,
        samples: 0,
        min: 0,
        max: 0,
        last: 0,
      };
    }
  }, [session?.id]);

  useEffect(() => {
    const sessionID = session?.id;
    if (!sessionID || !session || session.status !== "in_progress" || !Number.isFinite(heartRate) || heartRate <= 0) {
      return;
    }
    const current = heartRateStatsRef.current[sessionID] || {
      sum: 0,
      samples: 0,
      min: 0,
      max: 0,
      last: 0,
    };
    current.sum += heartRate;
    current.samples += 1;
    current.last = heartRate;
    current.min = current.min > 0 ? Math.min(current.min, heartRate) : heartRate;
    current.max = Math.max(current.max, heartRate);
    heartRateStatsRef.current[sessionID] = current;
  }, [heartRate, session]);

  const doAction = async (path, body) => {
    setActionLoading(true);
    try {
      const data = await api(path, body);
      applySession(data);
      if (data?.session?.status === "completed" || !data?.session) {
        await loadData();
      }
    } catch (err) {
      toast(formatApiError(err));
    } finally {
      setActionLoading(false);
    }
  };

  const startWorkout = () => doAction("/api/workout/session/start", { day: selectedDay });
  const endWarmup = () => doAction("/api/workout/session/warmup/end");
  const endRest = () => doAction("/api/workout/session/rest/end");
  const pause = () => doAction("/api/workout/session/pause");
  const resume = () => doAction("/api/workout/session/resume");
  const stopWorkout = () => doAction("/api/workout/session/stop");
  const savePlan = async () => {
    const raw = String(planEditorText || "").trim();
    if (!raw) {
      toast("Вставь план");
      return;
    }

    let textToSave = raw;
    if (!raw.startsWith("{")) {
      const weekPlan = parsePastedWeekPlan(raw);
      if (weekPlan.length === 0) {
        toast("Не удалось распознать дни плана");
        return;
      }
      textToSave = JSON.stringify({ week_plan: weekPlan });
    }

    setPlanSaving(true);
    try {
      await api("/api/plan/set", { text: textToSave });
      await loadData();
      await loadPlanEditor();
      toast("План обновлён");
    } catch (err) {
      toast(formatApiError(err));
    } finally {
      setPlanSaving(false);
    }
  };

  const exercises = plan?.exercises || [];
  const exercise = exercises[session?.exerciseIndex];

  const finishSet = () => {
    const isWarmup = session?.setIndex === 0;
    const plannedWeight = Number(exercise?.weight) || 0;
    const plannedReps = Number(exercise?.reps) || 0;
    const actualWeight = isWarmup ? 0 : Number(weight === "" ? plannedWeight : weight) || 0;
    const actualReps = isWarmup ? 0 : Math.round(Number(reps === "" ? plannedReps : reps) || 0);

    doAction("/api/workout/session/set/finish", {
      exerciseIndex: session.exerciseIndex,
      setIndex: session.setIndex,
      actualWeight,
      actualReps,
    });
  };

  if (loading) {
    return <div className="screen-loader is-loading"><div className="screen-spinner" /></div>;
  }

  const phase = session?.phase;
  const status = session?.status;
  const isActive = status === "in_progress";
  const sessionActive = Boolean(session && status !== "finished" && status !== "stopped" && status !== "completed");
  const hasValidPlan = exercises.length > 0 && planIssues.length === 0;
  const canAdjustMetrics = phase === "set" && session?.setIndex > 0 && exercise?.type !== "cardio";
  const metricWeight = exercise?.type === "cardio" ? "—" : (weight === "" ? (exercise?.weight ?? "—") : weight);
  const metricReps = exercise?.type === "cardio" ? "—" : (reps === "" ? (exercise?.reps ?? "—") : reps);
  const metricSets = exercise?.type === "cardio" ? "—" : (setsMetric === "" ? (exercise?.sets ?? "—") : setsMetric);
  const effectiveSets = exercise?.type === "cardio"
    ? 0
    : Math.max(1, Math.round(Number(setsMetric === "" ? exercise?.sets : setsMetric) || 1));

  const adjustMetric = (kind, direction, multiplier = 1) => {
    if (!canAdjustMetrics || !exercise || direction === 0) return;
    const step = kind === "weight" ? 2.5 : 1;
    const delta = direction * step * multiplier;

    if (kind === "weight") {
      setWeight((prev) => {
        const fallback = Number(exercise.weight) || 0;
        const base = prev === "" ? fallback : Number(prev);
        const safeBase = Number.isFinite(base) ? base : fallback;
        return String(Math.max(0, Math.round((safeBase + delta) * 10) / 10));
      });
      return;
    }

    if (kind === "reps") {
      setReps((prev) => {
        const fallback = Number(exercise.reps) || 0;
        const base = prev === "" ? fallback : Number(prev);
        const safeBase = Number.isFinite(base) ? base : fallback;
        return String(Math.max(0, Math.round(safeBase + delta)));
      });
      return;
    }

    setSetsMetric((prev) => {
      const fallback = Number(exercise.sets) || 1;
      const base = prev === "" ? fallback : Number(prev);
      const safeBase = Number.isFinite(base) ? base : fallback;
      return String(Math.max(1, Math.round(safeBase + delta)));
    });
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

  let setLabelText = "";
  if (exercise && exercise.type !== "cardio") {
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

  let actionLabel = "";
  let actionFn = null;
  if (sessionActive) {
    if (status === "paused") {
      actionLabel = "Продолжить";
      actionFn = resume;
    } else if (phase === "warmup") {
      actionLabel = "Закончить разминку";
      actionFn = endWarmup;
    } else if (phase === "set") {
      actionLabel = "Готово";
      actionFn = finishSet;
    } else if (phase === "rest" || phase === "cardio") {
      actionLabel = phase === "cardio" ? "Пропустить" : "Закончить отдых";
      actionFn = endRest;
    }
  }

  return (
    <div className="screen active">
      {availablePlanDays.length > 0 && (
        <div className="card">
          <div className="card-title">День тренировки</div>
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            {availablePlanDays.map((day) => (
              <button
                key={day}
                className={selectedDay === day ? "btn btn-accent" : "btn btn-outline"}
                onClick={() => setSelectedDay(day)}
                disabled={sessionActive}
                style={{ minWidth: 58, padding: "10px 14px" }}
              >
                {day}
              </button>
            ))}
          </div>
          <div className="muted" style={{ marginTop: 10, fontSize: 12 }}>
            {sessionActive ? "День нельзя менять во время активной тренировки." : `Выбран день ${selectedDay}.`}
          </div>
        </div>
      )}

      <div className="card">
        <div className="card-title">Таймер</div>

        {!sessionActive ? (
          <div>
            {planIssues.length > 0 ? (
              <div className="muted" style={{ textAlign: "center", padding: 16 }}>План тренировки требует правки</div>
            ) : !hasValidPlan ? (
              <div className="muted" style={{ textAlign: "center", padding: 16 }}>Сегодня нет тренировки</div>
            ) : (
              <button
                onClick={startWorkout}
                disabled={actionLoading}
                style={{
                  width: "100%",
                  padding: "20px 0",
                  border: "none",
                  borderRadius: 999,
                  background: "var(--accent)",
                  color: "#fff",
                  fontSize: 18,
                  fontWeight: 700,
                  letterSpacing: 2,
                  cursor: "pointer",
                  textTransform: "uppercase",
                  boxShadow: "0 4px 16px rgba(255,3,62,0.25)",
                  opacity: actionLoading ? 0.7 : 1,
                }}
              >
                НАЧАТЬ
              </button>
            )}
          </div>
        ) : (
          <div style={{ minHeight: 320, display: "flex", flexDirection: "column" }}>
            <div
              style={{
                border: "1px solid var(--border)",
                borderRadius: 16,
                padding: "12px 16px",
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                background: "var(--white)",
                marginBottom: 12,
              }}
            >
              <div>
                <div style={{ fontSize: 11, color: "var(--muted)", marginBottom: 2 }}>Общее время</div>
                <div style={{ fontSize: 28, fontWeight: 700, fontVariantNumeric: "tabular-nums", letterSpacing: 1 }}>
                  {formatTotal(totalTimeMs)}
                </div>
              </div>
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <HeartIcon size={20} color={heartRate ? "#ff033e" : "rgba(0,0,0,0.15)"} />
                <span
                  style={{
                    fontSize: 22,
                    fontWeight: 700,
                    fontVariantNumeric: "tabular-nums",
                    color: heartRate ? "#ff033e" : "var(--muted)",
                  }}
                >
                  {heartRate || "—"}
                </span>
              </div>
            </div>

            <div className="workout-phase-label" style={{ marginBottom: 8, fontWeight: 600, minHeight: 20 }}>
              {phase === "warmup" && "Разминка"}
              {phase === "set" && "Подход"}
              {phase === "rest" && (session.timerKind === "between" ? "Отдых между упражнениями" : "Отдых между подходами")}
              {phase === "cardio" && "Кардио"}
            </div>

            <div style={{ marginBottom: 8, minHeight: 170 }}>
              {exercise && (
                <div className="workout-focus-block">
                  <div className="workout-focus-name">{exercise.name || "—"}</div>
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
                  {exercise.type === "cardio" && (
                    <div className="workout-set-label muted" style={{ fontSize: 13 }}>
                      Длительность: {formatMinutes(exercise.durationSec)} мин
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

            <div style={{ display: "flex", gap: 8, marginTop: "auto", alignItems: "stretch" }}>
              <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
                {isActive && status !== "paused" && (
                  <button
                    className="btn btn-outline"
                    onClick={pause}
                    disabled={actionLoading}
                    style={{ padding: "10px 14px", fontSize: 13, whiteSpace: "nowrap" }}
                  >
                    Пауза
                  </button>
                )}
                <button
                  className="btn btn-outline"
                  onClick={stopWorkout}
                  disabled={actionLoading}
                  style={{ padding: "10px 14px", fontSize: 13, color: "var(--accent)", whiteSpace: "nowrap" }}
                >
                  Стоп
                </button>
              </div>
              {actionFn && (
                <button className="btn btn-accent" onClick={actionFn} disabled={actionLoading} style={{ flex: 1, fontSize: 14 }}>
                  {actionLabel}
                </button>
              )}
            </div>
          </div>
        )}
      </div>

      {planIssues.length > 0 && (
        <div className="card" style={{ color: "var(--accent)" }}>
          {planIssues.map((issue, index) => <div key={index}>{issue}</div>)}
        </div>
      )}

      {exercises.length > 0 && (
        <div className="card">
          <div className="card-title">План тренировки</div>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {exercises.map((item, index) => (
              <div key={index} className="list-item" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                <div>
                  <div style={{ fontWeight: 600 }}>{item.name || "—"}</div>
                  <div className="meta">
                    {item.type === "cardio"
                      ? `Длительность: ${formatMinutes(item.durationSec)} мин`
                      : `${item.sets}x${item.reps} · ${item.weight || "—"} кг · Отдых: ${formatRest(item.restSec || 120)}`}
                  </div>
                </div>
                <div
                  style={{
                    padding: "2px 8px",
                    borderRadius: 999,
                    fontSize: 11,
                    fontWeight: 600,
                    background: item.type === "cardio" ? "rgba(34,197,94,0.1)" : "rgba(255,3,62,0.08)",
                    color: item.type === "cardio" ? "#22c55e" : "var(--accent)",
                  }}
                >
                  {item.type === "cardio" ? "Кардио" : "Силовое"}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="card">
        <div className="card-title">Редактор плана</div>
        <textarea
          value={planEditorText}
          onChange={(event) => setPlanEditorText(event.target.value)}
          placeholder={"1\nПодтягивания с весом | 4х6 | — | 180\n\n2\nОтдых"}
          style={{ minHeight: 260, fontSize: 14, lineHeight: 1.45, marginBottom: 12 }}
        />
        <div className="muted" style={{ marginBottom: 12, fontSize: 12 }}>
          Вставляй дни блоками: номер дня, затем упражнения по строкам. Кардио вроде `Единоборства 60 мин` тоже нормализуется.
        </div>
        <button className="btn btn-accent" onClick={savePlan} disabled={planSaving} style={{ width: "100%" }}>
          {planSaving ? "Сохраняю..." : "Сохранить план"}
        </button>
      </div>
    </div>
  );
}
