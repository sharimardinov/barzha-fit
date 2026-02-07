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
        <div className="meal-totals">
          <span>{totals.kcal} ккал</span>
          <span className="muted"> · Б {totals.protein} · Ж {totals.fat} · У {totals.carbs}</span>
        </div>
        {meals.length === 0 && <div className="muted" style={{ textAlign: "center", padding: 16 }}>Ещё нет записей</div>}
        <div className="meal-list-items">
          {meals.map((meal) => (
            <div key={meal.id} className="list-item">
              <div>
                <div>{meal.text || meal.name || "Приём пищи"}</div>
                <div className="meta">{meal.kcal} ккал • Б{meal.protein_g || 0} Ж{meal.fat_g || 0} У{meal.carbs_g || 0}</div>
              </div>
              <div className="actions">
                <button className="btn btn-ghost" onClick={() => deleteMeal(meal.id)}>Удалить</button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
