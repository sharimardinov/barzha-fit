export default function StepsCard({ steps = 0, target = 0, distance, kcal }) {
  const ratio = target > 0 ? Math.min(steps / target, 1) : 0;
  const deg = Math.round(ratio * 360);
  const formatInt = (value) => (Number.isFinite(value) ? Math.round(value).toLocaleString("ru-RU") : "—");

  return (
    <div className="card steps-card">
      <div className="steps-card-left">
        <div className="steps-ring" style={{ "--steps-progress": `${deg}deg` }}>
          <div className="steps-ring-inner">
            <div className="steps-value">{steps.toLocaleString("ru-RU")}</div>
            <div className="steps-label">Шагов</div>
          </div>
        </div>
      </div>
      <div className="steps-card-right">
        <div className="steps-title">Ходьба</div>
        <div className="steps-goal">Цель: <span>{target > 0 ? target.toLocaleString("ru-RU") : "—"}</span></div>
        <div className="steps-divider" />
        <div className="steps-meta">
          <div><span>{formatInt(distance)}</span> м</div>
          <div><span>{formatInt(kcal)}</span> ккал</div>
        </div>
      </div>
    </div>
  );
}
