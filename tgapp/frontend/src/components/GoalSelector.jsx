export default function GoalSelector({ value, onChange, disabled = false }) {
  const options = [
    { id: "cut", label: "CUT" },
    { id: "balance", label: "BALANCE" },
    { id: "bulk", label: "BULK" },
  ];

  return (
    <div className="radio-inputs">
      {options.map((opt) => (
        <label className="radio" key={opt.id}>
          <input
            type="radio"
            name="goal-tabs"
            value={opt.id}
            checked={value === opt.id}
            onChange={() => onChange(opt.id)}
            disabled={disabled}
          />
          <span className="name">{opt.label}</span>
        </label>
      ))}
    </div>
  );
}
