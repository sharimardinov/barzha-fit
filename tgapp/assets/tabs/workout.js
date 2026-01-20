import { $, api, toast, parseNumberInput, formatApiError } from "../core.js";

const defaultRestSec = 120;

let plan = null;
let planIssues = [];
let session = null;
let tickHandle = null;
let refreshPending = false;
let lastSetKey = null;

export function initWorkoutTab() {
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

  const stopBtn = $("workout-stop");
  if (stopBtn) {
    stopBtn.addEventListener("click", async () => {
      try {
        const data = await api("/api/workout/session/stop");
        applySession(data);
      } catch (err) {
        toast(formatApiError(err, "Не удалось выйти из тренировки"));
      }
    });
  }
}

export async function loadWorkout() {
  plan = await fetchPlan();
  const sessionData = await fetchSession();
  session = sessionData?.session || null;
  if (sessionData?.plan) {
    plan = sessionData.plan;
    planIssues = [];
  }
  renderPlanSection();
  renderSession();
}

async function fetchPlan() {
  planIssues = [];
  try {
    const data = await api("/api/workout/plan/get");
    return data.plan || null;
  } catch (err) {
    if (err.message === "workout_plan_invalid") {
      if (Array.isArray(err.data?.issues)) {
        planIssues = err.data.issues;
      }
      return null;
    }
    if (err.message === "workout_plan_not_found") {
      planIssues = ["План тренировок не найден"];
      return null;
    }
    toast(formatApiError(err, "Не удалось загрузить план"));
    return null;
  }
}

async function fetchSession() {
  try {
    const data = await api("/api/workout/session/get");
    return { session: data.session || null, plan: data.plan || null };
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
  renderPlanSection();
}

function renderPlanSection() {
  const list = $("workout-plan-list");
  if (!list) return;
  list.innerHTML = "";

  const errors = $("workout-plan-errors");
  if (errors) {
    errors.innerHTML = "";
    if (planIssues.length > 0) {
      planIssues.forEach((issue) => {
        const item = document.createElement("div");
        item.textContent = issue;
        errors.appendChild(item);
      });
      errors.classList.remove("hidden");
    } else {
      errors.classList.add("hidden");
    }
  }

  const exercises = plan?.exercises || [];
  if (exercises.length === 0) {
    updatePlanEmptyState();
    return;
  }

  exercises.forEach((ex) => list.appendChild(createPlanItem(ex)));
  updatePlanEmptyState();
}

function updatePlanEmptyState() {
  const empty = $("workout-plan-empty");
  const list = $("workout-plan-list");
  if (!empty || !list) return;
  const hasRows = list.querySelectorAll(".workout-plan-item").length > 0;
  empty.classList.toggle("hidden", hasRows || planIssues.length > 0);
}

function createPlanItem(exercise) {
  const item = document.createElement("div");
  item.className = "list-item workout-plan-item";

  const body = document.createElement("div");
  const title = document.createElement("div");
  title.className = "workout-plan-item-title";
  title.textContent = exercise.name || "—";
  body.appendChild(title);

  const meta = document.createElement("div");
  meta.className = "workout-plan-item-meta";
  if (exercise.type === "cardio") {
    meta.textContent = `Длительность: ${formatMinutes(exercise.durationSec)} мин`;
  } else {
    const weightText = exercise.weight ? `${exercise.weight} кг` : "—";
    const restText = formatRest(exercise.restSec || defaultRestSec);
    meta.textContent = `Подходы: ${exercise.sets}x${exercise.reps} · Вес: ${weightText} · Отдых: ${restText}`;
  }
  body.appendChild(meta);

  const tag = document.createElement("div");
  tag.className = "workout-plan-item-tag";
  tag.textContent = exercise.type === "cardio" ? "Кардио" : "Силовое";

  item.appendChild(body);
  item.appendChild(tag);
  return item;
}

function renderSession() {
  const sessionEmpty = $("workout-session-empty");
  const sessionActive = $("workout-session-active");
  const startRow = $("workout-start-row");

  if (!session) {
    if (sessionEmpty) {
      if (planIssues.length > 0) {
        sessionEmpty.textContent = "Проверь формат плана в разделе «План»";
      } else if (!plan || !(plan.exercises || []).length) {
        sessionEmpty.textContent = "Сегодня нет тренировки";
      } else {
        sessionEmpty.textContent = "Нет активной тренировки";
      }
      sessionEmpty.classList.remove("hidden");
    }
    if (sessionActive) sessionActive.classList.add("hidden");
    if (startRow) startRow.classList.toggle("hidden", planIssues.length > 0 || !plan || !(plan.exercises || []).length);
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
      exerciseTarget.textContent = `Длительность: ${formatMinutes(ex.durationSec)} мин`;
    } else {
      const weightText = ex.weight ? `${ex.weight} кг` : "—";
      exerciseTarget.textContent = `Вес: ${weightText} · Повторы: ${ex.reps} · Подходы: ${ex.sets}`;
    }
  }

  if (setLabel) {
    if (!ex || ex.type === "cardio") {
      setLabel.textContent = "";
    } else if (session.phase === "set") {
      if (session.setIndex === 0) {
        setLabel.textContent = "Разминочный подход";
      } else {
        setLabel.textContent = `Подход ${session.setIndex} из ${ex.sets}`;
      }
    } else if (session.phase === "rest") {
      if (session.timerKind === "rest") {
        const finishedSet = Math.max(0, session.setIndex - 1);
        if (finishedSet === 0) {
          setLabel.textContent = "Отдых после разминки";
        } else {
          setLabel.textContent = `Отдых после подхода ${finishedSet} из ${ex.sets}`;
        }
      } else {
        setLabel.textContent = "Отдых между упражнениями";
      }
    } else {
      setLabel.textContent = "";
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
  const stopBtn = $("workout-stop");

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
  if (stopBtn) stopBtn.classList.toggle("hidden", !session);

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
  } catch (_) {
    refreshPending = false;
  }
}

function formatDuration(seconds) {
  const total = Math.max(0, Math.round(seconds));
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

function formatMinutes(seconds) {
  const total = Math.max(0, Math.round(seconds || 0));
  return Math.max(1, Math.round(total / 60));
}

function formatRest(seconds) {
  const total = Math.max(0, Math.round(seconds || 0));
  if (total % 60 === 0) {
    return `${total / 60} мин`;
  }
  return `${total} сек`;
}
