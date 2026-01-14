import { useState } from 'react';
import { Card } from '../Card';
import { Button } from '../Button';
import { Input } from '../Input';

export function FoodScreen() {
  const [foodText, setFoodText] = useState('');

  return (
    <div className="space-y-4">
      {/* Добавление еды */}
      <Card>
        <h3 className="mb-4">🍽️ Добавить еду</h3>
        <Input
          type="textarea"
          value={foodText}
          onChange={setFoodText}
          placeholder="Например:&#10;Куриная грудка 200г&#10;Гречка 150г&#10;Овощи 100г"
          rows={6}
        />
        <Button variant="success" fullWidth className="mt-4">
          ✅ Добавить
        </Button>
      </Card>

      {/* Приёмы за сегодня */}
      <Card>
        <h3 className="mb-4">📝 Приёмы пищи сегодня</h3>
        <div className="space-y-3">
          <div className="p-3 bg-muted rounded-xl">
            <div className="flex items-start justify-between mb-2">
              <div className="text-xs text-muted-foreground">08:30</div>
              <button className="text-red-500 min-w-[44px] min-h-[44px] flex items-center justify-center -mr-2 -mt-2">
                🗑️
              </button>
            </div>
            <div className="text-sm mb-2">Овсянка 100г, банан, молоко</div>
            <div className="text-xs text-muted-foreground">
              450 ккал • Б: 15г Ж: 10г У: 70г
            </div>
          </div>

          <div className="p-3 bg-muted rounded-xl">
            <div className="flex items-start justify-between mb-2">
              <div className="text-xs text-muted-foreground">13:00</div>
              <button className="text-red-500 min-w-[44px] min-h-[44px] flex items-center justify-center -mr-2 -mt-2">
                🗑️
              </button>
            </div>
            <div className="text-sm mb-2">Куриная грудка 200г, гречка 150г, овощи</div>
            <div className="text-xs text-muted-foreground">
              550 ккал • Б: 45г Ж: 12г У: 60г
            </div>
          </div>

          <div className="p-3 bg-muted rounded-xl">
            <div className="flex items-start justify-between mb-2">
              <div className="text-xs text-muted-foreground">18:30</div>
              <button className="text-red-500 min-w-[44px] min-h-[44px] flex items-center justify-center -mr-2 -mt-2">
                🗑️
              </button>
            </div>
            <div className="text-sm mb-2">Рыба 150г, салат</div>
            <div className="text-xs text-muted-foreground">
              350 ккал • Б: 30г Ж: 15г У: 20г
            </div>
          </div>
        </div>
      </Card>

      {/* Итого за день */}
      <Card className="bg-primary/5">
        <h4 className="mb-3">📊 Итого за день</h4>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <div className="text-muted-foreground">Калории</div>
            <div className="font-medium">1350 ккал</div>
          </div>
          <div>
            <div className="text-muted-foreground">Белок</div>
            <div className="font-medium">90 г</div>
          </div>
          <div>
            <div className="text-muted-foreground">Жиры</div>
            <div className="font-medium">35 г</div>
          </div>
          <div>
            <div className="text-muted-foreground">Углеводы</div>
            <div className="font-medium">150 г</div>
          </div>
        </div>
      </Card>
    </div>
  );
}
