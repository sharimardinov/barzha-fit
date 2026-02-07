import { useEffect, useState, useCallback } from "react";
import { api } from "../services/api";
import { useAppState } from "../hooks/useAppState";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import { parsePlan } from "../services/planUtils";
import StepsCard from "../components/StepsCard";
import { AccordionItem } from "../components/Accordion";
import ProgressBar from "../components/ProgressBar";

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
    if (state.activeTab === "today") {
      loadToday();
    }
  }, [state.activeTab, loadToday]);

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
  ];

  return (
    <div className="screen active">
      <StepsCard steps={steps.steps} target={steps.target} distance={steps.distance} kcal={steps.kcal} />

      <div className="card">
        <div className="card-title">Прогресс</div>
        {metrics.map((m) => {
          const status = m.icon === "green" ? "ok" : m.icon === "red" ? "bad" : "none";
          return (
            <div key={m.id} className="progress-item">
              <div className="progress-head">
                <span className="label">{m.label}</span>
                <span className="value">
                  {Math.round(m.value)} / {m.target}
                  <span className={`indicator ${status}`} />
                </span>
              </div>
              <ProgressBar current={m.value} target={m.target} />
            </div>
          );
        })}
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
