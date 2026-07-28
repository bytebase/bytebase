import { createContext, type ReactNode, useContext } from "react";
import type { IssueDetailPageState } from "../hooks/useIssueDetailPage";

const IssueDetailContext = createContext<IssueDetailPageState | null>(null);

export const IssueDetailProvider = ({
  value,
  children,
}: {
  value: IssueDetailPageState;
  children: ReactNode;
}) => {
  return (
    <IssueDetailContext.Provider value={value}>
      {children}
    </IssueDetailContext.Provider>
  );
};

export const useIssueDetailContext = () => {
  const context = useContext(IssueDetailContext);
  if (!context) {
    throw new Error(
      "useIssueDetailContext must be used within IssueDetailProvider"
    );
  }
  return context;
};
