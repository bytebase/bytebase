export type LeaveAction =
  | { action: "allow" }
  | { action: "intercept"; pendingTarget: string };

export function decideLeaveAction(input: {
  editingScopes: Record<string, true>;
  isBypassed: boolean;
  targetPath: string;
}): LeaveAction {
  if (input.isBypassed) return { action: "allow" };
  const hasEditing = Object.keys(input.editingScopes).length > 0;
  if (!hasEditing) return { action: "allow" };
  return { action: "intercept", pendingTarget: input.targetPath };
}
