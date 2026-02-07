import PillNav from "./PillNav";
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

  return (
    <PillNav
      items={tabs}
      activeId={state.activeTab}
      onItemClick={(item) => dispatch({ type: "SET_TAB", payload: item.id })}
      baseColor="#ffffff"
      pillColor="#f7f7f7"
      hoveredPillTextColor="#ffffff"
      pillTextColor="#000000"
    />
  );
}
