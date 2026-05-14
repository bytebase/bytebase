import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/router";

/**
 * Warn the user before they lose unsaved edits.
 *
 * - Browser refresh / tab close: fires the native `beforeunload` prompt.
 * - In-app Vue router navigation (sidebar links, back button, etc.):
 *   shows a `window.confirm` with `common.leave-without-saving` and aborts
 *   the navigation if the user cancels.
 *
 * Pass the current dirty boolean — it's stored in a ref each render so the
 * listeners always observe the latest value without re-binding.
 */
export function useUnsavedChangesGuard(isDirty: boolean): void {
  const { t } = useTranslation();
  const dirtyRef = useRef(isDirty);
  dirtyRef.current = isDirty;

  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (dirtyRef.current) {
        e.preventDefault();
        // Older browsers (and some WebKit builds) still require returnValue
        // to be set for the unload confirmation to fire; preventDefault alone
        // is enough on modern Chrome/Firefox but not universal.
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", handleBeforeUnload);

    const removeGuard = router.beforeEach((_to, _from, next) => {
      if (dirtyRef.current) {
        if (!window.confirm(t("common.leave-without-saving"))) {
          next(false);
          return;
        }
      }
      next();
    });

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
      removeGuard();
    };
  }, [t]);
}
