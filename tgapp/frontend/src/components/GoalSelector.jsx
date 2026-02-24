import { motion as Motion } from "motion/react";

export default function GoalSelector({ value, onChange, disabled = false }) {
  const options = [
    { id: "cut", label: "CUT", desc: "Сушка" },
    { id: "balance", label: "BALANCE", desc: "Поддержание" },
    { id: "bulk", label: "BULK", desc: "Набор массы" },
  ];

  return (
    <div style={{
      display: "flex", gap: 4, background: "#f0f0f0", borderRadius: 999, padding: 4,
      opacity: disabled ? 0.6 : 1,
      pointerEvents: disabled ? "none" : "auto",
    }}>
      {options.map((opt) => {
        const active = value === opt.id;
        return (
          <button
            key={opt.id}
            onClick={() => !disabled && onChange(opt.id)}
            disabled={disabled}
            style={{
              flex: 1, position: "relative", border: "none", cursor: disabled ? "not-allowed" : "pointer",
              borderRadius: 999, padding: "14px 8px",
              background: "transparent",
              zIndex: 1,
            }}
          >
            {/* Animated pill background */}
            {active && (
              <Motion.div
                layoutId="goal-pill-bg-profile"
                style={{
                  position: "absolute", inset: 0, borderRadius: 999,
                  background: "#ff033e",
                }}
                transition={{ type: "spring", stiffness: 400, damping: 30 }}
              />
            )}
            <div style={{
              position: "relative", zIndex: 2,
              fontSize: 14, fontWeight: 700, letterSpacing: 1,
              fontFamily: "'Aptos', Arial, sans-serif",
              color: active ? "#fff" : "rgba(0,0,0,0.35)",
              transition: "color 0.2s ease",
              lineHeight: 1.2,
            }}>{opt.label}</div>
            <div style={{
              position: "relative", zIndex: 2,
              fontSize: 10, marginTop: 3,
              color: active ? "rgba(255,255,255,0.7)" : "rgba(0,0,0,0.2)",
              transition: "color 0.2s ease",
              lineHeight: 1,
            }}>{opt.desc}</div>
          </button>
        );
      })}
    </div>
  );
}
