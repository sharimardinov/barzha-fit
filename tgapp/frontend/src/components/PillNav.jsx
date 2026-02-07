import { useEffect, useRef } from "react";
import "./PillNav.css";

const PillNav = ({
  items = [],
  activeId,
  onItemClick,
  className = "",
  baseColor = "#ff033e",
  pillColor = "#ffffff",
  hoveredPillTextColor = "#ffffff",
  pillTextColor = "#000000",
}) => {
  const circleRefs = useRef([]);
  const pillRefs = useRef([]);

  useEffect(() => {
    const layout = () => {
      pillRefs.current.forEach((pill, index) => {
        const circle = circleRefs.current[index];
        if (!pill || !circle) return;

        const rect = pill.getBoundingClientRect();
        const w = rect.width;
        const h = rect.height;
        if (!w || !h) return;

        // Geometry to make the hover circle perfectly hug the pill.
        const R = ((w * w) / 4 + h * h) / (2 * h);
        const D = Math.ceil(2 * R) + 2;
        const delta = Math.ceil(R - Math.sqrt(Math.max(0, R * R - (w * w) / 4))) + 1;
        const originY = D - delta;

        circle.style.width = `${D}px`;
        circle.style.height = `${D}px`;
        circle.style.bottom = `-${delta}px`;
        circle.style.setProperty("--circle-origin", `${originY}px`);

        pill.style.setProperty("--pill-h", `${h}px`);
        pill.style.setProperty("--pill-h-neg", `${-h}px`);
      });
    };

    layout();
    const onResize = () => layout();
    window.addEventListener("resize", onResize);

    if (document.fonts?.ready) {
      document.fonts.ready.then(layout).catch(() => {});
    }

    return () => window.removeEventListener("resize", onResize);
  }, [items]);

  const cssVars = {
    ["--base"]: baseColor,
    ["--pill-bg"]: pillColor,
    ["--hover-text"]: hoveredPillTextColor,
    ["--pill-text"]: pillTextColor,
  };

  return (
    <div className={`pill-nav-container ${className}`} style={cssVars}>
      <nav className="pill-nav" aria-label="Primary">
        <div className="pill-nav-items">
          <ul className="pill-list" role="menubar">
            {items.map((item, i) => (
              <li key={item.id || item.label || i} role="none">
                <button
                  type="button"
                  role="menuitem"
                  className={`pill${activeId === item.id ? " is-active" : ""}`}
                  aria-label={item.ariaLabel || item.label}
                  onClick={() => onItemClick?.(item)}
                  ref={(el) => {
                    pillRefs.current[i] = el;
                  }}
                >
                  <span
                    className="hover-circle"
                    aria-hidden="true"
                    ref={(el) => {
                      circleRefs.current[i] = el;
                    }}
                  />
                  <span className="label-stack">
                    <span className="pill-label">{item.label}</span>
                    <span className="pill-label-hover" aria-hidden="true">
                      {item.label}
                    </span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      </nav>
    </div>
  );
};

export default PillNav;
