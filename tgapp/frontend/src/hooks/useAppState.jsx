import { createContext, useContext, useReducer, useCallback } from "react";

const AppContext = createContext(null);

const initialState = {
  today: null,
  planText: "",
  planPayload: null,
  planStructured: false,
  onboarding: false,
  targets: null,
  profile: null,
  trainingProfile: null,
  activeTab: "today",
};

function reducer(state, action) {
  switch (action.type) {
    case "SET_TODAY":
      return { ...state, today: action.payload };
    case "SET_PLAN":
      return { ...state, planText: action.payload.text, planPayload: action.payload.payload, planStructured: action.payload.structured };
    case "SET_TARGETS":
      return { ...state, targets: action.payload };
    case "SET_PROFILE":
      return { ...state, profile: action.payload };
    case "SET_TRAINING_PROFILE":
      return { ...state, trainingProfile: action.payload };
    case "SET_ONBOARDING":
      return { ...state, onboarding: action.payload };
    case "SET_TAB":
      return { ...state, activeTab: action.payload };
    case "MERGE":
      return { ...state, ...action.payload };
    default:
      return state;
  }
}

export function AppProvider({ children }) {
  const [state, dispatch] = useReducer(reducer, initialState);
  return (
    <AppContext.Provider value={{ state, dispatch }}>
      {children}
    </AppContext.Provider>
  );
}

export function useAppState() {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useAppState must be inside AppProvider");
  return ctx;
}
