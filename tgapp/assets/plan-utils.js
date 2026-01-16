import { $ } from "./core.js";

export function safeParseJSON(raw) {
  const start = raw.indexOf("{");
  const end = raw.lastIndexOf("}");
  const sliced = start >= 0 && end > start ? raw.slice(start, end + 1) : raw;
  const cleaned = normalizeJSONString(sliced)
    .replace(/^\uFEFF/, "")
    .replace(/,\s*([}\]])/g, "$1")
    .replace(/\]\s*,\s*{/g, ", {");
  return JSON.parse(cleaned);
}

function normalizeJSONString(raw) {
  let out = "";
  let inString = false;
  let escape = false;
  for (let i = 0; i < raw.length; i += 1) {
    const ch = raw[i];
    if (escape) {
      out += ch;
      escape = false;
      continue;
    }
    if (ch === "\\" && inString) {
      out += ch;
      escape = true;
      continue;
    }
    if (ch === "\"") {
      inString = !inString;
      out += ch;
      continue;
    }
    if (inString && ch === "\n") {
      out += "\\n";
      continue;
    }
    if (inString && ch === "\r") {
      continue;
    }
    out += ch;
  }
  return out;
}

export function formatWeekPlan(items) {
  const lines = [];
  items.forEach((dayItem, idx) => {
    const dayNum = Number(dayItem.day || idx + 1);
    const name = String(dayItem.name || "—").trim();
    const focus = String(dayItem.focus || "").trim();
    const title = focus ? `${name} (${focus})` : name;
    const chunks = [];
    let counter = 0;
    let hasExercises = false;
    const itemsList = Array.isArray(dayItem.items) ? dayItem.items : [];
    if (itemsList.length) {
      itemsList.forEach((entry) => {
        const text = String(entry || "").trim();
        if (!text) return;
        hasExercises = true;
        counter += 1;
        chunks.push(`${counter}. ${text}`);
      });
      chunks.push("");
    }

    const groups = Array.isArray(dayItem.groups) ? dayItem.groups : [];
    groups.forEach((group) => {
      const groupName = String(group.muscle_group || "").trim();
      if (groupName) chunks.push(groupName);
      const exercises = Array.isArray(group.exercises) ? group.exercises : [];
      exercises.forEach((ex) => {
        const exName = String(ex.name || "").trim();
        if (!exName) return;
        hasExercises = true;
        counter += 1;
        const sets = String(ex.sets || "").trim();
        const reps = String(ex.reps || "").trim();
        const duration = String(ex.duration || "").trim();
        const notes = String(ex.notes || "").trim();
        let tail = "";
        if (duration) tail = duration;
        else if (sets || reps) tail = `${sets}${sets && reps ? "x" : ""}${reps}`;
        let line = `${counter}. ${exName}`;
        if (tail) line += ` — ${tail}`;
        if (notes) line += ` (${notes})`;
        chunks.push(line);
      });
      if (exercises.length) chunks.push("");
    });
    if (!hasExercises) {
      const activities = Array.isArray(dayItem.activities) ? dayItem.activities : [];
      activities.forEach((act) => {
        const text = String(act || "").trim();
        if (!text) return;
        counter += 1;
        chunks.push(`${counter}. ${text}`);
      });
      if (activities.length) chunks.push("");
    }
    if (dayItem.notes) {
      const note = String(dayItem.notes || "").trim();
      if (note) chunks.push(note);
    }
    const body = chunks.filter((line) => line !== "").join("\n");
    lines.push(`${dayNum ? `${title}` : title}${body ? `\n${body}` : ""}`);
  });
  return lines.join("\n\n");
}

export function formatPlanForDisplay(plan) {
  const raw = String(plan || "").trim();
  if (!raw.startsWith("{")) return raw || "—";
  try {
    const data = safeParseJSON(raw);
    if (Array.isArray(data.week_plan)) {
      return formatWeekPlan(data.week_plan);
    }
    const days = Array.isArray(data.days) ? data.days : null;
    if (!days || days.length === 0) return raw;
    const lines = [];
    for (let i = 0; i < 7; i += 1) {
      const rawDay = String(days[i] || "").replace(/\r\n/g, "\n");
      const dayLines = rawDay
        .split("\n")
        .map((line) => line.trim().replace(/^[_*]+|[_*]+$/g, ""))
        .filter(Boolean);
      const title = dayLines.length ? dayLines[0] : "—";
      const body = dayLines.length > 1 ? dayLines.slice(1).join("\n") : "";
      lines.push(`${title}${body ? `\n${body}` : ""}`);
    }
    return lines.join("\n\n");
  } catch (_) {
    return raw || "—";
  }
}

export function parsePlan(plan) {
  const raw = String(plan || "").trim();
  if (!raw.startsWith("{")) {
    return { text: raw || "—", structured: false };
  }
  try {
    const data = safeParseJSON(raw);
    if (Array.isArray(data.week_plan) && data.week_plan.length) {
      return { text: formatWeekPlan(data.week_plan), structured: true, weekPlan: data.week_plan };
    }
    const days = Array.isArray(data.days) ? data.days : null;
    if (days && days.length) {
      return { text: formatPlanForDisplay(raw), structured: true };
    }
  } catch (_) {
    return { text: raw || "—", structured: false };
  }
  return { text: raw || "—", structured: false };
}

export function extractPlanPayload(planText) {
  const raw = String(planText || "").trim();
  if (!raw.startsWith("{")) return null;
  try {
    return safeParseJSON(raw);
  } catch (_) {
    return null;
  }
}

export function renderTrainingAccordion(items, containerId = "training-accordion") {
  const container = $(containerId);
  if (!container) return;
  container.innerHTML = "";
  const seenDays = new Set();
  const hasContent = (dayItem) => {
    if (!dayItem || typeof dayItem !== "object") return false;
    const itemsList = Array.isArray(dayItem.items) ? dayItem.items.filter((v) => String(v || "").trim()) : [];
    const groups = Array.isArray(dayItem.groups) ? dayItem.groups : [];
    const activities = Array.isArray(dayItem.activities) ? dayItem.activities : [];
    const notes = String(dayItem.notes || "").trim();
    if (itemsList.length > 0) return true;
    if (groups.some((g) => Array.isArray(g.exercises) && g.exercises.length)) return true;
    if (activities.some((a) => String(a || "").trim() !== "")) return true;
    if (notes) return true;
    return false;
  };
  items.forEach((dayItem) => {
    if (!hasContent(dayItem)) return;
    const dayNum = Number(dayItem.day || 0);
    if (dayNum && (dayNum < 1 || dayNum > 7)) return;
    if (dayNum) {
      if (seenDays.has(dayNum)) return;
      seenDays.add(dayNum);
    }
    const name = String(dayItem.name || "—").trim();
    const focus = String(dayItem.focus || "").trim();
    const title = focus ? `${name} (${focus})` : name;

    const item = document.createElement("div");
    item.className = "accordion-item";

    const toggle = document.createElement("button");
    toggle.type = "button";
    toggle.className = "accordion-toggle";
    const label = document.createElement("span");
    label.textContent = `${title}`;
    toggle.appendChild(label);
    const burger = document.createElement("span");
    burger.className = "burger";
    burger.innerHTML = "<span></span><span></span><span></span>";
    toggle.appendChild(burger);
    const openBody = () => {
      item.classList.add("open");
      body.style.height = `${body.offsetHeight}px`;
      requestAnimationFrame(() => {
        const target = body.scrollHeight;
        body.style.height = `${target}px`;
      });
    };

    const closeBody = () => {
      body.style.height = `${body.scrollHeight}px`;
      requestAnimationFrame(() => {
        body.style.height = "0px";
      });
      item.classList.remove("open");
    };

    toggle.addEventListener("click", () => {
      if (item.classList.contains("open")) {
        closeBody();
      } else {
        openBody();
      }
    });

    const body = document.createElement("div");
    body.className = "accordion-body";
    body.style.height = "0px";
    body.addEventListener("transitionend", (event) => {
      if (event.propertyName !== "height") return;
      if (item.classList.contains("open")) {
        body.style.height = "auto";
      }
    });

    let hasExercises = false;
    let counter = 0;
    const itemsList = Array.isArray(dayItem.items) ? dayItem.items : [];
    if (itemsList.length) {
      const list = document.createElement("ol");
      list.className = "accordion-list";
      itemsList.forEach((entry) => {
        const text = String(entry || "").trim();
        if (!text) return;
        hasExercises = true;
        counter += 1;
        const li = document.createElement("li");
        li.textContent = text;
        list.appendChild(li);
      });
      if (list.children.length) body.appendChild(list);
    }

    const groups = Array.isArray(dayItem.groups) ? dayItem.groups : [];
    groups.forEach((group) => {
      const groupName = String(group.muscle_group || "").trim();
      if (groupName) {
        const h = document.createElement("div");
        h.className = "accordion-group-title";
        h.textContent = groupName;
        body.appendChild(h);
      }
      const list = document.createElement("ol");
      list.className = "accordion-list";
      const exercises = Array.isArray(group.exercises) ? group.exercises : [];
      exercises.forEach((ex) => {
        const exName = String(ex.name || "").trim();
        if (!exName) return;
        hasExercises = true;
        counter += 1;
        const sets = String(ex.sets || "").trim();
        const reps = String(ex.reps || "").trim();
        const duration = String(ex.duration || "").trim();
        const notes = String(ex.notes || "").trim();
        let tail = "";
        if (duration) tail = duration;
        else if (sets || reps) tail = `${sets}${sets && reps ? "x" : ""}${reps}`;
        let text = `${exName}`;
        if (tail) text += ` — ${tail}`;
        if (notes) text += ` (${notes})`;
        const li = document.createElement("li");
        li.textContent = text;
        list.appendChild(li);
      });
      if (list.children.length) body.appendChild(list);
    });

    if (!hasExercises) {
      const activities = Array.isArray(dayItem.activities) ? dayItem.activities : [];
      if (activities.length) {
        const list = document.createElement("ol");
        list.className = "accordion-list";
        activities.forEach((act) => {
          const text = String(act || "").trim();
          if (!text) return;
          const li = document.createElement("li");
          li.textContent = text;
          list.appendChild(li);
        });
        if (list.children.length) body.appendChild(list);
      }
    }

    item.appendChild(toggle);
    item.appendChild(body);
    container.appendChild(item);

    if (item.classList.contains("open")) {
      body.style.height = "auto";
    }
  });
}
