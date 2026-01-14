interface InputProps {
  label?: string;
  value: string | number;
  onChange: (value: string) => void;
  type?: 'text' | 'number' | 'textarea';
  placeholder?: string;
  error?: string;
  rows?: number;
}

export function Input({ 
  label, 
  value, 
  onChange, 
  type = 'text', 
  placeholder,
  error,
  rows = 4 
}: InputProps) {
  const baseStyles = 'w-full px-4 py-3 bg-input-background border border-border rounded-xl focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all';
  
  return (
    <div className="space-y-2">
      {label && <label className="text-sm">{label}</label>}
      {type === 'textarea' ? (
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          rows={rows}
          className={`${baseStyles} resize-none min-h-[100px]`}
        />
      ) : (
        <input
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className={`${baseStyles} min-h-[44px]`}
        />
      )}
      {error && <p className="text-xs text-red-500">{error}</p>}
    </div>
  );
}
