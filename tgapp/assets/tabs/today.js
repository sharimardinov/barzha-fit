import { $, api, state, toast, setProgress, setActiveTab } from "../core.js";
import { parsePlan, renderTrainingAccordion } from "../plan-utils.js";

let workoutCardProgress = 0;
let workoutCardAnim = null;

export async function loadToday() {
  const data = await api("/api/today");
  state.today = data;
  let planText = state.planText;
  if (!planText) {
    try {
      const plan = await api("/api/plan/get");
      planText = plan.text || "";
      state.planText = planText;
    } catch (_) {
      planText = "";
    }
  }
  const parsed = parsePlan(planText || data.plan || "");
  const todayResult = $("today-training-result");
  if (parsed.weekPlan && parsed.weekPlan.length) {
    const todayNum = Number(data.cycleDay || 0);
    const todayItem = parsed.weekPlan.find((item) => Number(item.day || 0) === todayNum);
    if (todayItem) {
      renderTrainingAccordion([todayItem], "today-accordion");
    } else {
      renderTrainingAccordion(parsed.weekPlan.slice(0, 1), "today-accordion");
    }
    if (todayResult) todayResult.style.display = "none";
  } else {
    if (todayResult) {
      todayResult.style.display = "block";
      todayResult.textContent = parsed.text || "—";
    }
    const container = $("today-accordion");
    if (container) container.innerHTML = "";
  }
  const workout = $("today-workout");
  if (workout) {
    const status = String(data.workout || "");
    const label = status === "done" ? "Сделано" : status === "skip" ? "Пропущено" : "—";
    workout.textContent = label;
    workout.classList.toggle("is-empty", label === "—");
  }

  setWorkoutCardProgress(data.workout === "done" ? 1 : 0);

  const doneBtn = $("workout-done");
  if (doneBtn) {
    const label = doneBtn.querySelector(".workout-ink");
    if (label) {
      label.textContent = data.workout === "done" ? "Сделано" : "Сделал";
    }
  }

  const setMetricValue = (id, text, icon) => {
    const el = $(id);
    if (!el) return;
    const status = icon === "🟢" ? "ok" : icon === "🔴" ? "bad" : "none";
    el.innerHTML = `${text} <span class="indicator ${status}"></span>`;
  };
  setMetricValue("today-kcal", `${data.kcal} / ${data.targets.kcal}`, data.icons.kcal);
  setMetricValue("today-protein", `${data.protein} / ${data.targets.protein}`, data.icons.protein);
  setMetricValue("today-fat", `${data.fat} / ${data.targets.fat}`, data.icons.fat);
  setMetricValue("today-carbs", `${data.carbs} / ${data.targets.carbs}`, data.icons.carbs);

  setProgress("progress-kcal", data.kcal, data.targets.kcal);
  setProgress("progress-protein", data.protein, data.targets.protein);
  setProgress("progress-fat", data.fat, data.targets.fat);
  setProgress("progress-carbs", data.carbs, data.targets.carbs);

  $("steps-summary").textContent = `${data.steps} / ${data.targets.steps}`;
  setProgress("progress-steps-screen", data.steps, data.targets.steps);
}

function setWorkoutCardProgress(target) {
  const card = $("workout-day-card");
  if (!card) return;
  const clamped = Math.max(0, Math.min(1, target));
  const current = workoutCardProgress;
  updateWorkoutInkPositions(card);
  if (Math.abs(current - clamped) < 0.001) {
    workoutCardProgress = clamped;
    card.style.setProperty("--workout-progress", `${clamped * 100}%`);
    card.classList.toggle("is-filled", clamped > 0);
    return;
  }
  card.classList.add("is-filled");
  if (workoutCardAnim) cancelAnimationFrame(workoutCardAnim);
  const start = performance.now();
  const duration = 360;
  const delta = clamped - current;
  const tick = (now) => {
    const t = Math.min(1, (now - start) / duration);
    const eased = 1 - Math.pow(1 - t, 3);
    const value = current + delta * eased;
    workoutCardProgress = value;
    card.style.setProperty("--workout-progress", `${value * 100}%`);
    if (t < 1) {
      workoutCardAnim = requestAnimationFrame(tick);
      return;
    }
    workoutCardAnim = null;
    if (clamped <= 0.001) {
      card.classList.remove("is-filled");
    }
  };
  workoutCardAnim = requestAnimationFrame(tick);
}

function updateWorkoutInkPositions(card) {
  const rect = card.getBoundingClientRect();
  card.style.setProperty("--workout-card-width", `${rect.width}px`);
  card.querySelectorAll(".workout-ink").forEach((el) => {
    const elRect = el.getBoundingClientRect();
    const offset = elRect.left - rect.left;
    el.style.setProperty("--workout-ink-offset", `${-offset}px`);
  });
}

export function initTodayTab() {
  const todayAddMeal = $("today-add-meal");
  if (todayAddMeal) {
    todayAddMeal.addEventListener("click", async () => {
      setActiveTab("meals");
      const { loadMeals } = await import("./meals.js");
      await loadMeals();
    });
  }
  const todayAddSteps = $("today-add-steps");
  if (todayAddSteps) {
    todayAddSteps.addEventListener("click", async () => {
      setActiveTab("steps");
      await loadToday();
    });
  }

  const workoutDone = $("workout-done");
  if (workoutDone) {
    workoutDone.addEventListener("click", async () => {
      await api("/api/workout/set", { status: "done" });
      await loadToday();
      toast("Отмечено");
    });
  }

  const workoutSkip = $("workout-skip");
  if (workoutSkip) {
    workoutSkip.addEventListener("click", async () => {
      await api("/api/workout/set", { status: "skip" });
      await loadToday();
      toast("Отмечено");
    });
  }

  const accordion = $("today-accordion");
  if (accordion) {
    accordion.addEventListener("click", () => {
      const card = $("workout-day-card");
      if (!card) return;
      requestAnimationFrame(() => updateWorkoutInkPositions(card));
    });
  }

  window.addEventListener("resize", () => {
    const card = $("workout-day-card");
    if (!card) return;
    updateWorkoutInkPositions(card);
  });
}
