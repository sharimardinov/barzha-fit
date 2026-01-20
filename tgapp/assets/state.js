export const tg = window.Telegram?.WebApp || null;
export const initData = tg?.initData || "";
const params = new URLSearchParams(window.location.search);
const tokenFromQuery = params.get("token");
if (tokenFromQuery) {
  localStorage.setItem("auth_token", tokenFromQuery);
}
export const authToken = tokenFromQuery || localStorage.getItem("auth_token") || "";

export const $ = (id) => document.getElementById(id);

export const state = {
  today: null,
  planText: "",
  planPayload: null,
  planStructured: false,
  onboarding: false,
};

export const targetFields = [
  { field: "kcal", id: "target-kcal", planId: "plan-target-kcal" },
  { field: "protein", id: "target-protein", planId: "plan-target-protein" },
  { field: "fat", id: "target-fat", planId: "plan-target-fat" },
  { field: "carbs", id: "target-carbs", planId: "plan-target-carbs" },
  { field: "steps", id: "target-steps", planId: "plan-target-steps" },
];
