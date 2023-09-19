import { inject, provide, InjectionKey } from "vue";
import GhostModeProvider from "./GhostModeProvider";
import GrantRequestModeProvider from "./GrantRequestModeProvider";
import type IssueLogic from "./IssueLogic";
import StandardModeProvider from "./StandardModeProvider";
import TenantModeProvider from "./TenantModeProvider";

export * from "./base";
export * from "./common";
export * from "./extra";
export * from "./assignee";
export * from "./transition";

export {
  IssueLogic,
  TenantModeProvider,
  GhostModeProvider,
  StandardModeProvider,
  GrantRequestModeProvider,
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
