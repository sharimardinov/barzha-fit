import { useEffect, useState, useCallback, useRef } from "react";
import { api } from "../services/api";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";

export default function MealsPage() {
  const toast = useToast();
  const [meals, setMeals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [adding, setAdding] = useState(false);
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

  return (
    <div className="screen active">
      <div className="card">
        <div className="card-title">Добавить еду</div>
        <textarea ref={inputRef} placeholder="Опиши что съел..." rows={3} />
        <button className="btn btn-accent" onClick={addMeal} disabled={adding} style={{ width: "100%", marginTop: 8 }}>
          {adding ? "Добавляю..." : "Добавить"}
        </button>
      </div>

      <div className="card">
        <div className="card-title">Сегодня</div>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", padding: "8px 0 12px", borderBottom: "1px solid var(--border)", marginBottom: 12 }}>
          <span style={{ fontSize: 20, fontWeight: 700 }}>{totals.kcal} <span style={{ fontSize: 13, fontWeight: 400, color: "var(--muted)" }}>ккал</span></span>
          <div style={{ display: "flex", gap: 12 }}>
            <span style={{ fontSize: 13, color: "var(--muted)" }}>Б <b style={{ color: "var(--black)" }}>{totals.protein}</b></span>
            <span style={{ fontSize: 13, color: "var(--muted)" }}>Ж <b style={{ color: "var(--black)" }}>{totals.fat}</b></span>
            <span style={{ fontSize: 13, color: "var(--muted)" }}>У <b style={{ color: "var(--black)" }}>{totals.carbs}</b></span>
          </div>
        </div>
        {meals.length === 0 && <div className="muted" style={{ textAlign: "center", padding: 16 }}>Ещё нет записей</div>}
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {meals.map((meal) => (
            <div key={meal.id} className="list-item" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 500, marginBottom: 4, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                  {meal.text || meal.name || "Приём пищи"}
                </div>
                <div style={{ display: "flex", gap: 8, fontSize: 12, color: "var(--muted)" }}>
                  <span style={{ fontWeight: 600, color: "var(--black)" }}>{meal.kcal} ккал</span>
                  <span>Б {meal.protein_g || 0}</span>
                  <span>Ж {meal.fat_g || 0}</span>
                  <span>У {meal.carbs_g || 0}</span>
                </div>
              </div>
              <button
                onClick={() => deleteMeal(meal.id)}
                style={{
                  background: "none", border: "none", cursor: "pointer",
                  color: "var(--muted)", fontSize: 18, padding: "4px 8px",
                  borderRadius: 8, transition: "color 0.15s, background 0.15s",
                  flexShrink: 0, marginLeft: 8,
                }}
                onMouseEnter={(e) => { e.target.style.color = "var(--accent)"; e.target.style.background = "rgba(255,3,62,0.08)"; }}
                onMouseLeave={(e) => { e.target.style.color = "var(--muted)"; e.target.style.background = "none"; }}
                title="Удалить"
              >✕</button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
