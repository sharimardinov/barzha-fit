import { initData, tg, toast, setActiveTab, updateNavHighlight, loadTargets } from "./core.js";
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
  loadStats,
  initStatsTab,
} from "./tabs/index.js";
import { ensureOnboarding, initOnboardingWizard } from "./onboarding.js";

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
        await loadStats();
      }
    });
  });
  window.addEventListener("resize", () => requestAnimationFrame(updateNavHighlight));
}

async function bootstrap() {
  if (!initData) {
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
  initStatsTab();

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
}

bootstrap().catch((err) => {
  console.error(err);
  toast("Ошибка мини-аппа");
});
