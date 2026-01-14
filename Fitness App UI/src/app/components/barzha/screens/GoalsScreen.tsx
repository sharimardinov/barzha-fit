import { useState } from 'react';
import { Card } from '../Card';
import { Button } from '../Button';
import { Input } from '../Input';

export function GoalsScreen() {
  const [calories, setCalories] = useState('2000');
  const [protein, setProtein] = useState('150');
  const [fat, setFat] = useState('60');
  const [carbs, setCarbs] = useState('200');
  const [steps, setSteps] = useState('10000');

  return (
    <div className="space-y-4">
      <Card>
        <h3 className="mb-4">🎯 Дневные цели</h3>
        <div className="space-y-4">
          <Input
            label="🔥 Калории (ккал)"
            type="number"
            value={calories}
            onChange={setCalories}
            placeholder="2000"
          />
          <Input
            label="🥩 Белок (г)"
            type="number"
            value={protein}
            onChange={setProtein}
            placeholder="150"
          />
          <Input
            label="🧈 Жиры (г)"
            type="number"
            value={fat}
            onChange={setFat}
            placeholder="60"
          />
          <Input
            label="🍞 Углеводы (г)"
            type="number"
            value={carbs}
            onChange={setCarbs}
            placeholder="200"
          />
          <Input
            label="👟 Шаги"
            type="number"
            value={steps}
            onChange={setSteps}
            placeholder="10000"
          />
        </div>
        <div className="flex gap-2 mt-6">
          <Button variant="success" fullWidth>
            💾 Сохранить
          </Button>
          <Button variant="secondary" fullWidth>
            🔄 Из профиля
          </Button>
        </div>
      </Card>

      <Card className="bg-primary/5">
        <h4 className="mb-2">📊 Текущий прогресс</h4>
        <div className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Калории</span>
            <span>1450 / 2000 (73%)</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Белок</span>
            <span>85 / 150 (57%)</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Жиры</span>
            <span>40 / 60 (67%)</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Углеводы</span>
            <span>120 / 200 (60%)</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Шаги</span>
            <span>6500 / 10000 (65%)</span>
          </div>
        </div>
      </Card>
    </div>
  );
}
