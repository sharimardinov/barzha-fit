import { Card } from '../Card';
import { Button } from '../Button';
import { ProgressBar } from '../ProgressBar';
import { EmptyState } from '../EmptyState';
import { useState } from 'react';

export function TodayScreen() {
  const [hasPlan] = useState(true);
  const [workoutStatus, setWorkoutStatus] = useState<'done' | 'skipped' | 'pending'>('pending');
  const [meals] = useState([
    {
      id: 1,
      time: '08:30',
      description: 'Овсянка 100г, банан, молоко',
      calories: 450,
      protein: 15,
      fat: 10,
      carbs: 70
    },
    {
      id: 2,
      time: '13:00',
      description: 'Куриная грудка 200г, гречка 150г, овощи',
      calories: 550,
      protein: 45,
      fat: 12,
      carbs: 60
    }
  ]);

  if (!hasPlan) {
    return (
      <EmptyState
        icon="📋"
        title="Нет плана на сегодня"
        description="Создайте план тренировок и питания, чтобы начать отслеживание прогресса"
        actionLabel="Создать план"
        onAction={() => console.log('Navigate to plan')}
      />
    );
  }

  return (
    <div className="space-y-4">
      {/* План дня */}
      <Card>
        <h3 className="mb-3">📋 План дня</h3>
        <div className="text-sm text-muted-foreground leading-relaxed whitespace-pre-line">
          Утро: овсянка с фруктами{'\n'}
          Обед: куриная грудка с гречкой{'\n'}
          Вечер: рыба с овощами{'\n'}
          Тренировка: силовая (грудь, трицепс)
        </div>
      </Card>

      {/* Тренировка */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h3>💪 Тренировка</h3>
          <span className="text-2xl">
            {workoutStatus === 'done' ? '✅' : workoutStatus === 'skipped' ? '❌' : '—'}
          </span>
        </div>
        {workoutStatus === 'pending' && (
          <div className="flex gap-2">
            <Button variant="success" fullWidth onClick={() => setWorkoutStatus('done')}>
              ✅ Сделал
            </Button>
            <Button variant="danger" fullWidth onClick={() => setWorkoutStatus('skipped')}>
              ❌ Пропустил
            </Button>
          </div>
        )}
        {workoutStatus === 'done' && (
          <div className="text-center py-2 text-green-500 font-medium">
            Отлично! Тренировка выполнена 💪
          </div>
        )}
        {workoutStatus === 'skipped' && (
          <div className="text-center py-2 text-red-500 font-medium">
            Пропущено. Не сдавайся!
          </div>
        )}
      </Card>

      {/* Прогресс */}
      <Card>
        <h3 className="mb-4">📊 Прогресс</h3>
        <div className="space-y-4">
          <ProgressBar
            label="Калории"
            current={1450}
            target={2000}
            unit="ккал"
            color="var(--chart-1)"
          />
          <ProgressBar
            label="Белок"
            current={85}
            target={150}
            unit="г"
            color="var(--chart-2)"
          />
          <ProgressBar
            label="Жир"
            current={40}
            target={60}
            unit="г"
            color="var(--chart-3)"
          />
          <ProgressBar
            label="Углеводы"
            current={120}
            target={200}
            unit="г"
            color="var(--chart-4)"
          />
          <ProgressBar
            label="Шаги"
            current={6500}
            target={10000}
            unit="шагов"
            color="var(--chart-5)"
          />
        </div>
      </Card>

      {/* Приёмы пищи */}
      <Card>
        <h3 className="mb-3">🍽️ Еда сегодня</h3>
        {meals.length === 0 ? (
          <div className="text-center py-6 text-muted-foreground">
            <p className="mb-4">Приёмов пищи пока нет</p>
            <Button variant="primary" fullWidth>
              + Добавить приём пищи
            </Button>
          </div>
        ) : (
          <>
            <div className="space-y-3">
              {meals.map((meal) => (
                <div key={meal.id} className="flex items-start justify-between p-3 bg-muted rounded-xl">
                  <div className="flex-1">
                    <div className="text-xs text-muted-foreground mb-1">{meal.time}</div>
                    <div className="text-sm">{meal.description}</div>
                    <div className="text-xs text-muted-foreground mt-1">
                      {meal.calories} ккал • Б: {meal.protein}г Ж: {meal.fat}г У: {meal.carbs}г
                    </div>
                  </div>
                  <button className="text-red-500 min-w-[44px] min-h-[44px] flex items-center justify-center -mr-2">
                    🗑️
                  </button>
                </div>
              ))}
            </div>
            <Button variant="primary" fullWidth className="mt-4">
              + Добавить приём пищи
            </Button>
          </>
        )}
      </Card>
    </div>
  );
}