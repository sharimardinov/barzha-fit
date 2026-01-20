import { $ } from "./state.js";

export function toast(message, anchor) {
  const el = $("toast");
  el.textContent = message;
  el.classList.add("show");

  if (anchor instanceof Element) {
    el.classList.add("anchored");
    requestAnimationFrame(() => {
      const rect = anchor.getBoundingClientRect();
      const toastRect = el.getBoundingClientRect();
      const padding = 12;
      let left = rect.right + padding;
      if (left + toastRect.width > window.innerWidth - padding) {
        left = rect.left - toastRect.width - padding;
      }
      left = Math.max(padding, Math.min(left, window.innerWidth - toastRect.width - padding));
      let top = rect.top + rect.height / 2 - toastRect.height / 2;
      top = Math.max(padding, Math.min(top, window.innerHeight - toastRect.height - padding));
      el.style.left = `${left}px`;
      el.style.top = `${top}px`;
      el.style.bottom = "";
      el.style.transform = "none";
    });
  } else {
    el.classList.remove("anchored");
    el.style.left = "50%";
    el.style.top = "";
    el.style.bottom = "150px";
    el.style.transform = "translateX(-50%)";
  }

  setTimeout(() => el.classList.remove("show"), 1800);
}

export function setButtonLoading(btn, loading, label) {
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

export function formatApiError(err, fallback) {
  const code = err?.message || "";
  if (code === "training_ai_unavailable") return "AI тренировок недоступен";
  if (code === "activity_ai_unavailable") return "AI активности недоступен";
  if (code === "training_generate_failed") return "Не удалось сгенерировать план";
  if (code === "activity_estimate_failed") return "Не удалось пересчитать активность";
  if (code === "plan_not_found") return "План не найден для пересчёта активности";
  if (code === "plan_save_failed") return "Не удалось сохранить план";
  if (code === "training_profile_save_failed") return "Ошибка сохранения тренировочного профиля";
  if (code === "profile_save_failed") return "Ошибка сохранения профиля";
  if (code === "workout_plan_not_found") return "План тренировки не найден";
  if (code === "workout_plan_invalid") return "План тренировки заполнен неверно";
  if (code === "workout_session_not_found") return "Активная тренировка не найдена";
  if (code === "workout_session_state") return "Состояние тренировки изменилось";
  if (code === "workout_session_paused") return "Тренировка на паузе";
  if (code === "strength_stats_unavailable") return "Статистика силовых недоступна";
  if (code === "strength_stats_failed") return "Не удалось загрузить статистику силовых";
  return fallback;
}

export function setActiveScreen(name) {
  document.querySelectorAll(".screen").forEach((panel) => {
    panel.classList.toggle("active", panel.id === `screen-${name}`);
  });
  const titles = {
    today: "Сегодня",
    meals: "Еда",
    steps: "Шаги",
    profile: "Профиль",
    plan: "План",
    discipline: "Дисциплина",
    stats: "Статистика",
    workout: "Тренировка",
    onboarding: "",
    "onboarding-done": "",
  };
  if ($("screen-title")) {
    $("screen-title").textContent = titles[name] || "Сегодня";
  }
}

export function setActiveTab(name) {
  document.querySelectorAll(".nav-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.tab === name);
  });
  setActiveScreen(name);
  requestAnimationFrame(updateNavHighlight);
}

export function updateNavHighlight() {
  const nav = document.querySelector(".nav");
  const highlight = $("nav-highlight");
  if (!nav || !highlight) return;
  const active = nav.querySelector(".nav-btn.active");
  if (!active) {
    highlight.style.opacity = "0";
    return;
  }
  const navRect = nav.getBoundingClientRect();
  const btnRect = active.getBoundingClientRect();
  const left = btnRect.left - navRect.left;
  const top = btnRect.top - navRect.top;
  highlight.style.width = `${btnRect.width}px`;
  highlight.style.height = `${btnRect.height}px`;
  highlight.style.transform = `translate(${left}px, ${top}px)`;
  highlight.style.opacity = "1";
}

export function setProgress(id, current, target) {
  const el = $(id);
  if (!el) return;
  const ratio = target > 0 ? current / target : 0;
  const pct = Math.min(Math.max(ratio * 100, 0), 100);
  el.style.width = `${pct}%`;
}
