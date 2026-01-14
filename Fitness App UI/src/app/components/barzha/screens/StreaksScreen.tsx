import { Card } from '../Card';

export function StreaksScreen() {
  return (
    <div className="space-y-4">
      <Card>
        <h3 className="mb-4">🔥 Серии тренировок</h3>
        <div className="space-y-4">
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">Текущая серия</span>
              <span className="text-2xl font-bold">5 дней</span>
            </div>
            <div className="w-full bg-muted rounded-full h-3 overflow-hidden">
              <div 
                className="h-full rounded-full transition-all"
                style={{ 
                  width: '71%',
                  background: 'linear-gradient(90deg, var(--chart-1), var(--chart-5))' 
                }}
              />
            </div>
            <div className="flex justify-between text-xs text-muted-foreground mt-1">
              <span>5 дней</span>
              <span>Рекорд: 7 дней</span>
            </div>
          </div>

          <div className="h-px bg-border" />

          <div>
            <div className="text-sm text-muted-foreground mb-3">История серий</div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <div className="w-16 text-xs text-muted-foreground">1-5 янв</div>
                <div className="flex-1 bg-muted rounded-full h-2 overflow-hidden">
                  <div 
                    className="h-full rounded-full"
                    style={{ 
                      width: '100%',
                      backgroundColor: 'var(--chart-1)' 
                    }}
                  />
                </div>
                <div className="w-10 text-xs text-right">5 дней</div>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-16 text-xs text-muted-foreground">7-13 янв</div>
                <div className="flex-1 bg-muted rounded-full h-2 overflow-hidden">
                  <div 
                    className="h-full rounded-full"
                    style={{ 
                      width: '100%',
                      backgroundColor: 'var(--chart-1)' 
                    }}
                  />
                </div>
                <div className="w-10 text-xs text-right font-medium">7 дней 🏆</div>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <Card>
        <h3 className="mb-4">🍽️ Серии питания</h3>
        <div className="space-y-4">
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">Текущая серия</span>
              <span className="text-2xl font-bold">3 дня</span>
            </div>
            <div className="w-full bg-muted rounded-full h-3 overflow-hidden">
              <div 
                className="h-full rounded-full transition-all"
                style={{ 
                  width: '60%',
                  background: 'linear-gradient(90deg, var(--chart-2), var(--chart-4))' 
                }}
              />
            </div>
            <div className="flex justify-between text-xs text-muted-foreground mt-1">
              <span>3 дня</span>
              <span>Рекорд: 5 дней</span>
            </div>
          </div>

          <div className="h-px bg-border" />

          <div>
            <div className="text-sm text-muted-foreground mb-3">История серий</div>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <div className="w-16 text-xs text-muted-foreground">1-5 янв</div>
                <div className="flex-1 bg-muted rounded-full h-2 overflow-hidden">
                  <div 
                    className="h-full rounded-full"
                    style={{ 
                      width: '100%',
                      backgroundColor: 'var(--chart-2)' 
                    }}
                  />
                </div>
                <div className="w-10 text-xs text-right font-medium">5 дней 🏆</div>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-16 text-xs text-muted-foreground">8-10 янв</div>
                <div className="flex-1 bg-muted rounded-full h-2 overflow-hidden">
                  <div 
                    className="h-full rounded-full"
                    style={{ 
                      width: '60%',
                      backgroundColor: 'var(--chart-2)' 
                    }}
                  />
                </div>
                <div className="w-10 text-xs text-right">3 дня</div>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <Card className="bg-primary/5">
        <h4 className="mb-3">🏆 Достижения</h4>
        <div className="space-y-3">
          <div className="flex items-center gap-3">
            <div className="text-3xl">🥇</div>
            <div className="flex-1">
              <div className="text-sm font-medium">Неделя без пропусков</div>
              <div className="text-xs text-muted-foreground">7 дней подряд</div>
            </div>
            <div className="text-green-500">✓</div>
          </div>
          <div className="flex items-center gap-3">
            <div className="text-3xl">🥈</div>
            <div className="flex-1">
              <div className="text-sm font-medium">Питание под контролем</div>
              <div className="text-xs text-muted-foreground">5 дней подряд</div>
            </div>
            <div className="text-green-500">✓</div>
          </div>
          <div className="flex items-center gap-3">
            <div className="text-3xl">🥉</div>
            <div className="flex-1">
              <div className="text-sm font-medium">Месяц стабильности</div>
              <div className="text-xs text-muted-foreground">30 дней активности</div>
            </div>
            <div className="text-muted-foreground">—</div>
          </div>
        </div>
      </Card>
    </div>
  );
}
