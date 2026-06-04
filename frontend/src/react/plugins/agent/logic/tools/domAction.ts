import type { AppRouterInstance } from "@/react/router";
import { lazyExecuteDomAction } from "../../dom";

export interface DomActionArgs {
  type: "click" | "input" | "select" | "read" | "scroll";
  ref: string;
  value?: string;
}

export function createDomActionTool(router: AppRouterInstance) {
  return async (args: DomActionArgs): Promise<string> => {
    const result = await lazyExecuteDomAction(args, router);
    return JSON.stringify(result);
  };
}
