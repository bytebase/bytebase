import type { ProjectHomeAction } from "./projectHomeActions";

const ACTION_IDS = new Set<ProjectHomeAction>([
  "plans",
  "query",
  "createDatabase",
  "access",
  "issues",
  "instances",
  "databases",
  "members",
  "gitops",
]);

export function readProjectHomeShortcuts(
  storage: Pick<Storage, "getItem">,
  key: string
): ProjectHomeAction[] | null {
  if (!key) return null;
  try {
    const raw = storage.getItem(key);
    if (raw === null) return null;
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return null;
    const seen = new Set<ProjectHomeAction>();
    return parsed.filter((id): id is ProjectHomeAction => {
      if (!ACTION_IDS.has(id) || seen.has(id)) return false;
      seen.add(id);
      return true;
    });
  } catch {
    return null;
  }
}

export function getVisibleProjectHomeShortcuts(
  saved: ProjectHomeAction[] | null,
  suggested: ProjectHomeAction[],
  available: ProjectHomeAction[]
): ProjectHomeAction[] {
  const availableIds = new Set(available);
  return (saved ?? suggested).filter((id) => availableIds.has(id));
}

export function reorderProjectHomeShortcuts(
  current: ProjectHomeAction[],
  visible: ProjectHomeAction[],
  source: ProjectHomeAction,
  target: ProjectHomeAction
): ProjectHomeAction[] {
  const reordered = [...visible];
  const sourceIndex = reordered.indexOf(source);
  const targetIndex = reordered.indexOf(target);
  if (sourceIndex < 0 || targetIndex < 0 || sourceIndex === targetIndex) {
    return current;
  }
  reordered.splice(sourceIndex, 1);
  reordered.splice(targetIndex, 0, source);
  const visibleIds = new Set(visible);
  let index = 0;
  return current.map((id) => (visibleIds.has(id) ? reordered[index++] : id));
}
