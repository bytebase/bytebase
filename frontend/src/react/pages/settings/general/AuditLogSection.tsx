import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { FeatureBadge } from "@/react/components/FeatureBadge";
import { Checkbox } from "@/react/components/ui/checkbox";
import { usePlanFeature } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { SectionHandle } from "./useSettingSection";

interface AuditLogSectionProps {
  allowEdit: boolean;
  onDirtyChange: () => void;
}

interface State {
  enableAuditLogStdout: boolean;
  enableDebug: boolean;
}

export const AuditLogSection = forwardRef<SectionHandle, AuditLogSectionProps>(
  function AuditLogSection({ allowEdit, onDirtyChange }, ref) {
    const { t } = useTranslation();

    const hasAuditLogFeature = usePlanFeature(PlanFeature.FEATURE_AUDIT_LOG);

    const getInitialState = useCallback((): State => {
      const profile = useAppStore.getState().getWorkspaceProfile();
      return {
        enableAuditLogStdout: profile.enableAuditLogStdout,
        enableDebug: profile.enableDebug,
      };
    }, []);

    const [state, setState] = useState<State>(getInitialState);

    const isDirty = useCallback(
      () => !isEqual(state, getInitialState()),
      [state, getInitialState]
    );

    const revert = useCallback(() => {
      setState(getInitialState());
    }, [getInitialState]);

    const update = useCallback(async () => {
      await useAppStore.getState().updateWorkspaceProfile({
        payload: {
          enableAuditLogStdout: state.enableAuditLogStdout,
          enableDebug: state.enableDebug,
        },
        updateMask: create(FieldMaskSchema, {
          paths: [
            "value.workspace_profile.enable_audit_log_stdout",
            "value.workspace_profile.enable_debug",
          ],
        }),
      });
    }, [state]);

    const title = t("settings.general.workspace.log");

    useImperativeHandle(ref, () => ({ isDirty, revert, update }));

    useEffect(() => {
      onDirtyChange();
    }, [state, onDirtyChange]);

    return (
      <div id="audit-log-stdout" className="py-6 lg:flex">
        <div className="text-left lg:w-1/4">
          <div className="flex items-center gap-x-2">
            <h1 className="text-2xl font-bold">{title}</h1>
            <FeatureBadge feature={PlanFeature.FEATURE_AUDIT_LOG} />
          </div>
        </div>
        <div className="flex-1 lg:px-4 flex flex-col gap-y-6">
          {/* Audit log stdout toggle */}
          <label className="flex items-start gap-x-3 cursor-pointer">
            <Checkbox
              checked={state.enableAuditLogStdout}
              className="mt-1"
              disabled={!allowEdit || !hasAuditLogFeature}
              onCheckedChange={(checked) =>
                setState((s) => ({
                  ...s,
                  enableAuditLogStdout: checked,
                }))
              }
            />
            <div className="flex flex-col gap-1">
              <div className="text-base font-semibold">
                {t("settings.general.workspace.audit-log-stdout.enable")}
              </div>
              <div className="textinfolabel">
                {t("settings.general.workspace.audit-log-stdout.description")}
              </div>
            </div>
          </label>

          {/* Debug mode toggle */}
          <label className="flex items-start gap-x-3 cursor-pointer">
            <Checkbox
              checked={state.enableDebug}
              className="mt-1"
              disabled={!allowEdit}
              onCheckedChange={(checked) =>
                setState((s) => ({
                  ...s,
                  enableDebug: checked,
                }))
              }
            />
            <div className="flex flex-col gap-1">
              <div className="text-base font-semibold">
                {t("settings.general.workspace.enable-debug.enable")}
              </div>
            </div>
          </label>
        </div>
      </div>
    );
  }
);
