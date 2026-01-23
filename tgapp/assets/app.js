import {
  authToken,
  initData,
  tg,
  toast,
  setActiveTab,
  updateNavHighlight,
  loadTargets,
  setScreenLoading,
} from "./core.js";
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

let loadersEnabled = true;

function initNav() {
  syncNavOffset();
  document.querySelectorAll(".nav-btn").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const tab = btn.dataset.tab;
      setActiveTab(tab);
      if (tab === "today") await runScreenLoad("today", loadToday);
      if (tab === "meals") await runScreenLoad("meals", loadMeals);
      if (tab === "plan") {
        await runScreenLoad("plan", async () => {
          await loadPlan();
          await loadTargets();
        });
      }
      if (tab === "steps") await runScreenLoad("steps", loadToday);
      if (tab === "profile") {
        await runScreenLoad("profile", async () => {
          await loadProfile();
          await loadTrainingProfile();
        });
      }
      if (tab === "discipline") await runScreenLoad("discipline", loadDiscipline);
      if (tab === "stats") await runScreenLoad("stats", loadStrengthStats);
      if (tab === "workout") await runScreenLoad("workout", loadWorkout);
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

function ensureScreenLoaders() {
  document.querySelectorAll(".screen").forEach((screen) => {
    if (screen.querySelector(".screen-loader")) return;
    const loader = document.createElement("div");
    loader.className = "screen-loader";
    loader.innerHTML = '<div class="screen-loader-spinner" aria-hidden="true"></div>';
    screen.appendChild(loader);
  });
}

async function runScreenLoad(name, task, opts = {}) {
  const showLoader = opts.showLoader ?? loadersEnabled;
  if (showLoader) setScreenLoading(name, true);
  try {
    await task();
  } finally {
    if (showLoader) setScreenLoading(name, false);
  }
}

async function prefetchScreens() {
  loadersEnabled = false;
  await Promise.allSettled([
    loadPlan().then(loadTargets),
    loadMeals(),
    loadProfile().then(loadTrainingProfile),
    loadDiscipline(),
    loadStrengthStats(),
    loadWorkout(),
  ]);
}

async function bootstrap() {
  ensureScreenLoaders();
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
  await runScreenLoad("today", loadToday);
  prefetchScreens();
  lucide?.createIcons();
  requestAnimationFrame(updateNavHighlight);
  syncNavOffset();
}

bootstrap().catch((err) => {
  console.error(err);
  toast("Ошибка мини-аппа");
});
