import { useEffect, useState, useCallback, useRef } from "react";
import { api } from "../services/api";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";

export default function MealsPage() {
  const toast = useToast();
  const [meals, setMeals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
  const [targets, setTargets] = useState({ kcal: 0, protein: 0, fat: 0, carbs: 0 });
  const inputRef = useRef(null);

  const loadMeals = useCallback(async () => {
    try {
      const data = await api("/api/meals/today");
      setMeals(Array.isArray(data) ? data : []);
    } catch (err) {
      toast(formatApiError(err, "Не удалось загрузить приёмы пищи"));
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { loadMeals(); }, [loadMeals]);

  // Load targets
  useEffect(() => {
    const loadTargets = async () => {
      try {
        const data = await api("/api/today");
        const t = data?.targets || {};
        setTargets({
          kcal: t.kcal || 0,
          protein: t.protein || 0,
          fat: t.fat || 0,
          carbs: t.carbs || 0,
        });
      } catch (err) {
        console.warn("Failed to load nutrition targets", err);
      }
    };
    loadTargets();
  }, []);

  const totals = meals.reduce((acc, m) => ({
    kcal: acc.kcal + (m.kcal || 0),
    protein: acc.protein + (m.protein_g || 0),
    fat: acc.fat + (m.fat_g || 0),
    carbs: acc.carbs + (m.carbs_g || 0),
  }), { kcal: 0, protein: 0, fat: 0, carbs: 0 });

  const addMeal = async () => {
    const text = inputRef.current?.value?.trim();
    if (!text) return;
    setAdding(true);
    try {
      await api("/api/meal/add", { text });
      inputRef.current.value = "";
      await loadMeals();
      toast("Добавлено");
    } catch (err) {
      toast(formatApiError(err, "Не удалось добавить"));
    } finally {
      setAdding(false);
    }
  };

  const deleteMeal = async (id) => {
    try {
      await api("/api/meal/delete", { id });
      setMeals((prev) => prev.filter((m) => m.id !== id));
      toast("Удалено");
    } catch (err) {
      toast(formatApiError(err, "Ошибка удаления"));
    }
  };

  if (loading) return <div className="screen-loader is-loading"><div className="screen-spinner" /></div>;

  const getProgress = (current, target) => {
    if (!target || target === 0) return 0;
    return Math.min((current / target) * 100, 100);
  };

  return (
    <div className="screen active">
      <div className="card">
        <div className="card-title">Добавить еду</div>
        <textarea ref={inputRef} placeholder="Опиши что съел..." rows={3} />
        <button className="btn btn-accent" onClick={addMeal} disabled={adding} style={{ width: "100%", marginTop: 8 }}>
          {adding ? "Добавляю..." : "Добавить"}
        </button>
      </div>

      {/* KBJU Info Block */}
      <div className="card">
        <div className="card-title">КБЖУ сегодня</div>
        <div style={{ display: "grid", gap: 16 }}>
          {/* Calories */}
          <div style={{
            padding: "16px",
            background: "rgba(0,0,0,0.02)",
            borderRadius: 12,
            border: "1px solid rgba(0,0,0,0.06)",
          }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
              <span style={{ fontSize: 13, fontWeight: 600, color: "var(--muted)" }}>Калории</span>
              <span style={{
                fontSize: 12, fontWeight: 700,
                color: "var(--accent)",
                padding: "2px 8px",
                borderRadius: 999,
                background: "rgba(255,3,62,0.1)",
              }}>
                {Math.round(getProgress(totals.kcal, targets.kcal))}%
              </span>
            </div>
            <div style={{ fontSize: 24, fontWeight: 700, marginBottom: 8 }}>
              {Math.round(totals.kcal)}
              <span style={{ fontSize: 14, fontWeight: 500, color: "var(--muted)", marginLeft: 4 }}>
                / {targets.kcal || "—"} ккал
              </span>
            </div>
            <div style={{
              height: 6,
              background: "rgba(0,0,0,0.08)",
              borderRadius: 999,
              overflow: "hidden",
            }}>
              <div style={{
                height: "100%",
                width: `${getProgress(totals.kcal, targets.kcal)}%`,
                background: "var(--accent)",
                borderRadius: 999,
                transition: "width 0.4s ease",
              }} />
            </div>
          </div>

          {/* Macros Grid */}
          <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 10 }}>
            {[
              { label: "Белки", value: totals.protein, target: targets.protein, unit: "г" },
              { label: "Жиры", value: totals.fat, target: targets.fat, unit: "г" },
              { label: "Углеводы", value: totals.carbs, target: targets.carbs, unit: "г" },
            ].map((macro) => (
              <div key={macro.label} style={{
                padding: "12px",
                background: "rgba(0,0,0,0.02)",
                borderRadius: 12,
                border: "1px solid rgba(0,0,0,0.06)",
                textAlign: "center",
              }}>
                <div style={{ fontSize: 11, fontWeight: 600, color: "var(--muted)", marginBottom: 6 }}>
                  {macro.label}
                </div>
                <div style={{ fontSize: 20, fontWeight: 700, color: "var(--black)", marginBottom: 4 }}>
                  {Math.round(macro.value)}
                </div>
                <div style={{ fontSize: 10, color: "var(--muted)" }}>
                  / {macro.target || "—"} {macro.unit}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Meals History */}
      <div className="card">
        <div className="card-title">История приёмов пищи</div>
        {meals.length === 0 ? (
          <div className="muted" style={{ textAlign: "center", padding: 24 }}>
            Ещё нет записей
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {meals.map((meal, idx) => (
              <div
                key={meal.id}
                style={{
                  padding: "12px 16px",
                  background: idx % 2 === 0 ? "rgba(0,0,0,0.02)" : "transparent",
                  borderRadius: 12,
                  border: "1px solid rgba(0,0,0,0.06)",
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                  transition: "all 0.2s ease",
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = "rgba(0,0,0,0.04)";
                  e.currentTarget.style.borderColor = "rgba(0,0,0,0.1)";
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = idx % 2 === 0 ? "rgba(0,0,0,0.02)" : "transparent";
                  e.currentTarget.style.borderColor = "rgba(0,0,0,0.06)";
                }}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{
                    fontWeight: 600,
                    fontSize: 14,
                    marginBottom: 4,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}>
                    {meal.text || meal.name || "Приём пищи"}
                  </div>
                  <div style={{
                    fontSize: 12,
                    color: "var(--muted)",
                    display: "flex",
                    alignItems: "center",
                    gap: 8,
                    flexWrap: "wrap",
                  }}>
                    <span style={{
                      fontWeight: 600,
                      color: "var(--accent)",
                      padding: "2px 6px",
                      borderRadius: 4,
                      background: "rgba(255,3,62,0.1)",
                    }}>
                      {meal.kcal || 0} ккал
                    </span>
                    <span>
                      Б {meal.protein_g || 0} • Ж {meal.fat_g || 0} • У {meal.carbs_g || 0}
                    </span>
                  </div>
                </div>
                <button
                  onClick={() => deleteMeal(meal.id)}
                  style={{
                    background: "none",
                    border: "none",
                    cursor: "pointer",
                    color: "var(--muted)",
                    fontSize: 18,
                    padding: "6px 10px",
                    borderRadius: 8,
                    transition: "all 0.15s ease",
                    flexShrink: 0,
                    marginLeft: 12,
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                  onMouseEnter={(e) => {
                    e.target.style.color = "var(--accent)";
                    e.target.style.background = "rgba(255,3,62,0.08)";
                  }}
                  onMouseLeave={(e) => {
                    e.target.style.color = "var(--muted)";
                    e.target.style.background = "none";
                  }}
                  title="Удалить"
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
