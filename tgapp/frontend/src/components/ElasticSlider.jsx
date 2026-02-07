/**
 * ElasticSlider — inspired by reactbits.dev/components/elastic-slider
 * Optimized: no state-driven re-renders during drag, cached rects, narrower track.
 */
import { animate, motion, useMotionValue, useMotionValueEvent, useTransform } from "motion/react";
import { useRef, useCallback } from "react";

const MAX_OVERFLOW = 50;

function decay(value, max) {
  if (max === 0) return 0;
  const entry = value / max;
  return (2 * (1 / (1 + Math.exp(-entry)) - 0.5)) * max;
}

function SliderCore({ value, startingValue, maxValue, stepSize, onChange, color, leftIcon, rightIcon }) {
  const sliderRef = useRef(null);
  const rectRef = useRef(null);
  const regionRef = useRef("middle");
  const clientX = useMotionValue(0);
  const overflow = useMotionValue(0);
  const scale = useMotionValue(1);

  const cacheRect = useCallback(() => {
    if (sliderRef.current) rectRef.current = sliderRef.current.getBoundingClientRect();
  }, []);

  useMotionValueEvent(clientX, "change", (latest) => {
    const r = rectRef.current;
    if (!r) return;
    let ov = 0;
    if (latest < r.left) { regionRef.current = "left"; ov = r.left - latest; }
    else if (latest > r.right) { regionRef.current = "right"; ov = latest - r.right; }
    else { regionRef.current = "middle"; }
    overflow.jump(decay(ov, MAX_OVERFLOW));
  });

  const handlePointerMove = (e) => {
    if (e.buttons > 0) {
      const r = rectRef.current;
      if (!r) return;
      let v = startingValue + ((e.clientX - r.left) / r.width) * (maxValue - startingValue);
      v = Math.round(v / stepSize) * stepSize;
      v = Math.min(Math.max(v, startingValue), maxValue);
      onChange(v);
      clientX.jump(e.clientX);
    }
  };

  const handlePointerDown = (e) => {
    cacheRect();
    handlePointerMove(e);
    e.currentTarget.setPointerCapture(e.pointerId);
  };

  const handlePointerUp = () => {
    animate(overflow, 0, { type: "spring", bounce: 0.5 });
  };

  const pct = maxValue === startingValue ? 0 : ((value - startingValue) / (maxValue - startingValue)) * 100;

  // Derived transforms (no getBoundingClientRect inside — uses cached overflow)
  const leftX = useTransform(overflow, (v) => regionRef.current === "left" ? -v : 0);
  const rightX = useTransform(overflow, (v) => regionRef.current === "right" ? v : 0);
  const trackScaleX = useTransform(overflow, (v) => {
    const w = rectRef.current?.width || 200;
    return 1 + v / w;
  });
  const trackScaleY = useTransform(overflow, [0, MAX_OVERFLOW], [1, 0.8]);
  const trackOrigin = useTransform(overflow, () => {
    const r = rectRef.current;
    if (!r) return "center";
    return clientX.get() < r.left + r.width / 2 ? "right" : "left";
  });

  return (
    <div
      style={{
        display: "flex", width: "100%", touchAction: "none", userSelect: "none",
        alignItems: "center", justifyContent: "center", gap: 10,
        padding: "0 4px",
      }}
    >
      {/* Left icon */}
      <motion.div style={{ x: leftX, flexShrink: 0, display: "flex", alignItems: "center" }}>
        {leftIcon}
      </motion.div>

      {/* Track — narrower with maxWidth for visible elastic */}
      <div
        ref={sliderRef}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        style={{
          position: "relative", display: "flex", flexGrow: 1,
          cursor: "grab", touchAction: "none", userSelect: "none",
          alignItems: "center", padding: "14px 0",
          maxWidth: "75%", margin: "0 auto",
        }}
      >
        <motion.div
          style={{
            display: "flex", flexGrow: 1,
            scaleX: trackScaleX,
            scaleY: trackScaleY,
            transformOrigin: trackOrigin,
            height: 6,
          }}
        >
          <div style={{
            position: "relative", height: "100%", flexGrow: 1,
            overflow: "hidden", borderRadius: 999, background: "rgba(0,0,0,0.08)",
          }}>
            <div style={{
              position: "absolute", height: "100%", width: `${pct}%`,
              background: color, borderRadius: 999,
            }} />
          </div>
        </motion.div>
      </div>

      {/* Right icon */}
      <motion.div style={{ x: rightX, flexShrink: 0, display: "flex", alignItems: "center" }}>
        {rightIcon}
      </motion.div>
    </div>
  );
}

function MinusIcon({ size = 18, color = "currentColor" }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2.5} strokeLinecap="round">
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

function PlusIcon({ size = 18, color = "currentColor" }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2.5} strokeLinecap="round">
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

export default function ElasticSlider({
  label = "",
  unit = "",
  min = 0,
  max = 100,
  step = 1,
  value = 50,
  onChange = () => {},
  color = "var(--accent)",
}) {
  return (
    <div style={{ marginBottom: 4 }}>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 6 }}>
        <span style={{ fontSize: 14, color: "var(--muted)" }}>{label}</span>
        <span style={{ fontSize: 28, fontWeight: 700, fontVariantNumeric: "tabular-nums" }}>
          {value}<span style={{ fontSize: 13, fontWeight: 400, color: "var(--muted)", marginLeft: 2 }}>{unit}</span>
        </span>
      </div>
      <SliderCore
        value={value} startingValue={min} maxValue={max} stepSize={step}
        onChange={onChange} color={color}
        leftIcon={<MinusIcon color={color} />}
        rightIcon={<PlusIcon color={color} />}
      />
      <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11, color: "var(--muted)", marginTop: 2, padding: "0 4px" }}>
        <span>{min}{unit}</span>
        <span>{max}{unit}</span>
      </div>
    </div>
  );
}
