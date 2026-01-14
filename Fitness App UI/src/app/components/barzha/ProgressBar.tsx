interface ProgressBarProps {
  current: number;
  target: number;
  label: string;
  unit: string;
  color?: string;
  showIndicator?: boolean;
}

export function ProgressBar({ 
  current, 
  target, 
  label, 
  unit, 
  color = 'var(--chart-1)',
  showIndicator = true 
}: ProgressBarProps) {
  const percentage = Math.min((current / target) * 100, 100);
  const isComplete = current >= target;
  const indicator = isComplete ? '🟢' : current > 0 ? '🟡' : '🔴';
  
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-sm">
        <div className="flex items-center gap-2">
          {showIndicator && <span>{indicator}</span>}
          <span className="text-foreground">{label}</span>
        </div>
        <span className="text-muted-foreground">
          {current} / {target} {unit}
        </span>
      </div>
      <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
        <div 
          className="h-full rounded-full transition-all duration-500"
          style={{ 
            width: `${percentage}%`,
            backgroundColor: color 
          }}
        />
      </div>
    </div>
  );
}
