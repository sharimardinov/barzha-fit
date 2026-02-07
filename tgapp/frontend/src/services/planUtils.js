export function safeParseJSON(raw) {
  if (!raw || typeof raw !== "string") return null;
  try {
    let s = raw.replace(/^\uFEFF/, "");
    s = s.replace(/,\s*([\]}])/g, "$1");
    return JSON.parse(s);
  } catch {
    return null;
  }
}

export function formatWeekPlan(items) {
  if (!Array.isArray(items)) return "";
  const dayNames = ["Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"];
  return items.map((day, i) => {
    const name = day.dayName || day.name || dayNames[i] || `День ${i + 1}`;
    const focus = day.focus ? ` — ${day.focus}` : "";
    const header = `${name}${focus}`;
    const lines = (day.items || []).map((item) => {
      if (typeof item === "string") return `  ${item}`;
      return `  ${item.name || ""}${item.sets ? ` ${item.sets}x${item.reps || ""}` : ""}${item.duration ? ` ${item.duration}` : ""}`;
    });
    return [header, ...lines].join("\n");
  }).join("\n\n");
}

export function parsePlan(plan) {
  if (!plan) return { items: [], text: "" };
  const parsed = safeParseJSON(plan);
  if (parsed?.week_plan) return { items: parsed.week_plan, text: formatWeekPlan(parsed.week_plan), structured: true, payload: parsed };
  if (parsed?.days) return { items: parsed.days, text: formatWeekPlan(parsed.days), structured: true, payload: parsed };
  return { items: [], text: String(plan), structured: false, payload: null };
}

export function buildWeekPlanTemplate(restText = "Отдых") {
  const dayNames = ["Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"];
  return dayNames.map((name, i) => ({
    day: i + 1,
    name: `День ${i + 1}`,
    dayName: name,
    type: "rest",
    focus: "",
    items: [restText],
  }));
}

export function normalizeWeekPlan(items) {
  if (!Array.isArray(items)) return buildWeekPlanTemplate();
  const dayNames = ["Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота", "Воскресенье"];
  return dayNames.map((name, i) => {
    const day = items[i] || {};
    const dayItems = (day.items || []).map((it) => typeof it === "string" ? it : it.name || "");
    const onlyRestWords = dayItems.every((s) => /^[\s]*(отдых|выходн|rest|off|—|-|)[\s]*$/i.test(s));
    const isRest = (onlyRestWords && dayItems.length <= 1) || dayItems.length === 0;
    return {
      day: i + 1,
      name: `День ${i + 1}`,
      dayName: name,
      type: isRest ? "rest" : "train",
      focus: day.focus || "",
      items: dayItems.length ? dayItems : ["Отдых"],
    };
  });
}
