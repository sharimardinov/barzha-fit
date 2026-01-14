import { useState } from 'react';
import { Card } from '../Card';
import { Button } from '../Button';
import { Input } from '../Input';

export function StepsScreen() {
  const [steps, setSteps] = useState('');

  return (
    <div className="space-y-4">
      <Card>
        <h3 className="mb-4">👟 Шаги за сегодня</h3>
        <Input
          label="Количество шагов"
          type="number"
          value={steps}
          onChange={setSteps}
          placeholder="Например: 8500"
        />
        <Button variant="success" fullWidth className="mt-4">
          ✅ Сохранить
        </Button>
      </Card>

      <Card>
        <h4 className="mb-3">📊 Прогресс</h4>
        <div className="text-center mb-4">
          <div className="text-4xl font-bold mb-2">6,500</div>
          <div className="text-sm text-muted-foreground">из 10,000 шагов</div>
        </div>
        <div className="w-full bg-muted rounded-full h-3 overflow-hidden">
          <div 
            className="h-full rounded-full transition-all duration-500"
            style={{ 
              width: '65%',
              backgroundColor: 'var(--chart-5)' 
            }}
          />
        </div>
        <div className="text-center mt-3 text-sm text-muted-foreground">
          65% от цели
        </div>
      </Card>

      <Card>
        <h4 className="mb-3">📅 История за неделю</h4>
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Понедельник</span>
            <span>8,500 шагов</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Вторник</span>
            <span>7,200 шагов</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Среда</span>
            <span>9,800 шагов</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Четверг</span>
            <span>8,800 шагов</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Пятница</span>
            <span>10,500 шагов ✅</span>
          </div>
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Суббота</span>
            <span>11,200 шагов ✅</span>
          </div>
          <div className="flex items-center justify-between text-sm font-medium">
            <span>Сегодня</span>
            <span>6,500 шагов</span>
          </div>
        </div>
      </Card>
    </div>
  );
}
