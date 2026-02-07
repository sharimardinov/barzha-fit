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

/* Custom range slider component */
function AccentSlider({ min, max, value, onChange, label, unit = "" }) {
  const pct = ((value - min) / (max - min)) * 100;
  return (
    <div style={{ marginBottom: 8 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 8 }}>
        <span style={{ fontSize: 14, color: "var(--muted)" }}>{label}</span>
        <span style={{ fontSize: 28, fontWeight: 700, fontVariantNumeric: "tabular-nums" }}>
          {value}<span style={{ fontSize: 14, fontWeight: 400, color: "var(--muted)", marginLeft: 2 }}>{unit}</span>
        </span>
      </div>
      <div style={{ position: "relative", height: 36, display: "flex", alignItems: "center" }}>
        <input
          type="range"
          min={min}
          max={max}
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
          style={{
            WebkitAppearance: "none", appearance: "none", width: "100%", height: 6,
            borderRadius: 999, outline: "none", cursor: "pointer",
            background: `linear-gradient(to right, var(--accent) 0%, var(--accent) ${pct}%, rgba(0,0,0,0.08) ${pct}%, rgba(0,0,0,0.08) 100%)`,
          }}
        />
      </div>
      <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11, color: "var(--muted)", marginTop: 2 }}>
        <span>{min}{unit}</span>
        <span>{max}{unit}</span>
      </div>
    </div>
  );
}

/* Bodyfat visual selector */
function BodyfatSlider({ value, onChange }) {
  const pct = ((value - 3) / (50 - 3)) * 100;
  const getCategory = (v) => {
    if (v <= 8) return { label: "Атлет", color: "#22c55e" };
    if (v <= 14) return { label: "Спортсмен", color: "#84cc16" };
    if (v <= 20) return { label: "Фитнес", color: "#eab308" };
    if (v <= 30) return { label: "Среднее", color: "#f97316" };
    return { label: "Высокое", color: "#ef4444" };
  };
  const cat = getCategory(value);
  return (
    <div style={{ textAlign: "center" }}>
      <div style={{
        width: 120, height: 120, borderRadius: "50%", margin: "0 auto 16px",
        background: `conic-gradient(${cat.color} ${pct * 3.6}deg, rgba(0,0,0,0.06) 0deg)`,
        display: "flex", alignItems: "center", justifyContent: "center",
      }}>
        <div style={{
          width: 96, height: 96, borderRadius: "50%", background: "var(--white)",
          display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
        }}>
          <span style={{ fontSize: 32, fontWeight: 700, lineHeight: 1 }}>{value}</span>
          <span style={{ fontSize: 12, color: "var(--muted)" }}>%</span>
        </div>
      </div>
      <div style={{
        display: "inline-block", padding: "4px 14px", borderRadius: 999,
        background: cat.color + "18", color: cat.color, fontWeight: 600, fontSize: 13, marginBottom: 16,
      }}>{cat.label}</div>
      <div style={{ position: "relative", height: 36, display: "flex", alignItems: "center" }}>
        <input
          type="range" min={3} max={50} value={value}
          onChange={(e) => onChange(Number(e.target.value))}
          style={{
            WebkitAppearance: "none", appearance: "none", width: "100%", height: 6,
            borderRadius: 999, outline: "none", cursor: "pointer",
            background: `linear-gradient(to right, ${cat.color} 0%, ${cat.color} ${pct}%, rgba(0,0,0,0.08) ${pct}%, rgba(0,0,0,0.08) 100%)`,
          }}
        />
      </div>
      <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11, color: "var(--muted)", marginTop: 2 }}>
        <span>3%</span>
        <span>50%</span>
      </div>
    </div>
  );
}

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
    planDays: Array(7).fill(null).map((_, i) => ({
      focus: "", items: i >= 5 ? "Отдых" : "", isRest: i >= 5,
    })),
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

  const toggleRest = (index) => {
    setData((prev) => {
      const days = [...prev.planDays];
      const isRest = !days[index].isRest;
      days[index] = { ...days[index], isRest, items: isRest ? "Отдых" : "", focus: isRest ? "" : days[index].focus };
      return { ...prev, planDays: days };
    });
  };

  const validate = () => {
    switch (currentStep.id) {
      case "sex": return !!data.sex;
      case "bodyMetrics": return data.age >= 14 && data.weight >= 30 && data.height >= 120;
      case "bodyfat": return data.bodyfat >= 1 && data.bodyfat <= 60;
      case "goalType": return !!data.goalType;
      case "planWeek": return data.planDays.every((d) => d.isRest || d.items?.trim());
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
        sex: data.sex, age: data.age, height_cm: data.height,
        weight_kg: data.weight, bodyfat_pct: data.bodyfat, goal: data.goalType,
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
        items: d.isRest ? ["Отдых"] : (d.items ? d.items.split("\n").map((s) => s.trim()).filter(Boolean) : ["Отдых"]),
        type: d.isRest ? "rest" : "train",
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
  const dayNamesFull = ["Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"];

  return (
    <div className="screen active" style={{ padding: 16 }}>
      {/* Progress bar */}
      <div style={{ display: "flex", gap: 4, marginBottom: 16 }}>
        {STEPS.map((_, i) => (
          <div key={i} style={{
            flex: 1, height: 4, borderRadius: 2,
            background: i <= step ? "var(--accent)" : "rgba(0,0,0,0.08)",
            transition: "background 0.3s ease",
          }} />
        ))}
      </div>

      <div className="muted" style={{ marginBottom: 4, fontSize: 13 }}>Шаг {step + 1} из {STEPS.length}</div>
      <h2 style={{ marginBottom: 4 }}>{currentStep.title}</h2>
      <p className="muted" style={{ marginBottom: 20 }}>{currentStep.description}</p>

      {currentStep.id === "sex" && (
        <div style={{ display: "flex", gap: 12 }}>
          {[{ v: "m", l: "Мужской", emoji: "👨" }, { v: "f", l: "Женский", emoji: "👩" }].map((opt) => (
            <button
              key={opt.v}
              onClick={() => update("sex", opt.v)}
              style={{
                flex: 1, padding: "24px 16px", fontSize: 16, fontWeight: 600,
                borderRadius: 16, border: `2px solid ${data.sex === opt.v ? "var(--accent)" : "var(--border)"}`,
                background: data.sex === opt.v ? "rgba(255,3,62,0.06)" : "var(--white)",
                color: data.sex === opt.v ? "var(--accent)" : "var(--black)",
                cursor: "pointer", transition: "all 0.2s ease",
                display: "flex", flexDirection: "column", alignItems: "center", gap: 8,
              }}
            >
              <span style={{ fontSize: 32 }}>{opt.emoji}</span>
              {opt.l}
            </button>
          ))}
        </div>
      )}

      {currentStep.id === "bodyMetrics" && (
        <div style={{ display: "grid", gap: 20 }}>
          <AccentSlider label="Возраст" min={14} max={80} value={data.age} onChange={(v) => update("age", v)} unit=" лет" />
          <AccentSlider label="Вес" min={30} max={150} value={data.weight} onChange={(v) => update("weight", v)} unit=" кг" />
          <AccentSlider label="Рост" min={120} max={220} value={data.height} onChange={(v) => update("height", v)} unit=" см" />
        </div>
      )}

      {currentStep.id === "bodyfat" && (
        <BodyfatSlider value={data.bodyfat} onChange={(v) => update("bodyfat", v)} />
      )}

      {currentStep.id === "goalType" && (
        <GoalSelector value={data.goalType} onChange={(v) => update("goalType", v)} />
      )}

      {currentStep.id === "planWeek" && (
        <div style={{ display: "grid", gap: 10 }}>
          {data.planDays.map((day, i) => (
            <div key={i} style={{
              border: `1px solid ${day.isRest ? "rgba(0,0,0,0.06)" : "var(--border)"}`,
              borderRadius: 14, padding: 12, background: day.isRest ? "rgba(0,0,0,0.02)" : "var(--white)",
              transition: "all 0.2s ease",
            }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: day.isRest ? 0 : 8 }}>
                <span style={{ fontWeight: 600, fontSize: 14 }}>{dayNamesFull[i]}</span>
                <button
                  onClick={() => toggleRest(i)}
                  style={{
                    padding: "4px 12px", borderRadius: 999, fontSize: 12, fontWeight: 600,
                    border: "none", cursor: "pointer", transition: "all 0.2s ease",
                    background: day.isRest ? "var(--accent)" : "rgba(0,0,0,0.06)",
                    color: day.isRest ? "var(--white)" : "var(--muted)",
                  }}
                >
                  {day.isRest ? "Отдых ✓" : "Отдых"}
                </button>
              </div>
              {!day.isRest && (
                <>
                  <input
                    type="text" placeholder="Фокус (грудь, спина...)"
                    value={day.focus} onChange={(e) => updateDay(i, "focus", e.target.value)}
                    style={{ marginBottom: 6 }}
                  />
                  <textarea
                    placeholder="Упражнения (по одному на строку)"
                    value={day.items} onChange={(e) => updateDay(i, "items", e.target.value)}
                    rows={2}
                  />
                </>
              )}
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
