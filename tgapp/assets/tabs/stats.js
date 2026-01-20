import { $, api } from "../core.js";

async function loadDisciplineWeek() {
  const data = await api("/api/stats/week");
  renderWeekCalendars(data);
}

async function loadDisciplineMonth() {
  const data = await api("/api/stats/month");
  renderMonthCalendars(data);
}

export async function loadDiscipline() {
  await loadDisciplineWeek();
  await loadDisciplineMonth();
  toggleStatsView("week");
}

export function initDisciplineTab() {
  const statsWeek = $("stats-week");
  if (statsWeek) {
    statsWeek.addEventListener("click", async () => {
      toggleStatsView("week");
    });
  }
  const statsMonth = $("stats-month");
  if (statsMonth) {
    statsMonth.addEventListener("click", async () => {
      toggleStatsView("month");
    });
  }
}

function toggleStatsView(view) {
  $("stats-week").classList.toggle("btn-accent", view === "week");
  $("stats-week").classList.toggle("btn-outline", view !== "week");
  $("stats-month").classList.toggle("btn-accent", view === "month");
  $("stats-month").classList.toggle("btn-outline", view !== "month");
  $("stats-week-section").classList.toggle("active", view === "week");
  $("stats-month-section").classList.toggle("active", view === "month");
}

function renderWeekCalendars(data) {
  const food = $("week-food-calendar");
  const steps = $("week-steps-calendar");
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
