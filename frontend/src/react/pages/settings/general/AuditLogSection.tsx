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
import { useVueState } from "@/react/hooks/useVueState";
import { useSettingV1Store, useSubscriptionV1Store } from "@/store";
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
    const settingV1Store = useSettingV1Store();
    const subscriptionV1Store = useSubscriptionV1Store();

    const hasAuditLogFeature = useVueState(() =>
      subscriptionV1Store.hasFeature(PlanFeature.FEATURE_AUDIT_LOG)
    );

    const getInitialState = useCallback(
      (): State => ({
        enableAuditLogStdout:
          settingV1Store.workspaceProfile.enableAuditLogStdout,
        enableDebug: settingV1Store.workspaceProfile.enableDebug,
      }),
      [settingV1Store]
    );

    const [state, setState] = useState<State>(getInitialState);

    const isDirty = useCallback(
      () => !isEqual(state, getInitialState()),
      [state, getInitialState]
    );

    const revert = useCallback(() => {
      setState(getInitialState());
    }, [getInitialState]);

    const update = useCallback(async () => {
      await settingV1Store.updateWorkspaceProfile({
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
    }, [state, settingV1Store]);

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
            <input
              type="checkbox"
              className="mt-1"
              disabled={!allowEdit || !hasAuditLogFeature}
              checked={state.enableAuditLogStdout}
              onChange={(e) =>
                setState((s) => ({
                  ...s,
                  enableAuditLogStdout: e.target.checked,
                }))
              }
            />
            <div className="flex flex-col gap-1">
              <div className="textinfo">
                {t("settings.general.workspace.audit-log-stdout.enable")}
              </div>
              <div className="textinfolabel">
                {t("settings.general.workspace.audit-log-stdout.description")}
              </div>
            </div>
          </label>

          {/* Debug mode toggle */}
          <label className="flex items-start gap-x-3 cursor-pointer">
            <input
              type="checkbox"
              className="mt-1"
              disabled={!allowEdit}
              checked={state.enableDebug}
              onChange={(e) =>
                setState((s) => ({
                  ...s,
                  enableDebug: e.target.checked,
                }))
              }
            />
            <div className="flex flex-col gap-1">
              <div className="textinfo">
                {t("settings.general.workspace.enable-debug.enable")}
              </div>
            </div>
          </label>
        </div>
      </div>
    );
  }
);
