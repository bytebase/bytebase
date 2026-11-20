import { useAppStore } from "@/stores/app";
import { UNKNOWN_USER_NAME } from "@/types/v1/user";

export interface AgentExecutionContext {
  readonly actor: string;
  readonly workspace: string;
}

export function captureAgentExecutionContext():
  | AgentExecutionContext
  | undefined {
  const state = useAppStore.getState();
  const actor = state.currentUser?.name;
  const workspace = state.workspaceResourceName();
  if (
    !actor ||
    actor === UNKNOWN_USER_NAME ||
    (state.currentUserName !== undefined && state.currentUserName !== actor) ||
    !/^workspaces\/(?!-$)[^/]+$/.test(workspace) ||
    (state.currentUser?.workspace && state.currentUser.workspace !== workspace)
  ) {
    return undefined;
  }
  return { actor, workspace };
}

export function isAgentExecutionContextCurrent(
  context: AgentExecutionContext | undefined,
  signal?: AbortSignal
): boolean {
  if (!context || signal?.aborted) return false;
  const current = captureAgentExecutionContext();
  return (
    current?.actor === context.actor && current.workspace === context.workspace
  );
}
