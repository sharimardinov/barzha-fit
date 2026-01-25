export * from "./state.js";
export * from "./api.js";
export * from "./ui.js";

export function postNativeMessage(name, payload) {
  if (typeof window === "undefined") return false;
  const handlers = window.webkit?.messageHandlers;
  const handler = handlers?.[name];
  if (!handler || typeof handler.postMessage !== "function") return false;
  try {
    handler.postMessage(payload ?? {});
    return true;
  } catch (_) {
    return false;
  }
}

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

  const itemSize = axis === "x" ? 50 : 40;
  let currentValue = null;
  let ticking = false;

  const getHorizontalMetrics = () => {
    const styles = getComputedStyle(list);
    const paddingLeft = Number.parseFloat(styles.paddingLeft || "0") || 0;
    const viewWidth = list.clientWidth || itemSize * 3;
    return { paddingLeft, viewWidth };
  };

  const indexFromOffset = (offset) => {
    if (axis !== "x") return Math.round(offset / itemSize);
    const { paddingLeft, viewWidth } = getHorizontalMetrics();
    const center = offset + viewWidth / 2;
    const raw = (center - paddingLeft - itemSize / 2) / itemSize;
    return Math.round(raw);
  };

  const scrollForIndex = (index) => {
    if (axis !== "x") return index * itemSize;
    const { paddingLeft, viewWidth } = getHorizontalMetrics();
    const target = index * itemSize + paddingLeft - (viewWidth / 2 - itemSize / 2);
    return Math.max(0, target);
  };

  const snapToIndex = (index) => {
    if (axis === "x") {
      list.scrollLeft = scrollForIndex(index);
    } else {
      list.scrollTop = index * itemSize;
    }
  };

  const syncActive = () => {
    if (ticking) return;
    ticking = true;
    requestAnimationFrame(() => {
      const offset = axis === "x" ? list.scrollLeft : list.scrollTop;
      const index = indexFromOffset(offset);
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
  list.addEventListener("mouseup", () => {
    const offset = axis === "x" ? list.scrollLeft : list.scrollTop;
    snapToIndex(Math.round(offset / itemSize));
  });

  wheel.appendChild(list);
  wheel.appendChild(selector);

  let initialIndex = values.indexOf(initial);
  if (initialIndex < 0 && initial !== undefined && initial !== null) {
    const target = Number(initial);
    let bestIdx = 0;
    let bestDiff = Infinity;
    values.forEach((val, idx) => {
      const diff = Math.abs(Number(val) - target);
      if (diff < bestDiff) {
        bestDiff = diff;
        bestIdx = idx;
      }
    });
    initialIndex = bestIdx;
  }
  if (initialIndex < 0) initialIndex = 0;
  requestAnimationFrame(() => {
    if (axis === "x") {
      list.scrollLeft = scrollForIndex(initialIndex);
    } else {
      list.scrollTop = initialIndex * itemSize;
    }
    syncActive();
  });

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
