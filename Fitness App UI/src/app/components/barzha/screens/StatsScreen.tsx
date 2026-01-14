import { useState } from 'react';
import { Card } from '../Card';

export function StatsScreen() {
  const [view, setView] = useState<'week' | 'month'>('week');

  return (
    <div className="space-y-4">
      {/* Табы */}
      <div className="flex gap-2">
        <button
          onClick={() => setView('week')}
          className={`flex-1 py-3 px-4 rounded-xl font-medium transition-all min-h-[44px] ${
            view === 'week'
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground'
          }`}
        >
          📅 Неделя
        </button>
        <button
          onClick={() => setView('month')}
          className={`flex-1 py-3 px-4 rounded-xl font-medium transition-all min-h-[44px] ${
            view === 'month'
              ? 'bg-primary text-primary-foreground'
              : 'bg-muted text-muted-foreground'
          }`}
        >
          📆 Месяц
        </button>
      </div>

      {view === 'week' ? (
        <Card>
          <h3 className="mb-4">📊 Статистика за неделю</h3>
          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">Понедельник</span>
                <span className="text-sm">✅ 🍽️</span>
              </div>
              <div className="text-xs text-muted-foreground">
                2,100 ккал • 160г белка • 9,500 шагов
              </div>
            </div>
            <div className="h-px bg-border" />
            
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">Вторник</span>
                <span className="text-sm">✅ ❌</span>
              </div>
              <div className="text-xs text-muted-foreground">
                1,850 ккал • 140г белка • 7,200 шагов
              </div>
            </div>
            <div className="h-px bg-border" />
            
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">Среда</span>
                <span className="text-sm">✅ 🍽️</span>
              </div>
              <div className="text-xs text-muted-foreground">
                2,200 ккал • 155г белка • 10,800 шагов
              </div>
            </div>
            <div className="h-px bg-border" />
            
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">Четверг</span>
                <span className="text-sm">✅ 🍽️</span>
              </div>
              <div className="text-xs text-muted-foreground">
                1,950 ккал • 145г белка • 8,800 шагов
              </div>
            </div>
            <div className="h-px bg-border" />
            
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">Пятница</span>
                <span className="text-sm">✅ 🍽️</span>
              </div>
              <div className="text-xs text-muted-foreground">
                2,050 ккал • 150г белка • 10,500 шагов
              </div>
            </div>
            <div className="h-px bg-border" />
            
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-muted-foreground">Суббота</span>
                <span className="text-sm">❌ 🍽️</span>
              </div>
              <div className="text-xs text-muted-foreground">
                2,300 ккал • 130г белка • 11,200 шагов
              </div>
            </div>
            <div className="h-px bg-border" />
            
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium">Сегодня</span>
                <span className="text-sm">— 🍽️</span>
              </div>
              <div className="text-xs text-muted-foreground">
                1,450 ккал • 85г белка • 6,500 шагов
              </div>
            </div>
          </div>
        </Card>
      ) : (
        <Card>
          <h3 className="mb-4">📆 Январь 2026</h3>
          <div className="grid grid-cols-7 gap-2 mb-4">
            {['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'].map(day => (
              <div key={day} className="text-center text-xs text-muted-foreground py-2">
                {day}
              </div>
            ))}
          </div>
          <div className="grid grid-cols-7 gap-2">
            {Array.from({ length: 31 }, (_, i) => i + 1).map(day => {
              const emoji = day < 14 ? (day % 3 === 0 ? '✅🍽️' : day % 5 === 0 ? '❌🍽���' : '✅') : day === 14 ? '—' : '';
              return (
                <div
                  key={day}
                  className={`aspect-square flex flex-col items-center justify-center text-xs rounded-lg ${
                    day === 14
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted'
                  }`}
                >
                  <div className="mb-1">{day}</div>
                  <div className="text-[10px]">{emoji}</div>
                </div>
              );
            })}
          </div>
          <div className="mt-4 space-y-2 text-xs">
            <div className="flex items-center gap-2">
              <span>✅</span>
              <span className="text-muted-foreground">Тренировка выполнена</span>
            </div>
            <div className="flex items-center gap-2">
              <span>🍽️</span>
              <span className="text-muted-foreground">Питание в норме</span>
            </div>
            <div className="flex items-center gap-2">
              <span>❌</span>
              <span className="text-muted-foreground">Пропущено</span>
            </div>
          </div>
        </Card>
      )}

      <Card className="bg-primary/5">
        <h4 className="mb-3">📈 Средние за неделю</h4>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <div className="text-muted-foreground">Калории</div>
            <div className="font-medium">2,050 ккал</div>
          </div>
          <div>
            <div className="text-muted-foreground">Белок</div>
            <div className="font-medium">145 г</div>
          </div>
          <div>
            <div className="text-muted-foreground">Тренировки</div>
            <div className="font-medium">5 из 7</div>
          </div>
          <div>
            <div className="text-muted-foreground">Шаги</div>
            <div className="font-medium">9,200</div>
          </div>
        </div>
      </Card>
    </div>
  );
}
