import { $, api, toast, formatApiError } from "../core.js";

export async function loadStrengthStats() {
  try {
    const data = await api("/api/stats/strength");
    renderStrengthStats(data);
  } catch (err) {
    toast(formatApiError(err, "Не удалось загрузить статистику"));
  }
}

export function initStrengthStatsTab() {
  // Reserved for future controls.
}

function renderStrengthStats(data) {
  const strength = data?.strength || data;
  renderOverall(data);
  renderTotals(strength?.totals);
  renderStatsList($("strength-top-tonnage"), strength?.topByTonnage, "tonnage");
  renderStatsList($("strength-top-reps"), strength?.topByReps, "reps");
  renderRecent($("strength-recent"), strength?.recent);
}

function renderOverall(data) {
  const steps = data?.stepsTotal;
  const macros = data?.macros || {};
  setText("stats-steps-total", formatSteps(steps));
  setText("stats-protein-total", formatGrams(macros.protein));
  setText("stats-fat-total", formatGrams(macros.fat));
  setText("stats-carbs-total", formatGrams(macros.carbs));
}

function renderTotals(totals) {
  setText("strength-total-sets", formatInt(totals?.sets));
  setText("strength-total-reps", formatInt(totals?.reps));
  setText("strength-total-tonnage", formatWeight(totals?.tonnage));
  setText("strength-avg-weight", formatWeight(totals?.avgWeight));
  setText("strength-max-weight", formatWeight(totals?.maxWeight));
}

function renderStatsList(target, items, mode) {
  if (!target) return;
  target.innerHTML = "";
  if (!Array.isArray(items) || items.length === 0) {
    target.appendChild(emptyLabel());
    return;
  }
  items.forEach((item) => {
    const row = document.createElement("div");
    row.className = "stats-row";

    const left = document.createElement("div");
    left.className = "stats-row-name";
    left.textContent = item.name || "—";

    const right = document.createElement("div");
    right.className = "stats-row-meta";
    if (mode === "tonnage") {
      right.textContent = `${formatWeight(item.tonnage)} · ${formatInt(item.reps)} повт. · ${formatInt(item.sets)} подх.`;
    } else {
      right.textContent = `${formatInt(item.reps)} повт. · ${formatWeight(item.tonnage)} · ${formatInt(item.sets)} подх.`;
    }

    row.appendChild(left);
    row.appendChild(right);
    target.appendChild(row);
  });
}

function renderRecent(target, items) {
  if (!target) return;
  target.innerHTML = "";
  if (!Array.isArray(items) || items.length === 0) {
    target.appendChild(emptyLabel());
    return;
  }
  items.forEach((block) => {
    const wrap = document.createElement("div");
    wrap.className = "stats-recent-block";

    const title = document.createElement("div");
    title.className = "stats-recent-title";
    title.textContent = block.name || "—";

    const list = document.createElement("div");
    list.className = "stats-recent-list";
    const entries = Array.isArray(block.entries) ? block.entries : [];
    if (entries.length === 0) {
      list.appendChild(emptyLabel());
    } else {
      entries.forEach((entry) => {
        const row = document.createElement("div");
        row.className = "stats-recent-item";
        row.textContent = `${formatDate(entry.completedAt)} · ${formatWeight(entry.weight)} × ${formatInt(entry.reps)}`;
        list.appendChild(row);
      });
    }

    wrap.appendChild(title);
    wrap.appendChild(list);
    target.appendChild(wrap);
  });
}

function emptyLabel() {
  const label = document.createElement("div");
  label.className = "muted";
  label.textContent = "Нет данных";
  return label;
}

function formatInt(value) {
  const num = Number(value) || 0;
  return Math.round(num).toLocaleString("ru-RU");
}

function formatWeight(value) {
  const num = Number(value) || 0;
  if (num <= 0) return "—";
  const rounded = Math.round(num * 10) / 10;
  return `${rounded.toLocaleString("ru-RU")} кг`;
}

function formatGrams(value) {
  const num = Number(value) || 0;
  return `${Math.round(num).toLocaleString("ru-RU")} г`;
}

function formatSteps(value) {
  const num = Number(value) || 0;
  if (num <= 0) return "0";
  return `${Math.round(num).toLocaleString("ru-RU")}`;
}

function formatDate(value) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleDateString("ru-RU", { day: "2-digit", month: "short" });
}

function setText(id, value) {
  const el = $(id);
  if (el) el.textContent = value;
}
