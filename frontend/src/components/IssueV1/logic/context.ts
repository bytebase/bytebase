import { InjectionKey, Ref, inject, provide } from "vue";
import { ComposedIssue } from "@/types";

export type IssueContext = {
  isCreating: Ref<boolean>;
  issue: Ref<ComposedIssue>;
};

const KEY = Symbol("bb.issue.context") as InjectionKey<IssueContext>;

export const useIssueContext = () => {
  return inject(KEY)!;
};

export const provideIssueContext = (
  context: Partial<IssueContext>,
  root = false
) => {
  if (!root) {
    const parent = useIssueContext();
    context = {
      ...parent,
      ...context,
    };
  }
  provide(KEY, context as IssueContext);
};
