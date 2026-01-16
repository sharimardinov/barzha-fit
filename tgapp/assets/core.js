export * from "./state.js";
export * from "./api.js";
export * from "./ui.js";

export function buildWheelValues(min, max, step) {
  const values = [];
  const isFloat = Math.abs(step % 1) > 0;
  for (let v = min; v <= max + 1e-9; v += step) {
    values.push(isFloat ? v.toFixed(1) : String(Math.round(v)));
  }
  return values;
}

export function createWheel(values, initial, onChange, axis = "y", extraClass = "") {
  const wheel = document.createElement("div");
  wheel.className = `wheel ${extraClass}`.trim();
  const list = document.createElement("div");
  list.className = "wheel-list";
  const selector = document.createElement("div");
  selector.className = "wheel-selector";

  values.forEach((value) => {
    const item = document.createElement("div");
    item.className = "wheel-item";
    item.textContent = value;
    item.dataset.value = value;
    list.appendChild(item);
  });

  const itemSize = axis === "x" ? 70 : 40;
  let currentValue = null;
  let ticking = false;

  const snapToIndex = (index) => {
    if (axis === "x") {
      list.scrollLeft = index * itemSize;
    } else {
      list.scrollTop = index * itemSize;
    }
  };

  const syncActive = () => {
    if (ticking) return;
    ticking = true;
    requestAnimationFrame(() => {
      const offset = axis === "x" ? list.scrollLeft : list.scrollTop;
      const index = Math.round(offset / itemSize);
      const value = values[Math.max(0, Math.min(values.length - 1, index))];
      if (value !== currentValue) {
        currentValue = value;
        if (typeof onChange === "function") onChange(value);
      }
      list.querySelectorAll(".wheel-item").forEach((item) => {
        item.classList.toggle("active", item.dataset.value === value);
      });
      ticking = false;
    });
  };

  list.addEventListener("scroll", syncActive);
  list.addEventListener("touchend", () => {
    const offset = axis === "x" ? list.scrollLeft : list.scrollTop;
    snapToIndex(Math.round(offset / itemSize));
  });
  list.addEventListener("mouseup", () => {
    const offset = axis === "x" ? list.scrollLeft : list.scrollTop;
    snapToIndex(Math.round(offset / itemSize));
  });

  wheel.appendChild(list);
  wheel.appendChild(selector);

  const initialIndex = Math.max(0, values.indexOf(initial));
  if (axis === "x") {
    list.scrollLeft = initialIndex * itemSize;
  } else {
    list.scrollTop = initialIndex * itemSize;
  }
  requestAnimationFrame(syncActive);

  return wheel;
}
export function normalizeNumberInput(value) {
  return String(value || "").replace(/,/g, ".").replace(/\s+/g, "");
}

export function parseNumberInput(value) {
  const normalized = normalizeNumberInput(value);
  return Number(normalized || 0);
}

export function normalizeGoalType(value) {
  const v = String(value || "").trim().toLowerCase();
  if (v === "cut" || v === "bulk" || v === "balance") return v;
  return "";
}

export function getGoalTypeFromTabs() {
  const checked = document.querySelector('input[name="goal-tabs"]:checked');
  return normalizeGoalType(checked ? checked.value : "");
}

export function setGoalTabs(value) {
  const v = normalizeGoalType(value) || "balance";
  const input = document.querySelector(`input[name="goal-tabs"][value="${v}"]`);
  if (input) input.checked = true;
}

export function splitGoalText(text) {
  const raw = String(text || "").trim();
  if (!raw) return { type: "", notes: "" };
  const lowered = raw.toLowerCase();
  for (const type of ["cut", "bulk", "balance"]) {
    if (lowered.startsWith(type)) {
      const rest = raw.slice(type.length).trim();
      const notes = rest.replace(/^[:\-–—]\s*/, "");
      return { type, notes };
    }
  }
  return { type: "", notes: raw };
}

export function buildTrainingGoalText(goalType, notes) {
  const type = normalizeGoalType(goalType);
  const extra = String(notes || "").trim();
  if (!type && !extra) return "";
  if (!extra) return type;
  if (!type) return extra;
  return `${type} — ${extra}`;
}
