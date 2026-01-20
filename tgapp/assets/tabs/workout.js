import { $, api, toast, setButtonLoading, parseNumberInput, formatApiError } from "../core.js";

const defaultRestSec = 120;

let plan = null;
let session = null;
let tickHandle = null;
let refreshPending = false;
let lastSetKey = null;

export function initWorkoutTab() {
  const addBtn = $("workout-add-exercise");
  if (addBtn) {
    addBtn.addEventListener("click", () => {
      const list = $("workout-plan-list");
      if (!list) return;
      list.appendChild(createExerciseRow());
      updatePlanEmptyState();
    });
  }

  const saveBtn = $("workout-plan-save");
  if (saveBtn) {
    saveBtn.addEventListener("click", async () => {
      const payload = collectPlanFromEditor();
      if (!payload) return;
      setButtonLoading(saveBtn, true, "Сохраняю...");
      try {
        await api("/api/workout/plan/save", { plan: payload });
        plan = payload;
        toast("План тренировки сохранен");
        await loadWorkout();
      } catch (err) {
        toast(formatApiError(err, "Не удалось сохранить план"));
      } finally {
        setButtonLoading(saveBtn, false);
      }
    });
  }

  const startBtn = $("workout-start");
  if (startBtn) {
    startBtn.addEventListener("click", async () => {
      try {
        const data = await api("/api/workout/session/start");
        applySession(data);
      } catch (err) {
        toast(formatApiError(err, "Не удалось начать тренировку"));
      }
    });
  }

  const warmupBtn = $("workout-warmup-done");
  if (warmupBtn) {
    warmupBtn.addEventListener("click", async () => {
      try {
        const data = await api("/api/workout/session/warmup/end");
        applySession(data);
      } catch (err) {
        toast(formatApiError(err, "Не удалось завершить разминку"));
      }
    });
  }

  const setDoneBtn = $("workout-set-done");
  if (setDoneBtn) {
    setDoneBtn.addEventListener("click", async () => {
      if (!session) return;
      const isWarmup = session.setIndex === 0;
      const actualWeight = isWarmup ? 0 : parseNumberInput($("workout-actual-weight")?.value);
      const actualReps = isWarmup ? 0 : Math.round(parseNumberInput($("workout-actual-reps")?.value));
      try {
        const data = await api("/api/workout/session/set/finish", {
          exerciseIndex: session.exerciseIndex,
          setIndex: session.setIndex,
          actualWeight,
          actualReps,
        });
        applySession(data);
      } catch (err) {
        toast(formatApiError(err, "Не удалось сохранить подход"));
      }
    });
  }

  const pauseBtn = $("workout-pause");
  if (pauseBtn) {
    pauseBtn.addEventListener("click", async () => {
      try {
        const data = await api("/api/workout/session/pause");
        applySession(data);
      } catch (err) {
        toast(formatApiError(err, "Не удалось поставить на паузу"));
      }
    });
  }

  const resumeBtn = $("workout-resume");
  if (resumeBtn) {
    resumeBtn.addEventListener("click", async () => {
      try {
        const data = await api("/api/workout/session/resume");
        applySession(data);
      } catch (err) {
        toast(formatApiError(err, "Не удалось продолжить"));
      }
    });
  }
}

export async function loadWorkout() {
  plan = await fetchPlan();
  session = await fetchSession();
  renderPlanEditor();
  renderSession();
}

async function fetchPlan() {
  try {
    const data = await api("/api/workout/plan/get");
    return data.plan || null;
  } catch (err) {
    if (err.message === "workout_plan_not_found") {
      return null;
    }
    toast(formatApiError(err, "Не удалось загрузить план"));
    return null;
  }
}

async function fetchSession() {
  try {
    const data = await api("/api/workout/session/get");
    return data.session || null;
  } catch (err) {
    if (err.message === "workout_session_not_found") {
      return null;
    }
    toast(formatApiError(err, "Не удалось загрузить тренировку"));
    return null;
  }
}

function applySession(data) {
  if (data?.plan) plan = data.plan;
  if (data?.session?.status === "completed") {
    session = null;
    toast("Тренировка завершена");
  } else {
    session = data?.session || null;
  }
  renderSession();
  renderPlanEditor();
}

function renderPlanEditor() {
  const list = $("workout-plan-list");
  if (!list) return;
  list.innerHTML = "";

  const exercises = plan?.exercises || [];
  if (exercises.length === 0) {
    list.appendChild(createExerciseRow());
  } else {
    exercises.forEach((ex) => list.appendChild(createExerciseRow(ex)));
  }

  updatePlanEmptyState();
  setPlanEditingEnabled(!session);
}

function updatePlanEmptyState() {
  const empty = $("workout-plan-empty");
  const list = $("workout-plan-list");
  if (!empty || !list) return;
  const hasRows = list.querySelectorAll(".workout-exercise-row").length > 0;
  empty.classList.toggle("hidden", hasRows);
}

function setPlanEditingEnabled(enabled) {
  const list = $("workout-plan-list");
  const addBtn = $("workout-add-exercise");
  const saveBtn = $("workout-plan-save");
  if (list) {
    list.querySelectorAll("input, select, button").forEach((el) => {
      el.disabled = !enabled;
    });
  }
  if (addBtn) addBtn.disabled = !enabled;
  if (saveBtn) saveBtn.disabled = !enabled;
  const note = $("workout-plan-note");
  if (note) {
    note.textContent = enabled
      ? "Отдых между подходами — 2 мин. Между упражнениями — +1 мин."
      : "План заблокирован — тренировка уже идет.";
  }
}

function createExerciseRow(exercise = {}) {
  const row = document.createElement("div");
  row.className = "workout-exercise-row";
  const type = exercise.type || "strength";
  row.dataset.type = type;

  const durationMin = exercise.durationSec ? Math.round(exercise.durationSec / 60) : "";
  const restValue = exercise.restSec || defaultRestSec;

  row.innerHTML = `
    <div class="row">
      <input type="text" class="workout-exercise-name" placeholder="Название" value="${escapeHtml(exercise.name || "")}">
      <select class="workout-exercise-type">
        <option value="strength" ${type === "strength" ? "selected" : ""}>Силовое</option>
        <option value="cardio" ${type === "cardio" ? "selected" : ""}>Кардио</option>
      </select>
      <button class="btn btn-outline workout-exercise-remove" type="button">Удалить</button>
    </div>
    <div class="workout-exercise-fields">
      <label class="strength-only">Вес (кг)
        <input type="number" step="0.5" class="workout-exercise-weight" placeholder="Например 60" value="${toInputValue(exercise.weight)}">
      </label>
      <label class="strength-only">Повторы
        <input type="number" class="workout-exercise-reps" placeholder="Например 10" value="${toInputValue(exercise.reps)}">
      </label>
      <label class="strength-only">Подходы
        <input type="number" class="workout-exercise-sets" placeholder="Например 3" value="${toInputValue(exercise.sets)}">
      </label>
      <label class="strength-only">Отдых (сек)
        <input type="number" class="workout-exercise-rest" placeholder="${defaultRestSec}" value="${toInputValue(restValue)}">
      </label>
      <label class="cardio-only">Длительность (мин)
        <input type="number" class="workout-exercise-duration" placeholder="Например 25" value="${toInputValue(durationMin)}">
      </label>
    </div>
  `;

  const typeSelect = row.querySelector(".workout-exercise-type");
  if (typeSelect) {
    typeSelect.addEventListener("change", (e) => {
      row.dataset.type = e.target.value || "strength";
    });
  }

  const removeBtn = row.querySelector(".workout-exercise-remove");
  if (removeBtn) {
    removeBtn.addEventListener("click", () => {
      row.remove();
      updatePlanEmptyState();
    });
  }

  return row;
}

function collectPlanFromEditor() {
  const list = $("workout-plan-list");
  if (!list) return null;
  const rows = Array.from(list.querySelectorAll(".workout-exercise-row"));
  if (rows.length === 0) {
    toast("Добавь хотя бы одно упражнение");
    return null;
  }

  const exercises = [];
  for (let i = 0; i < rows.length; i += 1) {
    const row = rows[i];
    const name = row.querySelector(".workout-exercise-name")?.value.trim();
    if (!name) {
      toast(`Упражнение ${i + 1}: укажи название`);
      return null;
    }
    const type = row.querySelector(".workout-exercise-type")?.value || "strength";
    if (type === "cardio") {
      const durationMin = parseNumberInput(row.querySelector(".workout-exercise-duration")?.value);
      if (durationMin <= 0) {
        toast(`Упражнение ${i + 1}: укажи длительность`);
        return null;
      }
      exercises.push({
        name,
        type,
        durationSec: Math.round(durationMin * 60),
        restSec: defaultRestSec,
      });
      continue;
    }

    const weight = parseNumberInput(row.querySelector(".workout-exercise-weight")?.value);
    const reps = Math.round(parseNumberInput(row.querySelector(".workout-exercise-reps")?.value));
    const sets = Math.round(parseNumberInput(row.querySelector(".workout-exercise-sets")?.value));
    const restRaw = parseNumberInput(row.querySelector(".workout-exercise-rest")?.value);
    const restSec = restRaw > 0 ? Math.round(restRaw) : defaultRestSec;

    if (reps <= 0 || sets <= 0) {
      toast(`Упражнение ${i + 1}: укажи подходы и повторы`);
      return null;
    }

    exercises.push({
      name,
      type,
      weight,
      reps,
      sets,
      restSec,
    });
  }

  return {
    defaultRestSec,
    exercises,
  };
}

function renderSession() {
  const sessionEmpty = $("workout-session-empty");
  const sessionActive = $("workout-session-active");
  const startRow = $("workout-start-row");

  if (!plan || !(plan.exercises || []).length) {
    if (sessionEmpty) {
      sessionEmpty.textContent = "Сначала собери план тренировки";
      sessionEmpty.classList.remove("hidden");
    }
    if (sessionActive) sessionActive.classList.add("hidden");
    if (startRow) startRow.classList.add("hidden");
    clearTick();
    return;
  }

  if (!session) {
    if (sessionEmpty) {
      sessionEmpty.textContent = "Нет активной тренировки";
      sessionEmpty.classList.remove("hidden");
    }
    if (sessionActive) sessionActive.classList.add("hidden");
    if (startRow) startRow.classList.remove("hidden");
    clearTick();
    return;
  }

  if (sessionEmpty) sessionEmpty.classList.add("hidden");
  if (sessionActive) sessionActive.classList.remove("hidden");
  if (startRow) startRow.classList.add("hidden");

  const ex = plan.exercises[session.exerciseIndex];
  const exerciseName = $("workout-exercise-name");
  const exerciseTarget = $("workout-exercise-target");
  const setLabel = $("workout-set-label");
  if (exerciseName) exerciseName.textContent = ex ? ex.name : "—";

  if (exerciseTarget) {
    if (!ex) {
      exerciseTarget.textContent = "—";
    } else if (ex.type === "cardio") {
      const mins = Math.round((ex.durationSec || 0) / 60);
      exerciseTarget.textContent = `Длительность: ${mins} мин`;
    } else {
      const weightText = ex.weight ? `${ex.weight} кг` : "—";
      exerciseTarget.textContent = `Вес: ${weightText} · Повторы: ${ex.reps} · Подходы: ${ex.sets}`;
    }
  }

  if (setLabel) {
    if (!ex || ex.type === "cardio") {
      setLabel.textContent = "";
    } else if (session.setIndex === 0) {
      setLabel.textContent = "Разминочный подход";
    } else {
      setLabel.textContent = `Подход ${session.setIndex} из ${ex.sets}`;
    }
  }

  updatePhaseControls();
  startTick();
}

function updatePhaseControls() {
  const phaseLabel = $("workout-phase-label");
  const timerBlock = $("workout-timer-block");
  const timerLabel = $("workout-timer-label");
  const inputs = $("workout-inputs");
  const warmupBtn = $("workout-warmup-done");
  const setDoneBtn = $("workout-set-done");
  const pauseBtn = $("workout-pause");
  const resumeBtn = $("workout-resume");

  const phase = session?.phase;
  const status = session?.status;

  if (phaseLabel) {
    if (phase === "warmup") phaseLabel.textContent = "Разминка";
    else if (phase === "rest") phaseLabel.textContent = session?.timerKind === "between" ? "Отдых между упражнениями" : "Отдых между подходами";
    else if (phase === "set") phaseLabel.textContent = "Подход";
    else if (phase === "cardio") phaseLabel.textContent = "Кардио";
    else phaseLabel.textContent = "—";
  }

  if (timerBlock) {
    const showTimer = phase === "rest" || phase === "cardio";
    timerBlock.classList.toggle("hidden", !showTimer);
    if (timerLabel) {
      if (phase === "cardio") timerLabel.textContent = "Осталось";
      else timerLabel.textContent = session?.timerKind === "between" ? "Отдых между упражнениями" : "Отдых";
    }
  }

  if (inputs) {
    const showInputs = phase === "set" && session?.setIndex > 0;
    inputs.classList.toggle("hidden", !showInputs);
  }

  if (warmupBtn) warmupBtn.classList.toggle("hidden", phase !== "warmup");
  if (setDoneBtn) setDoneBtn.classList.toggle("hidden", phase !== "set");

  if (pauseBtn) pauseBtn.classList.toggle("hidden", status !== "in_progress");
  if (resumeBtn) resumeBtn.classList.toggle("hidden", status !== "paused");

  if (session?.setIndex > 0 && phase === "set") {
    const ex = plan?.exercises?.[session.exerciseIndex];
    if (ex && ex.type !== "cardio") {
      const weightInput = $("workout-actual-weight");
      const repsInput = $("workout-actual-reps");
      const key = `${session.exerciseIndex}:${session.setIndex}`;
      if (lastSetKey !== key) {
        if (weightInput) weightInput.value = ex.weight || "";
        if (repsInput) repsInput.value = ex.reps || "";
        lastSetKey = key;
      }
    }
  } else {
    lastSetKey = null;
  }
}

function startTick() {
  clearTick();
  if (!session) return;
  updateTimers();
  tickHandle = setInterval(updateTimers, 1000);
}

function clearTick() {
  if (tickHandle) {
    clearInterval(tickHandle);
    tickHandle = null;
  }
  refreshPending = false;
}

function updateTimers() {
  if (!session) return;
  updateTotalTime();

  const timerValue = $("workout-timer-value");
  if (timerValue) {
    if (session.timerDurationSec > 0) {
      const remaining = getRemainingSeconds();
      timerValue.textContent = formatDuration(remaining);
      if (remaining <= 0) {
        triggerRefresh();
      }
    } else {
      timerValue.textContent = "00:00";
    }
  }
}

function updateTotalTime() {
  const totalEl = $("workout-total-time");
  if (!totalEl || !session?.startedAt) return;
  const now = Date.now();
  const startedAt = new Date(session.startedAt).getTime();
  let elapsed = Math.floor((now - startedAt) / 1000) - (session.pausedTotalSec || 0);
  if (session.status === "paused" && session.pausedAt) {
    const pausedAt = new Date(session.pausedAt).getTime();
    elapsed -= Math.floor((now - pausedAt) / 1000);
  }
  if (elapsed < 0) elapsed = 0;
  totalEl.textContent = formatDuration(elapsed);
}

function getRemainingSeconds() {
  if (!session) return 0;
  if (session.status === "paused") {
    return Math.max(0, session.timerDurationSec || 0);
  }
  if (!session.timerStartedAt || !session.timerDurationSec) return 0;
  const end = new Date(session.timerStartedAt).getTime() + session.timerDurationSec * 1000;
  const remaining = Math.ceil((end - Date.now()) / 1000);
  return Math.max(0, remaining);
}

function triggerRefresh() {
  if (refreshPending) return;
  if (!session || session.status !== "in_progress") return;
  if (session.phase !== "rest" && session.phase !== "cardio") return;
  refreshPending = true;
  refreshSession();
}

async function refreshSession() {
  try {
    const data = await api("/api/workout/session/get");
    refreshPending = false;
    applySession(data);
  } catch (err) {
    refreshPending = false;
  }
}

function formatDuration(seconds) {
  const total = Math.max(0, Math.round(seconds));
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

function toInputValue(value) {
  if (value === undefined || value === null) return "";
  if (Number.isNaN(value)) return "";
  return value === 0 ? "" : value;
}

function escapeHtml(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;");
}
