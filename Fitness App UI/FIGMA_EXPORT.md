# 🎨 Экспорт в Figma

## Что нужно экспортировать из кода в Figma

### 1. Цвета (Color Styles)

**Light Theme:**
```
Primary: #030213
Background: #ffffff
Card: #ffffff
Muted: #ececf0
Muted Foreground: #717182
Border: rgba(0, 0, 0, 0.1)
Input Background: #f3f3f5

Chart-1 (Калории): oklch(0.646 0.222 41.116) → #FF7A3D
Chart-2 (Белок): oklch(0.6 0.118 184.704) → #5BA8C8
Chart-3 (Жиры): oklch(0.398 0.07 227.392) → #495D87
Chart-4 (Углеводы): oklch(0.828 0.189 84.429) → #D4E157
Chart-5 (Шаги): oklch(0.769 0.188 70.08) → #FFB74D

Success: #22c55e
Danger: #ef4444
Warning: #f59e0b
```

### 2. Типографика (Text Styles)

**H2:** 20px / Line height 1.5 / Weight 500 / Color: Foreground  
**H3:** 18px / Line height 1.5 / Weight 500 / Color: Foreground  
**H4:** 16px / Line height 1.5 / Weight 500 / Color: Foreground  
**Body:** 16px / Line height 1.5 / Weight 400 / Color: Foreground  
**Body Small:** 14px / Line height 1.5 / Weight 400 / Color: Foreground  
**Caption:** 12px / Line height 1.5 / Weight 400 / Color: Muted Foreground  
**Caption Bold:** 12px / Line height 1.5 / Weight 500 / Color: Foreground  

### 3. Spacing (Размеры для Auto Layout)

**Base unit: 8px**

- 4px (0.5)
- 8px (1)
- 12px (1.5)
- 16px (2)
- 24px (3)
- 32px (4)
- 48px (6)

### 4. Border Radius

- 8px - small (sm)
- 12px - medium (xl)
- 16px - large (2xl)
- 9999px - full (круглый)

### 5. Компоненты (Components)

#### Card
- Width: Fill container
- Padding: 16px
- Border radius: 16px
- Background: Card
- Border: 1px solid Border

#### Button Variants
**Primary:**
- Min height: 44px
- Padding: 12px 16px
- Border radius: 12px
- Background: Primary
- Text: Primary Foreground

**Secondary:**
- Background: Muted
- Text: Foreground

**Success:**
- Background: #22c55e
- Text: White

**Danger:**
- Background: #ef4444
- Text: White

**Ghost:**
- Background: Transparent
- Text: Foreground
- Hover: Background Muted

#### Input/Textarea
- Min height: 44px
- Padding: 12px 16px
- Border radius: 12px
- Background: Input Background (#f3f3f5)
- Border: 1px solid Border
- Text: 16px / 400

#### Progress Bar
- Height: 8px
- Border radius: 9999px (full)
- Background: Muted
- Fill: Chart colors
- Width: Fill container

### 6. Иконки

**Источник:** Lucide Icons (https://lucide.dev)

Используемые иконки:
- Home (Сегодня)
- Utensils (Еда)
- Calendar (План)
- Target (Цели)
- Activity (Шаги)
- User (Профиль)
- BarChart2 (Статистика)
- Flame (Серии)

**Размеры:** 20px (size-5), 24px (size-6)

### 7. Навигация

**Bottom Navigation:**
- Fixed bottom
- Height: ~76px (с учётом safe area)
- Padding top: 8px
- Padding bottom: 24px (safe area)
- Grid: 4 columns × 2 rows
- Gap: 4px

**Nav Button:**
- Min size: 60px height
- Padding: 8px 4px
- Border radius: 12px
- Icon: 20px
- Label: 10px

**Active state:**
- Background: Primary with 10% opacity
- Text: Primary

**Inactive state:**
- Text: Muted Foreground

### 8. Frames для экранов

**Размер frames:**
- Width: 390px (стандартный iPhone)
- Height: Auto (не фиксировать)

**Альтернативные размеры:**
- iPhone SE: 375px
- Android: 360px, 412px
- iPad Mini: 600px (max-width)

### 9. Auto Layout настройки

**Вертикальные списки:**
- Direction: Vertical
- Gap: 12px или 16px
- Padding: 16px

**Горизонтальные кнопки:**
- Direction: Horizontal
- Gap: 8px
- Fill container

**Grid (2 колонки):**
- Columns: 2
- Gap: 12px

### 10. Effects (Эффекты)

**Card shadow (опционально):**
- Y: 1px
- Blur: 3px
- Color: #000000 with 5% opacity

**Button hover (для прототипа):**
- Opacity: 90%

**Transition:**
- Duration: 200ms
- Easing: Ease out

### 11. Эмодзи

Использовать нативные эмодзи для:
- Статусы (✅ ❌ 🟢 🟡 🔴)
- Категории (💪 🍽️ 🔥 📊 🎯 👟)
- Достижения (🏆 🥇 🥈 🥉)

### 12. Экраны для создания

1. **Today Screen** - главный с планом, тренировкой, прогрессом
2. **Food Screen** - добавление еды и список приёмов
3. **Plan Screen** - редактирование плана на неделю
4. **Goals Screen** - установка целей
5. **Steps Screen** - ввод и история шагов
6. **Profile Screen** - форма профиля
7. **Stats Screen** - недельная таблица + месячный календарь
8. **Streaks Screen** - серии с визуализацией

### 13. Состояния компонентов

Создать варианты (Variants) для:
- Button: primary, secondary, success, danger, ghost
- Input: default, error, filled
- Navigation item: active, inactive
- Workout status: pending, done, skipped
- Progress indicator: empty, in-progress, complete

---

## 📋 Чеклист экспорта

- [ ] Создать Color Styles (15 цветов)
- [ ] Создать Text Styles (7 стилей)
- [ ] Настроить Local Variables для spacing
- [ ] Создать компонент Card
- [ ] Создать компонент Button со всеми вариантами
- [ ] Создать компонент Input/Textarea
- [ ] Создать компонент Progress Bar
- [ ] Создать компонент Navigation Item
- [ ] Импортировать иконки из Lucide
- [ ] Создать 8 экранов
- [ ] Настроить прототип переключения экранов
- [ ] Добавить все состояния (empty, loading, error)
- [ ] Проверить responsive (375px, 390px, 600px)

---

**Время на создание в Figma:** ~4-6 часов для полного дизайна с компонентами и всеми экранами.
