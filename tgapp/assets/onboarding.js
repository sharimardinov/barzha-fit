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
      help: "Требуется для корректных норм и нагрузки",
      required: true,
    },
    {
      id: "bodyMetrics",
      title: "Твои параметры",
      type: "range-group",
      fields: [
        {
          id: "age",
          label: "Выбери возраст",
          min: 14,
          max: 80,
          step: 1,
          unit: "лет",
          defaultValue: 28,
        },
        {
          id: "weight",
          label: "Выбери вес",
          min: 30,
          max: 300,
          step: 0.5,
          unit: "кг",
          defaultValue: 75,
        },
        {
          id: "height",
          label: "Выбери рост",
          min: 100,
          max: 250,
          step: 1,
          unit: "м",
          display: "meters",
          defaultValue: 170,
        },
      ],
      help: "Эти данные нужны для расчёта норм и целей.",
      required: true,
    },
    {
      id: "trainingStage",
      title: "Уровень подготовки",
      type: "goal-combo",
      options: [
        { value: "core", label: "CORE" },
        { value: "flow", label: "FLOW" },
        { value: "peak", label: "PEAK" },
      ],
      defaultValue: "core",
      help: "Выбери свой текущий уровень",
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
      help: "Для точных целей по весу и форме.",
      required: true,
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
      notesPlaceholder: "Например: подсушиться, больше спины, меньше кардио",
      help: "Выбери цель.",
      required: true,
      showNotes: false,
      wide: true,
      big: true,
    },
    {
      id: "planMode",
      title: "Как получим план?",
      type: "options",
      options: [
        { value: "ai", label: "Сгенерировать", disabled: true },
        { value: "manual", label: "Вставить вручную" },
      ],
      help: "Соберём данные и сгенерируем план.",
      required: true,
    },
    {
      id: "planInputMode",
      title: "Как удобнее заполнить план?",
      type: "options",
      options: [
        { value: "simple", label: "По дням (простые поля)" },
        { value: "advanced", label: "Как сейчас (структурно)" },
      ],
      defaultValue: "simple",
      help: "Можно выбрать удобный способ ввода.",
      required: true,
      when: (d) => d.planMode === "manual",
    },
    {
      id: "injuries",
      title: "Травмы и ограничения",
      type: "multi-options",
      options: injuriesList,
      help: "Можно выбрать несколько или пропустить.",
      required: false,
      when: (d) => (d.planMode || "manual") !== "manual",
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
      : step.type === "range-group"
        ? "Передвинь ползунки"
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
      if (data[step.id] === undefined && step.defaultValue !== undefined) {
        data[step.id] = step.defaultValue;
      }
      const list = document.createElement("div");
      list.className = "option-list";
      step.options.forEach((opt) => {
        const btn = document.createElement("button");
        btn.type = "button";
        let className = "option-card";
        if (step.id === "sex" && opt.value === "m") className += " option-male";
        if (step.id === "sex" && opt.value === "f") className += " option-female";
        if (step.id === "planMode" || step.id === "planInputMode") className += " accent-fill";
        btn.className = className;
        btn.textContent = opt.label;
        btn.classList.toggle("active", data[step.id] === opt.value);
        if (opt.disabled) {
          btn.disabled = true;
          btn.classList.add("option-disabled");
        } else {
          btn.addEventListener("click", () => {
            data[step.id] = opt.value;
            renderStep();
          });
        }
        list.appendChild(btn);
      });
      bodyEl.appendChild(list);
    }

    if (step.type === "multi-options") {
      const list = document.createElement("div");
      list.className = "option-list";
      const selected = Array.isArray(data[step.id]) ? data[step.id] : [];
      step.options.forEach((opt) => {
        const btn = document.createElement("button");
        btn.type = "button";
        btn.className = "option-card";
        btn.textContent = opt.label;
        btn.classList.toggle("active", selected.includes(opt.value));
        btn.addEventListener("click", () => {
          const next = Array.isArray(data[step.id]) ? [...data[step.id]] : [];
          const idx = next.indexOf(opt.value);
          if (idx >= 0) {
            next.splice(idx, 1);
          } else {
            next.push(opt.value);
          }
          data[step.id] = next;
          btn.classList.toggle("active", next.includes(opt.value));
        });
        list.appendChild(btn);
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

    if (step.type === "range-group") {
      const group = document.createElement("div");
      group.className = "range-group";

      const formatRangeValue = (field, value) => {
        if (!Number.isFinite(value)) return "—";
        if (field.display === "meters") {
          const meters = value / 100;
          return meters.toLocaleString("ru-RU", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
        }
        if (field.step && Math.abs(field.step % 1) > 0) {
          return Number(value).toLocaleString("ru-RU", { minimumFractionDigits: 1, maximumFractionDigits: 1 });
        }
        return Math.round(value).toLocaleString("ru-RU");
      };

      const updateSliderTrack = (slider, value, min, max) => {
        if (max <= min) return;
        const pct = Math.max(0, Math.min(100, ((value - min) / (max - min)) * 100));
        slider.style.background = `linear-gradient(90deg, var(--accent) 0%, var(--accent) ${pct}%, #d6d7db ${pct}%, #d6d7db 100%)`;
      };

      step.fields.forEach((field) => {
        const min = field.min ?? 0;
        const max = field.max ?? 100;
        const stepValue = field.step ?? 1;
        if (data[field.id] === undefined || data[field.id] === null || data[field.id] === "") {
          data[field.id] = field.defaultValue ?? min;
        }
        const card = document.createElement("div");
        card.className = "range-card";

        const header = document.createElement("div");
        header.className = "range-header";

        const label = document.createElement("div");
        label.className = "range-title";
        label.textContent = field.label || "";

        const valueBox = document.createElement("div");
        valueBox.className = "range-value";
        const number = document.createElement("span");
        number.className = "range-number";
        number.textContent = formatRangeValue(field, Number(data[field.id]));
        const unit = document.createElement("span");
        unit.className = "range-unit";
        unit.textContent = field.unit || "";
        valueBox.appendChild(number);
        valueBox.appendChild(unit);

        header.appendChild(label);
        header.appendChild(valueBox);

        const slider = document.createElement("input");
        slider.type = "range";
        slider.min = String(min);
        slider.max = String(max);
        slider.step = String(stepValue);
        slider.value = String(data[field.id]);
        slider.className = "range-slider";
        updateSliderTrack(slider, Number(slider.value), min, max);
        slider.addEventListener("input", () => {
          const raw = Number(slider.value);
          data[field.id] = raw;
          number.textContent = formatRangeValue(field, raw);
          updateSliderTrack(slider, raw, min, max);
        });

        card.appendChild(header);
        card.appendChild(slider);
        group.appendChild(card);
      });

      bodyEl.appendChild(group);
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
      const isSimplePlan = (data.planInputMode || "simple") === "simple";
      if (isSimplePlan) {
        helpEl.textContent = "Заполни упражнения по отдельным полям. Для дня отдыха включи переключатель.";
      }
      const editor = document.createElement("div");
      editor.className = `training-editor active${isSimplePlan ? " simple" : ""}`;
      if (isSimplePlan) {
        const parseItemLine = (line) => {
          const raw = String(line || "").trim();
          if (!raw) return { type: "strength", name: "", sets: "", reps: "", weight: "", rest: "", duration: "" };
          const isCardio = /^(кардио|cardio)\b/i.test(raw);
          if (isCardio) {
            let text = raw.replace(/^(кардио|cardio)\s*[:/|-]?\s*/i, "").trim();
            let name = "";
            let duration = "";
            if (text.includes("/")) {
              const [left, right] = text.split("/").map((p) => p.trim());
              name = left || "";
              const m = (right || "").match(/(\d+(\.\d+)?)/);
              duration = m ? m[1] : "";
            } else {
              const parts = text.split("|").map((p) => p.trim()).filter(Boolean);
              name = parts[0] || "";
              const m = (parts[1] || "").match(/(\d+(\.\d+)?)/);
              duration = m ? m[1] : "";
            }
            return { type: "cardio", name, duration };
          }
          const parts = raw.split("|").map((p) => p.trim()).filter(Boolean);
          const name = parts[0] || "";
          let sets = "";
          let reps = "";
          if (parts[1]) {
            const match = parts[1].match(/(\d+)\s*[xх]\s*(\d+)/i);
            if (match) {
              sets = match[1] || "";
              reps = match[2] || "";
            } else {
              sets = parts[1] || "";
            }
          }
          const weight = parts[2] || "";
          const rest = parts[3] || "";
          return { type: "strength", name, sets, reps, weight, rest };
        };

        const isRestDay = (day) => {
          if (day.simpleIsRest !== undefined) return day.simpleIsRest;
          const items = Array.isArray(day.items) ? day.items : [];
          if (items.length !== 1) return false;
          const text = String(items[0] || "").trim().toLowerCase();
          return /(отдых|rest|off)/i.test(text);
        };

        const syncItems = (day) => {
          if (day.simpleIsRest) {
            day.items = ["Отдых"];
            return;
          }
          const items = Array.isArray(day.simpleItems) ? day.simpleItems : [];
          const lines = items
            .map((item) => {
              const type = item.type === "cardio" ? "cardio" : "strength";
              const name = String(item.name || "").trim();
              if (type === "cardio") {
                const duration = String(item.duration || "").trim();
                const title = name || "Кардио";
                const parts = [title];
                if (duration) parts.push(`${duration} мин`);
                return `Кардио: ${parts.join(" | ")}`.trim();
              }
              if (!name) return "";
              const sets = String(item.sets || "").trim();
              const reps = String(item.reps || "").trim();
              const weight = String(item.weight || "").trim();
              const rest = String(item.rest || "").trim();
              const parts = [name];
              if (sets || reps) {
                const sr = sets && reps ? `${sets}x${reps}` : `${sets}${reps ? `x${reps}` : ""}`;
                parts.push(sr);
              }
              if (weight) parts.push(weight);
              if (rest) parts.push(rest);
              return parts.join(" | ");
            })
            .filter(Boolean);
          day.items = lines;
        };

        data[step.id].forEach((day, index) => {
          const wrapper = document.createElement("div");
          wrapper.className = "training-day-editor";
          wrapper.dataset.index = String(index);

          const title = document.createElement("button");
          title.type = "button";
          title.className = "training-day-toggle";
          title.textContent = `День ${index + 1}`;
          wrapper.appendChild(title);

          const content = document.createElement("div");
          content.className = "training-day-content";
          wrapper.appendChild(content);

          if (!Array.isArray(day.simpleItems)) {
            const items = Array.isArray(day.items) ? day.items : [];
            day.simpleItems = items.length ? items.map(parseItemLine) : [];
          }
          day.simpleIsRest = isRestDay(day);
          if (!day.simpleIsRest && day.simpleItems.length === 0) {
            day.simpleItems.push({ name: "", sets: "", reps: "", weight: "", rest: "" });
          }

          const restRow = document.createElement("label");
          restRow.className = "container";
          const restToggle = document.createElement("input");
          restToggle.type = "checkbox";
          restToggle.checked = Boolean(day.simpleIsRest);
          const restCheckmark = document.createElement("div");
          restCheckmark.className = "checkmark";
          const restText = document.createElement("span");
          restText.className = "check-label";
          restText.textContent = "День отдыха";
          restRow.appendChild(restToggle);
          restRow.appendChild(restCheckmark);
          restRow.appendChild(restText);
          content.appendChild(restRow);

          const rows = document.createElement("div");
          rows.className = "stack";
          content.appendChild(rows);

          const renderRows = () => {
            rows.innerHTML = "";
            if (day.simpleIsRest) {
              const hint = document.createElement("div");
              hint.className = "muted";
              hint.textContent = "Отдых";
              rows.appendChild(hint);
              syncItems(day);
              return;
            }
            const makeField = (labelText, input) => {
              const field = document.createElement("label");
              field.className = "training-exercise-field";
              const label = document.createElement("span");
              label.textContent = labelText;
              field.appendChild(label);
              field.appendChild(input);
              return field;
            };

            day.simpleItems.forEach((item, itemIndex) => {
              const row = document.createElement("div");
              row.className = "training-exercise-row";
              row.classList.toggle("is-cardio", item.type === "cardio");

              const name = document.createElement("input");
              name.type = "text";
              name.placeholder = "Название";
              name.value = item.name || "";
              name.addEventListener("input", () => {
                item.name = name.value;
                syncItems(day);
              });

              const sets = document.createElement("input");
              sets.type = "number";
              sets.placeholder = "Подходы";
              sets.value = item.sets || "";
              sets.addEventListener("input", () => {
                item.sets = sets.value;
                syncItems(day);
              });

              const reps = document.createElement("input");
              reps.type = "number";
              reps.placeholder = "Повторения";
              reps.value = item.reps || "";
              reps.addEventListener("input", () => {
                item.reps = reps.value;
                syncItems(day);
              });

              const weight = document.createElement("input");
              weight.type = "number";
              weight.step = "0.5";
              weight.placeholder = "Вес";
              weight.value = item.weight || "";
              weight.addEventListener("input", () => {
                item.weight = weight.value;
                syncItems(day);
              });

              const rest = document.createElement("input");
              rest.type = "number";
              rest.placeholder = "Отдых (сек)";
              rest.value = item.rest || "";
              rest.addEventListener("input", () => {
                item.rest = rest.value;
                syncItems(day);
              });

              const type = document.createElement("select");
              const strengthOpt = document.createElement("option");
              strengthOpt.value = "strength";
              strengthOpt.textContent = "Силовая";
              const cardioOpt = document.createElement("option");
              cardioOpt.value = "cardio";
              cardioOpt.textContent = "Кардио";
              type.appendChild(strengthOpt);
              type.appendChild(cardioOpt);
              type.value = item.type === "cardio" ? "cardio" : "strength";
              type.addEventListener("change", () => {
                item.type = type.value;
                syncItems(day);
                renderRows();
              });

              const duration = document.createElement("input");
              duration.type = "number";
              duration.placeholder = "Минуты";
              duration.value = item.duration || "";
              duration.addEventListener("input", () => {
                item.duration = duration.value;
                syncItems(day);
              });

              const remove = document.createElement("button");
              remove.type = "button";
              remove.className = "btn btn-outline";
              remove.textContent = "Удалить";
              remove.addEventListener("click", () => {
                day.simpleItems.splice(itemIndex, 1);
                syncItems(day);
                renderRows();
              });

              row.appendChild(makeField("Тип", type));
              row.appendChild(makeField("Название", name));
              if (item.type === "cardio") {
                row.appendChild(makeField("Длительность, мин", duration));
              } else {
                row.appendChild(makeField("Подходы", sets));
                row.appendChild(makeField("Повторения", reps));
                row.appendChild(makeField("Вес", weight));
                row.appendChild(makeField("Отдых, сек", rest));
              }
              row.appendChild(remove);
              rows.appendChild(row);
            });
          };

          const addBtn = document.createElement("button");
          addBtn.type = "button";
          addBtn.className = "btn btn-outline";
          addBtn.textContent = "Добавить упражнение";
          addBtn.addEventListener("click", () => {
            day.simpleItems.push({ type: "strength", name: "", sets: "", reps: "", weight: "", rest: "" });
            renderRows();
          });

          restToggle.addEventListener("change", () => {
            day.simpleIsRest = restToggle.checked;
            renderRows();
          });

          renderRows();
          content.appendChild(addBtn);
          const isOpen = index === 0;
          wrapper.classList.toggle("open", isOpen);
          content.style.display = isOpen ? "grid" : "none";
          title.addEventListener("click", () => {
            const next = !wrapper.classList.contains("open");
            wrapper.classList.toggle("open", next);
            content.style.display = next ? "grid" : "none";
          });
          editor.appendChild(wrapper);
        });
      } else {
        renderPlanEditor(editor, data[step.id], (updated) => {
          data[step.id] = updated;
        });
      }
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
    if (step.type === "range-group") {
      const fields = Array.isArray(step.fields) ? step.fields : [];
      for (const field of fields) {
        const raw = Number(data[field.id]);
        if (!Number.isFinite(raw)) {
          toast("Заполни все значения");
          return false;
        }
        if (field.min !== undefined && raw < field.min) {
          toast(`Минимум ${field.min}`);
          return false;
        }
        if (field.max !== undefined && raw > field.max) {
          toast(`Максимум ${field.max}`);
          return false;
        }
      }
      return true;
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
}
