import { authToken, initData, targetFields } from "./state.js";

export async function api(path, body = {}) {
  const headers = {
    "Content-Type": "application/json",
  };
  if (initData) {
    headers["X-Tg-Init-Data"] = initData;
  } else if (authToken) {
    headers.Authorization = `Bearer ${authToken}`;
  }
  const res = await fetch(path, {
    method: "POST",
    headers,
    body: JSON.stringify(body),
  });
  const data = await res.json().catch(() => ({}));
  if (!data.ok) {
    const err = new Error(data.error || "request_failed");
    err.data = data.data;
    throw err;
  }
  return data.data;
}

export async function loadTargets() {
  const t = await api("/api/targets/get");
  targetFields.forEach(({ id, planId, field }) => {
    const value = t[field];
    const main = document.getElementById(id);
    if (main) main.value = value;
    const plan = document.getElementById(planId);
    if (plan) plan.value = value;
  });
}
