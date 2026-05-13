import { useCallback, useMemo } from "react";
import { usePlanDetailStore } from "../../shared/stores/usePlanDetailStore";

export function useEditingScopes() {
  const editingScopes = usePlanDetailStore((s) => s.editingScopes);
  const setEditing = usePlanDetailStore((s) => s.setEditing);
  const bypassLeaveGuardOnce = usePlanDetailStore(
    (s) => s.bypassLeaveGuardOnce
  );
  const pendingLeaveConfirm = usePlanDetailStore((s) => s.pendingLeaveConfirm);
  const setPendingLeaveConfirm = usePlanDetailStore(
    (s) => s.setPendingLeaveConfirm
  );

  const isEditing = useMemo(
    () => Object.keys(editingScopes).length > 0,
    [editingScopes]
  );

  const setScopeEditing = useCallback(
    (scope: string, editing: boolean) => setEditing(scope, editing),
    [setEditing]
  );

  return {
    editingScopes,
    isEditing,
    setEditing: setScopeEditing,
    bypassLeaveGuardOnce,
    pendingLeaveConfirm,
    setPendingLeaveConfirm,
  };
}
