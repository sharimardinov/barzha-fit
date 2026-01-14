import { useState } from 'react';
import { Card } from '../Card';
import { Button } from '../Button';
import { Input } from '../Input';

export function PlanScreen() {
  const [plan, setPlan] = useState(`День 1: Грудь, трицепс
Утро: овсянка, банан
Обед: курица, гречка
Ужин: рыба, овощи

День 2: Спина, бицепс
Утро: яйца, хлеб
Обед: говядина, рис
Ужин: творог, орехи

День 3: Ноги, плечи
Утро: омлет, фрукты
Обед: индейка, паста
Ужин: курица, салат`);

  return (
    <div className="space-y-4">
      <Card>
        <h3 className="mb-4">📅 План на неделю</h3>
        <Input
          type="textarea"
          value={plan}
          onChange={setPlan}
          placeholder="Опишите план на 7 дней..."
          rows={12}
        />
        <div className="flex gap-2 mt-4">
          <Button variant="success" fullWidth>
            💾 Сохранить
          </Button>
          <Button variant="secondary" fullWidth>
            🔄 Сбросить
          </Button>
        </div>
      </Card>

      <Card className="bg-muted/50">
        <h4 className="mb-2">💡 Совет</h4>
        <p className="text-sm text-muted-foreground">
          Распишите план тренировок и питания на каждый день недели. 
          Это поможет вам следовать структуре и не пропускать важные моменты.
        </p>
      </Card>
    </div>
  );
}
