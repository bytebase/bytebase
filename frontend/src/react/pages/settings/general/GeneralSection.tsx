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
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useActuatorV1Store, useSettingV1Store } from "@/store";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import type { SectionHandle } from "./useSettingSection";

interface GeneralSectionProps {
  title: string;
  onDirtyChange: () => void;
}

interface LocalState {
  databaseChangeMode: DatabaseChangeMode;
  externalUrl: string;
}

export const GeneralSection = forwardRef<SectionHandle, GeneralSectionProps>(
  function GeneralSection({ title, onDirtyChange }, ref) {
    const { t } = useTranslation();
    const settingV1Store = useSettingV1Store();
    const actuatorV1Store = useActuatorV1Store();

    const isSaaSMode = useVueState(() => actuatorV1Store.isSaaSMode);
    const externalUrlFromFlag = useVueState(
      () => actuatorV1Store.serverInfo?.externalUrlFromFlag ?? false
    );

    const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);

    const getInitialState = useCallback((): LocalState => {
      const mode = settingV1Store.workspaceProfile.databaseChangeMode;
      return {
        databaseChangeMode:
          mode === DatabaseChangeMode.PIPELINE ||
          mode === DatabaseChangeMode.EDITOR
            ? mode
            : DatabaseChangeMode.PIPELINE,
        externalUrl: actuatorV1Store.serverInfo?.externalUrl ?? "",
      };
    }, [settingV1Store, actuatorV1Store]);

    const [state, setState] = useState<LocalState>(getInitialState);
    const [showModal, setShowModal] = useState(false);

    const isDirty = useCallback(
      () => !isEqual(state, getInitialState()),
      [state, getInitialState]
    );

    const revert = useCallback(() => {
      setState(getInitialState());
    }, [getInitialState]);

    const update = useCallback(async () => {
      const initState = getInitialState();
      if (state.externalUrl !== initState.externalUrl) {
        await settingV1Store.updateWorkspaceProfile({
          payload: {
            externalUrl: state.externalUrl,
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.external_url"],
          }),
        });
      }
      if (state.databaseChangeMode !== initState.databaseChangeMode) {
        await settingV1Store.updateWorkspaceProfile({
          payload: {
            databaseChangeMode: state.databaseChangeMode,
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.database_change_mode"],
          }),
        });
        if (state.databaseChangeMode === DatabaseChangeMode.EDITOR) {
          setShowModal(true);
        }
      }
    }, [state, getInitialState, settingV1Store]);

    useImperativeHandle(
      ref,
      () => ({
        isDirty,
        revert,
        update,
      }),
      [isDirty, revert, update]
    );

    // Notify parent when state changes.
    useEffect(() => {
      onDirtyChange();
    }, [state, onDirtyChange]);

    const goToSQLEditor = () => {
      router.push({ name: SQL_EDITOR_HOME_MODULE });
    };

    return (
      <div className="pb-6 lg:flex">
        <div className="text-left lg:w-1/4">
          <div className="flex items-center gap-x-2">
            <h1 className="text-2xl font-bold">{title}</h1>
          </div>
        </div>
        <PermissionGuard
          permissions={["bb.settings.setWorkspaceProfile"]}
          display="block"
        >
          <div className="flex-1 mt-4 lg:px-4 lg:mt-0 flex flex-col gap-y-6">
            {/* Database change mode */}
            <div>
              <div className="mb-4 font-medium">
                {t("settings.general.workspace.default-landing-page.self")}
              </div>
              <div className="w-full flex flex-col gap-8">
                <label className="flex items-start gap-x-3 cursor-pointer">
                  <input
                    type="radio"
                    name="databaseChangeMode"
                    className="mt-1"
                    disabled={!canEdit}
                    checked={
                      state.databaseChangeMode === DatabaseChangeMode.PIPELINE
                    }
                    onChange={() =>
                      setState((s) => ({
                        ...s,
                        databaseChangeMode: DatabaseChangeMode.PIPELINE,
                      }))
                    }
                  />
                  <div className="flex flex-col gap-1">
                    <div className="textinfo">
                      {t(
                        "settings.general.workspace.default-landing-page.workspace.self"
                      )}
                    </div>
                    <div className="textinfolabel">
                      {t(
                        "settings.general.workspace.default-landing-page.workspace.description"
                      )}
                    </div>
                  </div>
                </label>
                <label className="flex items-start gap-x-3 cursor-pointer">
                  <input
                    type="radio"
                    name="databaseChangeMode"
                    className="mt-1"
                    disabled={!canEdit}
                    checked={
                      state.databaseChangeMode === DatabaseChangeMode.EDITOR
                    }
                    onChange={() =>
                      setState((s) => ({
                        ...s,
                        databaseChangeMode: DatabaseChangeMode.EDITOR,
                      }))
                    }
                  />
                  <div className="flex flex-col gap-1">
                    <div className="textinfo">
                      {t(
                        "settings.general.workspace.default-landing-page.sql-editor.self"
                      )}
                    </div>
                    <div className="textinfolabel">
                      {t(
                        "settings.general.workspace.default-landing-page.sql-editor.description"
                      )}
                    </div>
                  </div>
                </label>
              </div>
            </div>

            {/* External URL */}
            <div>
              <label className="flex items-center gap-x-2">
                <span className="font-medium">
                  {t("settings.general.workspace.external-url.self")}
                </span>
              </label>
              <div className="mb-3 text-sm text-gray-400">
                {t("settings.general.workspace.external-url.description")}{" "}
                <a
                  href="https://docs.bytebase.com/get-started/self-host/external-url?source=console"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-accent hover:underline"
                >
                  {t("common.learn-more")}
                </a>
              </div>
              {externalUrlFromFlag && !isSaaSMode && (
                <div className="mb-3 p-3 bg-blue-50 text-blue-700 text-sm rounded-xs">
                  {t(
                    "settings.general.workspace.external-url.cannot-edit-flag"
                  )}
                </div>
              )}
              <Input
                value={state.externalUrl}
                className="mb-4 w-full"
                disabled={!canEdit || isSaaSMode || externalUrlFromFlag}
                onChange={(e) =>
                  setState((s) => ({ ...s, externalUrl: e.target.value }))
                }
              />
            </div>
          </div>
        </PermissionGuard>

        {/* Modal after switching to Editor mode */}
        {showModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center">
            <div
              className="fixed inset-0 bg-black/40"
              onClick={() => setShowModal(false)}
            />
            <div className="relative bg-white rounded-sm shadow-lg p-6 max-w-md w-full mx-4 flex flex-col gap-2">
              <h2 className="text-lg font-semibold">
                {t("settings.general.workspace.config-updated")}
              </h2>
              <div className="py-2">
                {t(
                  "settings.general.workspace.default-landing-page.default-view-changed-to-sql-editor"
                )}
              </div>
              <div className="flex items-center justify-end gap-2">
                <Button variant="outline" onClick={() => setShowModal(false)}>
                  {t("common.ok")}
                </Button>
                <Button onClick={goToSQLEditor}>
                  {t(
                    "settings.general.workspace.default-landing-page.go-to-sql-editor"
                  )}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    );
  }
);
