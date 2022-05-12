import { inject, provide, InjectionKey } from "vue";
import type IssueLogic from "./IssueLogic";
import TenantModeProvider from "./TenantModeProvider";
import GhostModeProvider from "./GhostModeProvider";
import StandardModeProvider from "./StandardModeProvider";

export * from "./base";
export * from "./common";
export * from "./extra";

export {
  IssueLogic,
  TenantModeProvider,
  GhostModeProvider,
  StandardModeProvider,
};

export const KEY = Symbol("bb.issue.logic") as InjectionKey<IssueLogic>;

export const useIssueLogic = () => {
  return inject(KEY)!;
};

export const provideIssueLogic = (logic: Partial<IssueLogic>, root = false) => {
  if (!root) {
    const parent = useIssueLogic();
    logic = {
      ...parent,
      ...logic,
    };
  }
  provide(KEY, logic as IssueLogic);
};
