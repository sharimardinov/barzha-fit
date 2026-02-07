const messages = {
  unknown_error: "Что-то пошло не так",
  profile_not_found: "Профиль не найден",
  missing_fields: "Заполни все поля",
  invalid_sex: "Укажи пол",
  invalid_age: "Проверь возраст",
  invalid_height: "Проверь рост",
  invalid_weight: "Проверь вес",
  invalid_bodyfat: "Проверь % жира",
  invalid_goal: "Выбери цель",
  targets_not_found: "Цели не найдены",
  meal_text_empty: "Введи текст",
  meal_parse_failed: "Не удалось распознать еду",
  workout_plan_not_found: "План тренировки не найден",
  workout_plan_invalid: "Некорректный план тренировки",
  workout_session_not_found: "Сессия не найдена",
  plan_invalid: "Некорректный план",
  plan_not_found: "План не найден",
};

export function formatApiError(err, fallback = "Ошибка") {
  const code = err?.message || "";
  return messages[code] || fallback;
}
