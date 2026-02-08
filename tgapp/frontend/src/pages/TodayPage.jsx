import { useEffect, useState, useCallback } from "react";
import { api } from "../services/api";
import { useAppState } from "../hooks/useAppState";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import { parsePlan } from "../services/planUtils";
import StepsCard from "../components/StepsCard";
import { AccordionItem } from "../components/Accordion";
import ProgressBar from "../components/ProgressBar";

function MacroIcon({ type }) {
  if (type === "protein") {
    return (
      <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <rect x="3" y="9" width="4" height="6" rx="1.2" />
        <rect x="17" y="9" width="4" height="6" rx="1.2" />
        <rect x="7" y="11" width="10" height="2" rx="1" />
      </svg>
    );
  }
  if (type === "carbs") {
    return (
      <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <rect x="4" y="6" width="16" height="3" rx="1.5" />
        <rect x="6" y="11" width="12" height="3" rx="1.5" />
        <rect x="8" y="16" width="8" height="3" rx="1.5" />
      </svg>
    );
  }
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12 2c-2.8 3.8-6 7.2-6 11a6 6 0 0 0 12 0c0-3.8-3.2-7.2-6-11z" />
    </svg>
  );
}

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
  const formatNumber = (value) => (Number.isFinite(value) ? Math.round(value).toLocaleString("ru-RU") : "—");

  const macroMetrics = [
    { id: "protein", label: "Белки", value: today?.protein || 0, target: t.protein || 0, unit: "г" },
    { id: "carbs", label: "Углеводы", value: today?.carbs || 0, target: t.carbs || 0, unit: "г" },
    { id: "fat", label: "Жиры", value: today?.fat || 0, target: t.fat || 0, unit: "г" },
  ];

  return (
    <div className="screen active">
      <StepsCard steps={steps.steps} target={steps.target} distance={steps.distance} kcal={steps.kcal} />

      <div className="card macro-panel">
        <div className="macro-header">
          <div className="macro-title">Макронутриенты</div>
          <button
            type="button"
            className="macro-link"
            onClick={() => dispatch({ type: "SET_TAB", payload: "meals" })}
          >
            Детали
          </button>
        </div>
        <div className="macro-grid">
          {macroMetrics.map((m) => {
            const targetText = m.target > 0 ? `${formatNumber(m.target)}${m.unit}` : "—";
            return (
              <div key={m.id} className="macro-card">
                <div className="macro-card-head">
                  <span className={`macro-icon macro-icon-${m.id}`} aria-hidden="true">
                    <MacroIcon type={m.id} />
                  </span>
                  <span className="macro-label">{m.label}</span>
                </div>
                <div className="macro-value">
                  <span className="macro-current">{formatNumber(m.value)}</span>
                  <span className="macro-unit">{m.unit}</span>
                  <span className="macro-target">/ {targetText}</span>
                </div>
                <ProgressBar current={m.value} target={m.target} className="macro-progress" />
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
