import { useEffect, useRef } from "react";
import { motion, AnimatePresence } from "motion/react";
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
  const pillRefs = useRef([]);

  useEffect(() => {
    const layout = () => {
      pillRefs.current.forEach((pill) => {
        if (!pill) return;

        const rect = pill.getBoundingClientRect();
        const w = rect.width;
        const h = rect.height;
        if (!w || !h) return;

        pill.style.setProperty("--pill-h", `${h}px`);
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
                <motion.button
                  type="button"
                  role="menuitem"
                  className={`pill${activeId === item.id ? " is-active" : ""}`}
                  aria-label={item.ariaLabel || item.label}
                  onClick={() => onItemClick?.(item)}
                  whileHover={{ y: -2 }}
                  whileTap={{ y: 1 }}
                  transition={{ type: "spring", stiffness: 420, damping: 30 }}
                  ref={(el) => {
                    pillRefs.current[i] = el;
                  }}
                >
                  <AnimatePresence>
                    {activeId === item.id && (
                      <motion.span
                        className="pill-active"
                        layoutId="pill-active"
                        transition={{ type: "spring", stiffness: 600, damping: 36 }}
                      />
                    )}
                  </AnimatePresence>
                  <span className="pill-content">
                    {item.icon && (
                      <span className="pill-icon" aria-hidden="true">
                        <item.icon size={18} strokeWidth={2} />
                      </span>
                    )}
                    <span className="pill-label">{item.label}</span>
                  </span>
                </motion.button>
              </li>
            ))}
          </ul>
        </div>
      </nav>
    </div>
  );
};

export default PillNav;
