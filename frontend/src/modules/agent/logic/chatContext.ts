import { type ReactRoute, router } from "@/app/router";
import { projectNamePrefix, workspaceNamePrefix } from "@/lib/resourceName";
import { useAppStore } from "@/stores/app";

export function resolveChatContext(
  route: ReactRoute,
  workspaceName: string
): string | undefined {
  const projectId = route.params.projectId ?? route.params.project;
  if (typeof projectId === "string" && projectId && projectId !== "-") {
    return `${projectNamePrefix}${projectId}`;
  }
  return workspaceName.startsWith(workspaceNamePrefix)
    ? workspaceName
    : undefined;
}

export function getCurrentChatContext(): string | undefined {
  const state = useAppStore.getState();
  const workspaceName =
    state.workspace?.name ||
    state.currentUser?.workspace ||
    state.serverInfo?.workspace ||
    "";
  return resolveChatContext(router.currentRoute.value, workspaceName);
}
