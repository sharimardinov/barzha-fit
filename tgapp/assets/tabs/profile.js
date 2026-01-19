import {
  $,
  api,
  state,
  toast,
  setButtonLoading,
  formatApiError,
  setGoalTabs,
  splitGoalText,
  buildTrainingGoalText,
  getGoalTypeFromTabs,
  normalizeNumberInput,
  setActiveScreen,
  loadTargets,
} from "../core.js";
import { loadToday } from "./today.js";
import { loadPlan } from "./plan.js";

let injuryOptions = [];
let selectedInjuries = [];

async function ensureInjuryOptions() {
  if (injuryOptions.length > 0) {
    return injuryOptions;
  }
  try {
    const list = await api("/api/training/injuries");
    injuryOptions = Array.isArray(list) ? list : [];
  } catch (_) {
    injuryOptions = [];
  }
  renderInjuryOptions();
  return injuryOptions;
}

function renderInjuryOptions() {
  const container = $("training-injuries");
  if (!container) return;
  container.innerHTML = "";
  injuryOptions.forEach((item) => {
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "option-card";
    btn.textContent = item.label;
    btn.classList.toggle("active", selectedInjuries.includes(item.code));
    btn.addEventListener("click", () => {
      if (selectedInjuries.includes(item.code)) {
        selectedInjuries = selectedInjuries.filter((v) => v !== item.code);
      } else {
        selectedInjuries = [...selectedInjuries, item.code];
      }
      btn.classList.toggle("active", selectedInjuries.includes(item.code));
    });
    container.appendChild(btn);
  });
}

function setSelectedInjuries(raw) {
  const list = String(raw || "")
    .split(/[,;]+/)
    .map((item) => item.trim())
    .filter(Boolean);
  if (injuryOptions.length > 0) {
    const allowed = new Set(injuryOptions.map((item) => item.code));
    selectedInjuries = list.filter((item) => allowed.has(item));
  } else {
    selectedInjuries = list;
  }
  renderInjuryOptions();
}

function formatSelectedInjuries() {
  return selectedInjuries.join(", ");
}

export async function loadProfile() {
  try {
    const p = await api("/api/profile/get");
    $("profile-sex").value = p.sex || "";
    $("profile-age").value = p.age ?? "";
    $("profile-height").value = p.height_cm ?? "";
    $("profile-weight").value = p.weight_kg ?? "";
    $("profile-bodyfat").value = p.bodyfat_pct ?? "";
    setGoalTabs(p.goal || "balance");
    $("profile-activity").textContent = p.activity_multiplier ? p.activity_multiplier.toFixed(2) : "—";
  } catch (_) {
    $("profile-sex").value = "";
    setGoalTabs("balance");
  }
}

export async function loadTrainingProfile() {
  await ensureInjuryOptions();
  try {
    const p = await api("/api/training/profile/get");
    setSelectedInjuries(p.injuries || "");
    const parsed = splitGoalText(p.goal || "");
    if (parsed.type) {
      setGoalTabs(parsed.type);
    } else {
      setGoalTabs(p.goal || "balance");
    }
    $("training-times").value = p.trainings_per_week || "";
    if (p.pharma === true) $("training-pharma").value = "yes";
    else if (p.pharma === false) $("training-pharma").value = "no";
    else $("training-pharma").value = "";
  } catch (_) {
    setSelectedInjuries("");
  }
}

function parseNumberField(id, label, opts) {
  const normalized = normalizeNumberInput($(id).value);
  if (!normalized) {
    return { ok: !opts.required, value: 0, label };
  }
  const value = Number(normalized);
  if (!Number.isFinite(value)) {
    return { ok: false, value: 0, label };
  }
  if (opts.integer && !Number.isInteger(value)) {
    return { ok: false, value: 0, label };
  }
  if (value < opts.min || value > opts.max) {
    return { ok: false, value: 0, label };
  }
  return { ok: true, value, label };
}

export function validateProfileInputs() {
  const issues = [];
  const age = parseNumberField("profile-age", "возраст", { required: true, integer: true, min: 14, max: 80 });
  const height = parseNumberField("profile-height", "рост", { required: true, integer: true, min: 100, max: 250 });
  const weight = parseNumberField("profile-weight", "вес", { required: true, integer: false, min: 30, max: 300 });
  const bodyfat = parseNumberField("profile-bodyfat", "процент жира", { required: true, integer: false, min: 1, max: 100 });
  const goalType = getGoalTypeFromTabs();

  if (!age.ok) issues.push(age.label);
  if (!height.ok) issues.push(height.label);
  if (!weight.ok) issues.push(weight.label);
  if (!bodyfat.ok) issues.push(bodyfat.label);
  if (!goalType) issues.push("цель");

  if (issues.length) {
    toast(`Заполни корректно: ${issues.join(", ")}`);
    return null;
  }

  return {
    age: age.value,
    height: height.value,
    weight: weight.value,
    bodyfat: bodyfat.value,
    trainingYears: 0,
    goalType,
  };
}

export function validateTrainingInputs() {
  const issues = [];
  const times = parseNumberField("training-times", "тренировок в неделю", { required: true, integer: true, min: 1, max: 7 });
  const goalType = getGoalTypeFromTabs();
  const goalNotes = "";
  const pharmaValue = $("training-pharma").value;
  const pharma = pharmaValue === "yes" ? true : pharmaValue === "no" ? false : null;

  if (!times.ok) issues.push(times.label);
  if (!goalType) issues.push("цель");
  if (pharma === null) issues.push("фармакология");

  if (issues.length) {
    toast(`Заполни корректно: ${issues.join(", ")}`);
    return null;
  }

  return {
    bench: 0,
    pullups: 0,
    run: 0,
    times: times.value,
    goalType,
    goalNotes,
    pharma,
  };
}

export async function saveProfileFlow(payload, trainingPayload, button, opts = {}) {
  setButtonLoading(button, true, "Сохраняю...");
  const activityEl = $("profile-activity");
  if (activityEl) activityEl.textContent = "…";
  try {
    await api("/api/profile/set", payload);
    await api("/api/targets/refresh");
    if (opts.planMode === "ai") {
      await api("/api/training/profile/set", trainingPayload);
      if (opts.trainingInput) {
        await api("/api/training/program/generate", opts.trainingInput);
        await loadPlan();
      }
    } else if (opts.planMode === "manual" && typeof opts.planText === "string") {
      await api("/api/plan/set", { text: opts.planText });
      await loadPlan();
    } else if (opts.planMode === "ai-profile") {
      await api("/api/training/profile/set", trainingPayload);
    }

    await loadTargets();
    await loadToday();
    toast("Профиль сохранён", button);
    if (state.onboarding) {
      setActiveScreen("onboarding-done");
    }
  } catch (err) {
    if (err.message === "missing_fields" && err.data?.fields?.length) {
      toast(`Заполни: ${err.data.fields.join(", ")}`);
      return false;
    }
    toast(formatApiError(err, "Ошибка пересчёта"));
    return false;
  } finally {
    setButtonLoading(button, false);
  }
  return true;
}

export function initProfileTab() {
  ensureInjuryOptions();
  const profileSave = $("profile-save");
  if (profileSave) {
    profileSave.addEventListener("click", async () => {
      const profileValidated = validateProfileInputs();
      const trainingValidated = validateTrainingInputs();
      if (!profileValidated || !trainingValidated) return;

      const payload = {
        sex: $("profile-sex").value,
        age: profileValidated.age,
        height_cm: profileValidated.height,
        weight_kg: profileValidated.weight,
        training_years: profileValidated.trainingYears,
        bodyfat_pct: profileValidated.bodyfat,
        goal: profileValidated.goalType,
      };
      const trainingGoal = buildTrainingGoalText(trainingValidated.goalType, trainingValidated.goalNotes);
      const trainingPayload = {
        bench_kg: trainingValidated.bench || 0,
        pullups: trainingValidated.pullups || 0,
        run_km: trainingValidated.run || 0,
        injuries: formatSelectedInjuries(),
        goal: trainingGoal,
        pharma: trainingValidated.pharma,
        trainings_per_week: trainingValidated.times,
        wishes: "",
      };
      await saveProfileFlow(payload, trainingPayload, profileSave, { planMode: "ai-profile" });
    });
  }
}

export function isProfileComplete(p) {
  return (
    p &&
    (p.sex === "m" || p.sex === "f") &&
    p.age >= 14 &&
    p.age <= 80 &&
    p.height_cm >= 100 &&
    p.height_cm <= 250 &&
    p.weight_kg >= 30 &&
    p.weight_kg <= 300 &&
    p.bodyfat_pct >= 1 &&
    p.bodyfat_pct <= 100 &&
    p.training_years >= 0 &&
    p.training_years <= 80
  );
}

export function isTrainingComplete(tp) {
  return (
    tp &&
    tp.bench_kg >= 0 &&
    tp.bench_kg <= 400 &&
    tp.pullups >= 0 &&
    tp.pullups <= 100 &&
    tp.run_km >= 0 &&
    tp.run_km <= 300 &&
    tp.trainings_per_week >= 1 &&
    tp.trainings_per_week <= 7 &&
    typeof tp.goal === "string" &&
    tp.goal.trim().length > 0 &&
    tp.pharma !== null &&
    tp.pharma !== undefined
  );
}
