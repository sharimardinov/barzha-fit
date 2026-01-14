import { useState } from 'react';
import { Home, Utensils, Calendar, Target, Activity, User, BarChart2, Flame } from 'lucide-react';
import { TodayScreen } from '@/app/components/barzha/screens/TodayScreen';
import { FoodScreen } from '@/app/components/barzha/screens/FoodScreen';
import { PlanScreen } from '@/app/components/barzha/screens/PlanScreen';
import { GoalsScreen } from '@/app/components/barzha/screens/GoalsScreen';
import { StepsScreen } from '@/app/components/barzha/screens/StepsScreen';
import { ProfileScreen } from '@/app/components/barzha/screens/ProfileScreen';
import { StatsScreen } from '@/app/components/barzha/screens/StatsScreen';
import { StreaksScreen } from '@/app/components/barzha/screens/StreaksScreen';

type Screen = 'today' | 'food' | 'plan' | 'goals' | 'steps' | 'profile' | 'stats' | 'streaks';

export default function App() {
  const [activeScreen, setActiveScreen] = useState<Screen>('today');

  const screens = {
    today: { title: 'Сегодня', icon: Home, component: TodayScreen },
    food: { title: 'Еда', icon: Utensils, component: FoodScreen },
    plan: { title: 'План', icon: Calendar, component: PlanScreen },
    goals: { title: 'Цели', icon: Target, component: GoalsScreen },
    steps: { title: 'Шаги', icon: Activity, component: StepsScreen },
    profile: { title: 'Профиль', icon: User, component: ProfileScreen },
    stats: { title: 'Статистика', icon: BarChart2, component: StatsScreen },
    streaks: { title: 'Серии', icon: Flame, component: StreaksScreen },
  };

  const CurrentScreen = screens[activeScreen].component;

  return (
    <div className="flex flex-col min-h-screen bg-background">
      {/* Header */}
      <header className="bg-card border-b border-border px-4 py-3 sticky top-0 z-10">
        <div className="max-w-[600px] mx-auto flex items-center justify-between">
          <h2 className="flex items-center gap-2">
            💪 BarzhaFit
          </h2>
          <div className="text-sm text-muted-foreground">
            {screens[activeScreen].title}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 px-4 py-4 pb-20 max-w-[600px] mx-auto w-full">
        <CurrentScreen />
      </main>

      {/* Bottom Navigation */}
      <nav className="fixed bottom-0 left-0 right-0 bg-card border-t border-border shadow-lg">
        <div className="max-w-[600px] mx-auto px-2 pt-2 pb-6">
          <div className="grid grid-cols-4 gap-1">
            <NavButton
              icon={Home}
              label="Сегодня"
              active={activeScreen === 'today'}
              onClick={() => setActiveScreen('today')}
            />
            <NavButton
              icon={Utensils}
              label="Еда"
              active={activeScreen === 'food'}
              onClick={() => setActiveScreen('food')}
            />
            <NavButton
              icon={Calendar}
              label="План"
              active={activeScreen === 'plan'}
              onClick={() => setActiveScreen('plan')}
            />
            <NavButton
              icon={Target}
              label="Цели"
              active={activeScreen === 'goals'}
              onClick={() => setActiveScreen('goals')}
            />
            <NavButton
              icon={Activity}
              label="Шаги"
              active={activeScreen === 'steps'}
              onClick={() => setActiveScreen('steps')}
            />
            <NavButton
              icon={User}
              label="Профиль"
              active={activeScreen === 'profile'}
              onClick={() => setActiveScreen('profile')}
            />
            <NavButton
              icon={BarChart2}
              label="Статистика"
              active={activeScreen === 'stats'}
              onClick={() => setActiveScreen('stats')}
            />
            <NavButton
              icon={Flame}
              label="Серии"
              active={activeScreen === 'streaks'}
              onClick={() => setActiveScreen('streaks')}
            />
          </div>
        </div>
      </nav>
    </div>
  );
}

interface NavButtonProps {
  icon: React.ElementType;
  label: string;
  active: boolean;
  onClick: () => void;
}

function NavButton({ icon: Icon, label, active, onClick }: NavButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`flex flex-col items-center gap-1 py-2 px-1 rounded-xl min-h-[60px] justify-center transition-colors ${
        active
          ? 'text-primary bg-primary/10'
          : 'text-muted-foreground hover:text-foreground'
      }`}
    >
      <Icon className="size-5" />
      <span className="text-[10px] leading-tight text-center">{label}</span>
    </button>
  );
}