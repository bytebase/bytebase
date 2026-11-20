import { type DomActionParams, lazyPrepareDomAction } from "../../dom";
import type {
  AgentDomContext,
  AgentDomPreparation,
} from "../../dom/interactionRegistry";

export type DomActionArgs = DomActionParams;

export function createDomActionTool() {
  return async (
    args: DomActionArgs,
    context?: AgentDomContext
  ): Promise<AgentDomPreparation> =>
    context
      ? lazyPrepareDomAction(args, context)
      : {
          kind: "result",
          result: {
            success: false,
            message: "The current page and run could not be verified.",
          },
        };
}
