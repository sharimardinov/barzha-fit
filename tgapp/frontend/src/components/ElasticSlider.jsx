/**
 * ElasticSlider — inspired by reactbits.dev/components/elastic-slider
 * Self-contained, no Chakra UI / react-icons deps.
 */
import { animate, motion, useMotionValue, useMotionValueEvent, useTransform } from "motion/react";
import { useEffect, useRef, useState } from "react";

const MAX_OVERFLOW = 50;

function decay(value, max) {
  if (max === 0) return 0;
  const entry = value / max;
  const sigmoid = 2 * (1 / (1 + Math.exp(-entry)) - 0.5);
  return sigmoid * max;
}

function SliderCore({ value, startingValue, maxValue, stepSize, onChange, color, leftIcon, rightIcon }) {
  const sliderRef = useRef(null);
  const [region, setRegion] = useState("middle");
  const clientX = useMotionValue(0);
  const overflow = useMotionValue(0);
  const scale = useMotionValue(1);

  useMotionValueEvent(clientX, "change", (latest) => {
    if (sliderRef.current) {
      const { left, right } = sliderRef.current.getBoundingClientRect();
      let newValue;
      if (latest < left) { setRegion("left"); newValue = left - latest; }
      else if (latest > right) { setRegion("right"); newValue = latest - right; }
      else { setRegion("middle"); newValue = 0; }
      overflow.jump(decay(newValue, MAX_OVERFLOW));
    }
  });

  const handlePointerMove = (e) => {
    if (e.buttons > 0 && sliderRef.current) {
      const { left, width } = sliderRef.current.getBoundingClientRect();
      let newVal = startingValue + ((e.clientX - left) / width) * (maxValue - startingValue);
      newVal = Math.round(newVal / stepSize) * stepSize;
      newVal = Math.min(Math.max(newVal, startingValue), maxValue);
      onChange(newVal);
      clientX.jump(e.clientX);
    }
  };

  const handlePointerDown = (e) => {
    handlePointerMove(e);
    e.currentTarget.setPointerCapture(e.pointerId);
  };

  const handlePointerUp = () => {
    animate(overflow, 0, { type: "spring", bounce: 0.5 });
  };

  const pct = maxValue === startingValue ? 0 : ((value - startingValue) / (maxValue - startingValue)) * 100;

  return (
    <motion.div
      onHoverStart={() => animate(scale, 1.15)}
      onHoverEnd={() => animate(scale, 1)}
      onTouchStart={() => animate(scale, 1.15)}
      onTouchEnd={() => animate(scale, 1)}
      style={{
        scale,
        opacity: useTransform(scale, [1, 1.15], [0.85, 1]),
        display: "flex", width: "100%", touchAction: "none", userSelect: "none",
        alignItems: "center", justifyContent: "center", gap: 12,
      }}
    >
      {/* Left icon */}
      <motion.div style={{
        x: useTransform(() => region === "left" ? -overflow.get() / scale.get() : 0),
        flexShrink: 0, color: "var(--muted)", display: "flex", alignItems: "center",
      }}>
        {leftIcon}
      </motion.div>

      {/* Track */}
      <div
        ref={sliderRef}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        style={{
          position: "relative", display: "flex", width: "100%", flexGrow: 1,
          cursor: "grab", touchAction: "none", userSelect: "none",
          alignItems: "center", padding: "14px 0",
        }}
      >
        <motion.div
          style={{
            display: "flex", flexGrow: 1,
            scaleX: useTransform(() => {
              if (sliderRef.current) {
                const { width } = sliderRef.current.getBoundingClientRect();
                return 1 + overflow.get() / width;
              }
              return 1;
            }),
            scaleY: useTransform(overflow, [0, MAX_OVERFLOW], [1, 0.8]),
            transformOrigin: useTransform(() => {
              if (sliderRef.current) {
                const { left, width } = sliderRef.current.getBoundingClientRect();
                return clientX.get() < left + width / 2 ? "right" : "left";
              }
              return "center";
            }),
            height: useTransform(scale, [1, 1.15], [6, 10]),
            marginTop: useTransform(scale, [1, 1.15], [0, -2]),
            marginBottom: useTransform(scale, [1, 1.15], [0, -2]),
          }}
        >
          <div style={{
            position: "relative", height: "100%", flexGrow: 1,
            overflow: "hidden", borderRadius: 999, background: "rgba(0,0,0,0.08)",
          }}>
            <div style={{
              position: "absolute", height: "100%", width: `${pct}%`,
              background: color, borderRadius: 999, transition: "width 0.05s",
            }} />
          </div>
        </motion.div>
      </div>

      {/* Right icon */}
      <motion.div style={{
        x: useTransform(() => region === "right" ? overflow.get() / scale.get() : 0),
        flexShrink: 0, color: "var(--muted)", display: "flex", alignItems: "center",
      }}>
        {rightIcon}
      </motion.div>
    </motion.div>
  );
}

/* Minus icon */
function MinusIcon({ size = 18, color = "currentColor" }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2.5} strokeLinecap="round">
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

/* Plus icon */
function PlusIcon({ size = 18, color = "currentColor" }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth={2.5} strokeLinecap="round">
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

/**
 * Full ElasticSlider with label, value display, and elastic track.
 */
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
      {/* Label + value */}
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 6 }}>
        <span style={{ fontSize: 14, color: "var(--muted)" }}>{label}</span>
        <span style={{ fontSize: 28, fontWeight: 700, fontVariantNumeric: "tabular-nums" }}>
          {value}<span style={{ fontSize: 13, fontWeight: 400, color: "var(--muted)", marginLeft: 2 }}>{unit}</span>
        </span>
      </div>

      {/* Slider */}
      <SliderCore
        value={value}
        startingValue={min}
        maxValue={max}
        stepSize={step}
        onChange={onChange}
        color={color}
        leftIcon={<MinusIcon color={color} />}
        rightIcon={<PlusIcon color={color} />}
      />

      {/* Min/max labels */}
      <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11, color: "var(--muted)", marginTop: 2 }}>
        <span>{min}{unit}</span>
        <span>{max}{unit}</span>
      </div>
    </div>
  );
}
