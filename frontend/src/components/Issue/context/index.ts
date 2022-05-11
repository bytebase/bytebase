import { inject, provide, InjectionKey } from "vue";
import IssueContext from "./IssueContext";

export const KEY = Symbol(
  "bb.issue.ui-state-context"
) as InjectionKey<IssueContext>;

export const useIssueContext = () => {
  return inject(KEY)!;
};

export const provideIssueContext = (
  context: Partial<IssueContext>,
  root = false
) => {
  if (!root) {
    const parentContext = useIssueContext();
    context = {
      ...parentContext,
      ...context,
    };
  }
  provide(KEY, context as IssueContext);
};
