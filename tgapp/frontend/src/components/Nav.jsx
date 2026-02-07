import PillNav from "./PillNav";
import { useAppState } from "../hooks/useAppState";

const tabs = [
  { id: "today", label: "Сегодня" },
  { id: "workout", label: "Тренировка" },
  { id: "meals", label: "Еда" },
  { id: "profile", label: "Профиль" },
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
