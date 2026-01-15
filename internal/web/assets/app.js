const tg = window.Telegram?.WebApp || null;
const initData = tg?.initData || "";

const $ = (id) => document.getElementById(id);

const state = {
  today: null,
  planText: "",
};

const targetFields = [
  { field: "kcal", id: "target-kcal", planId: "plan-target-kcal" },
  { field: "protein", id: "target-protein", planId: "plan-target-protein" },
  { field: "fat", id: "target-fat", planId: "plan-target-fat" },
  { field: "carbs", id: "target-carbs", planId: "plan-target-carbs" },
  { field: "steps", id: "target-steps", planId: "plan-target-steps" },
];

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
    const err = new Error(data.error || "request_failed");
    err.data = data.data;
    throw err;
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
    training: "Тренинг",
    stats: "Статистика",
  };
  $("screen-title").textContent = titles[name] || "Сегодня";
}

function setProgress(id, current, target) {
  const el = $(id);
  if (!el) return;
  const ratio = target > 0 ? current / target : 0;
  const pct = Math.min(Math.max(ratio * 100, 0), 100);
  el.style.width = `${pct}%`;
}

function formatPlanForDisplay(plan) {
  const raw = String(plan || "").trim();
  if (!raw.startsWith("{")) return raw || "—";
  try {
    const data = JSON.parse(raw);
    const days = Array.isArray(data.days) ? data.days : null;
    if (!days || days.length < 7) return raw;
    const lines = [];
    for (let i = 0; i < 7; i += 1) {
      const text = String(days[i] || "—").trim() || "—";
      lines.push(`День ${i + 1}\n${text}`);
    }
    if (data.comment) {
      lines.push(`Комментарий:\n${String(data.comment).trim()}`);
    }
    return lines.join("\n\n");
  } catch (_) {
    return raw || "—";
  }
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

  setProgress("progress-kcal", data.kcal, data.targets.kcal);
  setProgress("progress-protein", data.protein, data.targets.protein);
  setProgress("progress-fat", data.fat, data.targets.fat);
  setProgress("progress-carbs", data.carbs, data.targets.carbs);
  setProgress("progress-steps", data.steps, data.targets.steps);

  $("steps-summary").textContent = `${data.steps} / ${data.targets.steps}`;
  setProgress("progress-steps-screen", data.steps, data.targets.steps);
}

async function loadMeals() {
  const items = await api("/api/meals/today");
  const list = $("meal-list");
  const listToday = $("meal-list-today");
  list.innerHTML = "";
  if (listToday) listToday.innerHTML = "";

  let totalKcal = 0;
  let totalP = 0;
  let totalF = 0;
  let totalC = 0;

  if (!items.length) {
    list.innerHTML = '<div class="hint">Пока пусто.</div>';
    if (listToday) listToday.innerHTML = '<div class="hint">Пока пусто.</div>';
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

      if (listToday) {
        const clone = card.cloneNode(true);
        clone.querySelector("button").addEventListener("click", async () => {
          await api("/api/meal/delete", { id: item.id });
          toast("Удалено");
          await loadMeals();
          await loadToday();
        });
        listToday.appendChild(clone);
      }
    });
  }

  $("meal-total-kcal").textContent = `${totalKcal} ккал`;
  $("meal-total-macros").textContent = `Б ${totalP} • Ж ${totalF} • У ${totalC}`;
  const totalTodayKcal = $("meal-total-kcal-today");
  const totalTodayMacros = $("meal-total-macros-today");
  if (totalTodayKcal) totalTodayKcal.textContent = `${totalKcal} ккал`;
  if (totalTodayMacros) totalTodayMacros.textContent = `Б ${totalP} • Ж ${totalF} • У ${totalC}`;
}

async function loadTargets() {
  const t = await api("/api/targets/get");
  targetFields.forEach(({ id, planId, field }) => {
    const value = t[field];
    const main = document.getElementById(id);
    if (main) main.value = value;
    const plan = document.getElementById(planId);
    if (plan) plan.value = value;
  });
}

async function loadPlan() {
  try {
    const data = await api("/api/plan/get");
    state.planText = data.text || "";
    $("plan-text").value = state.planText;
    const trainingResult = $("training-result");
    if (trainingResult) trainingResult.textContent = formatPlanForDisplay(state.planText);
  } catch (_) {
    state.planText = "";
    $("plan-text").value = "";
    const trainingResult = $("training-result");
    if (trainingResult) trainingResult.textContent = "—";
  }
}

async function loadProfile() {
  try {
    const p = await api("/api/profile/get");
    $("profile-sex").value = p.sex || "";
    $("profile-age").value = p.age || "";
    $("profile-height").value = p.height_cm || "";
    $("profile-weight").value = p.weight_kg || "";
    $("profile-training-years").value = p.training_years || "";
    $("profile-bodyfat").value = p.bodyfat_pct || "";
    $("profile-activity").value = p.activity_multiplier ? p.activity_multiplier.toFixed(2) : "";
    $("profile-goal").value = p.goal || "";
  } catch (_) {
    $("profile-sex").value = "";
  }
}

async function loadTrainingProfile() {
  try {
    const p = await api("/api/training/profile/get");
    $("training-bench").value = p.bench_kg || "";
    $("training-pullups").value = p.pullups || "";
    $("training-run").value = p.run_km || "";
    $("training-injuries").value = p.injuries || "";
    $("training-goal").value = p.goal || "";
    $("training-times").value = p.trainings_per_week || "";
    $("training-wishes").value = p.wishes || "";
    $("training-dislikes").value = p.dislikes || "";
    $("training-cannot").value = p.cannot_do || "";
    if (p.pharma === true) $("training-pharma").value = "yes";
    else if (p.pharma === false) $("training-pharma").value = "no";
    else $("training-pharma").value = "";
  } catch (_) {
    $("training-bench").value = "";
  }
}

async function loadStatsWeek() {
  const data = await api("/api/stats/week");
  renderWeekCalendars(data);
}

async function loadStatsMonth() {
  const data = await api("/api/stats/month");
  renderMonthCalendars(data);
}

async function loadStreak() {
  const data = await api("/api/streak");
  $("streak-workout").textContent = data.workoutStreak;
  $("streak-meal").textContent = data.mealStreak;
  const max = 7;
  const pct = Math.min((data.mealStreak / max) * 100, 100);
  $("streak-meal-progress").textContent = `${data.mealStreak} / ${max} дней`;
  const bar = $("streak-meal-bar");
  if (bar) bar.style.width = `${pct}%`;
}

function initNav() {
  document.querySelectorAll(".nav-btn").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const tab = btn.dataset.tab;
      setActiveTab(tab);
      if (tab === "today") {
        await loadToday();
        await loadMeals();
        await loadStreak();
      }
      if (tab === "meals") await loadMeals();
      if (tab === "plan") {
        await loadPlan();
        await loadTargets();
      }
      if (tab === "targets") await loadTargets();
      if (tab === "steps") await loadToday();
      if (tab === "profile") await loadProfile();
      if (tab === "training") {
        await loadTrainingProfile();
        await loadPlan();
      }
      if (tab === "stats") {
        await loadStatsWeek();
        await loadStatsMonth();
        toggleStatsView("week");
      }
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

  initNav();

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

  const saveTargets = async (prefix) => {
    for (const { field, id, planId } of targetFields) {
      const fieldId = prefix === "plan" ? planId : id;
      const el = document.getElementById(fieldId);
      const value = Number(el?.value || 0);
      await api("/api/targets/set", { field, value });
    }
    toast("Цели обновлены");
    await loadToday();
  };

  $("targets-save").addEventListener("click", async () => {
    await saveTargets("main");
  });

  $("targets-refresh").addEventListener("click", async () => {
    await api("/api/targets/refresh");
    await loadTargets();
    toast("Пересчитано");
    await loadToday();
  });

  const planTargetsSave = document.getElementById("plan-targets-save");
  if (planTargetsSave) {
    planTargetsSave.addEventListener("click", async () => {
      await saveTargets("plan");
    });
  }

  const planTargetsRefresh = document.getElementById("plan-targets-refresh");
  if (planTargetsRefresh) {
    planTargetsRefresh.addEventListener("click", async () => {
      await api("/api/targets/refresh");
      await loadTargets();
      toast("Пересчитано");
      await loadToday();
    });
  }

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
      training_years: Number($("profile-training-years").value || 0),
      bodyfat_pct: Number($("profile-bodyfat").value || 0),
      goal: $("profile-goal").value,
    };
    await api("/api/profile/set", payload);
    toast("Профиль сохранён");
  });

  const activityCalc = document.getElementById("profile-activity-calc");
  if (activityCalc) {
    activityCalc.addEventListener("click", async () => {
      try {
        const p = await api("/api/activity/estimate");
        $("profile-activity").value = p.activity_multiplier ? p.activity_multiplier.toFixed(2) : "";
        toast("Коэффициент рассчитан");
      } catch (err) {
        if (err.message === "plan_not_found") {
          toast("Сначала заполни план тренировок");
          return;
        }
        toast("Ошибка расчёта");
      }
    });
  }

  const trainingSave = document.getElementById("training-save");
  if (trainingSave) {
    trainingSave.addEventListener("click", async () => {
      const payload = {
        bench_kg: Number($("training-bench").value || 0),
        pullups: Number($("training-pullups").value || 0),
        run_km: Number($("training-run").value || 0),
        injuries: $("training-injuries").value.trim(),
        goal: $("training-goal").value,
        pharma: $("training-pharma").value === "yes" ? true : $("training-pharma").value === "no" ? false : null,
        trainings_per_week: Number($("training-times").value || 0),
        wishes: $("training-wishes").value.trim(),
        dislikes: $("training-dislikes").value.trim(),
        cannot_do: $("training-cannot").value.trim(),
      };
      await api("/api/training/profile/set", payload);
      toast("Анкета сохранена");
    });
  }

  const trainingGenerate = document.getElementById("training-generate");
  if (trainingGenerate) {
    trainingGenerate.addEventListener("click", async () => {
      const payload = {
        bench_kg: Number($("training-bench").value || 0),
        pullups: Number($("training-pullups").value || 0),
        run_km: Number($("training-run").value || 0),
        injuries: $("training-injuries").value.trim(),
        goal: $("training-goal").value,
        pharma: $("training-pharma").value === "yes" ? true : $("training-pharma").value === "no" ? false : null,
        trainings_per_week: Number($("training-times").value || 0),
        wishes: $("training-wishes").value.trim(),
        dislikes: $("training-dislikes").value.trim(),
        cannot_do: $("training-cannot").value.trim(),
      };
      try {
        await api("/api/training/profile/set", payload);
        const res = await api("/api/training/generate");
        $("training-result").textContent = formatPlanForDisplay(res.plan);
        await loadPlan();
        toast("План сохранён");
      } catch (err) {
        if (err.message === "missing_fields" && err.data?.fields?.length) {
          toast(`Заполни: ${err.data.fields.join(", ")}`);
          return;
        }
        toast("Ошибка генерации");
      }
    });
  }

  $("stats-week").addEventListener("click", async () => {
    toggleStatsView("week");
    await loadStatsWeek();
  });

  $("stats-month").addEventListener("click", async () => {
    toggleStatsView("month");
    await loadStatsMonth();
  });

  $("streak-refresh").addEventListener("click", async () => {
    await loadStreak();
  });

  await loadToday();
  await loadMeals();
  await loadStreak();
  lucide?.createIcons();
}

function toggleStatsView(view) {
  $("stats-week").classList.toggle("btn-accent", view === "week");
  $("stats-week").classList.toggle("btn-outline", view !== "week");
  $("stats-month").classList.toggle("btn-accent", view === "month");
  $("stats-month").classList.toggle("btn-outline", view !== "month");
  $("stats-week-section").classList.toggle("active", view === "week");
  $("stats-month-section").classList.toggle("active", view === "month");
}

function renderWeekCalendars(data) {
  const food = $("week-food-calendar");
  const steps = $("week-steps-calendar");
  food.innerHTML = "";
  steps.innerHTML = "";
  if (!data || !Array.isArray(data.days) || data.days.length === 0) {
    renderEmptyCalendar(food);
    renderEmptyCalendar(steps);
    return;
  }
  data.days.forEach((d) => {
    const future = isFutureDate(d.date);
    food.appendChild(makeCalendarCell(d.day, d.foodOk, future));
    steps.appendChild(makeCalendarCell(d.day, d.stepsOk, future));
  });
}

function renderMonthCalendars(data) {
  const food = $("month-food-calendar");
  const steps = $("month-steps-calendar");
  food.innerHTML = "";
  steps.innerHTML = "";
  if (!data || !Array.isArray(data.days) || data.days.length === 0) {
    renderEmptyCalendar(food);
    renderEmptyCalendar(steps);
    return;
  }
  for (let i = 0; i < data.offset; i += 1) {
    food.appendChild(makeEmptyCell());
    steps.appendChild(makeEmptyCell());
  }
  data.days.forEach((d) => {
    const future = isFutureDate(d.date);
    food.appendChild(makeCalendarCell(d.day, d.foodOk, future));
    steps.appendChild(makeCalendarCell(d.day, d.stepsOk, future));
  });
}

function makeCalendarCell(day, active, future) {
  const cell = document.createElement("div");
  cell.className = future ? "calendar-cell future" : "calendar-cell";
  const dot = document.createElement("div");
  dot.className = `calendar-dot${active ? " active" : ""}`;
  const label = document.createElement("div");
  label.textContent = String(day);
  cell.appendChild(dot);
  cell.appendChild(label);
  return cell;
}

function makeEmptyCell() {
  const cell = document.createElement("div");
  cell.className = "calendar-cell calendar-empty";
  const dot = document.createElement("div");
  dot.className = "calendar-dot";
  const label = document.createElement("div");
  label.textContent = "-";
  cell.appendChild(dot);
  cell.appendChild(label);
  return cell;
}

function renderEmptyCalendar(target) {
  const label = document.createElement("div");
  label.className = "calendar-empty-label";
  label.textContent = "Нет данных";
  target.appendChild(label);
}

function isFutureDate(value) {
  if (!value) return false;
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const date = new Date(`${value}T00:00:00`);
  return date > today;
}

bootstrap().catch((err) => {
  console.error(err);
  toast("Ошибка мини-аппа");
});
