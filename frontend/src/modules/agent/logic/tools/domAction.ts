import type { AppRouterInstance } from "@/app/router";
import { lazyExecuteDomAction } from "../../dom";

export interface DomActionArgs {
  type: "click" | "input" | "select" | "read" | "scroll";
  ref: string;
  value?: string;
}

export function createDomActionTool(
  router: AppRouterInstance,
  canUseCurrentPage?: () => boolean
) {
  return async (args: DomActionArgs): Promise<string> => {
    const result = await lazyExecuteDomAction(args, router, canUseCurrentPage);
    return JSON.stringify(result);
  };
}
