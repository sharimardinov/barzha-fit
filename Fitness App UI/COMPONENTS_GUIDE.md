# BarzhaFit Components Guide

## 🧩 Базовые компоненты

### Card
```tsx
import { Card } from '@/app/components/barzha/Card';

<Card>
  <h3>Заголовок</h3>
  <p>Содержимое карточки</p>
</Card>

<Card className="bg-primary/5">
  <p>Карточка с кастомным фоном</p>
</Card>
```

### Button
```tsx
import { Button } from '@/app/components/barzha/Button';

// Варианты
<Button variant="primary">Основная кнопка</Button>
<Button variant="secondary">Вторичная</Button>
<Button variant="success">Успех</Button>
<Button variant="danger">Опасность</Button>
<Button variant="ghost">Прозрачная</Button>

// Размеры
<Button size="sm">Маленькая</Button>
<Button size="md">Средняя</Button>
<Button size="lg">Большая</Button>

// На всю ширину
<Button fullWidth>Полная ширина</Button>

// С действием
<Button onClick={() => console.log('Clicked!')}>
  Нажми меня
</Button>
```

### Input
```tsx
import { Input } from '@/app/components/barzha/Input';

// Обычный input
<Input
  label="Название"
  value={value}
  onChange={setValue}
  placeholder="Введите текст"
/>

// Число
<Input
  label="Количество"
  type="number"
  value={count}
  onChange={setCount}
/>

// Textarea
<Input
  label="Описание"
  type="textarea"
  value={description}
  onChange={setDescription}
  rows={6}
/>

// С ошибкой
<Input
  label="Email"
  value={email}
  onChange={setEmail}
  error="Неверный формат email"
/>
```

### Select
```tsx
import { Select } from '@/app/components/barzha/Select';

<Select
  label="Выберите цель"
  value={goal}
  onChange={setGoal}
  options={[
    { value: 'cut', label: '🔥 Сушка' },
    { value: 'balance', label: '⚖️ Поддержка' },
    { value: 'bulk', label: '💪 Набор массы' }
  ]}
/>
```

### ProgressBar
```tsx
import { ProgressBar } from '@/app/components/barzha/ProgressBar';

<ProgressBar
  label="Калории"
  current={1450}
  target={2000}
  unit="ккал"
  color="var(--chart-1)"
  showIndicator={true}
/>
```

## 🎭 Состояния

### EmptyState
```tsx
import { EmptyState } from '@/app/components/barzha/EmptyState';

<EmptyState
  icon="📋"
  title="Нет данных"
  description="Создайте первую запись, чтобы начать"
  actionLabel="Создать"
  onAction={() => console.log('Create')}
/>
```

### LoadingSpinner
```tsx
import { LoadingSpinner, LoadingState } from '@/app/components/barzha/LoadingSpinner';

// Только спиннер
<LoadingSpinner />

// Полное состояние загрузки
<LoadingState />
```

### Toast
```tsx
import { useToast } from '@/app/components/barzha/Toast';

function MyComponent() {
  const { showToast, ToastComponent } = useToast();

  return (
    <>
      <Button onClick={() => showToast('Сохранено!', 'success')}>
        Сохранить
      </Button>
      <Button onClick={() => showToast('Ошибка!', 'error')}>
        Ошибка
      </Button>
      <Button onClick={() => showToast('Информация', 'info')}>
        Инфо
      </Button>
      {ToastComponent}
    </>
  );
}
```

## 📱 Экраны

### Структура экрана
```tsx
export function MyScreen() {
  return (
    <div className="space-y-4">
      <Card>
        <h3 className="mb-4">Секция 1</h3>
        {/* Контент */}
      </Card>

      <Card>
        <h3 className="mb-4">Секция 2</h3>
        {/* Контент */}
      </Card>
    </div>
  );
}
```

### Форма с кнопками
```tsx
<Card>
  <h3 className="mb-4">Редактирование</h3>
  <div className="space-y-4">
    <Input label="Поле 1" value={val1} onChange={setVal1} />
    <Input label="Поле 2" value={val2} onChange={setVal2} />
  </div>
  <div className="flex gap-2 mt-6">
    <Button variant="success" fullWidth>
      💾 Сохранить
    </Button>
    <Button variant="secondary" fullWidth>
      ❌ Отмена
    </Button>
  </div>
</Card>
```

### Список с удалением
```tsx
<Card>
  <h3 className="mb-3">Список</h3>
  <div className="space-y-3">
    {items.map(item => (
      <div key={item.id} className="flex items-start justify-between p-3 bg-muted rounded-xl">
        <div className="flex-1">
          <div className="text-xs text-muted-foreground mb-1">
            {item.time}
          </div>
          <div className="text-sm">{item.description}</div>
        </div>
        <button 
          onClick={() => handleDelete(item.id)}
          className="text-red-500 min-w-[44px] min-h-[44px] flex items-center justify-center -mr-2"
        >
          🗑️
        </button>
      </div>
    ))}
  </div>
</Card>
```

### Статистика
```tsx
<Card className="bg-primary/5">
  <h4 className="mb-3">📊 Статистика</h4>
  <div className="grid grid-cols-2 gap-3 text-sm">
    <div>
      <div className="text-muted-foreground">Калории</div>
      <div className="font-medium">2,050 ккал</div>
    </div>
    <div>
      <div className="text-muted-foreground">Белок</div>
      <div className="font-medium">145 г</div>
    </div>
  </div>
</Card>
```

## 🎨 Tailwind классы

### Spacing
- `space-y-4` - вертикальный отступ между элементами
- `gap-2`, `gap-3`, `gap-4` - отступы в grid/flex
- `p-4` - padding 16px
- `px-4 py-3` - padding horizontal/vertical
- `mb-4`, `mt-4` - margin bottom/top

### Layout
- `flex` - flexbox
- `grid grid-cols-2` - grid 2 колонки
- `items-center` - выравнивание по центру
- `justify-between` - пространство между элементами

### Размеры
- `w-full` - ширина 100%
- `h-full` - высота 100%
- `min-h-[44px]` - минимальная высота
- `max-w-[600px]` - максимальная ширина

### Цвета
- `bg-card` - фон карточки
- `bg-muted` - приглушенный фон
- `bg-primary` - основной цвет
- `text-muted-foreground` - приглушенный текст
- `text-foreground` - основной текст

### Скругление
- `rounded-xl` - 12px
- `rounded-2xl` - 16px
- `rounded-full` - полностью круглый

### Прочее
- `transition-all` - плавный переход
- `hover:opacity-90` - эффект при наведении
- `truncate` - обрезка текста с ...
- `whitespace-pre-line` - сохранение переносов строк
