import { createContext, type ReactNode, useContext } from "react";
import type { PlanDetailPageState } from "./hooks/types";

const PlanDetailContext = createContext<PlanDetailPageState | null>(null);

export const PlanDetailProvider = ({
  value,
  children,
}: {
  value: PlanDetailPageState;
  children: ReactNode;
}) => {
  return (
    <PlanDetailContext.Provider value={value}>
      {children}
    </PlanDetailContext.Provider>
  );
};

export const usePlanDetailContext = () => {
  const context = useContext(PlanDetailContext);
  if (!context) {
    throw new Error(
      "usePlanDetailContext must be used within PlanDetailProvider"
    );
  }
  return context;
};
