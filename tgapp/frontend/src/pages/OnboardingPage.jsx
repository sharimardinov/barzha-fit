import { useState, useCallback } from "react";
import { api } from "../services/api";
import { useAppState } from "../hooks/useAppState";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import { motion, AnimatePresence } from "motion/react";
import Stepper, { Step } from "../components/Stepper";
import ElasticSlider from "../components/ElasticSlider";

const STEPS = [
  { id: "sex", title: "Твой пол", description: "Требуется для корректных норм и нагрузки" },
  { id: "bodyMetrics", title: "Твои параметры", description: "Для расчёта норм и целей" },
  { id: "trainingStage", title: "Уровень подготовки", description: "Выбери свой текущий уровень" },
  { id: "bodyfat", title: "Процент жира", description: "Для точных целей по весу и форме" },
  { id: "goalType", title: "Твоя цель", description: "Выбери направление тренировок" },
  { id: "planWeek", title: "План на неделю", description: "Опиши свой тренировочный план" },
];

/* Bodyfat circle + elastic slider */
function BodyfatSlider({ value, onChange }) {
  const pct = ((value - 3) / (50 - 3)) * 100;
  const getCategory = (v) => {
    if (v <= 8) return { label: "Атлет", color: "#22c55e" };
    if (v <= 14) return { label: "Спортсмен", color: "#65a30d" };
    if (v <= 20) return { label: "Фитнес", color: "#ca8a04" };
    if (v <= 30) return { label: "Среднее", color: "#ea580c" };
    return { label: "Высокое", color: "#dc2626" };
  };
  const cat = getCategory(value);
  return (
    <div style={{ textAlign: "center" }}>
      <div style={{
        width: 130, height: 130, borderRadius: "50%", margin: "0 auto 16px",
        background: `conic-gradient(${cat.color} ${pct * 3.6}deg, rgba(0,0,0,0.06) 0deg)`,
        display: "flex", alignItems: "center", justifyContent: "center",
        transition: "background 0.3s ease",
      }}>
        <div style={{
          width: 104, height: 104, borderRadius: "50%", background: "var(--white)",
          display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center",
        }}>
          <span style={{ fontSize: 36, fontWeight: 700, lineHeight: 1 }}>{value}</span>
          <span style={{ fontSize: 12, color: "var(--muted)" }}>%</span>
        </div>
      </div>
      <div style={{
        display: "inline-block", padding: "4px 16px", borderRadius: 999,
        background: cat.color + "15", color: cat.color, fontWeight: 600, fontSize: 13, marginBottom: 16,
        transition: "all 0.3s ease",
      }}>{cat.label}</div>
      <ElasticSlider min={3} max={50} value={value} onChange={onChange} label="" unit="%" color={cat.color} />
    </div>
  );
}

/* Goal selector — PillNav style with animated sliding indicator */
function GoalPill({ value, onChange }) {
  const options = [
    { id: "cut", label: "CUT", desc: "Сушка" },
    { id: "balance", label: "BALANCE", desc: "Поддержание" },
    { id: "bulk", label: "BULK", desc: "Набор массы" },
  ];

  return (
    <div style={{
      display: "flex", gap: 4, background: "#f0f0f0", borderRadius: 999, padding: 4,
    }}>
      {options.map((opt) => {
        const active = value === opt.id;
        return (
          <button
            key={opt.id}
            onClick={() => onChange(opt.id)}
            style={{
              flex: 1, position: "relative", border: "none", cursor: "pointer",
              borderRadius: 999, padding: "14px 8px",
              background: "transparent",
              zIndex: 1,
            }}
          >
            {/* Animated pill background */}
            {active && (
              <motion.div
                layoutId="goal-pill-bg"
                style={{
                  position: "absolute", inset: 0, borderRadius: 999,
                  background: "#ff033e",
                }}
                transition={{ type: "spring", stiffness: 400, damping: 30 }}
              />
            )}
            <div style={{
              position: "relative", zIndex: 2,
              fontSize: 14, fontWeight: 700, letterSpacing: 1,
              fontFamily: "'Aptos', Arial, sans-serif",
              color: active ? "#fff" : "rgba(0,0,0,0.35)",
              transition: "color 0.2s ease",
              lineHeight: 1.2,
            }}>{opt.label}</div>
            <div style={{
              position: "relative", zIndex: 2,
              fontSize: 10, marginTop: 3,
              color: active ? "rgba(255,255,255,0.7)" : "rgba(0,0,0,0.2)",
              transition: "color 0.2s ease",
              lineHeight: 1,
            }}>{opt.desc}</div>
          </button>
        );
      })}
    </div>
  );
}

/* Training stage cards — CORE / FLOW / PEAK */
/* Uses mask-image for pixel-perfect #ff033e coloring instead of CSS filter */
function StageSelector({ value, onChange }) {
  const stages = [
    { id: "core", label: "CORE", desc: "Начинающий", img: "/app/core.svg" },
    { id: "flow", label: "FLOW", desc: "Средний", img: "/app/flow.svg" },
    { id: "peak", label: "PEAK", desc: "Продвинутый", img: "/app/peak.svg" },
  ];
  return (
    <div style={{ display: "flex", gap: 10 }}>
      {stages.map((s) => {
        const active = value === s.id;
        return (
          <button key={s.id} onClick={() => onChange(s.id)} style={{
            flex: 1, padding: "16px 8px", borderRadius: 16, cursor: "pointer",
            border: "none",
            background: active ? "rgba(255,3,62,0.06)" : "rgba(0,0,0,0.03)",
            boxShadow: active ? "0 4px 20px rgba(255,3,62,0.15)" : "none",
            transition: "all 0.35s ease", display: "flex", flexDirection: "column",
            alignItems: "center", gap: 8,
          }}>
            {/* Use mask-image to color SVG precisely */}
            <div style={{
              width: 60, height: 100,
              WebkitMaskImage: `url(${s.img})`,
              maskImage: `url(${s.img})`,
              WebkitMaskSize: "contain",
              maskSize: "contain",
              WebkitMaskRepeat: "no-repeat",
              maskRepeat: "no-repeat",
              WebkitMaskPosition: "center",
              maskPosition: "center",
              backgroundColor: active ? "#ff033e" : "rgba(0,0,0,0.25)",
              transition: "background-color 0.35s ease",
            }} />
            <div style={{
              fontSize: 14, fontWeight: 700, letterSpacing: 1,
              color: active ? "#ff033e" : "var(--muted)",
              transition: "color 0.35s ease",
            }}>{s.label}</div>
            <div style={{
              fontSize: 11,
              color: active ? "#ff033e" : "var(--muted)",
              transition: "color 0.35s ease",
            }}>{s.desc}</div>
          </button>
        );
      })}
    </div>
  );
}

export default function OnboardingPage({ onComplete }) {
  const { dispatch } = useAppState();
  const toast = useToast();
  const [step, setStep] = useState(0);
  const [saving, setSaving] = useState(false);
  const [planMode, setPlanMode] = useState("text"); // "text" or "structured"
  const [data, setData] = useState({
    sex: "",
    age: 25, weight: 70, height: 175,
    trainingStage: "core",
    bodyfat: 15,
    goalType: "balance",
    planDays: Array(7).fill(null).map((_, i) => ({
      focus: "", items: i >= 5 ? "Отдых" : "", isRest: i >= 5,
      exercises: [], // for structured mode
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
      days[index] = { ...days[index], isRest, items: isRest ? "Отдых" : "", focus: isRest ? "" : days[index].focus, exercises: [] };
      return { ...prev, planDays: days };
    });
  };

  const updateExercise = (dayIdx, exIdx, field, value) => {
    setData((prev) => {
      const days = [...prev.planDays];
      const exercises = [...(days[dayIdx].exercises || [])];
      exercises[exIdx] = { ...exercises[exIdx], [field]: value };
      days[dayIdx] = { ...days[dayIdx], exercises };
      return { ...prev, planDays: days };
    });
  };

  const addExercise = (dayIdx, type = "strength") => {
    setData((prev) => {
      const days = [...prev.planDays];
      const base = type === "cardio"
        ? { name: "", type: "cardio", duration: "", sets: "", reps: "", rest: "" }
        : { name: "", type: "strength", sets: "", reps: "", rest: "", duration: "" };
      const exercises = [...(days[dayIdx].exercises || []), base];
      days[dayIdx] = { ...days[dayIdx], exercises };
      return { ...prev, planDays: days };
    });
  };

  const removeExercise = (dayIdx, exIdx) => {
    setData((prev) => {
      const days = [...prev.planDays];
      days[dayIdx] = { ...days[dayIdx], exercises: days[dayIdx].exercises.filter((_, j) => j !== exIdx) };
      return { ...prev, planDays: days };
    });
  };

  const validate = () => {
    switch (currentStep.id) {
      case "sex": return !!data.sex;
      case "bodyMetrics": return data.age >= 14 && data.weight >= 30 && data.height >= 120;
      case "trainingStage": return !!data.trainingStage;
      case "bodyfat": return data.bodyfat >= 1 && data.bodyfat <= 60;
      case "goalType": return !!data.goalType;
      case "planWeek":
        return data.planDays.every((d) => {
          if (d.isRest) return true;
          if (planMode === "structured") return (d.exercises || []).some((ex) => ex.name.trim());
          return d.items?.trim();
        });
      default: return true;
    }
  };

  const handleStepChange = (newStep) => {
    const goingForward = newStep > step + 1;
    if (goingForward && !validate()) {
      toast("Заполни все поля");
      return;
    }
    setStep(newStep - 1);
  };

  const handleComplete = () => {
    if (!validate()) { toast("Заполни все поля"); return; }
    submit();
  };

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
      const weekPlan = data.planDays.map((d, i) => {
        let items;
        if (d.isRest) {
          items = ["Отдых"];
        } else if (planMode === "structured") {
          items = (d.exercises || [])
            .filter((ex) => ex.name.trim())
            .map((ex) => {
              let line = ex.name.trim();
              if (ex.type === "cardio") {
                if (ex.duration) line += ` | ${ex.duration} мин`;
              } else {
                if (ex.sets && ex.reps) line += ` | ${ex.sets}x${ex.reps}`;
                else if (ex.sets) line += ` | ${ex.sets}`;
                if (ex.rest) line += ` | ${ex.rest}`;
              }
              return line;
            });
          if (items.length === 0) items = ["Отдых"];
        } else {
          items = d.items ? d.items.split("\n").map((s) => s.trim()).filter(Boolean) : ["Отдых"];
        }
        return {
          day: i + 1,
          name: `День ${i + 1}`,
          dayName: dayNames[i],
          focus: d.focus || "",
          items,
          type: d.isRest ? "rest" : "train",
        };
      });
      await api("/api/plan/set", { text: JSON.stringify({ week_plan: weekPlan }) });

      dispatch({ type: "SET_ONBOARDING", payload: false });
      onComplete();
    } catch (err) {
      toast(formatApiError(err, "Ошибка сохранения"));
    } finally {
      setSaving(false);
    }
  }, [data, dispatch, onComplete, toast, planMode]);

  const dayNamesFull = ["Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"];

  return (
    <div className="screen active" style={{ padding: 16 }}>
      <Stepper
        currentStep={step + 1}
        onStepChange={handleStepChange}
        onComplete={handleComplete}
        canProceed={validate()}
        loading={saving}
      >
        {/* Step 1 — Sex */}
        <Step>
          <h2 style={{ marginBottom: 4 }}>{STEPS[0].title}</h2>
          <p className="muted" style={{ marginBottom: 20 }}>{STEPS[0].description}</p>
          <div style={{ display: "flex", gap: 12 }}>
            {[{ v: "m", l: "М" }, { v: "f", l: "Ж" }].map((opt) => {
              const active = data.sex === opt.v;
              return (
                <button key={opt.v} onClick={() => update("sex", opt.v)} style={{
                  flex: 1, padding: "32px 16px", fontSize: 32, fontWeight: 700,
                  borderRadius: 18, border: "none", cursor: "pointer",
                  background: active ? "var(--accent)" : "rgba(0,0,0,0.04)",
                  color: active ? "var(--white)" : "var(--black)",
                  boxShadow: active ? "0 4px 20px rgba(255,3,62,0.3)" : "none",
                  transition: "all 0.2s ease",
                }}>{opt.l}</button>
              );
            })}
          </div>
        </Step>

        {/* Step 2 — Body Metrics (ElasticSlider) */}
        <Step>
          <h2 style={{ marginBottom: 4 }}>{STEPS[1].title}</h2>
          <p className="muted" style={{ marginBottom: 20 }}>{STEPS[1].description}</p>
          <div style={{ display: "grid", gap: 20 }}>
            <ElasticSlider label="Возраст" min={14} max={80} value={data.age} onChange={(v) => update("age", v)} unit=" лет" />
            <ElasticSlider label="Вес" min={30} max={150} value={data.weight} onChange={(v) => update("weight", v)} unit=" кг" step={0.5} />
            <ElasticSlider label="Рост" min={120} max={220} value={data.height} onChange={(v) => update("height", v)} unit=" см" />
          </div>
        </Step>

        {/* Step 3 — Training Stage */}
        <Step>
          <h2 style={{ marginBottom: 4 }}>{STEPS[2].title}</h2>
          <p className="muted" style={{ marginBottom: 20 }}>{STEPS[2].description}</p>
          <StageSelector value={data.trainingStage} onChange={(v) => update("trainingStage", v)} />
        </Step>

        {/* Step 4 — Bodyfat (ElasticSlider via BodyfatSlider) */}
        <Step>
          <h2 style={{ marginBottom: 4 }}>{STEPS[3].title}</h2>
          <p className="muted" style={{ marginBottom: 20 }}>{STEPS[3].description}</p>
          <BodyfatSlider value={data.bodyfat} onChange={(v) => update("bodyfat", v)} />
        </Step>

        {/* Step 5 — Goal Type */}
        <Step>
          <h2 style={{ marginBottom: 4 }}>{STEPS[4].title}</h2>
          <p className="muted" style={{ marginBottom: 20 }}>{STEPS[4].description}</p>
          <GoalPill value={data.goalType} onChange={(v) => update("goalType", v)} />
        </Step>

        {/* Step 6 — Plan Week */}
        <Step>
          <h2 style={{ marginBottom: 4 }}>{STEPS[5].title}</h2>
          <p className="muted" style={{ marginBottom: 20 }}>{STEPS[5].description}</p>
          <div style={{ display: "grid", gap: 10 }}>
            {/* Mode toggle */}
            <div style={{ display: "flex", gap: 4, background: "rgba(0,0,0,0.04)", borderRadius: 10, padding: 3 }}>
              {[
                { id: "text", label: "Текстом" },
                { id: "structured", label: "По полям" },
              ].map((m) => (
                <button key={m.id} onClick={() => setPlanMode(m.id)} style={{
                  flex: 1, padding: "6px 8px", borderRadius: 8, border: "none", cursor: "pointer",
                  fontSize: 12, fontWeight: 600,
                  background: planMode === m.id ? "var(--white)" : "transparent",
                  color: planMode === m.id ? "var(--black)" : "var(--muted)",
                  boxShadow: planMode === m.id ? "0 1px 3px rgba(0,0,0,0.08)" : "none",
                  transition: "all 0.2s ease",
                }}>{m.label}</button>
              ))}
            </div>

            {planMode === "text" && (
              <div style={{ fontSize: 11, color: "var(--muted)", padding: "0 2px" }}>
                Формат: Название | 3x10 | 60кг | 120сек (по одному на строку)
              </div>
            )}

            {data.planDays.map((day, i) => (
              <div key={i} style={{
                border: `1px solid ${day.isRest ? "rgba(0,0,0,0.06)" : "var(--border)"}`,
                borderRadius: 14, padding: 12, background: day.isRest ? "rgba(0,0,0,0.02)" : "var(--white)",
                transition: "all 0.2s ease",
              }}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: day.isRest ? 0 : 8 }}>
                  <span style={{ fontWeight: 600, fontSize: 14 }}>{dayNamesFull[i]}</span>
                  <button onClick={() => toggleRest(i)} style={{
                    padding: "4px 12px", borderRadius: 999, fontSize: 12, fontWeight: 600,
                    border: "none", cursor: "pointer", transition: "all 0.2s ease",
                    background: day.isRest ? "var(--accent)" : "rgba(0,0,0,0.06)",
                    color: day.isRest ? "var(--white)" : "var(--muted)",
                  }}>{day.isRest ? "Отдых ✓" : "Отдых"}</button>
                </div>
                {!day.isRest && (
                  <>
                    <input type="text" placeholder="Фокус (грудь, спина...)" value={day.focus} onChange={(e) => updateDay(i, "focus", e.target.value)} style={{ marginBottom: 6 }} />

                    {planMode === "text" ? (
                      <textarea placeholder="Упражнения (по одному на строку)" value={day.items} onChange={(e) => updateDay(i, "items", e.target.value)} rows={2} />
                    ) : (
                      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                        {(day.exercises || []).map((ex, j) => (
                          <div key={j}>
                            {/* Exercise type badge */}
                            <div style={{ display: "flex", gap: 4, marginBottom: 4 }}>
                              <button onClick={() => updateExercise(i, j, "type", "strength")} style={{
                                padding: "2px 8px", borderRadius: 6, fontSize: 10, fontWeight: 600, border: "none", cursor: "pointer",
                                background: (ex.type || "strength") === "strength" ? "rgba(255,3,62,0.1)" : "rgba(0,0,0,0.04)",
                                color: (ex.type || "strength") === "strength" ? "var(--accent)" : "var(--muted)",
                                transition: "all 0.2s",
                              }}>Силовое</button>
                              <button onClick={() => updateExercise(i, j, "type", "cardio")} style={{
                                padding: "2px 8px", borderRadius: 6, fontSize: 10, fontWeight: 600, border: "none", cursor: "pointer",
                                background: ex.type === "cardio" ? "rgba(34,197,94,0.1)" : "rgba(0,0,0,0.04)",
                                color: ex.type === "cardio" ? "#22c55e" : "var(--muted)",
                                transition: "all 0.2s",
                              }}>Кардио</button>
                            </div>

                            <div style={{ display: "flex", gap: 4, alignItems: "center" }}>
                              <input type="text" placeholder={ex.type === "cardio" ? "Бег, велосипед..." : "Упражнение"} value={ex.name}
                                onChange={(e) => updateExercise(i, j, "name", e.target.value)}
                                style={{ flex: 3, fontSize: 12, padding: "6px 8px" }} />

                              {ex.type === "cardio" ? (
                                <input type="text" placeholder="Мин" value={ex.duration || ""}
                                  onChange={(e) => updateExercise(i, j, "duration", e.target.value)}
                                  style={{ flex: 1, fontSize: 12, padding: "6px 4px", textAlign: "center" }} />
                              ) : (
                                <>
                                  <input type="text" placeholder="Подх" value={ex.sets}
                                    onChange={(e) => updateExercise(i, j, "sets", e.target.value)}
                                    style={{ flex: 0.7, fontSize: 12, padding: "6px 4px", textAlign: "center" }} />
                                  <span style={{ fontSize: 12, color: "var(--muted)" }}>×</span>
                                  <input type="text" placeholder="Повт" value={ex.reps}
                                    onChange={(e) => updateExercise(i, j, "reps", e.target.value)}
                                    style={{ flex: 0.7, fontSize: 12, padding: "6px 4px", textAlign: "center" }} />
                                  <input type="text" placeholder="Отдых" value={ex.rest}
                                    onChange={(e) => updateExercise(i, j, "rest", e.target.value)}
                                    style={{ flex: 0.8, fontSize: 12, padding: "6px 4px", textAlign: "center" }} />
                                </>
                              )}

                              <button onClick={() => removeExercise(i, j)} style={{
                                background: "none", border: "none", cursor: "pointer", color: "var(--muted)",
                                fontSize: 14, padding: "2px 4px", flexShrink: 0,
                              }}>✕</button>
                            </div>
                          </div>
                        ))}
                        <div style={{ display: "flex", gap: 4 }}>
                          <button onClick={() => addExercise(i, "strength")} style={{
                            flex: 1, padding: "6px 10px", borderRadius: 8, border: "1px dashed var(--border)",
                            background: "none", cursor: "pointer", fontSize: 12, color: "var(--muted)",
                          }}>+ силовое</button>
                          <button onClick={() => addExercise(i, "cardio")} style={{
                            flex: 1, padding: "6px 10px", borderRadius: 8, border: "1px dashed rgba(34,197,94,0.3)",
                            background: "none", cursor: "pointer", fontSize: 12, color: "#22c55e",
                          }}>+ кардио</button>
                        </div>
                      </div>
                    )}
                  </>
                )}
              </div>
            ))}
          </div>
        </Step>
      </Stepper>
    </div>
  );
}
