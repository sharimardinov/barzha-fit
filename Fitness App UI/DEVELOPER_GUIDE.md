# 👨‍💻 Руководство для разработчика

## Быстрый старт

Приложение уже полностью работает. Просто откройте его и переключайтесь между экранами через нижнюю навигацию.

## Структура приложения

```
src/app/
├── App.tsx                          # Главный компонент с навигацией
└── components/barzha/               # Все компоненты BarzhaFit
    ├── Card.tsx                     # Базовая карточка
    ├── Button.tsx                   # Кнопки (5 вариантов)
    ├── Input.tsx                    # Input и Textarea
    ├── Select.tsx                   # Выпадающий список
    ├── ProgressBar.tsx              # Прогресс-бар с индикатором
    ├── Toast.tsx                    # Уведомления
    ├── EmptyState.tsx               # Пустые состояния
    ├── LoadingSpinner.tsx           # Загрузка
    └── screens/                     # 8 экранов приложения
        ├── TodayScreen.tsx          # Главный экран
        ├── FoodScreen.tsx           # Еда
        ├── PlanScreen.tsx           # План
        ├── GoalsScreen.tsx          # Цели
        ├── StepsScreen.tsx          # Шаги
        ├── ProfileScreen.tsx        # Профиль
        ├── StatsScreen.tsx          # Статистика
        └── StreaksScreen.tsx        # Серии
```

## Добавление нового экрана

1. Создайте компонент в `screens/`:
```tsx
// screens/NewScreen.tsx
import { Card } from '../Card';
import { Button } from '../Button';

export function NewScreen() {
  return (
    <div className="space-y-4">
      <Card>
        <h3>Новый экран</h3>
        <p>Контент...</p>
      </Card>
    </div>
  );
}
```

2. Добавьте экран в `App.tsx`:
```tsx
import { NewScreen } from '@/app/components/barzha/screens/NewScreen';

// В type Screen:
type Screen = 'today' | 'food' | ... | 'newScreen';

// В screens объект:
const screens = {
  // ...
  newScreen: { title: 'Новое', icon: Plus, component: NewScreen },
};

// В навигацию:
<NavButton
  icon={Plus}
  label="Новое"
  active={activeScreen === 'newScreen'}
  onClick={() => setActiveScreen('newScreen')}
/>
```

## Использование компонентов

### Button
```tsx
<Button variant="success" fullWidth onClick={handleClick}>
  ✅ Сохранить
</Button>
```

### Input
```tsx
const [value, setValue] = useState('');

<Input
  label="Название"
  value={value}
  onChange={setValue}
  placeholder="Введите..."
/>
```

### ProgressBar
```tsx
<ProgressBar
  label="Калории"
  current={1450}
  target={2000}
  unit="ккал"
  color="var(--chart-1)"
/>
```

### Toast
```tsx
import { useToast } from '@/app/components/barzha/Toast';

function MyComponent() {
  const { showToast, ToastComponent } = useToast();

  const handleSave = () => {
    // ... логика сохранения
    showToast('Сохранено!', 'success');
  };

  return (
    <>
      <Button onClick={handleSave}>Сохранить</Button>
      {ToastComponent}
    </>
  );
}
```

## Работа с данными

Сейчас все данные хранятся в `useState` внутри компонентов. Для реального приложения нужно:

### 1. Создать Store (Zustand или Context)

```tsx
// stores/useAppStore.ts
import { create } from 'zustand';

interface AppState {
  workoutStatus: 'done' | 'skipped' | 'pending';
  setWorkoutStatus: (status: 'done' | 'skipped' | 'pending') => void;
  meals: Meal[];
  addMeal: (meal: Meal) => void;
  // ...
}

export const useAppStore = create<AppState>((set) => ({
  workoutStatus: 'pending',
  setWorkoutStatus: (status) => set({ workoutStatus: status }),
  meals: [],
  addMeal: (meal) => set((state) => ({ meals: [...state.meals, meal] })),
}));
```

### 2. Подключить API

```tsx
// services/api.ts
export async function saveMeal(meal: Meal) {
  const response = await fetch('/api/meals', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(meal)
  });
  return response.json();
}

// В компоненте:
const handleSave = async () => {
  try {
    await saveMeal(newMeal);
    showToast('Сохранено!', 'success');
  } catch (error) {
    showToast('Ошибка сохранения', 'error');
  }
};
```

### 3. Telegram WebApp API

```tsx
// utils/telegram.ts
export const tg = window.Telegram?.WebApp;

export function initTelegramWebApp() {
  tg?.ready();
  tg?.expand();
  
  // Получить данные пользователя
  const user = tg?.initDataUnsafe?.user;
  
  // Показать главную кнопку
  tg?.MainButton.setText('Сохранить');
  tg?.MainButton.show();
  tg?.MainButton.onClick(handleSave);
}

// В App.tsx:
useEffect(() => {
  initTelegramWebApp();
}, []);
```

## Валидация форм

Используйте react-hook-form (уже установлен):

```tsx
import { useForm } from 'react-hook-form';

function ProfileScreen() {
  const { register, handleSubmit, formState: { errors } } = useForm();

  const onSubmit = (data) => {
    console.log(data);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <Input
        label="Вес"
        {...register('weight', { required: true, min: 30, max: 200 })}
        error={errors.weight && 'Введите корректный вес'}
      />
      <Button type="submit">Сохранить</Button>
    </form>
  );
}
```

## Адаптивность

Все компоненты уже адаптивны. Основные брейкпоинты:
- Mobile: default (360-430px)
- Tablet: используйте `max-w-[600px] mx-auto`

```tsx
<div className="max-w-[600px] mx-auto w-full">
  {/* Контент центрируется на больших экранах */}
</div>
```

## Оптимизация

### 1. Lazy loading экранов

```tsx
import { lazy, Suspense } from 'react';
import { LoadingState } from '@/app/components/barzha/LoadingSpinner';

const TodayScreen = lazy(() => import('./screens/TodayScreen'));

// В App:
<Suspense fallback={<LoadingState />}>
  <CurrentScreen />
</Suspense>
```

### 2. Мемоизация

```tsx
import { memo } from 'react';

export const ProgressBar = memo(({ label, current, target, unit, color }) => {
  // ...
});
```

### 3. useCallback для обработчиков

```tsx
const handleDelete = useCallback((id: number) => {
  // ...
}, []);
```

## Тестирование

```tsx
// __tests__/Button.test.tsx
import { render, screen, fireEvent } from '@testing-library/react';
import { Button } from '@/app/components/barzha/Button';

test('кнопка вызывает onClick', () => {
  const handleClick = jest.fn();
  render(<Button onClick={handleClick}>Нажми</Button>);
  
  fireEvent.click(screen.getByText('Нажми'));
  expect(handleClick).toHaveBeenCalled();
});
```

## Деплой

### Vercel
```bash
npm run build
vercel deploy
```

### Netlify
```bash
npm run build
netlify deploy --prod
```

### GitHub Pages
```bash
npm run build
gh-pages -d dist
```

## Типы данных

Создайте файл с типами:

```tsx
// types/index.ts
export interface Meal {
  id: number;
  time: string;
  description: string;
  calories: number;
  protein: number;
  fat: number;
  carbs: number;
  date: string;
}

export interface Goal {
  calories: number;
  protein: number;
  fat: number;
  carbs: number;
  steps: number;
}

export interface Profile {
  gender: 'male' | 'female';
  age: number;
  height: number;
  weight: number;
  bodyFat: number;
  activity: 'sedentary' | 'light' | 'moderate' | 'active' | 'very_active';
  goal: 'cut' | 'balance' | 'bulk';
}

export interface WorkoutDay {
  date: string;
  completed: boolean;
  skipped: boolean;
}

export interface Streak {
  type: 'workout' | 'food';
  current: number;
  best: number;
  history: { start: string; end: string; length: number }[];
}
```

## Полезные хуки

```tsx
// hooks/useLocalStorage.ts
export function useLocalStorage<T>(key: string, initialValue: T) {
  const [value, setValue] = useState<T>(() => {
    const item = window.localStorage.getItem(key);
    return item ? JSON.parse(item) : initialValue;
  });

  useEffect(() => {
    window.localStorage.setItem(key, JSON.stringify(value));
  }, [key, value]);

  return [value, setValue] as const;
}

// Использование:
const [goals, setGoals] = useLocalStorage('goals', defaultGoals);
```

## Debugging

```tsx
// Включить React DevTools
// Использовать console.log для отладки состояния

useEffect(() => {
  console.log('Current screen:', activeScreen);
}, [activeScreen]);

// Показать границы элементов (для проверки layout)
// Добавьте в tailwind.config.js:
// * { @apply border border-red-500; }
```

## Troubleshooting

**Проблема:** Компоненты не обновляются  
**Решение:** Проверьте, что useState используется правильно

**Проблема:** Стили не применяются  
**Решение:** Убедитесь что используете правильные Tailwind классы

**Проблема:** Навигация не работает  
**Решение:** Проверьте, что activeScreen установлен правильно

---

**Готово к разработке!** 🚀

Все компоненты работают, стили применены, навигация настроена.
