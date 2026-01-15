const tg = window.Telegram?.WebApp || null;
const initData = tg?.initData || "";

const $ = (id) => document.getElementById(id);

const state = {
  today: null,
  planText: "",
  planPayload: null,
  planStructured: false,
  onboarding: false,
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

function setButtonLoading(btn, loading, label) {
  if (!btn) return;
  const spinnerId = btn.dataset.spinner;
  const spinner = spinnerId ? $(spinnerId) : null;
  if (loading) {
    if (!btn.dataset.label) {
      btn.dataset.label = btn.textContent || "";
    }
    btn.classList.add("loading");
    btn.disabled = true;
    btn.setAttribute("aria-busy", "true");
    if (label) btn.textContent = label;
    if (spinner) spinner.classList.add("active");
    return;
  }
  btn.classList.remove("loading");
  btn.disabled = false;
  btn.removeAttribute("aria-busy");
  if (btn.dataset.label) {
    btn.textContent = btn.dataset.label;
  }
  if (spinner) spinner.classList.remove("active");
}

function formatApiError(err, fallback) {
  const code = err?.message || "";
  if (code === "training_ai_unavailable") return "AI тренировок недоступен";
  if (code === "activity_ai_unavailable") return "AI активности недоступен";
  if (code === "training_generate_failed") return "Не удалось сгенерировать план";
  if (code === "activity_estimate_failed") return "Не удалось пересчитать активность";
  if (code === "plan_not_found") return "План не найден для пересчёта активности";
  if (code === "plan_save_failed") return "Не удалось сохранить план";
  if (code === "training_profile_save_failed") return "Ошибка сохранения тренировочного профиля";
  if (code === "profile_save_failed") return "Ошибка сохранения профиля";
  return fallback;
}

function buildWheelValues(min, max, step) {
  const values = [];
  const isFloat = Math.abs(step % 1) > 0;
  for (let v = min; v <= max + 1e-9; v += step) {
    values.push(isFloat ? v.toFixed(1) : String(Math.round(v)));
  }
  return values;
}

function createWheel(values, initial, onChange, axis = "y", extraClass = "") {
  const wheel = document.createElement("div");
  wheel.className = `wheel ${extraClass}`.trim();
  const list = document.createElement("div");
  list.className = "wheel-list";
  const selector = document.createElement("div");
  selector.className = "wheel-selector";

  values.forEach((value) => {
    const item = document.createElement("div");
    item.className = "wheel-item";
    item.textContent = value;
    item.dataset.value = value;
    list.appendChild(item);
  });

  const itemSize = axis === "x" ? 70 : 40;
  let currentValue = null;
  let ticking = false;

  const syncActive = () => {
    const scrollPos = axis === "x" ? list.scrollLeft : list.scrollTop;
    const idx = Math.min(values.length - 1, Math.max(0, Math.round(scrollPos / itemSize)));
    const children = list.children;
    for (let i = 0; i < children.length; i += 1) {
      children[i].classList.toggle("active", i === idx);
    }
    const value = values[idx];
    if (value !== currentValue) {
      currentValue = value;
      onChange(value);
    }
  };

  list.addEventListener("scroll", () => {
    if (ticking) return;
    ticking = true;
    requestAnimationFrame(() => {
      syncActive();
      ticking = false;
    });
  });

  list.addEventListener("click", (event) => {
    const item = event.target.closest(".wheel-item");
    if (!item) return;
    const idx = values.indexOf(item.dataset.value || "");
    if (idx >= 0) {
      if (axis === "x") {
        list.scrollTo({ left: idx * itemSize, behavior: "smooth" });
      } else {
        list.scrollTo({ top: idx * itemSize, behavior: "smooth" });
      }
    }
  });

  wheel.appendChild(list);
  wheel.appendChild(selector);

  const initialIndex = Math.max(0, values.indexOf(initial));
  if (axis === "x") {
    list.scrollLeft = initialIndex * itemSize;
  } else {
    list.scrollTop = initialIndex * itemSize;
  }
  requestAnimationFrame(syncActive);

  return wheel;
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

function setActiveScreen(name) {
  document.querySelectorAll(".screen").forEach((panel) => {
    panel.classList.toggle("active", panel.id === `screen-${name}`);
  });
  const titles = {
    today: "Сегодня",
    meals: "Еда",
    steps: "Шаги",
    profile: "Профиль",
    plan: "План",
    stats: "Статистика",
    onboarding: "",
    "onboarding-done": "",
  };
  if ($("screen-title")) {
    $("screen-title").textContent = titles[name] || "Сегодня";
  }
}

function setActiveTab(name) {
  document.querySelectorAll(".nav-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.tab === name);
  });
  setActiveScreen(name);
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
    const data = safeParseJSON(raw);
    if (Array.isArray(data.week_plan)) {
      return formatWeekPlan(data.week_plan);
    }
    const days = Array.isArray(data.days) ? data.days : null;
    if (!days || days.length === 0) return raw;
    const lines = [];
    for (let i = 0; i < 7; i += 1) {
      const rawDay = String(days[i] || "").replace(/\r\n/g, "\n");
      const dayLines = rawDay
        .split("\n")
        .map((line) => line.trim().replace(/^[_*]+|[_*]+$/g, ""))
        .filter(Boolean);
      const title = dayLines.length ? dayLines[0] : "—";
      const body = dayLines.length > 1 ? dayLines.slice(1).join("\n") : "";
      lines.push(`${title}${body ? `\n${body}` : ""}`);
    }
    return lines.join("\n\n");
  } catch (_) {
    return raw || "—";
  }
}

function parsePlan(plan) {
  const raw = String(plan || "").trim();
  if (!raw.startsWith("{")) {
    return { text: raw || "—", structured: false };
  }
  try {
    const data = safeParseJSON(raw);
    if (Array.isArray(data.week_plan) && data.week_plan.length) {
      return { text: formatWeekPlan(data.week_plan), structured: true, weekPlan: data.week_plan };
    }
    const days = Array.isArray(data.days) ? data.days : null;
    if (days && days.length) {
      return { text: formatPlanForDisplay(raw), structured: true };
    }
  } catch (_) {
    return { text: raw || "—", structured: false };
  }
  return { text: raw || "—", structured: false };
}

function renderTrainingAccordion(items, containerId = "training-accordion") {
  const container = $(containerId);
  if (!container) return;
  container.innerHTML = "";
  const hasContent = (dayItem) => {
    if (!dayItem || typeof dayItem !== "object") return false;
    const name = String(dayItem.name || "").trim();
    const focus = String(dayItem.focus || "").trim();
    const itemsList = Array.isArray(dayItem.items) ? dayItem.items.filter((v) => String(v || "").trim()) : [];
    const groups = Array.isArray(dayItem.groups) ? dayItem.groups : [];
    const activities = Array.isArray(dayItem.activities) ? dayItem.activities : [];
    const notes = String(dayItem.notes || "").trim();
    if (itemsList.length > 0) return true;
    if (groups.some((g) => Array.isArray(g.exercises) && g.exercises.length)) return true;
    if (activities.some((a) => String(a || "").trim() !== "")) return true;
    if (notes) return true;
    return name !== "" || focus !== "";
  };
  items.forEach((dayItem) => {
    if (!hasContent(dayItem)) return;
    const dayNum = Number(dayItem.day || 0);
    const name = String(dayItem.name || "—").trim();
    const focus = String(dayItem.focus || "").trim();
    const title = focus ? `${name} (${focus})` : name;

    const item = document.createElement("div");
    item.className = "accordion-item";

    const toggle = document.createElement("button");
    toggle.type = "button";
    toggle.className = "accordion-toggle";
    const label = document.createElement("span");
    label.textContent = `${title}`;
    toggle.appendChild(label);
    toggle.addEventListener("click", () => item.classList.toggle("open"));

    const body = document.createElement("div");
    body.className = "accordion-body";

    let hasExercises = false;
    let counter = 0;
    const items = Array.isArray(dayItem.items) ? dayItem.items : [];
    if (items.length) {
      const list = document.createElement("ol");
      list.className = "accordion-list";
      items.forEach((entry) => {
        const text = String(entry || "").trim();
        if (!text) return;
        hasExercises = true;
        counter += 1;
        const li = document.createElement("li");
        li.textContent = text;
        list.appendChild(li);
      });
      if (list.children.length) body.appendChild(list);
    }

    const groups = Array.isArray(dayItem.groups) ? dayItem.groups : [];
    groups.forEach((group) => {
      const groupName = String(group.muscle_group || "").trim();
      if (groupName) {
        const h = document.createElement("div");
        h.className = "accordion-group-title";
        h.textContent = groupName;
        body.appendChild(h);
      }
      const list = document.createElement("ol");
      list.className = "accordion-list";
      const exercises = Array.isArray(group.exercises) ? group.exercises : [];
      exercises.forEach((ex) => {
        const exName = String(ex.name || "").trim();
        if (!exName) return;
        hasExercises = true;
        counter += 1;
        const sets = String(ex.sets || "").trim();
        const reps = String(ex.reps || "").trim();
        const duration = String(ex.duration || "").trim();
        const notes = String(ex.notes || "").trim();
        let tail = "";
        if (duration) tail = duration;
        else if (sets || reps) tail = `${sets}${sets && reps ? "x" : ""}${reps}`;
        let text = `${exName}`;
        if (tail) text += ` — ${tail}`;
        if (notes) text += ` (${notes})`;
        const li = document.createElement("li");
        li.textContent = text;
        list.appendChild(li);
      });
      if (list.children.length) body.appendChild(list);
    });

    if (!hasExercises) {
      const activities = Array.isArray(dayItem.activities) ? dayItem.activities : [];
      if (activities.length) {
        const list = document.createElement("ol");
        list.className = "accordion-list";
        activities.forEach((act) => {
          const text = String(act || "").trim();
          if (!text) return;
          const li = document.createElement("li");
          li.textContent = text;
          list.appendChild(li);
        });
        if (list.children.length) body.appendChild(list);
      }
    }

    item.appendChild(toggle);
    item.appendChild(body);
    container.appendChild(item);
  });
}

function safeParseJSON(raw) {
  const start = raw.indexOf("{");
  const end = raw.lastIndexOf("}");
  const sliced = start >= 0 && end > start ? raw.slice(start, end + 1) : raw;
  const cleaned = normalizeJSONString(sliced)
    .replace(/^\uFEFF/, "")
    .replace(/,\s*([}\]])/g, "$1");
  return JSON.parse(cleaned);
}

function extractPlanPayload(planText) {
  const raw = String(planText || "").trim();
  if (!raw.startsWith("{")) return null;
  try {
    return safeParseJSON(raw);
  } catch (_) {
    return null;
  }
}

function normalizeJSONString(raw) {
  let out = "";
  let inString = false;
  let escape = false;
  for (let i = 0; i < raw.length; i += 1) {
    const ch = raw[i];
    if (escape) {
      out += ch;
      escape = false;
      continue;
    }
    if (ch === "\\" && inString) {
      out += ch;
      escape = true;
      continue;
    }
    if (ch === "\"") {
      inString = !inString;
      out += ch;
      continue;
    }
    if (inString && ch === "\n") {
      out += "\\n";
      continue;
    }
    if (inString && ch === "\r") {
      continue;
    }
    out += ch;
  }
  return out;
}

function formatWeekPlan(items) {
  const lines = [];
  items.forEach((dayItem, idx) => {
    const dayNum = Number(dayItem.day || idx + 1);
    const name = String(dayItem.name || "—").trim();
    const focus = String(dayItem.focus || "").trim();
    const title = focus ? `${name} (${focus})` : name;
    const chunks = [];
    let counter = 0;
    let hasExercises = false;
    const items = Array.isArray(dayItem.items) ? dayItem.items : [];
    if (items.length) {
      items.forEach((entry) => {
        const text = String(entry || "").trim();
        if (!text) return;
        hasExercises = true;
        counter += 1;
        chunks.push(`${counter}. ${text}`);
      });
      chunks.push("");
    }

    const groups = Array.isArray(dayItem.groups) ? dayItem.groups : [];
    groups.forEach((group) => {
      const groupName = String(group.muscle_group || "").trim();
      if (groupName) chunks.push(groupName);
      const exercises = Array.isArray(group.exercises) ? group.exercises : [];
      exercises.forEach((ex) => {
        const exName = String(ex.name || "").trim();
        if (!exName) return;
        hasExercises = true;
        counter += 1;
        const sets = String(ex.sets || "").trim();
        const reps = String(ex.reps || "").trim();
        const duration = String(ex.duration || "").trim();
        const notes = String(ex.notes || "").trim();
        let tail = "";
        if (duration) tail = duration;
        else if (sets || reps) tail = `${sets}${sets && reps ? "x" : ""}${reps}`;
        let line = `${counter}. ${exName}`;
        if (tail) line += ` — ${tail}`;
        if (notes) line += ` (${notes})`;
        chunks.push(line);
      });
      if (exercises.length) chunks.push("");
    });
    if (!hasExercises) {
      const activities = Array.isArray(dayItem.activities) ? dayItem.activities : [];
      activities.forEach((act) => {
        const text = String(act || "").trim();
        if (!text) return;
        counter += 1;
        chunks.push(`${counter}. ${text}`);
      });
    }
    if (dayItem.notes) {
      const note = String(dayItem.notes || "").trim();
      if (note) chunks.push(note);
    }
    const body = chunks.filter((line) => line !== "").join("\n");
    lines.push(`${title}${body ? `\n${body}` : ""}`);
  });
  return lines.join("\n\n");
}

function setPlanEditMode(active) {
  const editor = $("training-editor");
  const accordion = $("training-accordion");
  const result = $("training-result");
  const editBtn = $("training-edit");
  const saveBtn = $("training-save");
  const cancelBtn = $("training-cancel");
  if (!editor || !editBtn || !saveBtn || !cancelBtn) return;

  if (active) {
    editor.classList.add("active");
    if (accordion) accordion.style.display = "none";
    if (result) result.style.display = "none";
    editBtn.style.display = "none";
    saveBtn.style.display = "inline-flex";
    cancelBtn.style.display = "inline-flex";
    return;
  }
  editor.classList.remove("active");
  if (accordion) accordion.style.display = "";
  if (result) result.style.display = "";
  if (state.planStructured && state.planPayload?.week_plan?.length) {
    editBtn.style.display = "inline-flex";
  } else {
    editBtn.style.display = "none";
  }
  saveBtn.style.display = "none";
  cancelBtn.style.display = "none";
}

function renderTrainingEditor(payload) {
  const editor = $("training-editor");
  if (!editor) return;
  editor.innerHTML = "";
  const weekPlan = Array.isArray(payload?.week_plan) ? payload.week_plan : [];
  weekPlan.forEach((day, index) => {
    const wrapper = document.createElement("div");
    wrapper.className = "training-day-editor";
    wrapper.dataset.index = String(index);

    const title = document.createElement("div");
    title.className = "muted";
    title.textContent = `День ${day.day || index + 1}`;
    wrapper.appendChild(title);

    const nameInput = document.createElement("input");
    nameInput.type = "text";
    nameInput.placeholder = "Название дня";
    const dayFallback = `День ${day.day || index + 1}`;
    const rawName = String(day.name || "").trim();
    nameInput.value = rawName || dayFallback;
    nameInput.dataset.field = "name";
    wrapper.appendChild(nameInput);

    const focusInput = document.createElement("input");
    focusInput.type = "text";
    focusInput.placeholder = "Фокус (можно пусто)";
    focusInput.value = String(day.focus || "").trim();
    focusInput.dataset.field = "focus";
    wrapper.appendChild(focusInput);

    const itemsArea = document.createElement("textarea");
    itemsArea.placeholder = "Упражнения, по одному на строку";
    itemsArea.value = Array.isArray(day.items) ? day.items.join("\n") : "";
    itemsArea.dataset.field = "items";
    wrapper.appendChild(itemsArea);

    editor.appendChild(wrapper);
  });
}

async function loadToday() {
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
    const icon = data.workoutIcon || "";
    workout.textContent = icon === "—" ? "" : icon;
    workout.classList.toggle("is-empty", icon === "" || icon === "—");
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
  setMetricValue("today-steps", `${data.steps} / ${data.targets.steps}`, data.icons.steps);

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
    state.planPayload = extractPlanPayload(state.planText);
    state.planStructured = Array.isArray(state.planPayload?.week_plan);
    const parsed = parsePlan(state.planText);
    const trainingResult = $("training-result");
    if (parsed.weekPlan && parsed.weekPlan.length) {
      renderTrainingAccordion(parsed.weekPlan);
      if (trainingResult) trainingResult.style.display = "none";
    } else {
      if (trainingResult) {
        trainingResult.style.display = "block";
        trainingResult.textContent = parsed.text || "—";
      }
      const container = $("training-accordion");
      if (container) container.innerHTML = "";
    }
    setPlanEditMode(false);
  } catch (_) {
    state.planText = "";
    state.planPayload = null;
    state.planStructured = false;
    const trainingResult = $("training-result");
    if (trainingResult) trainingResult.textContent = "—";
    setPlanEditMode(false);
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
    $("profile-activity").textContent = p.activity_multiplier ? p.activity_multiplier.toFixed(2) : "—";
  } catch (_) {
    $("profile-sex").value = "";
  }
}

function normalizeNumberInput(value) {
  return String(value || "").replace(/,/g, ".").replace(/\s+/g, "");
}

function parseNumberInput(value) {
  const normalized = normalizeNumberInput(value);
  return Number(normalized || 0);
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

function validateProfileInputs() {
  const issues = [];
  const age = parseNumberField("profile-age", "возраст", { required: true, integer: true, min: 14, max: 80 });
  const height = parseNumberField("profile-height", "рост", { required: true, integer: true, min: 100, max: 250 });
  const weight = parseNumberField("profile-weight", "вес", { required: true, integer: false, min: 30, max: 300 });
  const bodyfat = parseNumberField("profile-bodyfat", "процент жира", { required: true, integer: false, min: 1, max: 100 });
  const trainingYears = parseNumberField("profile-training-years", "стаж тренировок", {
    required: true,
    integer: true,
    min: 1,
    max: 80,
  });

  if (!age.ok) issues.push(age.label);
  if (!height.ok) issues.push(height.label);
  if (!weight.ok) issues.push(weight.label);
  if (!bodyfat.ok) issues.push(bodyfat.label);
  if (!trainingYears.ok) issues.push(trainingYears.label);

  if (issues.length) {
    toast(`Заполни корректно: ${issues.join(", ")}`);
    return null;
  }

  return {
    age: age.value,
    height: height.value,
    weight: weight.value,
    bodyfat: bodyfat.value,
    trainingYears: trainingYears.value,
  };
}

function validateTrainingInputs() {
  const issues = [];
  const bench = parseNumberField("training-bench", "жим лёжа", { required: true, integer: true, min: 0, max: 400 });
  const pullups = parseNumberField("training-pullups", "подтягивания", { required: true, integer: true, min: 0, max: 100 });
  const run = parseNumberField("training-run", "бег", { required: true, integer: false, min: 0, max: 300 });
  const times = parseNumberField("training-times", "тренировок в неделю", { required: true, integer: true, min: 1, max: 7 });
  const goal = $("training-goal").value.trim();
  const pharmaValue = $("training-pharma").value;
  const pharma = pharmaValue === "yes" ? true : pharmaValue === "no" ? false : null;

  if (!bench.ok) issues.push(bench.label);
  if (!pullups.ok) issues.push(pullups.label);
  if (!run.ok) issues.push(run.label);
  if (!times.ok) issues.push(times.label);
  if (!goal) issues.push("цель");
  if (pharma === null) issues.push("фармакология");

  if (issues.length) {
    toast(`Заполни корректно: ${issues.join(", ")}`);
    return null;
  }

  return {
    bench: bench.value,
    pullups: pullups.value,
    run: run.value,
    times: times.value,
    goal,
    pharma,
  };
}

function setOnboardingActive(active) {
  state.onboarding = active;
  document.body.classList.toggle("onboarding", active);
}

function isProfileComplete(p) {
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
    p.training_years >= 1 &&
    p.training_years <= 80
  );
}

function isTrainingComplete(tp) {
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

async function ensureOnboarding() {
  let profile = null;
  try {
    profile = await api("/api/profile/get");
  } catch (err) {
    if (err.message !== "profile_not_found") {
      toast("Ошибка загрузки профиля");
      return false;
    }
  }
  let training = null;
  try {
    training = await api("/api/training/profile/get");
  } catch (_) {
    training = null;
  }

  if (!isProfileComplete(profile) || !isTrainingComplete(training)) {
    setOnboardingActive(true);
    setActiveScreen("onboarding");
    return false;
  }

  setOnboardingActive(false);
  return true;
}

async function runProfilePipeline() {
  await api("/api/training/generate");
  const activity = await api("/api/activity/estimate");
  $("profile-activity").textContent = activity.activity_multiplier ? activity.activity_multiplier.toFixed(2) : "—";
  await api("/api/targets/refresh");
}

async function saveProfileFlow(payload, trainingPayload, button) {
  setButtonLoading(button, true, "Сохраняю и считаю...");
  const activityEl = $("profile-activity");
  if (activityEl) activityEl.textContent = "…";
  try {
    await api("/api/profile/set", payload);
    await api("/api/training/profile/set", trainingPayload);
    let pipelineOk = false;
    for (let attempt = 1; attempt <= 3; attempt += 1) {
      try {
        await runProfilePipeline();
        pipelineOk = true;
        break;
      } catch (err) {
        if (err.message === "training_plan_invalid" && attempt < 3) {
          continue;
        }
        if (err.message === "training_plan_invalid") {
          toast("План кривой. Нажми ещё раз.");
          return false;
        }
        throw err;
      }
    }
    if (!pipelineOk) {
      return false;
    }
    await loadPlan();
    await loadTargets();
    await loadToday();
    toast("Профиль сохранён");
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

function initNav() {
  document.querySelectorAll(".nav-btn").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const tab = btn.dataset.tab;
      setActiveTab(tab);
      if (tab === "today") {
        await loadToday();
      }
      if (tab === "meals") await loadMeals();
      if (tab === "plan") {
        await loadPlan();
        await loadTargets();
      }
      if (tab === "steps") await loadToday();
      if (tab === "profile") {
        await loadProfile();
        await loadTrainingProfile();
      }
      if (tab === "stats") {
        await loadStatsWeek();
        await loadStatsMonth();
        toggleStatsView("week");
      }
    });
  });
}

function initOnboardingWizard() {
  const progressEl = $("onboarding-progress");
  const titleEl = $("onboarding-title");
  const descEl = $("onboarding-desc");
  const bodyEl = $("onboarding-body");
  const helpEl = $("onboarding-help");
  const backBtn = $("onboarding-back");
  const nextBtn = $("onboarding-next");

  if (!progressEl || !titleEl || !descEl || !bodyEl || !helpEl || !nextBtn || !backBtn) {
    return;
  }

  const steps = [
    {
      id: "sex",
      title: "Твой пол",
      type: "options",
      options: [
        { value: "m", label: "Мужской" },
        { value: "f", label: "Женский" },
      ],
      help: "Нужно для корректных норм и нагрузки.",
      required: true,
    },
    {
      id: "age",
      title: "Сколько тебе лет?",
      type: "input",
      inputType: "number",
      min: 14,
      max: 80,
      placeholder: "Например: 28",
      help: "Возраст влияет на восстановление и объём.",
      required: true,
    },
    {
      id: "height",
      title: "Твой рост",
      type: "input",
      inputType: "number",
      min: 100,
      max: 250,
      placeholder: "Например: 175",
      help: "Используем для расчёта калорий и целей.",
      required: true,
    },
    {
      id: "weight",
      title: "Твой вес",
      type: "input",
      inputType: "number",
      min: 30,
      max: 300,
      placeholder: "Например: 70",
      help: "Нужен для расчёта нагрузки и калорий.",
      required: true,
    },
    {
      id: "trainingYears",
      title: "Стаж тренировок (лет)",
      type: "input",
      inputType: "number",
      min: 1,
      max: 80,
      placeholder: "Например: 3",
      help: "Определяет уровень и подбор упражнений.",
      required: true,
    },
    {
      id: "bodyfat",
      title: "Процент жира",
      type: "input",
      inputType: "number",
      min: 1,
      max: 100,
      placeholder: "Например: 18",
      help: "Для точных целей по весу и форме.",
      required: true,
    },
    {
      id: "bench",
      title: "Жим лёжа (кг)",
      type: "input",
      inputType: "number",
      min: 0,
      max: 400,
      placeholder: "Например: 80",
      help: "Понимаем силу верхнего тела.",
      required: true,
    },
    {
      id: "pullups",
      title: "Подтягивания (количество)",
      type: "input",
      inputType: "number",
      min: 0,
      max: 100,
      placeholder: "Например: 8",
      help: "Оценка тяговой силы.",
      required: true,
    },
    {
      id: "run",
      title: "Бег (сколько км сможешь пробежать)",
      type: "input",
      inputType: "number",
      min: 0,
      max: 100,
      placeholder: "Например: 5",
      help: "Помогает оценить выносливость.",
      required: true,
    },
    {
      id: "injuries",
      title: "Травмы и ограничения",
      type: "textarea",
      placeholder: "Например: грыжа L5-S1",
      help: "Чтобы исключить рискованные упражнения.",
      required: false,
    },
    {
      id: "goal",
      title: "Твоя цель",
      type: "textarea",
      placeholder: "Коротко: цель и фокус",
      help: "Определяет акцент программы.",
      required: true,
    },
    {
      id: "pharma",
      title: "Фармакология",
      type: "options",
      options: [
        { value: true, label: "Да" },
        { value: false, label: "Нет" },
      ],
      help: "Влияет на восстановление и объём.",
      required: true,
    },
    {
      id: "trainingsPerWeek",
      title: "Тренировок в неделю",
      type: "input",
      inputType: "number",
      min: 1,
      max: 7,
      placeholder: "Например: 4",
      help: "Формируем недельную структуру.",
      required: true,
    },
    {
      id: "wishes",
      title: "Пожелания",
      type: "textarea",
      placeholder: "Например: больше спины, не люблю бег",
      help: "Учтём предпочтения и ограничения.",
      required: false,
    },
  ];

  const data = {};
  let stepIndex = 0;

  const getDefaultWheelValue = (step) => {
    if (step.defaultValue !== undefined) {
      return String(step.defaultValue);
    }
    const midpoint = (step.min + step.max) / 2;
    const isFloat = Math.abs(step.step % 1) > 0;
    const rounded = isFloat
      ? (Math.round(midpoint / step.step) * step.step).toFixed(1)
      : String(Math.round(midpoint));
    return rounded;
  };

  const formatWheelValue = (step, value) => {
    if (value === undefined || value === null || Number.isNaN(value)) {
      return getDefaultWheelValue(step);
    }
    const isFloat = Math.abs(step.step % 1) > 0;
    return isFloat ? Number(value).toFixed(1) : String(Math.round(Number(value)));
  };

  const setValue = (step, raw) => {
    if (step.type === "wheel") {
      const val = Math.abs(step.step % 1) > 0 ? Number.parseFloat(raw) : Number(raw);
      data[step.id] = Number.isNaN(val) ? null : val;
      return;
    }
    data[step.id] = raw;
  };

  const renderStep = () => {
    const step = steps[stepIndex];
    progressEl.textContent = `Шаг ${stepIndex + 1} из ${steps.length}`;
    titleEl.textContent = step.title;
    descEl.textContent = step.type === "wheel" || step.type === "wheel-horizontal"
      ? `Прокрути колесо${step.unit ? ` (${step.unit})` : ""}`
      : step.type === "options"
        ? "Выбери один вариант"
        : "Введи значение";
    helpEl.textContent = step.help || "";
    bodyEl.innerHTML = "";

    if (step.type === "options") {
      const list = document.createElement("div");
      list.className = "option-list";
      step.options.forEach((opt) => {
        const btn = document.createElement("button");
        btn.type = "button";
        let className = "option-card";
        if (step.id === "sex" && opt.value === "m") className += " option-male";
        if (step.id === "sex" && opt.value === "f") className += " option-female";
        btn.className = className;
        btn.textContent = opt.label;
        btn.classList.toggle("active", data[step.id] === opt.value);
        btn.addEventListener("click", () => {
          data[step.id] = opt.value;
          renderStep();
        });
        list.appendChild(btn);
      });
      bodyEl.appendChild(list);
    }

    if (step.type === "wheel" || step.type === "wheel-horizontal") {
      const values = buildWheelValues(step.min, step.max, step.step);
      const initial = formatWheelValue(step, data[step.id]);
      if (data[step.id] === undefined) {
        setValue(step, initial);
      }
      const axis = step.type === "wheel-horizontal" ? "x" : "y";
      const extraClass = step.type === "wheel-horizontal"
        ? "horizontal"
        : step.variant === "side"
          ? "side"
          : "";
      const wheel = createWheel(values, initial, (value) => setValue(step, value), axis, extraClass);
      bodyEl.appendChild(wheel);
    }

    if (step.type === "input") {
      const field = document.createElement("input");
      field.type = step.inputType || "text";
      field.inputMode = field.type === "number" ? "decimal" : "text";
      if (step.min !== undefined) field.min = String(step.min);
      if (step.max !== undefined) field.max = String(step.max);
      field.value = data[step.id] || "";
      field.placeholder = step.placeholder || "";
      field.addEventListener("input", () => {
        data[step.id] = field.value;
      });
      bodyEl.appendChild(field);
    }

    if (step.type === "textarea") {
      const field = document.createElement("textarea");
      field.placeholder = step.placeholder || "";
      field.value = data[step.id] || "";
      field.addEventListener("input", () => {
        data[step.id] = field.value;
      });
      bodyEl.appendChild(field);
    }

    backBtn.style.visibility = stepIndex === 0 ? "hidden" : "visible";
    nextBtn.textContent = stepIndex === steps.length - 1 ? "Готово" : "Далее";
  };

  const validateStep = (step) => {
    const value = data[step.id];
    if (!step.required) return true;
    if (step.type === "options" && (value === undefined || value === null || value === "")) {
      toast("Выбери вариант");
      return false;
    }
    if ((step.type === "wheel" || step.type === "wheel-horizontal") && (value === undefined || value === null || Number.isNaN(value))) {
      toast("Выбери значение");
      return false;
    }
    if (step.type === "input") {
      const raw = String(value || "").trim();
      if (!raw) {
        toast("Заполни поле");
        return false;
      }
      if (step.inputType === "number") {
        const numeric = Number(raw.replace(/,/g, "."));
        if (!Number.isFinite(numeric)) {
          toast("Введи число");
          return false;
        }
        if (step.min !== undefined && numeric < step.min) {
          toast(`Минимум ${step.min}`);
          return false;
        }
        if (step.max !== undefined && numeric > step.max) {
          toast(`Максимум ${step.max}`);
          return false;
        }
        data[step.id] = numeric;
      } else {
        if (step.id === "sex") {
          const normalized = raw.toLowerCase();
          if (["м", "m", "male", "муж", "мужской"].includes(normalized)) {
            data[step.id] = "m";
          } else if (["ж", "f", "female", "жен", "женский"].includes(normalized)) {
            data[step.id] = "f";
          } else {
            toast("Введи м или ж");
            return false;
          }
          return true;
        }
        if (step.id === "pharma") {
          const normalized = raw.toLowerCase();
          if (["да", "yes", "y", "true", "1"].includes(normalized)) {
            data[step.id] = true;
          } else if (["нет", "no", "n", "false", "0"].includes(normalized)) {
            data[step.id] = false;
          } else {
            toast("Введи да или нет");
            return false;
          }
          return true;
        }
        data[step.id] = raw;
      }
    }
    if (step.type === "textarea" && String(value || "").trim() === "") {
      toast("Заполни поле");
      return false;
    }
    return true;
  };

  const submitOnboarding = async () => {
    const payload = {
      sex: data.sex,
      age: Number(data.age || 0),
      height_cm: Number(data.height || 0),
      weight_kg: Number(data.weight || 0),
      training_years: Number(data.trainingYears || 0),
      bodyfat_pct: Number(data.bodyfat || 0),
    };
    const trainingPayload = {
      bench_kg: Number(data.bench || 0),
      pullups: Number(data.pullups || 0),
      run_km: Number(data.run || 0),
      injuries: String(data.injuries || "").trim(),
      goal: String(data.goal || "").trim(),
      pharma: data.pharma,
      trainings_per_week: Number(data.trainingsPerWeek || 0),
      wishes: String(data.wishes || "").trim(),
    };
    await saveProfileFlow(payload, trainingPayload, nextBtn);
  };

  backBtn.addEventListener("click", () => {
    if (stepIndex === 0) return;
    stepIndex -= 1;
    renderStep();
  });

  nextBtn.addEventListener("click", async () => {
    const step = steps[stepIndex];
    if (!validateStep(step)) return;
    if (stepIndex === steps.length - 1) {
      await submitOnboarding();
      return;
    }
    stepIndex += 1;
    renderStep();
  });

  renderStep();
}

async function bootstrap() {
  if (!initData) {
    toast("Открой мини-апп из Telegram");
    return;
  }
  tg?.ready();
  tg?.expand();

  initNav();

  const onboardingReady = await ensureOnboarding();
  if (!onboardingReady) {
    initOnboardingWizard();
    const onboardingGoToday = $("onboarding-go-today");
    if (onboardingGoToday) {
      onboardingGoToday.addEventListener("click", async () => {
        setOnboardingActive(false);
        setActiveTab("today");
        await loadToday();
      });
    }
  }

  const todayAddMeal = $("today-add-meal");
  if (todayAddMeal) {
    todayAddMeal.addEventListener("click", async () => {
      setActiveTab("meals");
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
      toast("Отмечено ✅");
    });
  }

  const workoutSkip = $("workout-skip");
  if (workoutSkip) {
    workoutSkip.addEventListener("click", async () => {
      await api("/api/workout/set", { status: "skip" });
      await loadToday();
      toast("Отмечено ❌");
    });
  }

  const mealAdd = $("meal-add");
  if (mealAdd) {
    mealAdd.addEventListener("click", async () => {
      const text = $("meal-text").value.trim();
      if (!text) return;
      const data = await api("/api/meal/add", { text });
      toast(data.aiError ? "AI упал, текст сохранён" : "Еда добавлена");
      $("meal-text").value = "";
      await loadMeals();
      await loadToday();
    });
  }

  const saveTargets = async (prefix) => {
    for (const { field, id, planId } of targetFields) {
      const fieldId = prefix === "plan" ? planId : id;
      const el = document.getElementById(fieldId);
      if (!el) continue;
      const value = parseNumberInput(el.value);
      await api("/api/targets/set", { field, value });
    }
    toast("Цели обновлены");
    await loadToday();
  };

  const trainingEdit = $("training-edit");
  if (trainingEdit) {
    trainingEdit.addEventListener("click", () => {
      if (!state.planStructured || !state.planPayload?.week_plan?.length) {
        toast("Редактирование доступно только для структурного плана");
        return;
      }
      renderTrainingEditor(state.planPayload);
      setPlanEditMode(true);
    });
  }

  const trainingCancel = $("training-cancel");
  if (trainingCancel) {
    trainingCancel.addEventListener("click", () => {
      setPlanEditMode(false);
    });
  }

  const trainingSave = $("training-save");
  if (trainingSave) {
    trainingSave.addEventListener("click", async () => {
      if (!state.planStructured || !state.planPayload?.week_plan?.length) {
        toast("Редактирование недоступно");
        return;
      }
      const editor = $("training-editor");
      if (!editor) return;
      const updated = JSON.parse(JSON.stringify(state.planPayload));
      const dayBlocks = Array.from(editor.querySelectorAll(".training-day-editor"));
      for (const block of dayBlocks) {
        const idx = Number(block.dataset.index || 0);
        const name = block.querySelector('[data-field="name"]')?.value.trim() || "";
        const focus = block.querySelector('[data-field="focus"]')?.value.trim() || "";
        const itemsRaw = block.querySelector('[data-field="items"]')?.value || "";
        const items = itemsRaw
          .split("\n")
          .map((line) => line.trim())
          .filter(Boolean);
        if (!name) {
          toast("Укажи название дня");
          return;
        }
        if (items.length === 0) {
          toast("Добавь упражнения");
          return;
        }
        updated.week_plan[idx].name = name;
        updated.week_plan[idx].focus = focus;
        updated.week_plan[idx].items = items;
      }

      const text = JSON.stringify(updated);
      setButtonLoading(trainingSave, true, "Сохраняю...");
      try {
        await api("/api/plan/set", { text });
        state.planText = text;
        state.planPayload = updated;
        await api("/api/activity/estimate");
        await loadPlan();
        await loadToday();
        const p = await api("/api/profile/get");
        const activityEl = $("profile-activity");
        if (activityEl) {
          activityEl.textContent = p.activity_multiplier ? p.activity_multiplier.toFixed(2) : "—";
        }
        toast("План обновлён");
      } catch (err) {
        toast(formatApiError(err, "Ошибка сохранения плана"));
      } finally {
        setButtonLoading(trainingSave, false);
      }
    });
  }

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

  const stepsSave = $("steps-save");
  if (stepsSave) {
    stepsSave.addEventListener("click", async () => {
      const steps = parseNumberInput($("steps-value").value);
      await api("/api/steps/set", { steps });
      toast("Шаги записаны");
      await loadToday();
    });
  }

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
      };
      const trainingPayload = {
        bench_kg: trainingValidated.bench,
        pullups: trainingValidated.pullups,
        run_km: trainingValidated.run,
        injuries: $("training-injuries").value.trim(),
        goal: trainingValidated.goal,
        pharma: trainingValidated.pharma,
        trainings_per_week: trainingValidated.times,
        wishes: $("training-wishes").value.trim(),
      };
      await saveProfileFlow(payload, trainingPayload, profileSave);
    });
  }

  const statsWeek = $("stats-week");
  if (statsWeek) {
    statsWeek.addEventListener("click", async () => {
      toggleStatsView("week");
      await loadStatsWeek();
    });
  }

  const statsMonth = $("stats-month");
  if (statsMonth) {
    statsMonth.addEventListener("click", async () => {
      toggleStatsView("month");
      await loadStatsMonth();
    });
  }

  if (!onboardingReady) {
    lucide?.createIcons();
    return;
  }

  setActiveTab("today");
  await loadToday();
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
