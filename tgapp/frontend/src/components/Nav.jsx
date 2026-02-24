import { useRef, useEffect, useCallback, createElement } from "react";
import { Home, Timer, Utensils, User } from "lucide-react";
import { useAppState } from "../hooks/useAppState";

const tabs = [
  { id: "today", label: "Сегодня", icon: Home },
  { id: "workout", label: "Тренировка", icon: Timer },
  { id: "meals", label: "Еда", icon: Utensils },
  { id: "profile", label: "Профиль", icon: User },
];

export default function Nav() {
  const { state, dispatch } = useAppState();
  const highlightRef = useRef(null);
  const navRef = useRef(null);

  const updateHighlight = useCallback(() => {
    const nav = navRef.current;
    const highlight = highlightRef.current;
    if (!nav || !highlight) return;
    const active = nav.querySelector(".nav-btn.active");
    if (!active) {
      highlight.style.opacity = "0";
      highlight.style.width = "0";
      return;
    }
    const navRect = nav.getBoundingClientRect();
    const btnRect = active.getBoundingClientRect();
    const left = btnRect.left - navRect.left;
    highlight.style.width = `${btnRect.width}px`;
    highlight.style.transform = `translate(${left}px, -50%)`;
    highlight.style.opacity = "1";
  }, []);

  useEffect(() => {
    updateHighlight();
  }, [state.activeTab, updateHighlight]);

  useEffect(() => {
    window.addEventListener("resize", updateHighlight);
    return () => window.removeEventListener("resize", updateHighlight);
  }, [updateHighlight]);

  return (
    <nav className="nav" ref={navRef}>
      <div className="nav-glass" />
      <div className="nav-highlight" ref={highlightRef} />
      {tabs.map((tab) => (
        <button
          key={tab.id}
          className={`nav-btn${state.activeTab === tab.id ? " active" : ""}`}
          onClick={() => dispatch({ type: "SET_TAB", payload: tab.id })}
        >
          {createElement(tab.icon, { size: 22, strokeWidth: 2 })}
          <span>{tab.label}</span>
        </button>
      ))}
    </nav>
  );
}
