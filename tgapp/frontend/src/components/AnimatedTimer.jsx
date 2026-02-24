/**
 * Animated Timer Display — inspired by reactbits.dev/components/counter
 * Rolling digit animation for workout timer in HH:MM:SS format.
 * No external deps besides motion/react.
 */
import { motion as Motion, useSpring, useTransform } from "motion/react";
import { useEffect, useMemo } from "react";

function RollingDigit({ value, fontSize, color }) {
  const height = fontSize * 1.2;
  const mv = useSpring(value, { stiffness: 80, damping: 15, mass: 0.5 });

  useEffect(() => { mv.set(value); }, [value, mv]);

  return (
    <div style={{
      position: "relative", width: "0.6em", height, overflow: "hidden",
      display: "inline-flex", fontVariantNumeric: "tabular-nums",
    }}>
      {Array.from({ length: 10 }, (_, i) => (
        <DigitSlot key={i} mv={mv} number={i} height={height} color={color} />
      ))}
    </div>
  );
}

function DigitSlot({ mv, number, height, color }) {
  const y = useTransform(mv, (latest) => {
    const place = latest % 10;
    let offset = (10 + number - place) % 10;
    if (offset > 5) offset -= 10;
    return offset * height;
  });

  return (
    <Motion.span
      style={{
        y,
        position: "absolute", inset: 0,
        display: "flex", alignItems: "center", justifyContent: "center",
        color: color || "inherit",
      }}
    >
      {number}
    </Motion.span>
  );
}

function Separator({ char, fontSize, color }) {
  return (
    <span style={{
      display: "inline-flex", alignItems: "center", justifyContent: "center",
      width: "0.4em", fontSize, fontWeight: 700,
      color: color || "inherit", opacity: 0.4,
    }}>
      {char}
    </span>
  );
}

/**
 * Parses elapsed ms into digit values for HH:MM:SS
 */
function parseTime(elapsedMs) {
  const totalSec = Math.max(0, Math.floor(elapsedMs / 1000));
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  return {
    h1: Math.floor(h / 10) % 10,
    h2: h % 10,
    m1: Math.floor(m / 10),
    m2: m % 10,
    s1: Math.floor(s / 10),
    s2: s % 10,
  };
}

export default function AnimatedTimer({
  elapsedMs = 0,
  fontSize = 28,
  color = "var(--black)",
  background = "var(--white)",
  style = {},
}) {
  const t = useMemo(() => parseTime(elapsedMs), [elapsedMs]);
  const gradientColor = background === "var(--white)" ? "#ffffff" : background;

  return (
    <div style={{
      display: "flex", alignItems: "center", justifyContent: "center",
      position: "relative", fontSize, fontWeight: 700, lineHeight: 1.2,
      background, borderRadius: 16, padding: "12px 0",
      border: "1px solid var(--border)",
      fontFamily: "'Aptos', Arial, sans-serif",
      width: "100%",
      ...style,
    }}>
      {/* Top/bottom gradient masks */}
      <div style={{ pointerEvents: "none", position: "absolute", top: 0, left: 0, right: 0, height: 10, borderRadius: "16px 16px 0 0", background: `linear-gradient(to bottom, ${gradientColor}, transparent)`, zIndex: 2 }} />
      <div style={{ pointerEvents: "none", position: "absolute", bottom: 0, left: 0, right: 0, height: 10, borderRadius: "0 0 16px 16px", background: `linear-gradient(to top, ${gradientColor}, transparent)`, zIndex: 2 }} />

      <RollingDigit value={t.h1} fontSize={fontSize} color={color} />
      <RollingDigit value={t.h2} fontSize={fontSize} color={color} />
      <Separator char=":" fontSize={fontSize} color={color} />
      <RollingDigit value={t.m1} fontSize={fontSize} color={color} />
      <RollingDigit value={t.m2} fontSize={fontSize} color={color} />
      <Separator char=":" fontSize={fontSize} color={color} />
      <RollingDigit value={t.s1} fontSize={fontSize} color={color} />
      <RollingDigit value={t.s2} fontSize={fontSize} color={color} />
    </div>
  );
}
