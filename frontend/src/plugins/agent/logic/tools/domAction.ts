import type { Router } from "vue-router";
import { lazyExecuteDomAction } from "../../dom";

export interface DomActionArgs {
  type: "click" | "input" | "select" | "scroll";
  index: number;
  value?: string;
}

export function createDomActionTool(router: Router) {
  return async (args: DomActionArgs): Promise<string> => {
    const result = await lazyExecuteDomAction(args, router);
    return JSON.stringify(result);
  };
}
