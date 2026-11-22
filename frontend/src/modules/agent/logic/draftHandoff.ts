import { router } from "@/app/router";
import { useAppStore } from "@/stores/app";
import { useAgentStore } from "../store/agent";
import type { MaskingRuleDraftContext } from "./maskingRuleDraft";

export function assertCurrentContext(context: MaskingRuleDraftContext): void {
  const app = useAppStore.getState();
  const agent = useAgentStore.getState();
  if (
    app.currentUser?.name !== context.userName ||
    app.workspaceResourceName() !== context.workspaceName ||
    agent.currentChatId !== context.chatId ||
    agent.getChat(context.chatId)?.workspaceName !== context.workspaceName
  ) {
    throw new Error("The draft handoff context is no longer current.");
  }
}

export function assertCurrentTarget(targetPath: string): void {
  if (router.currentRoute.value.fullPath !== targetPath)
    throw new Error("The draft handoff route is no longer current.");
}

export async function waitForTarget(
  context: MaskingRuleDraftContext,
  targetPath: string,
  isTargetReady: (context: MaskingRuleDraftContext) => boolean
): Promise<void> {
  for (let attempt = 0; attempt < 60; attempt++) {
    assertCurrentContext(context);
    assertCurrentTarget(targetPath);
    if (isTargetReady(context)) return;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error("The draft handoff editor is not available.");
}
