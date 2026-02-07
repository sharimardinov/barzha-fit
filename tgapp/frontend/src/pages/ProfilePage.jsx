import { useEffect, useState, useCallback } from "react";
import { api } from "../services/api";
import { useAppState } from "../hooks/useAppState";
import { useToast } from "../components/Toast";
import { formatApiError } from "../services/errors";
import GoalSelector from "../components/GoalSelector";
import { Settings } from "lucide-react";
import { postNativeMessage } from "../services/telegram";

export default function ProfilePage() {
  const { state, dispatch } = useAppState();
  const toast = useToast();
  const [loading, setLoading] = useState(true);
  const [editMode, setEditMode] = useState(false);
  const [saving, setSaving] = useState(false);
  const [profile, setProfile] = useState({ sex: "", age: "", height_cm: "", weight_kg: "", bodyfat_pct: "" });
  const [goal, setGoal] = useState("balance");
  const [activity, setActivity] = useState("—");

  const loadProfile = useCallback(async () => {
    try {
      const p = await api("/api/profile/get");
      setProfile({
        sex: p.sex || "",
        age: p.age || "",
        height_cm: p.height_cm || "",
        weight_kg: p.weight_kg || "",
        bodyfat_pct: p.bodyfat_pct || "",
      });
      setGoal(p.goal || "balance");
      setActivity(p.activity || "—");
      dispatch({ type: "SET_PROFILE", payload: p });
    } catch (err) {
      if (err.message !== "profile_not_found") {
        toast(formatApiError(err, "Ошибка загрузки профиля"));
      }
    }
    try {
      const tp = await api("/api/training/profile/get");
      const goalText = tp.goal || "";
      const match = goalText.match(/^(cut|balance|bulk)/i);
      if (match) setGoal(match[1].toLowerCase());
    } catch { /* ignore */ }
    setLoading(false);
  }, [dispatch, toast]);

  useEffect(() => { loadProfile(); }, [loadProfile]);

  const saveProfile = async () => {
    setSaving(true);
    try {
      await api("/api/profile/set", {
        sex: profile.sex,
        age: Number(profile.age),
        height_cm: Number(profile.height_cm),
        weight_kg: Number(profile.weight_kg),
        bodyfat_pct: Number(profile.bodyfat_pct),
        goal,
      });
      await api("/api/targets/refresh");
      await api("/api/training/profile/set", {
        bench_kg: 0, pullups: 0, run_km: 0,
        injuries: "", goal: `${goal}`, pharma: false,
        trainings_per_week: 0, wishes: "",
      });
      toast("Профиль сохранён");
      setEditMode(false);
      const p = await api("/api/profile/get");
      setActivity(p.activity || "—");
    } catch (err) {
      toast(formatApiError(err, "Ошибка сохранения"));
    } finally {
      setSaving(false);
    }
  };

  const logout = () => {
    localStorage.removeItem("auth_token");
    if (postNativeMessage("authState", { action: "logout", source: "profile" })) return;
    if (window.Telegram?.WebApp?.close) { window.Telegram.WebApp.close(); return; }
    window.location.reload();
  };

  if (loading) return <div className="screen-loader is-loading"><div className="screen-spinner" /></div>;

  return (
    <div className="screen active">
      <div className="card">
        <div className="card-title" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          Профиль
          <button
            onClick={() => setEditMode(!editMode)}
            title="Настройки"
            style={{
              background: editMode ? "rgba(255,3,62,0.08)" : "rgba(0,0,0,0.04)",
              border: "none", borderRadius: 10, padding: "6px 8px", cursor: "pointer",
              display: "flex", alignItems: "center", justifyContent: "center",
              transition: "background 0.2s, transform 0.2s",
              transform: editMode ? "rotate(90deg)" : "rotate(0deg)",
            }}
          >
            <Settings size={18} color={editMode ? "var(--accent)" : "#71717a"} strokeWidth={2} />
          </button>
        </div>
        <div className="form-grid">
          <label>Пол
            <select value={profile.sex} onChange={(e) => setProfile({ ...profile, sex: e.target.value })} disabled={!editMode}>
              <option value="">Выбери</option>
              <option value="m">Мужской</option>
              <option value="f">Женский</option>
            </select>
          </label>
          <label>Возраст
            <input type="number" value={profile.age} onChange={(e) => setProfile({ ...profile, age: e.target.value })} disabled={!editMode} />
          </label>
          <label>Рост (см)
            <input type="number" value={profile.height_cm} onChange={(e) => setProfile({ ...profile, height_cm: e.target.value })} disabled={!editMode} />
          </label>
          <label>Вес (кг)
            <input type="number" value={profile.weight_kg} onChange={(e) => setProfile({ ...profile, weight_kg: e.target.value })} disabled={!editMode} />
          </label>
          <label>% жира
            <input type="number" value={profile.bodyfat_pct} onChange={(e) => setProfile({ ...profile, bodyfat_pct: e.target.value })} disabled={!editMode} />
          </label>
        </div>
        <div className="form-grid" style={{ marginTop: 12 }}>
          <label>Цель
            <GoalSelector value={goal} onChange={setGoal} disabled={!editMode} />
          </label>
        </div>
        <div className="row activity-hint" style={{ marginTop: 8 }}>
          <div className="muted">Коэффициент активности: <span>{activity}</span></div>
        </div>
        {editMode && (
          <button className="btn btn-accent" onClick={saveProfile} disabled={saving} style={{ width: "100%", marginTop: 12 }}>
            {saving ? "Сохраняю..." : "Сохранить"}
          </button>
        )}
      </div>
      <div className="row profile-logout-row" style={{ marginTop: 12 }}>
        <button className="btn btn-outline" onClick={logout} style={{ width: "100%" }}>Выйти</button>
      </div>
    </div>
  );
}
