const tg = window.Telegram?.WebApp || null;
const initData = tg?.initData || "";

const $ = (id) => document.getElementById(id);

const state = {
  today: null,
  planText: "",
};

function toast(message) {
  const el = $("toast");
  el.textContent = message;
  el.classList.add("show");
  setTimeout(() => el.classList.remove("show"), 1800);
}

async function api(path, body = {}) {
  const res = await fetch(path, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Tg-Init-Data": initData,
    },
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!data.ok) {
    throw new Error(data.error || "request_failed");
  }
  return data.data;
}

function setActiveTab(name) {
  document.querySelectorAll(".nav-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.tab === name);
  });
  document.querySelectorAll(".screen").forEach((panel) => {
    panel.classList.toggle("active", panel.id === `screen-${name}`);
  });
  const titles = {
    today: "Сегодня",
    meals: "Еда",
    plan: "План",
    targets: "Цели",
    steps: "Шаги",
    profile: "Профиль",
    stats: "Статистика",
    streak: "Серии",
  };
  $("screen-title").textContent = titles[name] || "Сегодня";
}

function setProgress(id, current, target, color) {
  const el = $(id);
  if (!el) return;
  const ratio = target > 0 ? current / target : 0;
  const pct = Math.min(Math.max(ratio * 100, 0), 100);
  el.style.width = `${pct}%`;
  if (color) el.style.background = color;
}

async function loadToday() {
  const data = await api("/api/today");
  state.today = data;
  $("today-plan").textContent = data.plan || "—";
  $("today-workout").textContent = data.workoutIcon || "—";
  $("today-kcal").textContent = `${data.kcal} / ${data.targets.kcal} ${data.icons.kcal}`;
  $("today-protein").textContent = `${data.protein} / ${data.targets.protein} ${data.icons.protein}`;
  $("today-fat").textContent = `${data.fat} / ${data.targets.fat} ${data.icons.fat}`;
  $("today-carbs").textContent = `${data.carbs} / ${data.targets.carbs} ${data.icons.carbs}`;
  $("today-steps").textContent = `${data.steps} / ${data.targets.steps} ${data.icons.steps}`;
  $("today-food").textContent = data.icons.food || "—";

  setProgress("progress-kcal", data.kcal, data.targets.kcal, "var(--accent-1)");
  setProgress("progress-protein", data.protein, data.targets.protein, "var(--accent-2)");
  setProgress("progress-fat", data.fat, data.targets.fat, "var(--accent-3)");
  setProgress("progress-carbs", data.carbs, data.targets.carbs, "var(--accent-4)");
  setProgress("progress-steps", data.steps, data.targets.steps, "var(--success)");

  $("steps-summary").textContent = `${data.steps} / ${data.targets.steps}`;
  setProgress("progress-steps-screen", data.steps, data.targets.steps, "var(--success)");
}

async function loadMeals() {
  const items = await api("/api/meals/today");
  const list = $("meal-list");
  list.innerHTML = "";
  let totalKcal = 0;
  let totalP = 0;
  let totalF = 0;
  let totalC = 0;

  if (!items.length) {
    list.innerHTML = '<div class="hint">Пока пусто.</div>';
  } else {
    items.forEach((item) => {
      totalKcal += item.kcal;
      totalP += item.protein_g;
      totalF += item.fat_g;
      totalC += item.carbs_g;
      const card = document.createElement("div");
      card.className = "list-item";
      card.innerHTML = `
        <div>${item.text}</div>
        <div class="meta">${item.kcal} ккал • Б${item.protein_g} Ж${item.fat_g} У${item.carbs_g}</div>
        <div class="actions">
          <button class="btn btn-ghost" data-id="${item.id}">Удалить</button>
        </div>
      `;
      card.querySelector("button").addEventListener("click", async () => {
        await api("/api/meal/delete", { id: item.id });
        toast("Удалено");
        await loadMeals();
        await loadToday();
      });
      list.appendChild(card);
    });
  }

  $("meal-total-kcal").textContent = `${totalKcal} ккал`;
  $("meal-total-macros").textContent = `Б ${totalP} • Ж ${totalF} • У ${totalC}`;
}

async function loadTargets() {
  const t = await api("/api/targets/get");
  $("target-kcal").value = t.kcal;
  $("target-protein").value = t.protein;
  $("target-fat").value = t.fat;
  $("target-carbs").value = t.carbs;
  $("target-steps").value = t.steps;
}

async function loadPlan() {
  try {
    const data = await api("/api/plan/get");
    state.planText = data.text || "";
    $("plan-text").value = state.planText;
  } catch (_) {
    state.planText = "";
    $("plan-text").value = "";
  }
}

async function loadProfile() {
  try {
    const p = await api("/api/profile/get");
    $("profile-sex").value = p.sex || "";
    $("profile-age").value = p.age || "";
    $("profile-height").value = p.height_cm || "";
    $("profile-weight").value = p.weight_kg || "";
    $("profile-bodyfat").value = p.bodyfat_pct || "";
    $("profile-activity").value = p.activity || "";
    $("profile-goal").value = p.goal || "";
  } catch (_) {
    $("profile-sex").value = "";
  }
}

async function loadStatsWeek() {
  const data = await api("/api/stats/week");
  $("stats-week-text").innerHTML = data.text;
}

async function loadStatsMonth() {
  const data = await api("/api/stats/month");
  $("stats-month-text").textContent = data.text;
}

async function loadStreak() {
  const data = await api("/api/streak");
  $("streak-workout").textContent = data.workoutStreak;
  $("streak-meal").textContent = data.mealStreak;
  $("streak-bar").textContent = data.mealBar;
}

function initNav() {
  document.querySelectorAll(".nav-btn").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const tab = btn.dataset.tab;
      setActiveTab(tab);
      if (tab === "today") await loadToday();
      if (tab === "meals") await loadMeals();
      if (tab === "plan") await loadPlan();
      if (tab === "targets") await loadTargets();
      if (tab === "steps") await loadToday();
      if (tab === "profile") await loadProfile();
      if (tab === "stats") {
        await loadStatsWeek();
        await loadStatsMonth();
      }
      if (tab === "streak") await loadStreak();
    });
  });
}

async function bootstrap() {
  if (!initData) {
    toast("Открой мини-апп из Telegram");
    return;
  }
  tg?.ready();
  tg?.expand();
  if (tg?.initDataUnsafe?.user) {
    const u = tg.initDataUnsafe.user;
    $("user-line").textContent = `Привет, ${u.first_name || u.username || "спортсмен"}!`;
  }

  initNav();

  $("refresh-today").addEventListener("click", async () => {
    await loadToday();
    toast("Обновлено");
  });

  $("workout-done").addEventListener("click", async () => {
    await api("/api/workout/set", { status: "done" });
    await loadToday();
    toast("Отмечено ✅");
  });

  $("workout-skip").addEventListener("click", async () => {
    await api("/api/workout/set", { status: "skip" });
    await loadToday();
    toast("Отмечено ❌");
  });

  $("meal-add").addEventListener("click", async () => {
    const text = $("meal-text").value.trim();
    if (!text) return;
    const data = await api("/api/meal/add", { text });
    toast(data.aiError ? "AI упал, текст сохранён" : "Еда добавлена");
    $("meal-text").value = "";
    await loadMeals();
    await loadToday();
  });

  $("meal-undo").addEventListener("click", async () => {
    const res = await api("/api/meal/undo");
    toast(res.deleted ? "Удалено" : "Нечего удалять");
    await loadMeals();
    await loadToday();
  });

  $("plan-save").addEventListener("click", async () => {
    const text = $("plan-text").value.trim();
    if (!text) return;
    await api("/api/plan/set", { text });
    state.planText = text;
    toast("План сохранён");
    await loadToday();
  });

  $("plan-reset").addEventListener("click", () => {
    $("plan-text").value = state.planText || "";
    toast("Сброшено");
  });

  $("targets-save").addEventListener("click", async () => {
    const fields = [
      ["kcal", "target-kcal"],
      ["protein", "target-protein"],
      ["fat", "target-fat"],
      ["carbs", "target-carbs"],
      ["steps", "target-steps"],
    ];
    for (const [field, id] of fields) {
      const value = Number($(id).value || 0);
      await api("/api/targets/set", { field, value });
    }
    toast("Цели обновлены");
    await loadToday();
  });

  $("targets-refresh").addEventListener("click", async () => {
    await api("/api/targets/refresh");
    await loadTargets();
    toast("Пересчитано");
    await loadToday();
  });

  $("steps-save").addEventListener("click", async () => {
    const steps = Number($("steps-value").value || 0);
    await api("/api/steps/set", { steps });
    toast("Шаги записаны");
    await loadToday();
  });

  $("profile-save").addEventListener("click", async () => {
    const payload = {
      sex: $("profile-sex").value,
      age: Number($("profile-age").value || 0),
      height_cm: Number($("profile-height").value || 0),
      weight_kg: Number($("profile-weight").value || 0),
      bodyfat_pct: Number($("profile-bodyfat").value || 0),
      activity: $("profile-activity").value,
      goal: $("profile-goal").value,
    };
    await api("/api/profile/set", payload);
    toast("Профиль сохранён");
  });

  $("stats-week").addEventListener("click", async () => {
    await loadStatsWeek();
  });

  $("stats-month").addEventListener("click", async () => {
    await loadStatsMonth();
  });

  $("streak-refresh").addEventListener("click", async () => {
    await loadStreak();
  });

  await loadToday();
  lucide?.createIcons();
}

bootstrap().catch((err) => {
  console.error(err);
  toast("Ошибка мини-аппа");
});
