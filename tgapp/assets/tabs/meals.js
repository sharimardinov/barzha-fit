import { $, api, toast } from "../core.js";
import { loadToday } from "./today.js";

export async function loadMeals() {
  const items = await api("/api/meals/today");
  const list = $("meal-list");
  const listToday = $("meal-list-today");
  list.innerHTML = "";
  if (listToday) listToday.innerHTML = "";

  let totalKcal = 0;
  let totalP = 0;
  let totalF = 0;
  let totalC = 0;

  if (!items.length) {
    list.innerHTML = '<div class="hint">Пока пусто.</div>';
    if (listToday) listToday.innerHTML = '<div class="hint">Пока пусто.</div>';
  } else {
    items.forEach((item) => {
      totalKcal += item.kcal;
      totalP += item.protein_g;
      totalF += item.fat_g;
      totalC += item.carbs_g;

      const card = document.createElement("div");
      card.className = "list-item";
      card.innerHTML = `
        <div>${item.text}</div>
        <div class="meta">${item.kcal} ккал • Б${item.protein_g} Ж${item.fat_g} У${item.carbs_g}</div>
        <div class="actions">
          <button class="btn btn-ghost" data-id="${item.id}">Удалить</button>
        </div>
      `;
      card.querySelector("button").addEventListener("click", async () => {
        await api("/api/meal/delete", { id: item.id });
        toast("Удалено");
        await loadMeals();
        await loadToday();
      });
      list.appendChild(card);

      if (listToday) {
        const clone = card.cloneNode(true);
        clone.querySelector("button").addEventListener("click", async () => {
          await api("/api/meal/delete", { id: item.id });
          toast("Удалено");
          await loadMeals();
          await loadToday();
        });
        listToday.appendChild(clone);
      }
    });
  }

  $("meal-total-kcal").textContent = `${totalKcal} ккал`;
  $("meal-total-macros").textContent = `Б ${totalP} • Ж ${totalF} • У ${totalC}`;
  const totalTodayKcal = $("meal-total-kcal-today");
  const totalTodayMacros = $("meal-total-macros-today");
  if (totalTodayKcal) totalTodayKcal.textContent = `${totalKcal} ккал`;
  if (totalTodayMacros) totalTodayMacros.textContent = `Б ${totalP} • Ж ${totalF} • У ${totalC}`;
}

export function initMealsTab() {
  const mealAdd = $("meal-add");
  if (mealAdd) {
    mealAdd.addEventListener("click", async () => {
      const text = $("meal-text").value.trim();
      if (!text) return;
      const data = await api("/api/meal/add", { text });
      toast(data.aiError ? "AI упал, текст сохранён" : "Еда добавлена");
      $("meal-text").value = "";
      await loadMeals();
      await loadToday();
    });
  }
}
