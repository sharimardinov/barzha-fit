interface CardProps {
  children: React.ReactNode;
  className?: string;
}

export function Card({ children, className = '' }: CardProps) {
  return (
    <div className={`bg-card rounded-2xl p-4 border border-border ${className}`}>
      {children}
    </div>
  );
}
