import { $ } from "./state.js";

export function toast(message, anchor) {
  const el = $("toast");
  if (!el) return;
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
  if (code === "google_auth_not_configured") return "Google вход недоступен";
  if (code === "google_auth_failed") return "Не удалось открыть Google";
  if (code === "stars_unavailable") return "Оплата звёздами недоступна";
  if (code === "stars_invoice_failed") return "Не удалось создать счёт на оплату";
  return fallback;
}

export function setActiveScreen(name) {
  document.querySelectorAll(".screen").forEach((panel) => {
    panel.classList.toggle("active", panel.id === `screen-${name}`);
  });
  const titles = {
    today: "Сегодня",
    meals: "Еда",
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
  document.dispatchEvent(new CustomEvent("screen-change", { detail: { name } }));
}

export function setActiveTab(name) {
  document.querySelectorAll(".nav-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.tab === name);
  });
  setActiveScreen(name);
  requestAnimationFrame(updateNavHighlight);
  try {
    if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.nativeNav) {
      window.webkit.messageHandlers.nativeNav.postMessage({ tab: name });
    }
  } catch (_) {}
}

export function setScreenLoading(name, loading) {
  const screen = document.getElementById(`screen-${name}`);
  if (!screen) return;
  screen.classList.toggle("is-loading", Boolean(loading));
}

export function updateNavHighlight() {
  const nav = document.querySelector(".nav");
  const highlight = $("nav-highlight");
  if (!nav || !highlight) return;
  const active = nav.querySelector(".nav-btn.active");
  if (!active) {
    highlight.style.opacity = "0";
    highlight.style.width = "0";
    return;
  }
  const navRect = nav.getBoundingClientRect();
  const btnRect = active.getBoundingClientRect();
  const left = btnRect.left - navRect.left;
  highlight.style.width = `${btnRect.width}px`;
  highlight.style.transform = `translate(${left}px, -50%)`;
  highlight.style.opacity = "1";
}

window.addEventListener("nativeTab", (event) => {
  const tab = event?.detail?.tab;
  if (tab) setActiveTab(tab);
});

export function setProgress(id, current, target) {
  const el = $(id);
  if (!el) return;
  const ratio = target > 0 ? current / target : 0;
  const pct = Math.min(Math.max(ratio * 100, 0), 100);
  el.style.width = `${pct}%`;
}
