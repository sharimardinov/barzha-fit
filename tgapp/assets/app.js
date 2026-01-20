import { authToken, initData, tg, toast, setActiveTab, updateNavHighlight, loadTargets } from "./core.js";
import {
  loadToday,
  initTodayTab,
  loadMeals,
  initMealsTab,
  loadPlan,
  initPlanTab,
  loadProfile,
  loadTrainingProfile,
  initProfileTab,
  initStepsTab,
  loadDiscipline,
  initDisciplineTab,
  loadStrengthStats,
  initStrengthStatsTab,
  loadWorkout,
  initWorkoutTab,
} from "./tabs/index.js";
import { ensureOnboarding, initOnboardingWizard } from "./onboarding.js";

function initNav() {
  syncNavOffset();
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
      if (tab === "discipline") {
        await loadDiscipline();
      }
      if (tab === "stats") {
        await loadStrengthStats();
      }
      if (tab === "workout") {
        await loadWorkout();
      }
    });
  });
  window.addEventListener("resize", () => {
    syncNavOffset();
    requestAnimationFrame(updateNavHighlight);
  });
}

function syncNavOffset() {
  const nav = document.querySelector(".nav");
  if (!nav) return;
  document.documentElement.style.setProperty("--nav-height", `${nav.offsetHeight}px`);
}

async function bootstrap() {
  const debugPanel = document.getElementById("debug-panel");
  const enableDebug = new URLSearchParams(window.location.search).has("debug");
  if (debugPanel && enableDebug) {
    debugPanel.classList.add("active");
    debugPanel.textContent = `authToken: ${authToken ? "yes" : "no"} | initData: ${initData ? "yes" : "no"}`;
    if (authToken) {
      try {
        const res = await fetch("/auth/verify", {
          headers: { Authorization: `Bearer ${authToken}` },
        });
        const data = await res.json().catch(() => ({}));
        if (data?.data?.user_id) {
          debugPanel.textContent = `user_id: ${data.data.user_id}`;
        } else if (data?.error) {
          debugPanel.textContent = `auth_verify: ${data.error}`;
        }
      } catch (_) {
        debugPanel.textContent = "auth_verify: failed";
      }
    }
  }
  if (!initData && !authToken) {
    toast("Открой мини-апп из Telegram");
    return;
  }
  tg?.ready();
  tg?.expand();

  initNav();
  initTodayTab();
  initMealsTab();
  initPlanTab();
  initStepsTab();
  initProfileTab();
  initDisciplineTab();
  initStrengthStatsTab();
  initWorkoutTab();

  const onboardingReady = await ensureOnboarding();
  if (!onboardingReady) {
    await initOnboardingWizard();
    lucide?.createIcons();
    requestAnimationFrame(updateNavHighlight);
    return;
  }

  setActiveTab("today");
  await loadToday();
  lucide?.createIcons();
  requestAnimationFrame(updateNavHighlight);
  syncNavOffset();
}

bootstrap().catch((err) => {
  console.error(err);
  toast("Ошибка мини-аппа");
});
