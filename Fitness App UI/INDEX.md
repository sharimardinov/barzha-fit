# 📑 BarzhaFit - Индекс проекта

## 🎯 С чего начать?

**Новый пользователь:** начните с [`QUICK_START.md`](./QUICK_START.md)  
**Дизайнер:** изучите [`DESIGN_SYSTEM.md`](./DESIGN_SYSTEM.md) и [`FIGMA_EXPORT.md`](./FIGMA_EXPORT.md)  
**Разработчик:** читайте [`DEVELOPER_GUIDE.md`](./DEVELOPER_GUIDE.md)  
**Менеджер проекта:** смотрите [`PROJECT_SUMMARY.md`](./PROJECT_SUMMARY.md)

---

## 📚 Документация

### 1. Основная документация
| Файл | Описание | Для кого |
|------|----------|----------|
| [`README.md`](./README.md) | Общее описание проекта | Все |
| [`QUICK_START.md`](./QUICK_START.md) | Быстрый старт и тестирование | Тестировщики, новые пользователи |
| [`PROJECT_SUMMARY.md`](./PROJECT_SUMMARY.md) | Итоговая сводка проекта | PM, stakeholders |

### 2. Дизайн
| Файл | Описание | Для кого |
|------|----------|----------|
| [`DESIGN_SYSTEM.md`](./DESIGN_SYSTEM.md) | Цвета, типографика, spacing | Дизайнеры, разработчики |
| [`FIGMA_EXPORT.md`](./FIGMA_EXPORT.md) | Инструкции для экспорта в Figma | Дизайнеры |

### 3. Разработка
| Файл | Описание | Для кого |
|------|----------|----------|
| [`DEVELOPER_GUIDE.md`](./DEVELOPER_GUIDE.md) | Руководство разработчика | Frontend developers |
| [`COMPONENTS_GUIDE.md`](./COMPONENTS_GUIDE.md) | Гайд по компонентам | Разработчики |

### 4. Продукт
| Файл | Описание | Для кого |
|------|----------|----------|
| [`DEMO_SCENARIOS.md`](./DEMO_SCENARIOS.md) | Пользовательские сценарии | PM, QA, дизайнеры |

---

## 🗂️ Структура проекта

```
BarzhaFit/
│
├── 📄 Документация (8 файлов)
│   ├── README.md              # Главное README
│   ├── INDEX.md               # Этот файл
│   ├── QUICK_START.md         # Быстрый старт
│   ├── PROJECT_SUMMARY.md     # Итоговая сводка
│   ├── DESIGN_SYSTEM.md       # Дизайн-система
│   ├── FIGMA_EXPORT.md        # Экспорт в Figma
│   ├── DEVELOPER_GUIDE.md     # Гайд разработчика
│   ├── COMPONENTS_GUIDE.md    # Гайд по компонентам
│   └── DEMO_SCENARIOS.md      # Сценарии использования
│
├── 🧩 Компоненты (16 файлов)
│   └── src/app/components/barzha/
│       ├── Card.tsx           # Базовая карточка
│       ├── Button.tsx         # Кнопки (5 вариантов)
│       ├── Input.tsx          # Input & Textarea
│       ├── Select.tsx         # Dropdown
│       ├── ProgressBar.tsx    # Прогресс-бар
│       ├── Toast.tsx          # Уведомления
│       ├── EmptyState.tsx     # Пустые состояния
│       ├── LoadingSpinner.tsx # Загрузка
│       └── screens/
│           ├── TodayScreen.tsx    # 🏠 Сегодня
│           ├── FoodScreen.tsx     # 🍽️ Еда
│           ├── PlanScreen.tsx     # 📅 План
│           ├── GoalsScreen.tsx    # 🎯 Цели
│           ├── StepsScreen.tsx    # 👟 Шаги
│           ├── ProfileScreen.tsx  # 👤 Профиль
│           ├── StatsScreen.tsx    # 📊 Статистика
│           └── StreaksScreen.tsx  # 🔥 Серии
│
└── 📱 Главный компонент
    └── src/app/App.tsx        # Навигация + роутинг
```

---

## 🎨 Функционал по экранам

### 🏠 Сегодня (TodayScreen)
- План дня
- Статус тренировки (✅/❌/—)
- Прогресс по 5 показателям
- Список приёмов пищи
- **Файл:** [`TodayScreen.tsx`](./src/app/components/barzha/screens/TodayScreen.tsx)

### 🍽️ Еда (FoodScreen)
- Добавление приёмов пищи
- Список за сегодня
- Итоговые показатели
- Удаление приёмов
- **Файл:** [`FoodScreen.tsx`](./src/app/components/barzha/screens/FoodScreen.tsx)

### 📅 План (PlanScreen)
- Редактирование плана на неделю
- Многострочный текст
- Сохранение/сброс
- **Файл:** [`PlanScreen.tsx`](./src/app/components/barzha/screens/PlanScreen.tsx)

### 🎯 Цели (GoalsScreen)
- Установка целей (калории, БЖУ, шаги)
- Пересчёт из профиля
- Текущий прогресс
- **Файл:** [`GoalsScreen.tsx`](./src/app/components/barzha/screens/GoalsScreen.tsx)

### 👟 Шаги (StepsScreen)
- Ввод шагов
- Прогресс к цели
- История за неделю
- **Файл:** [`StepsScreen.tsx`](./src/app/components/barzha/screens/StepsScreen.tsx)

### 👤 Профиль (ProfileScreen)
- Форма данных пользователя
- Расчётные показатели
- Выбор цели (cut/balance/bulk)
- **Файл:** [`ProfileScreen.tsx`](./src/app/components/barzha/screens/ProfileScreen.tsx)

### 📊 Статистика (StatsScreen)
- Недельная таблица
- Месячный календарь
- Средние показатели
- **Файл:** [`StatsScreen.tsx`](./src/app/components/barzha/screens/StatsScreen.tsx)

### 🔥 Серии (StreaksScreen)
- Серии тренировок и питания
- Визуализация прогресса
- История серий
- Достижения
- **Файл:** [`StreaksScreen.tsx`](./src/app/components/barzha/screens/StreaksScreen.tsx)

---

## 🧩 Компоненты

### Базовые
- **Card** - универсальная карточка
- **Button** - 5 вариантов (primary, secondary, success, danger, ghost)
- **Input** - text, number, textarea
- **Select** - выпадающий список
- **ProgressBar** - с индикаторами 🟢🟡🔴

### Состояния
- **Toast** - уведомления
- **EmptyState** - пустые состояния
- **LoadingSpinner** - загрузка

**Детали:** [`COMPONENTS_GUIDE.md`](./COMPONENTS_GUIDE.md)

---

## 🎨 Дизайн-система

### Цвета
- Primary: `#030213`
- Background: `#ffffff`
- 5 Chart colors для графиков

### Типографика
- H2: 20px / medium
- H3: 18px / medium
- Body: 16px / normal

### Spacing
- Базовая сетка: 8px
- Padding: 16px
- Gap: 12-16px

**Детали:** [`DESIGN_SYSTEM.md`](./DESIGN_SYSTEM.md)

---

## 📊 Статистика проекта

| Метрика | Значение |
|---------|----------|
| Экранов | 8 |
| Компонентов | 8 базовых + 8 экранов |
| Строк кода | ~2,100 |
| Строк документации | ~1,500 |
| Файлов документации | 8 |
| Языков | TypeScript, CSS |
| Framework | React 18 |
| Стили | Tailwind CSS v4 |

---

## ✅ Чеклист готовности

### Дизайн
- [x] Цветовая палитра
- [x] Типографика
- [x] Spacing система
- [x] Компоненты
- [x] 8 экранов
- [x] Responsive дизайн
- [x] Эмодзи и иконки

### Разработка
- [x] React компоненты
- [x] TypeScript типизация
- [x] Tailwind стили
- [x] Навигация
- [x] Состояния (useState)
- [x] Responsive layout
- [x] Touch-friendly UI

### Документация
- [x] README
- [x] Design System
- [x] Components Guide
- [x] Developer Guide
- [x] Figma Export
- [x] Demo Scenarios
- [x] Quick Start
- [x] Project Summary

### Тестирование
- [x] Все экраны работают
- [x] Навигация функционирует
- [x] Формы интерактивны
- [x] Кнопки кликабельны
- [x] Responsive проверен

---

## 🚀 Следующие шаги

### Backend (не реализовано)
- [ ] API endpoints
- [ ] База данных
- [ ] Telegram Bot API
- [ ] Аутентификация

### Интеграции (не реализовано)
- [ ] Telegram WebApp SDK
- [ ] Push-уведомления
- [ ] Analytics
- [ ] Облачная синхронизация

### Дополнительно (опционально)
- [ ] Фото приёмов пищи
- [ ] AI рекомендации
- [ ] Социальные функции
- [ ] Интеграция с носимыми устройствами

---

## 💡 Полезные команды

```bash
# Запуск dev сервера
npm run dev

# Сборка production
npm run build

# Предпросмотр production
npm run preview
```

---

## 📞 Техническая информация

**Проект:** BarzhaFit Telegram Mini App  
**Версия:** 1.0.0  
**Статус:** ✅ MVP Ready  
**Дата создания:** Январь 2026  
**Платформа:** Telegram Mini App (WebView)  
**Целевые устройства:** iOS/Android (360-600px)  

---

## 🎉 Готово к использованию!

Проект полностью реализован и готов к:
- ✅ Тестированию
- ✅ Экспорту в Figma
- ✅ Разработке бэкенда
- ✅ Деплою

**Начните с [`QUICK_START.md`](./QUICK_START.md) для быстрого старта!** 🚀

---

**BarzhaFit** - ваш персональный фитнес-трекер в Telegram! 💪
