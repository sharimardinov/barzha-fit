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

export function formatWeekPlanForEditor(items) {
  if (!Array.isArray(items) || items.length === 0) return "";
  return items.map((day, index) => {
    if (typeof day === "string") {
      const dayLines = String(day)
        .split("\n")
        .map((line) => line.trim())
        .filter(Boolean);
      return [String(index + 1), ...(dayLines.length ? dayLines : ["Отдых"])].join("\n");
    }
    const dayNumber = Number(day?.day) || index + 1;
    const lines = Array.isArray(day?.items)
      ? day.items.map((item) => String(item || "").trim()).filter(Boolean)
      : [];
    return [String(dayNumber), ...(lines.length ? lines : ["Отдых"])].join("\n");
  }).join("\n\n");
}

export function parsePlan(plan) {
  if (!plan) return { items: [], text: "" };
  const parsed = safeParseJSON(plan);
  if (parsed?.week_plan) return { items: parsed.week_plan, text: formatWeekPlan(parsed.week_plan), structured: true, payload: parsed };
  if (parsed?.days) return { items: parsed.days, text: formatWeekPlan(parsed.days), structured: true, payload: parsed };
  return { items: [], text: String(plan), structured: false, payload: null };
}

function isDayHeader(line) {
  return line.match(/^\s*(?:день\s*)?(\d+)\s*[:\-–—.]?\s*$/i);
}

function isRestWord(line) {
  return /^(отдых|rest|off)$/i.test(String(line || "").trim());
}

function isIncompleteExerciseLine(line) {
  return /[|/]\s*$/.test(String(line || "").trim());
}

function normalizeSetRepField(value) {
  const raw = String(value || "").trim();
  if (!raw) return raw;
  const match = raw.match(/^(\d+)\s*[xх×]\s*(\d+)(?:\s*(?:сек|sec|s))?(?:\s+\S.*)?$/i);
  if (!match) return raw;
  return `${match[1]}x${match[2]}`;
}

function normalizeCardioLine(line) {
  const match = String(line || "").trim().match(/^(.+?)\s+(\d+(?:[.,]\d+)?)\s*(мин|min|m)\s*$/i);
  if (!match) return String(line || "").trim();
  const name = match[1].trim();
  const duration = `${match[2].replace(",", ".")} мин`;
  return `Кардио: ${name} | ${duration}`;
}

function normalizeExerciseLine(line) {
  const raw = String(line || "").trim();
  if (!raw) return "";
  if (isRestWord(raw)) return "Отдых";
  if (!raw.includes("|")) return normalizeCardioLine(raw);

  const parts = raw.split("|").map((part) => part.trim());
  if (parts.length >= 2) {
    parts[1] = normalizeSetRepField(parts[1]);
  }
  return parts.join(" | ");
}

function preprocessPlanLines(raw) {
  const normalized = String(raw || "").replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  const sourceLines = normalized.split("\n");
  const out = [];

  for (const source of sourceLines) {
    let line = source.replace(/\t/g, " ").trim();
    if (!line) continue;

    const header = isDayHeader(line);
    if (out.length > 0 && isIncompleteExerciseLine(out[out.length - 1]) && !header) {
      const split = line.match(/^(\d+(?:[.,]\d+)?)\s*(.*)$/);
      if (split) {
        out[out.length - 1] = `${out[out.length - 1].trim()} ${split[1]}`.trim();
        line = String(split[2] || "").trim();
        if (!line) continue;
      } else {
        out[out.length - 1] = `${out[out.length - 1].trim()} ${line}`.trim();
        continue;
      }
    }

    out.push(line);
  }

  return out;
}

export function parsePastedWeekPlan(raw) {
  const lines = preprocessPlanLines(raw);
  const blocks = [];
  let current = null;

  for (const line of lines) {
    const header = isDayHeader(line);
    if (header) {
      if (current) blocks.push(current);
      current = { day: Number(header[1]), items: [] };
      continue;
    }
    if (!current) continue;
    current.items.push(line);
  }
  if (current) blocks.push(current);
  if (blocks.length === 0) {
    return [];
  }

  return blocks.map((block) => {
    const items = block.items
      .map((line) => normalizeExerciseLine(line))
      .filter(Boolean);
    const isRest = items.length === 0 || items.every(isRestWord);
    return {
      day: block.day,
      name: `День ${block.day}`,
      focus: "",
      type: isRest ? "rest" : "train",
      items: isRest ? ["Отдых"] : items,
    };
  });
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
