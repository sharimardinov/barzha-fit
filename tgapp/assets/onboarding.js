import {
  $,
  api,
  state,
  toast,
  buildWheelValues,
  createWheel,
  normalizeGoalType,
  buildTrainingGoalText,
  setActiveScreen,
  setActiveTab,
} from "./core.js";
import { loadToday } from "./tabs/today.js";
import { loadPlan, normalizeWeekPlanForSave, renderPlanEditor } from "./tabs/plan.js";
import { loadTargets } from "./core.js";
import { isProfileComplete, isTrainingComplete, saveProfileFlow } from "./tabs/profile.js";

function setOnboardingActive(active) {
  state.onboarding = active;
  document.body.classList.toggle("onboarding", active);
}

export async function ensureOnboarding() {
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
  let hasPlan = false;
  try {
    const plan = await api("/api/plan/get");
    hasPlan = typeof plan.text === "string" && plan.text.trim().length > 0;
  } catch (_) {
    hasPlan = false;
  }

  if (!isProfileComplete(profile) || (!isTrainingComplete(training) && !hasPlan)) {
    setOnboardingActive(true);
    setActiveScreen("onboarding");
    return false;
  }

  setOnboardingActive(false);
  return true;
}

export async function initOnboardingWizard() {
  const progressEl = $("onboarding-progress");
  const titleEl = $("onboarding-title");
  const descEl = $("onboarding-desc");
  const bodyEl = $("onboarding-body");
  const helpEl = $("onboarding-help");
  const backBtn = $("onboarding-back");
  const nextBtn = $("onboarding-next");

  let injuryOptions = [];
  try {
    injuryOptions = await api("/api/training/injuries");
  } catch (_) {
    injuryOptions = [];
  }
  const injuriesList = Array.isArray(injuryOptions) && injuryOptions.length
    ? injuryOptions.map((item) => ({
      value: item.code,
      label: item.label,
    }))
    : [];

  const steps = [
    {
      id: "sex",
      title: "Твой пол",
      type: "sex-buttons",
      options: [
        { value: "m", label: "М" },
        { value: "f", label: "Ж" },
      ],
      help: "Нужно для корректных норм и нагрузки",
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
      help: "Возраст влияет на восстановление и объём",
      required: true,
    },
    {
      id: "height",
      title: "Твой рост",
      type: "input",
      inputType: "number",
      min: 140,
      max: 210,
      placeholder: "Например: 175",
      help: "Для расчёта калорий и целей",
      required: true,
    },
    {
      id: "weight",
      title: "Твой вес",
      type: "wheel-horizontal",
      min: 30,
      max: 160,
      step: 0.5,
      defaultValue: 75,
      unit: "кг",
      help: "Для расчёта нагрузки и калорий",
      required: true,
    },
    {
      id: "trainingStage",
      title: "Твоя фаза",
      type: "goal-combo",
      options: [
        { value: "core", label: "CORE" },
        { value: "flow", label: "FLOW" },
        { value: "peak", label: "PEAK" },
      ],
      defaultValue: "core",
      help: "Выбери свой текущий уровень подготовки",
      required: true,
      showNotes: false,
      wide: true,
      big: true,
    },
    {
      id: "bodyfat",
      title: "Процент жира",
      type: "input",
      inputType: "number",
      min: 1,
      max: 100,
      placeholder: "Например: 18",
      help: "Для точных целей по весу и форме",
      required: true,
    },
    {
      id: "planMode",
      title: "Как получим план?",
      type: "options",
      options: [
        { value: "ai", label: "Сгенерировать", disabled: true },
        { value: "manual", label: "Вставить вручную" },
      ],
      help: "Генерация тренировок временно недоступна. Можно вставить план вручную.",
      required: true,
    },        
    {
      id: "injuries",
      title: "Слабые места",
      type: "multi-options",
      options: injuriesList,
      help: "Требуется для корректной генерации безопасной программы",
      required: false,
      when: (d) => (d.planMode || "manual") !== "manual",
    },
    {
      id: "goalType",
      title: "Твоя цель",
      type: "goal-combo",
      options: [
        { value: "cut", label: "CUT" },
        { value: "balance", label: "BALANCE" },
        { value: "bulk", label: "BULK" },
      ],
      defaultValue: "balance",
      notesId: "goalNotes",
      help: "Выбери к чему стремишься",
      required: true,
      showNotes: false,
      wide: true,
      big: true,
    },
    {
      id: "trainingsPerWeek",
      title: "Тренировок в неделю",
      type: "input",
      inputType: "number",
      min: 2,
      max: 6,
      placeholder: "Например: 4",
      help: "Формируем недельную структуру.",
      required: true,
      when: (d) => (d.planMode || "manual") !== "manual",
    },
    {
      id: "planWeek",
      title: "Заполни план на неделю",
      type: "plan-editor",
      help: "Нужно заполнить все 7 дней. Если отдых — просто напиши 'Отдых'.",
      required: true,
      when: (d) => d.planMode === "manual",
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

  const getVisibleSteps = () => steps.filter((s) => !s.when || s.when(data));

  const renderStep = () => {
    const visibleSteps = getVisibleSteps();
    if (stepIndex < 0) stepIndex = 0;
    if (stepIndex >= visibleSteps.length) stepIndex = visibleSteps.length - 1;
    const step = visibleSteps[stepIndex];
    progressEl.textContent = `Шаг ${stepIndex + 1} из ${visibleSteps.length}`;
    titleEl.textContent = step.title;
    descEl.textContent = step.type === "wheel" || step.type === "wheel-horizontal"
      ? `Прокрути колесо${step.unit ? ` (${step.unit})` : ""}`
      : step.type === "options" || step.type === "goal-combo" || step.type === "sex-buttons"
        ? "Выбери один вариант"
        : step.type === "multi-options"
          ? "Можно выбрать несколько"
          : step.type === "plan-editor"
            ? "Заполни план на 7 дней"
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
        if (step.id === "planMode") className += " accent-fill";
        btn.className = className;
        btn.textContent = opt.label;
        btn.classList.toggle("active", data[step.id] === opt.value);

          if (opt.disabled) {
          btn.classList.add("option-disabled");
          btn.disabled = true;
          } else {
          btn.addEventListener("click", () => {
            data[step.id] = opt.value;
            renderStep();
          });
          }
        });
        list.appendChild(btn);
      };
      bodyEl.appendChild(list);
    }

    if (step.type === "multi-options") {
      const list = document.createElement("div");
      list.className = "injury-options";
    
      const selected = Array.isArray(data[step.id]) ? data[step.id] : [];
    
      step.options.forEach((opt, idx) => {
        const row = document.createElement("div");
        row.className = "injury-row";
    
        const text = document.createElement("div");
        text.className = "injury-label";
        text.textContent = opt.label;
    
        const pinLabel = document.createElement("label");
        pinLabel.className = "pinbox";
    
        const input = document.createElement("input");
        input.type = "checkbox";
        input.checked = selected.includes(opt.value);
    
        const pinWrap = document.createElement("div");
        pinWrap.innerHTML = PIN_SVG;
    
        input.addEventListener("change", () => {
          const next = Array.isArray(data[step.id]) ? [...data[step.id]] : [];
          const i = next.indexOf(opt.value);
    
          if (input.checked) {
            if (i < 0) next.push(opt.value);
          } else {
            if (i >= 0) next.splice(i, 1);
          }
    
          data[step.id] = next;
        });
    
        // делаем кликабельной всю строку (кроме того что уже label)
        row.addEventListener("click", (e) => {
          // если кликнули по самому input/label — не дублируем
          if (e.target instanceof HTMLElement && e.target.closest(".pinbox")) return;
          input.checked = !input.checked;
          input.dispatchEvent(new Event("change", { bubbles: true }));
        });
    
        pinLabel.appendChild(input);
        pinLabel.appendChild(pinWrap.firstElementChild);
    
        row.appendChild(pinLabel);
        row.appendChild(text);
    
        list.appendChild(row);
      });
    
      bodyEl.appendChild(list);
    }
    

    if (step.type === "sex-buttons") {
      const list = document.createElement("div");
      list.className = "option-list sex-options";
      step.options.forEach((opt) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "option-card sex-option";
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

    if (step.type === "goal-combo") {
      if (data[step.id] === undefined && step.defaultValue !== undefined) {
        data[step.id] = step.defaultValue;
      }
      const container = document.createElement("div");
      container.className = "tabs-container";
      const tabs = document.createElement("div");
      const tabsClasses = ["tabs"];
      if (step.layout === "vertical") tabsClasses.push("vertical");
      if (step.wide) tabsClasses.push("wide");
      if (step.big) tabsClasses.push("big");
      if (step.id === "trainingStage") tabsClasses.push("accent-text");
      tabs.className = tabsClasses.join(" ");
      step.options.forEach((opt, index) => {
        const input = document.createElement("input");
        input.type = "radio";
        input.name = "onboarding-goal-tabs";
        input.id = `onboarding-goal-${index}`;
        input.value = opt.value;
        input.checked = data[step.id] === opt.value;
        const label = document.createElement("label");
        label.className = "tab";
        label.setAttribute("for", input.id);
        if (step.id === "trainingStage") {
          label.classList.add("tab-with-icon");
          label.style.setProperty("--icon", `url(images/${opt.value}.svg)`);
          const icon = document.createElement("span");
          icon.className = "tab-icon";
          const text = document.createElement("span");
          text.className = "tab-text";
          text.textContent = opt.label;
          label.appendChild(icon);
          label.appendChild(text);
        } else {
          label.textContent = opt.label;
        }
        input.addEventListener("change", () => {
          data[step.id] = opt.value;
        });
        tabs.appendChild(input);
        tabs.appendChild(label);
      });
      const glider = document.createElement("span");
      glider.className = "glider";
      tabs.appendChild(glider);
      const updateGlider = () => {
        const idx = step.options.findIndex((opt) => opt.value === data[step.id]);
        const index = idx >= 0 ? idx : 0;
        if (step.wide) {
          glider.style.transform = "none";
          glider.style.left = `calc(${index} * (100% / ${step.options.length}))`;
        } else if (step.layout === "vertical") {
          glider.style.transform = `translateY(${index * 100}%)`;
        } else {
          glider.style.transform = `translateX(${index * 100}%)`;
        }
      };
      updateGlider();
      tabs.addEventListener("change", updateGlider);
      container.appendChild(tabs);
      bodyEl.appendChild(container);

      if (step.showNotes !== false && step.notesId) {
        const field = document.createElement("textarea");
        field.placeholder = step.notesPlaceholder || "";
        field.value = data[step.notesId] || "";
        field.addEventListener("input", () => {
          data[step.notesId] = field.value;
        });
        bodyEl.appendChild(field);
      }
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

    if (step.type === "plan-editor") {
      if (!Array.isArray(data[step.id]) || data[step.id].length !== 7) {
        data[step.id] = Array.from({ length: 7 }).map((_, idx) => ({
          day: idx + 1,
          name: `День ${idx + 1}`,
          focus: "",
          type: "rest",
          items: [],
        }));
      }
      const editor = document.createElement("div");
      editor.className = "training-editor active";
      renderPlanEditor(editor, data[step.id], (updated) => {
        data[step.id] = updated;
      });
      bodyEl.appendChild(editor);
    }

    backBtn.style.visibility = stepIndex === 0 ? "hidden" : "visible";
    nextBtn.textContent = stepIndex === visibleSteps.length - 1 ? "Готово" : "Далее";
  };

  const validateStep = (step) => {
    const value = data[step.id];
    if (!step.required) return true;
    if (step.type === "options" && (value === undefined || value === null || value === "")) {
      toast("Выбери вариант");
      return false;
    }
    if (step.type === "goal-combo" && (value === undefined || value === null || value === "")) {
      toast("Выбери цель");
      return false;
    }
    if (step.type === "sex-buttons" && (value === undefined || value === null || value === "")) {
      toast("Выбери вариант");
      return false;
    }
    if (step.type === "multi-options" && step.required) {
      const list = Array.isArray(value) ? value : [];
      if (list.length === 0) {
        toast("Выбери хотя бы один вариант");
        return false;
      }
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
        data[step.id] = raw;
      }
    }
    if (step.type === "textarea" && step.required && String(value || "").trim() === "") {
      toast("Заполни поле");
      return false;
    }
    if (step.type === "plan-editor") {
      const weekPlan = Array.isArray(value) ? value : [];
      if (weekPlan.length !== 7) {
        toast("Заполни все 7 дней (можно писать отдых)");
        return false;
      }
      for (const day of weekPlan) {
        const items = Array.isArray(day.items)
          ? day.items.map((line) => String(line || "").trim()).filter(Boolean)
          : [];
        if (items.length === 0) {
          toast("Заполни все 7 дней (можно писать отдых)");
          return false;
        }
      }
      return true;
    }
    return true;
  };

  const submitOnboarding = async () => {
    const stageMap = {
      core: 0,
      flow: 3,
      peak: 6,
    };
    const trainingYears = stageMap[data.trainingStage] ?? 0;
    const payload = {
      sex: data.sex,
      age: Number(data.age || 0),
      height_cm: Number(data.height || 0),
      weight_kg: Number(data.weight || 0),
      training_years: trainingYears,
      bodyfat_pct: Number(data.bodyfat || 0),
      goal: normalizeGoalType(data.goalType || ""),
    };
    const trainingGoal = buildTrainingGoalText(data.goalType, data.goalNotes);
    const injuriesList = Array.isArray(data.injuries)
      ? data.injuries
      : String(data.injuries || "")
        .split(/[,;]+/)
        .map((item) => item.trim())
        .filter(Boolean);
    const injuries = injuriesList.join(", ");
    const trainingPayload = {
      bench_kg: 0,
      pullups: 0,
      run_km: 0,
      injuries,
      goal: trainingGoal,
      pharma: false,
      trainings_per_week: Number(data.trainingsPerWeek || 0),
      wishes: "",
    };
    const planMode = data.planMode === "manual" ? "manual" : "ai";
    const planWeek = Array.isArray(data.planWeek) ? data.planWeek : null;
    const normalizedPlan = planMode === "manual"
      ? JSON.stringify({ week_plan: normalizeWeekPlanForSave(planWeek), comment: "" })
      : "";
    const trainingInput = planMode === "ai"
      ? {
        fitness_level: data.trainingStage,
        goal: data.goalType,
        days_per_week: Number(data.trainingsPerWeek || 0),
        injuries: injuriesList,
      }
      : null;
    await saveProfileFlow(payload, trainingPayload, nextBtn, {
      planMode,
      planText: normalizedPlan,
      trainingInput,
    });
  };

  backBtn.addEventListener("click", () => {
    if (stepIndex === 0) return;
    stepIndex -= 1;
    renderStep();
  });

  nextBtn.addEventListener("click", async () => {
    const visibleSteps = getVisibleSteps();
    const step = visibleSteps[stepIndex];
    if (!validateStep(step)) return;
    if (stepIndex === visibleSteps.length - 1) {
      await submitOnboarding();
      return;
    }
    stepIndex += 1;
    renderStep();
  });

  renderStep();

  const onboardingGoToday = $("onboarding-go-today");
  if (onboardingGoToday) {
    onboardingGoToday.addEventListener("click", async () => {
      setOnboardingActive(false);
      setActiveTab("today");
      await loadToday();
      await loadPlan();
      await loadTargets();
    });
  }

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Enter") return;
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;
    if (target.tagName === "TEXTAREA") return;
    if (nextBtn && !nextBtn.disabled) {
      event.preventDefault();
      nextBtn.click();
    }
  });


const PIN_SVG = `
<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 75 100" class="pin">
  <line stroke-width="12" stroke="black" y2="100" x2="37" y1="64" x1="37"></line>
  <path stroke-width="10" stroke="black"
    d="M16.5 36V4.5H58.5V36V53.75V54.9752L59.1862 55.9903L66.9674 67.5H8.03256L15.8138 55.9903L16.5 54.9752V53.75V36Z">
  </path>
</svg>`;

