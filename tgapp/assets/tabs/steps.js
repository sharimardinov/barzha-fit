import { $, api, toast, parseNumberInput } from "../core.js";
import { loadToday } from "./today.js";

export function initStepsTab() {
  const stepsSave = $("steps-save");
  if (stepsSave) {
    stepsSave.addEventListener("click", async () => {
      const steps = parseNumberInput($("steps-value").value);
      await api("/api/steps/set", { steps });
      toast("Шаги записаны");
      await loadToday();
    });
  }
}
