import { useState, useCallback } from "react";
import { api } from "../services/api";
import { useAppState } from "../hooks/useAppState";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import GoalSelector from "../components/GoalSelector";

const STEPS = [
  { id: "sex", title: "Пол", description: "Выбери свой пол" },
  { id: "bodyMetrics", title: "Параметры тела", description: "Укажи свои параметры" },
  { id: "bodyfat", title: "Процент жира", description: "Укажи примерный процент жира" },
  { id: "goalType", title: "Твоя цель", description: "Выбери направление тренировок" },
  { id: "planWeek", title: "План на неделю", description: "Опиши свой тренировочный план" },
];

export default function OnboardingPage({ onComplete }) {
  const { dispatch } = useAppState();
  const toast = useToast();
  const [step, setStep] = useState(0);
  const [saving, setSaving] = useState(false);
  const [data, setData] = useState({
    sex: "",
    age: 25,
    weight: 70,
    height: 175,
    bodyfat: 15,
    goalType: "balance",
    planDays: Array(7).fill("").map((_, i) => ({ focus: "", items: i >= 5 ? "Отдых" : "" })),
  });

  const currentStep = STEPS[step];

  const update = (field, value) => setData((prev) => ({ ...prev, [field]: value }));

  const updateDay = (index, field, value) => {
    setData((prev) => {
      const days = [...prev.planDays];
      days[index] = { ...days[index], [field]: value };
      return { ...prev, planDays: days };
    });
  };

  const validate = () => {
    switch (currentStep.id) {
      case "sex": return !!data.sex;
      case "bodyMetrics": return data.age >= 14 && data.weight >= 30 && data.height >= 120;
      case "bodyfat": return data.bodyfat >= 1 && data.bodyfat <= 60;
      case "goalType": return !!data.goalType;
      case "planWeek": return data.planDays.every((d) => d.items?.trim());
      default: return true;
    }
  };

  const next = () => {
    if (!validate()) { toast("Заполни все поля"); return; }
    if (step < STEPS.length - 1) { setStep(step + 1); return; }
    submit();
  };

  const back = () => { if (step > 0) setStep(step - 1); };

  const submit = useCallback(async () => {
    setSaving(true);
    try {
      await api("/api/profile/set", {
        sex: data.sex,
        age: data.age,
        height_cm: data.height,
        weight_kg: data.weight,
        bodyfat_pct: data.bodyfat,
        goal: data.goalType,
      });
      await api("/api/targets/refresh");
      await api("/api/training/profile/set", {
        bench_kg: 0, pullups: 0, run_km: 0,
        injuries: "", goal: data.goalType, pharma: false,
        trainings_per_week: 0, wishes: "",
      });

      const dayNames = ["Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"];
      const weekPlan = data.planDays.map((d, i) => ({
        dayName: dayNames[i],
        focus: d.focus || "",
        items: d.items ? d.items.split("\n").map((s) => s.trim()).filter(Boolean) : ["Отдых"],
        type: /(отдых|rest|off)/i.test(d.items || "") ? "rest" : "train",
      }));
      await api("/api/plan/set", { text: JSON.stringify({ week_plan: weekPlan }) });

      dispatch({ type: "SET_ONBOARDING", payload: false });
      onComplete();
    } catch (err) {
      toast(formatApiError(err, "Ошибка сохранения"));
    } finally {
      setSaving(false);
    }
  }, [data, dispatch, onComplete, toast]);

  const dayNames = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];

  return (
    <div className="screen active" style={{ padding: 16 }}>
      <div className="muted" style={{ marginBottom: 8 }}>Шаг {step + 1} из {STEPS.length}</div>
      <h2 style={{ marginBottom: 4 }}>{currentStep.title}</h2>
      <p className="muted" style={{ marginBottom: 16 }}>{currentStep.description}</p>

      {currentStep.id === "sex" && (
        <div style={{ display: "flex", gap: 12 }}>
          {[{ v: "m", l: "Мужской" }, { v: "f", l: "Женский" }].map((opt) => (
            <button
              key={opt.v}
              className={`option-card sex-option${data.sex === opt.v ? " active" : ""}`}
              onClick={() => update("sex", opt.v)}
              style={{ flex: 1, padding: 20, fontSize: 16, fontWeight: 600 }}
            >
              {opt.l}
            </button>
          ))}
        </div>
      )}

      {currentStep.id === "bodyMetrics" && (
        <div style={{ display: "grid", gap: 16 }}>
          <label>Возраст: <strong>{data.age}</strong>
            <input type="range" min={14} max={80} value={data.age} onChange={(e) => update("age", Number(e.target.value))} />
          </label>
          <label>Вес: <strong>{data.weight} кг</strong>
            <input type="range" min={30} max={150} value={data.weight} onChange={(e) => update("weight", Number(e.target.value))} />
          </label>
          <label>Рост: <strong>{data.height} см</strong>
            <input type="range" min={120} max={220} value={data.height} onChange={(e) => update("height", Number(e.target.value))} />
          </label>
        </div>
      )}

      {currentStep.id === "bodyfat" && (
        <div>
          <label>Процент жира: <strong>{data.bodyfat}%</strong>
            <input type="range" min={3} max={50} value={data.bodyfat} onChange={(e) => update("bodyfat", Number(e.target.value))} />
          </label>
        </div>
      )}

      {currentStep.id === "goalType" && (
        <GoalSelector value={data.goalType} onChange={(v) => update("goalType", v)} />
      )}

      {currentStep.id === "planWeek" && (
        <div style={{ display: "grid", gap: 12 }}>
          {data.planDays.map((day, i) => (
            <div key={i} className="card" style={{ padding: 12 }}>
              <div style={{ fontWeight: 600, marginBottom: 8 }}>{dayNames[i]}</div>
              <input
                type="text"
                placeholder="Фокус (грудь, спина...)"
                value={day.focus}
                onChange={(e) => updateDay(i, "focus", e.target.value)}
                style={{ marginBottom: 6 }}
              />
              <textarea
                placeholder="Упражнения (по одному на строку) или 'Отдых'"
                value={day.items}
                onChange={(e) => updateDay(i, "items", e.target.value)}
                rows={3}
              />
            </div>
          ))}
        </div>
      )}

      <div style={{ display: "flex", gap: 12, marginTop: 24 }}>
        {step > 0 && <button className="btn btn-outline" onClick={back} style={{ flex: 1 }}>Назад</button>}
        <button className="btn btn-accent" onClick={next} disabled={saving} style={{ flex: 1 }}>
          {step === STEPS.length - 1 ? (saving ? "Сохраняю..." : "Готово") : "Далее"}
        </button>
      </div>
    </div>
  );
}
