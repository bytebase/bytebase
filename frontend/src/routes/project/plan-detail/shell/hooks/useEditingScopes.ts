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

  // Stable facade: every field is already reference-stable, so memoize the
  // wrapper too. This keeps the derived `page` context value stable across
  // renders (it depends on this object), so context consumers re-render only on
  // real changes instead of on every page-hook render.
  return useMemo(
    () => ({
      editingScopes,
      isEditing,
      setEditing: setScopeEditing,
      bypassLeaveGuardOnce,
      pendingLeaveConfirm,
      setPendingLeaveConfirm,
    }),
    [
      editingScopes,
      isEditing,
      setScopeEditing,
      bypassLeaveGuardOnce,
      pendingLeaveConfirm,
      setPendingLeaveConfirm,
    ]
  );
}
