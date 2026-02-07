import { useMemo } from "react";

/**
 * Lightweight GradualBlur — inspired by reactbits.dev/animations/gradual-blur
 * Creates a smooth gradient-blur overlay, typically placed at the bottom of the screen
 * above a floating navbar to blend content into the background.
 *
 * No external dependencies (no mathjs).
 */
export default function GradualBlur({
  position = "bottom",
  layers = 6,
  strength = 2,
  height = "7rem",
  zIndex = 15,
  style = {},
}) {
  const direction = position === "top" ? "to top" : "to bottom";

  const divs = useMemo(() => {
    const result = [];
    const step = 100 / layers;

    for (let i = 1; i <= layers; i++) {
      const progress = i / layers;
      // Quadratic ease-out curve for smoother blur ramp
      const curved = 1 - Math.pow(1 - progress, 2);
      const blur = (curved * strength).toFixed(2);

      const p1 = Math.round((step * i - step) * 10) / 10;
      const p2 = Math.round(step * i * 10) / 10;
      const p3 = Math.min(100, Math.round((step * i + step) * 10) / 10);

      const gradient = `linear-gradient(${direction}, transparent ${p1}%, black ${p2}%, black ${p3}%, transparent ${Math.min(100, p3 + step)}%)`;

      result.push(
        <div
          key={i}
          style={{
            position: "absolute",
            inset: 0,
            WebkitMaskImage: gradient,
            maskImage: gradient,
            backdropFilter: `blur(${blur}rem)`,
            WebkitBackdropFilter: `blur(${blur}rem)`,
          }}
        />
      );
    }
    return result;
  }, [layers, strength, direction]);

  return (
    <div
      style={{
        position: "fixed",
        left: 0,
        right: 0,
        [position]: 0,
        height,
        pointerEvents: "none",
        zIndex,
        isolation: "isolate",
        ...style,
      }}
    >
      <div style={{ position: "relative", width: "100%", height: "100%" }}>
        {divs}
      </div>
    </div>
  );
}
