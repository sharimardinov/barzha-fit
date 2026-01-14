import { useState } from 'react';
import { Card } from '../Card';
import { Button } from '../Button';
import { Input } from '../Input';
import { Select } from '../Select';

export function ProfileScreen() {
  const [gender, setGender] = useState('male');
  const [age, setAge] = useState('25');
  const [height, setHeight] = useState('175');
  const [weight, setWeight] = useState('75');
  const [bodyFat, setBodyFat] = useState('15');
  const [activity, setActivity] = useState('moderate');
  const [goal, setGoal] = useState('cut');

  return (
    <div className="space-y-4">
      <Card>
        <h3 className="mb-4">👤 Профиль</h3>
        <div className="space-y-4">
          <Select
            label="⚧️ Пол"
            value={gender}
            onChange={setGender}
            options={[
              { value: 'male', label: 'Мужской' },
              { value: 'female', label: 'Женский' }
            ]}
          />
          <Input
            label="🎂 Возраст (лет)"
            type="number"
            value={age}
            onChange={setAge}
            placeholder="25"
          />
          <Input
            label="📏 Рост (см)"
            type="number"
            value={height}
            onChange={setHeight}
            placeholder="175"
          />
          <Input
            label="⚖️ Вес (кг)"
            type="number"
            value={weight}
            onChange={setWeight}
            placeholder="75"
          />
          <Input
            label="📊 Процент жира (%)"
            type="number"
            value={bodyFat}
            onChange={setBodyFat}
            placeholder="15"
          />
          <Select
            label="🏃 Уровень активности"
            value={activity}
            onChange={setActivity}
            options={[
              { value: 'sedentary', label: 'Сидячий образ' },
              { value: 'light', label: 'Лёгкая активность' },
              { value: 'moderate', label: 'Умеренная активность' },
              { value: 'active', label: 'Активный' },
              { value: 'very_active', label: 'Очень активный' }
            ]}
          />
          <Select
            label="🎯 Цель"
            value={goal}
            onChange={setGoal}
            options={[
              { value: 'cut', label: '🔥 Сушка (Cut)' },
              { value: 'balance', label: '⚖️ Поддержка (Balance)' },
              { value: 'bulk', label: '💪 Набор массы (Bulk)' }
            ]}
          />
        </div>
        <Button variant="success" fullWidth className="mt-6">
          💾 Сохранить профиль
        </Button>
      </Card>

      <Card className="bg-primary/5">
        <h4 className="mb-3">📊 Расчётные показатели</h4>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">BMI</span>
            <span className="font-medium">24.5 (Норма)</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Базовый метаболизм</span>
            <span className="font-medium">1,750 ккал</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Дневная норма</span>
            <span className="font-medium">2,450 ккал</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Для цели (Cut)</span>
            <span className="font-medium">2,000 ккал</span>
          </div>
        </div>
      </Card>
    </div>
  );
}
