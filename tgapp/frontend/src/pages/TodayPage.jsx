import { useEffect, useState, useCallback } from "react";
import { api } from "../services/api";
import { useAppState } from "../hooks/useAppState";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import { parsePlan } from "../services/planUtils";
import StepsCard from "../components/StepsCard";
import { AccordionItem } from "../components/Accordion";

export default function TodayPage() {
  const { state, dispatch } = useAppState();
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [today, setToday] = useState(null);
  const [steps, setSteps] = useState({ steps: 0, target: 0, distance: null, kcal: null });

  const loadToday = useCallback(async () => {
    try {
      const data = await api("/api/today");
      setToday(data);
      dispatch({ type: "SET_TODAY", payload: data });

      const s = Number(data.steps || 0);
      const t = Number(data.targets?.steps || 0);
      setSteps((prev) => ({ ...prev, steps: s, target: t }));

      if (!state.planText && data.plan) {
        const plan = parsePlan(data.plan);
        dispatch({ type: "SET_PLAN", payload: { text: plan.text, payload: plan.payload, structured: plan.structured } });
      }
    } catch (err) {
      toast(formatApiError(err, "Не удалось загрузить данные"));
    } finally {
      setLoading(false);
    }
  }, [dispatch, toast, state.planText]);

  useEffect(() => {
    loadToday();
  }, [loadToday]);

  useEffect(() => {
    const handler = (event) => {
      const detail = event.detail || {};
      const s = Number(detail.steps || 0);
      const d = detail.distance != null ? Number(detail.distance) : null;
      const k = detail.kcal != null ? Number(detail.kcal) : null;
      setSteps((prev) => ({ ...prev, steps: s, distance: d, kcal: k }));
    };
    window.addEventListener("nativeSteps", handler);
    return () => window.removeEventListener("nativeSteps", handler);
  }, []);

  if (loading) return <div className="screen-loader is-loading"><div className="screen-spinner" /></div>;

  const t = today?.targets || {};
  const icons = today?.icons || {};

  const metrics = [
    { label: "Ккал", value: today?.kcal || 0, target: t.kcal || 0, icon: icons.kcal, id: "kcal" },
    { label: "Белки", value: today?.protein || 0, target: t.protein || 0, icon: icons.protein, id: "protein" },
    { label: "Жиры", value: today?.fat || 0, target: t.fat || 0, icon: icons.fat, id: "fat" },
    { label: "Углеводы", value: today?.carbs || 0, target: t.carbs || 0, icon: icons.carbs, id: "carbs" },
    { label: "Шаги", value: steps.steps, target: steps.target, icon: icons.steps, id: "steps" },
  ];

  return (
    <div className="screen active">
      <StepsCard steps={steps.steps} target={steps.target} distance={steps.distance} kcal={steps.kcal} />

      <div className="card">
        <div className="card-title">Прогресс</div>
        <div style={{ display: "grid", gap: 16 }}>
          {metrics.map((m) => {
            const ratio = m.target > 0 ? m.value / m.target : 0;
            const pct = Math.min(Math.max(ratio * 100, 0), 100);
            const isOver = m.value > m.target;
            const color = m.icon === "green" ? "#22c55e" : m.icon === "red" ? "#ef4444" : "var(--accent)";
            
            return (
              <div key={m.id} style={{
                padding: "12px 16px",
                background: isOver ? "rgba(34,197,94,0.06)" : "rgba(0,0,0,0.02)",
                borderRadius: 12,
                border: `1px solid ${isOver ? "rgba(34,197,94,0.2)" : "rgba(0,0,0,0.06)"}`,
                transition: "all 0.2s ease",
              }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", marginBottom: 10 }}>
                  <div>
                    <div style={{ fontSize: 13, fontWeight: 600, color: "var(--muted)", marginBottom: 4 }}>
                      {m.label}
                    </div>
                    <div style={{ fontSize: 20, fontWeight: 700, color: isOver ? "#22c55e" : "var(--black)" }}>
                      {Math.round(m.value)}
                      <span style={{ fontSize: 14, fontWeight: 500, color: "var(--muted)", marginLeft: 4 }}>
                        / {m.target}
                      </span>
                    </div>
                  </div>
                  <div style={{
                    fontSize: 12, fontWeight: 700,
                    color: isOver ? "#22c55e" : pct >= 80 ? color : "var(--muted)",
                    padding: "4px 10px",
                    borderRadius: 999,
                    background: isOver ? "rgba(34,197,94,0.1)" : pct >= 80 ? `${color}15` : "rgba(0,0,0,0.04)",
                  }}>
                    {isOver ? "✓" : `${Math.round(pct)}%`}
                  </div>
                </div>
                <div style={{
                  height: 8,
                  background: isOver ? "rgba(34,197,94,0.15)" : "rgba(0,0,0,0.06)",
                  borderRadius: 999,
                  overflow: "hidden",
                  position: "relative",
                }}>
                  <div style={{
                    height: "100%",
                    width: `${Math.min(pct, 100)}%`,
                    background: isOver 
                      ? "linear-gradient(90deg, #22c55e 0%, #16a34a 100%)"
                      : `linear-gradient(90deg, ${color} 0%, ${color}dd 100%)`,
                    borderRadius: 999,
                    transition: "width 0.6s cubic-bezier(0.4, 0, 0.2, 1)",
                    boxShadow: pct > 0 ? `0 2px 6px ${color}40` : "none",
                  }} />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {today?.todayTraining && (
        <div className="card">
          <AccordionItem title="Тренировка сегодня" defaultOpen>
            <div className="accordion-list">
              {(Array.isArray(today.todayTraining) ? today.todayTraining : [today.todayTraining]).map((item, i) => (
                <div key={i} className="list-item">{typeof item === "string" ? item : item.name || JSON.stringify(item)}</div>
              ))}
            </div>
          </AccordionItem>
        </div>
      )}

      <button
        className="btn btn-accent"
        style={{ width: "100%" }}
        onClick={() => dispatch({ type: "SET_TAB", payload: "meals" })}
      >
        <span>+ Добавить еду</span>
      </button>
    </div>
  );
}
