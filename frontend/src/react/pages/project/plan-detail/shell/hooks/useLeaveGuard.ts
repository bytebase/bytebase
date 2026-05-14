import { useCallback, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/router";
import {
  usePlanDetailStore,
  usePlanDetailStoreApi,
} from "../../shared/stores/usePlanDetailStore";
import { decideLeaveAction } from "../leaveGuard";

export function useLeaveGuard() {
  const { t } = useTranslation();
  const storeApi = usePlanDetailStoreApi();
  const editingScopes = usePlanDetailStore((s) => s.editingScopes);
  const setPendingLeaveTarget = usePlanDetailStore(
    (s) => s.setPendingLeaveTarget
  );
  const setPendingLeaveConfirm = usePlanDetailStore(
    (s) => s.setPendingLeaveConfirm
  );

  const isEditing = Object.keys(editingScopes).length > 0;

  const resolveLeaveConfirm = useCallback(
    (confirmed: boolean) => {
      const state = storeApi.getState();
      const target = confirmed ? state.pendingLeaveTarget : null;
      setPendingLeaveTarget(null);
      setPendingLeaveConfirm(false);
      if (target) {
        state.bypassLeaveGuardOnce();
        // Replace (not push) so a confirmed-discard navigation doesn't leave
        // an extra entry that lets Back return to the discarded plan. Works
        // correctly whether the original navigation was push, replace, or
        // browser back/forward.
        void router.replace(target);
      }
    },
    [setPendingLeaveConfirm, setPendingLeaveTarget, storeApi]
  );

  useEffect(() => {
    if (!isEditing) {
      // Editing scope ended (e.g. async save completed) while a leave
      // prompt is open — there's nothing unsaved anymore, so navigate to
      // the captured target without further confirmation.
      if (storeApi.getState().pendingLeaveTarget) {
        resolveLeaveConfirm(true);
      }
      return;
    }

    const onBeforeUnload = (event: BeforeUnloadEvent) => {
      event.returnValue = t("common.leave-without-saving");
      event.preventDefault();
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    const removeGuard = router.beforeEach((to, _from, next) => {
      const state = storeApi.getState();
      const decision = decideLeaveAction({
        editingScopes: state.editingScopes,
        isBypassed: state.isLeaveGuardBypassed(),
        targetPath: to.fullPath,
      });
      if (decision.action === "allow") {
        next();
        return;
      }
      // Cancel the navigation synchronously and remember the target so we
      // can re-issue it from resolveLeaveConfirm after the user confirms.
      // Always overwrite the pending target — the latest navigation wins.
      setPendingLeaveTarget(decision.pendingTarget);
      setPendingLeaveConfirm(true);
      next(false);
    });

    return () => {
      window.removeEventListener("beforeunload", onBeforeUnload);
      removeGuard();
    };
  }, [
    isEditing,
    resolveLeaveConfirm,
    setPendingLeaveConfirm,
    setPendingLeaveTarget,
    storeApi,
    t,
  ]);

  return { resolveLeaveConfirm };
}
