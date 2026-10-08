import { storageKeyWorkspaceSetupFinished } from "@/utils/storage-keys";

export function readWorkspaceSetupFinished(
  workspace: string
): boolean | undefined {
  if (!workspace) return undefined;

  try {
    const raw = localStorage.getItem(
      storageKeyWorkspaceSetupFinished(workspace)
    );
    if (raw === null) return undefined;

    const value: unknown = JSON.parse(raw);
    return typeof value === "boolean" ? value : undefined;
  } catch {
    return undefined;
  }
}

export function saveWorkspaceSetupFinished(
  workspace: string,
  finished: boolean
): boolean {
  if (!workspace) return false;

  try {
    localStorage.setItem(
      storageKeyWorkspaceSetupFinished(workspace),
      JSON.stringify(finished)
    );
    return true;
  } catch {
    return false;
  }
}
