import { $, api } from "../core.js";

let monthOffset = 0;

async function loadDisciplineMonth() {
  const data = await api("/api/stats/month", { offset: monthOffset });
  renderMonthCalendars(data);
  updateMonthLabel(data);
  updateMonthControls();
}

export async function loadDiscipline() {
  monthOffset = 0;
  await loadDisciplineMonth();
}

export function initDisciplineTab() {
  const prev = $("discipline-prev");
  if (prev) {
    prev.addEventListener("click", async () => {
      monthOffset -= 1;
      await loadDisciplineMonth();
    });
  }
  const next = $("discipline-next");
  if (next) {
    next.addEventListener("click", async () => {
      if (monthOffset >= 0) return;
      monthOffset += 1;
      await loadDisciplineMonth();
    });
  }
}

function renderMonthCalendars(data) {
  const food = $("month-food-calendar");
  const steps = $("month-steps-calendar");
  food.innerHTML = "";
  steps.innerHTML = "";
  if (!data || !Array.isArray(data.days) || data.days.length === 0) {
    renderEmptyCalendar(food);
    renderEmptyCalendar(steps);
    return;
  }
  for (let i = 0; i < data.offset; i += 1) {
    food.appendChild(makeEmptyCell());
    steps.appendChild(makeEmptyCell());
  }
  data.days.forEach((d) => {
    const future = isFutureDate(d.date);
    food.appendChild(makeCalendarCell(d.day, d.foodOk, future));
    steps.appendChild(makeCalendarCell(d.day, d.stepsOk, future));
  });
}

function updateMonthLabel(data) {
  const label = $("discipline-month-label");
  if (!label) return;
  const start = data?.monthStart;
  if (!start) {
    label.textContent = "—";
    return;
  }
  const date = new Date(`${start}T00:00:00`);
  if (Number.isNaN(date.getTime())) {
    label.textContent = "—";
    return;
  }
  const text = date.toLocaleDateString("ru-RU", { month: "long", year: "numeric" });
  label.textContent = text.charAt(0).toUpperCase() + text.slice(1);
}

function updateMonthControls() {
  const next = $("discipline-next");
  if (!next) return;
  next.disabled = monthOffset >= 0;
  next.classList.toggle("btn-outline", monthOffset >= 0);
  next.classList.toggle("btn-accent", monthOffset < 0);
}

function makeCalendarCell(day, active, future) {
  const cell = document.createElement("div");
  cell.className = future ? "calendar-cell future" : "calendar-cell";
  const dot = document.createElement("div");
  dot.className = `calendar-dot${active ? " active" : ""}`;
  const label = document.createElement("div");
  label.textContent = String(day);
  cell.appendChild(dot);
  cell.appendChild(label);
  return cell;
}

function makeEmptyCell() {
  const cell = document.createElement("div");
  cell.className = "calendar-cell calendar-empty";
  const dot = document.createElement("div");
  dot.className = "calendar-dot";
  const label = document.createElement("div");
  label.textContent = "-";
  cell.appendChild(dot);
  cell.appendChild(label);
  return cell;
}

function renderEmptyCalendar(target) {
  const label = document.createElement("div");
  label.className = "calendar-empty-label";
  label.textContent = "Нет данных";
  target.appendChild(label);
}

function isFutureDate(value) {
  if (!value) return false;
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const date = new Date(`${value}T00:00:00`);
  return date > today;
}
