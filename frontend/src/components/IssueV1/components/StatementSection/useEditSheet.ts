import { useIssueContext } from "../../logic";
import { useSpecSheet } from "./useSpecSheet";
import { useTaskSheet } from "./useTaskSheet";

export const useEditSheet = () => {
  const { issue } = useIssueContext();
  const taskSheet = useTaskSheet();
  const specSheet = useSpecSheet();

  return issue.value.rollout ? taskSheet : specSheet;
};
