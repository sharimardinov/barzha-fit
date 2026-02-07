const BASE = "";

function getAuthHeaders() {
  const headers = { "Content-Type": "application/json" };
  const initData = window.Telegram?.WebApp?.initData;
  if (initData) {
    headers["X-Tg-Init-Data"] = initData;
  }
  const token = localStorage.getItem("auth_token") || new URLSearchParams(window.location.search).get("token");
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  return headers;
}

export async function api(path, body) {
  const res = await fetch(BASE + path, {
    method: "POST",
    headers: getAuthHeaders(),
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  const data = await res.json();
  if (!data.ok) {
    const err = new Error(data.error || "unknown_error");
    err.data = data.data;
    throw err;
  }
  return data.data;
}

export function getAuthToken() {
  return localStorage.getItem("auth_token") || new URLSearchParams(window.location.search).get("token") || "";
}

export function getInitData() {
  return window.Telegram?.WebApp?.initData || "";
}
