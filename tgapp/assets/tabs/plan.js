import {
  $,
  api,
  state,
  toast,
  setButtonLoading,
  formatApiError,
  loadTargets,
  targetFields,
} from "../core.js";
import { extractPlanPayload, parsePlan, renderTrainingAccordion } from "../plan-utils.js";
import { loadToday } from "./today.js";

function buildWeekPlanTemplate(defaultItem = "") {
  const weekPlan = [];
  for (let day = 1; day <= 7; day += 1) {
    weekPlan.push({
      day,
      name: `День ${day}`,
      focus: "",
      type: "rest",
      items: defaultItem ? [defaultItem] : [],
    });
  }
  return weekPlan;
}

function buildEmptyWeekPlan() {
  return buildWeekPlanTemplate("Отдых");
}

export function normalizeWeekPlanForSave(weekPlan) {
  const out = [];
  for (let i = 0; i < 7; i += 1) {
    const src = weekPlan?.[i] || {};
    const focus = String(src.focus || "").trim();
    const items = Array.isArray(src.items)
      ? src.items.map((line) => String(line || "").trim()).filter(Boolean)
      : [];
    const lowered = items.join(" ").toLowerCase();
    const hasRestWord = /(выходн|отдых|rest|off)/i.test(lowered);
    const isRest = hasRestWord || items.length === 0;
    out.push({
      day: i + 1,
      name: `День ${i + 1}`,
      focus,
      type: isRest ? "rest" : "train",
      items: items.length ? items : ["Отдых"],
    });
  }
  return out;
}

export function renderPlanEditor(container, weekPlan, onChange) {
  if (!container) return;
  container.innerHTML = "";
  weekPlan.forEach((day, index) => {
    const wrapper = document.createElement("div");
    wrapper.className = "training-day-editor";
    wrapper.dataset.index = String(index);

    const title = document.createElement("div");
    title.className = "muted";
    title.textContent = `День ${index + 1}`;
    wrapper.appendChild(title);

    const focusInput = document.createElement("input");
    focusInput.type = "text";
    focusInput.placeholder = "Фокус (можно пусто)";
    focusInput.value = String(day.focus || "").trim();
    focusInput.addEventListener("input", () => {
      day.focus = focusInput.value.trim();
      if (onChange) onChange(weekPlan);
    });
    wrapper.appendChild(focusInput);

    const itemsArea = document.createElement("textarea");
    itemsArea.placeholder = "По одному на строку. Формат: Название | 3x10 | 60 | 120 (или через /). Кардио: Название | 25 мин";
    itemsArea.value = Array.isArray(day.items) ? day.items.join("\n") : "";
    itemsArea.addEventListener("input", () => {
      day.items = itemsArea.value
        .split("\n")
        .map((line) => line.trim())
        .filter(Boolean);
      if (onChange) onChange(weekPlan);
    });
    wrapper.appendChild(itemsArea);

    container.appendChild(wrapper);
  });
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
  const hasStructuredPlan = state.planStructured && state.planPayload?.week_plan?.length;
  if (accordion) accordion.style.display = hasStructuredPlan ? "" : "none";
  if (result) result.style.display = hasStructuredPlan ? "none" : "";
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

    const focusInput = document.createElement("input");
    focusInput.type = "text";
    focusInput.placeholder = "Фокус (можно пусто)";
    focusInput.value = String(day.focus || "").trim();
    focusInput.dataset.field = "focus";
    wrapper.appendChild(focusInput);

    const itemsArea = document.createElement("textarea");
    itemsArea.placeholder = "По одному на строку. Формат: Название | 3x10 | 60 | 120 (или через /). Кардио: Название | 25 мин";
    itemsArea.value = Array.isArray(day.items) ? day.items.join("\n") : "";
    itemsArea.dataset.field = "items";
    wrapper.appendChild(itemsArea);

    editor.appendChild(wrapper);
  });
}

async function saveTargets(prefix) {
  for (const { field, id, planId } of targetFields) {
    const fieldId = prefix === "plan" ? planId : id;
    const el = document.getElementById(fieldId);
    if (!el) continue;
    const value = Number(el.value || 0);
    await api("/api/targets/set", { field, value });
  }
  toast("Цели обновлены");
  await loadToday();
}

export async function loadPlan() {
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
      const rawPlan = String(state.planText || "").trim();
      if (rawPlan) {
        state.planPayload = { week_plan: buildEmptyWeekPlan(), comment: "" };
        state.planStructured = true;
        renderPlanEditor($("training-editor"), state.planPayload.week_plan);
        setPlanEditMode(true);
        if (trainingResult) trainingResult.style.display = "none";
        const container = $("training-accordion");
        if (container) container.innerHTML = "";
        return;
      }
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

export function initPlanTab() {
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
        const focus = block.querySelector('[data-field="focus"]')?.value.trim() || "";
        const itemsRaw = block.querySelector('[data-field="items"]')?.value || "";
        const items = itemsRaw
          .split("\n")
          .map((line) => line.trim())
          .filter(Boolean);
        if (items.length === 0) {
          toast("Добавь упражнения");
          return;
        }
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
      toast("Пересчитано", planTargetsRefresh);
      await loadToday();
    });
  }
}
