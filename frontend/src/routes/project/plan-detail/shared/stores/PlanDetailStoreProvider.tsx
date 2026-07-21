import { type ReactNode, useState } from "react";
import {
  createPlanDetailStore,
  PlanDetailStoreContext,
} from "./usePlanDetailStore";

export const PlanDetailStoreProvider = ({
  children,
}: {
  children: ReactNode;
}) => {
  const [store] = useState(createPlanDetailStore);
  return (
    <PlanDetailStoreContext.Provider value={store}>
      {children}
    </PlanDetailStoreContext.Provider>
  );
};
