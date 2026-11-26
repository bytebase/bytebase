import { router } from "@/app/router";
import { getCurrentAgentSqlEditorContext } from "./sqlEditorHost";
import type { AgentChatSnapshot } from "./types";

export function captureAgentPageSnapshot(): AgentChatSnapshot | undefined {
  const page = {
    path: router.currentRoute.value.fullPath,
    title: document.title,
  };
  const context = getCurrentAgentSqlEditorContext();
  if (!context) return page;
  return context.kind === "ready"
    ? { ...page, sqlEditor: context.snapshot }
    : undefined;
}
