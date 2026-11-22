import { type AgentState, useAgentStore } from "../store/agent";

interface AgentPageHandoffContext
  extends Pick<AgentState, "workspaceName" | "userEmail"> {
  chatId: string;
}

export function captureAgentPageHandoffContext(
  chatId?: string
): AgentPageHandoffContext | undefined {
  const state = useAgentStore.getState();
  const sourceId = chatId ?? state.currentChatId;
  if (
    !state.visible ||
    !sourceId ||
    state.currentChatId !== sourceId ||
    !state.getChat(sourceId)
  )
    return undefined;
  return {
    chatId: sourceId,
    workspaceName: state.workspaceName,
    userEmail: state.userEmail,
  };
}

// Call only after the destination or unsaved form has accepted the handoff.
export function minimizeAgentAfterPageHandoff(
  context: AgentPageHandoffContext | undefined
): void {
  if (!context) return;
  const state = useAgentStore.getState();
  if (
    state.visible &&
    state.currentChatId === context.chatId &&
    state.workspaceName === context.workspaceName &&
    state.userEmail === context.userEmail &&
    state.getChat(context.chatId)
  )
    state.minimize();
}
