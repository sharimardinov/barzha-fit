import { useEffect, useState, useCallback, useRef } from "react";
import { api } from "../services/api";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import { AccordionItem, Accordion } from "../components/Accordion";
import { postNativeMessage } from "../services/telegram";

function formatDuration(sec) {
  if (!Number.isFinite(sec) || sec < 0) return "00:00";
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

export default function WorkoutPage() {
  const toast = useToast();
  const [plan, setPlan] = useState(null);
  const [session, setSession] = useState(null);
  const [loading, setLoading] = useState(true);
  const [actionLoading, setActionLoading] = useState(false);
  const [weight, setWeight] = useState("");
  const [reps, setReps] = useState("");
  const [timerDisplay, setTimerDisplay] = useState("00:00");
  const [timerProgress, setTimerProgress] = useState(0);
  const tickRef = useRef(null);

  const loadData = useCallback(async () => {
    try {
      const [planData, sessionData] = await Promise.all([
        api("/api/workout/plan/get").catch(() => null),
        api("/api/workout/session/get").catch(() => null),
      ]);
      setPlan(planData);
      setSession(sessionData);
    } catch (err) {
      toast(formatApiError(err));
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { loadData(); }, [loadData]);

  // Timer tick
  useEffect(() => {
    if (!session || (session.phase !== "rest" && session.phase !== "cardio")) {
      setTimerDisplay("00:00");
      setTimerProgress(0);
      return;
    }
    const tick = () => {
      if (!session.timerStartedAt || !session.timerDurationSec) return;
      const end = new Date(session.timerStartedAt).getTime() + session.timerDurationSec * 1000;
      const remaining = Math.max(0, Math.round((end - Date.now()) / 1000));
      setTimerDisplay(formatDuration(remaining));
      const total = session.timerDurationSec || 1;
      setTimerProgress(Math.min(1, 1 - remaining / total));
      if (remaining <= 0) { clearInterval(tickRef.current); loadData(); }
    };
    tick();
    tickRef.current = setInterval(tick, 200);
    return () => clearInterval(tickRef.current);
  }, [session, loadData]);

  const doAction = async (path, body) => {
    setActionLoading(true);
    try {
      const data = await api(path, body);
      setSession(data);
    } catch (err) {
      toast(formatApiError(err));
    } finally {
      setActionLoading(false);
    }
  };

  const startWorkout = () => doAction("/api/workout/session/start");
  const endWarmup = () => doAction("/api/workout/session/warmup/end");
  const finishSet = () => doAction("/api/workout/session/set/finish", { weight_kg: Number(weight) || 0, reps: Number(reps) || 0 });
  const endRest = () => doAction("/api/workout/session/rest/end");
  const pause = () => doAction("/api/workout/session/pause");
  const resume = () => doAction("/api/workout/session/resume");
  const stopWorkout = () => doAction("/api/workout/session/stop");

  if (loading) return <div className="screen-loader is-loading"><div className="screen-spinner" /></div>;

  const ex = session?.exercises?.[session?.exerciseIndex];
  const phase = session?.phase;
  const status = session?.status;
  const isActive = status === "in_progress";

  return (
    <div className="screen active">
      {/* Session card */}
      <div className="card">
        <div className="card-title">Таймер</div>
        {!session || status === "finished" || status === "stopped" ? (
          <div style={{ textAlign: "center", padding: 16 }}>
            <div className="muted" style={{ marginBottom: 12 }}>Нет активной тренировки</div>
            <button className="btn btn-accent" onClick={startWorkout} disabled={actionLoading || !plan}>
              Начать тренировку
            </button>
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
                  {ex.sets}x{ex.reps} · {ex.weightKg || 0} кг
                </div>
                {phase === "set" && (
                  <div className="workout-set-label muted" style={{ fontSize: 13 }}>
                    Подход {(session.setIndex || 0) + 1} из {ex.sets}
                  </div>
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
            {phase === "set" && (
              <div className="workout-inputs" style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8, marginTop: 8 }}>
                <label>Вес (кг)
                  <input type="number" value={weight} onChange={(e) => setWeight(e.target.value)} placeholder={String(ex?.weightKg || 0)} />
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
              {phase === "rest" && <button className="btn btn-outline" onClick={endRest} disabled={actionLoading}>Закончить отдых</button>}
              {isActive && status !== "paused" && <button className="btn btn-outline" onClick={pause} disabled={actionLoading}>Пауза</button>}
              {status === "paused" && <button className="btn btn-accent" onClick={resume} disabled={actionLoading}>Продолжить</button>}
              <button className="btn btn-outline" onClick={stopWorkout} disabled={actionLoading} style={{ color: "var(--accent)" }}>Стоп</button>
            </div>
          </div>
        )}
      </div>

      {/* Plan */}
      {plan?.exercises && (
        <div className="card">
          <AccordionItem title="Программа тренировок">
            <div className="workout-plan-list">
              {plan.exercises.map((ex, i) => (
                <div key={i} className="workout-exercise-block">
                  <div className="workout-exercise-name">{ex.name}</div>
                  <div className="workout-exercise-target muted">
                    {ex.type === "cardio" ? `${ex.durationMin || 0} мин` : `${ex.sets}x${ex.reps} · ${ex.weightKg || 0} кг · Отдых: ${ex.restSec || 120}с`}
                  </div>
                </div>
              ))}
            </div>
          </AccordionItem>
        </div>
      )}
    </div>
  );
}
