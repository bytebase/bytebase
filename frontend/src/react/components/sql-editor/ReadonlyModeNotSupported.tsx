import { ShieldAlert } from "lucide-react";
import { Trans, useTranslation } from "react-i18next";
import { EngineIcon } from "@/react/components/EngineIcon";
import { useVueState } from "@/react/hooks/useVueState";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import { AdminModeButton } from "./AdminModeButton";

/**
 * Replaces frontend/src/views/sql-editor/EditorPanel/ReadonlyModeNotSupported.vue.
 * Shown when the current tab targets an instance without read-only mode —
 * prompts the user to switch to admin mode instead.
 */
export function ReadonlyModeNotSupported() {
  const { t } = useTranslation();
  const { instance: instanceRef } = useConnectionOfCurrentSQLEditorTab();
  const instance = useVueState(() => instanceRef.value);

  return (
    <div className="w-full h-full flex flex-col items-center justify-center gap-y-2 p-4">
      <div className="flex items-center gap-x-2 text-error">
        <ShieldAlert className="size-6" />
        <span className="text-base font-medium">
          {t("common.missing-permission")}
        </span>
      </div>
      <div className="text-sm text-control-light flex items-center gap-x-1">
        <Trans
          t={t}
          i18nKey="sql-editor.allow-admin-mode-only"
          components={{
            instance: (
              <span className="inline-flex items-center gap-x-1 font-medium text-control">
                <EngineIcon engine={instance.engine} className="size-4" />
                <span>{instance.title}</span>
              </span>
            ),
          }}
        />
      </div>
      <AdminModeButton />
    </div>
  );
}
