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
import { LearnMoreLink } from "@/react/components/LearnMoreLink";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import { Alert } from "@/react/components/ui/alert";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { useServerState } from "@/react/hooks/useAppState";
import { useAppStore } from "@/react/stores/app";
import { router } from "@/router";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import { useSettingV1Store } from "@/store";
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
    const refreshServerInfo = useAppStore((state) => state.refreshServerInfo);

    const { isSaaSMode, externalUrl, serverInfo } = useServerState();
    const externalUrlFromFlag = serverInfo?.externalUrlFromFlag ?? false;

    const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);

    const getInitialState = useCallback((): LocalState => {
      const mode = settingV1Store.workspaceProfile.databaseChangeMode;
      return {
        databaseChangeMode:
          mode === DatabaseChangeMode.PIPELINE ||
          mode === DatabaseChangeMode.EDITOR
            ? mode
            : DatabaseChangeMode.PIPELINE,
        externalUrl,
      };
    }, [settingV1Store, externalUrl]);

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
        await refreshServerInfo();
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
    }, [state, getInitialState, settingV1Store, refreshServerInfo]);

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
              <div className="mb-4 text-base font-semibold">
                {t("settings.general.workspace.default-landing-page.self")}
              </div>
              <div className="w-full flex flex-col gap-4">
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
                    <div className="textinfo font-semibold">
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
                    <div className="textinfo font-semibold">
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

            {/* External URL — hidden in SaaS/cloud mode */}
            {!isSaaSMode && (
              <div>
                <label className="flex items-center gap-x-2">
                  <span className="text-base font-semibold">
                    {t("settings.general.workspace.external-url.self")}
                  </span>
                </label>
                <div className="mb-3 text-sm text-control-placeholder">
                  {t("settings.general.workspace.external-url.description")}{" "}
                  <LearnMoreLink
                    href="https://docs.bytebase.com/get-started/self-host/external-url?source=console"
                    className="text-accent"
                  />
                </div>
                {externalUrlFromFlag && (
                  <Alert
                    variant="info"
                    className="mb-3"
                    description={t(
                      "settings.general.workspace.external-url.cannot-edit-flag"
                    )}
                  />
                )}
                <Input
                  value={state.externalUrl}
                  className="w-full"
                  disabled={!canEdit || externalUrlFromFlag}
                  onChange={(e) =>
                    setState((s) => ({ ...s, externalUrl: e.target.value }))
                  }
                />
              </div>
            )}
          </div>
        </PermissionGuard>

        {/* Modal after switching to Editor mode */}
        {showModal && (
          <Dialog
            open
            onOpenChange={(nextOpen) => !nextOpen && setShowModal(false)}
          >
            <DialogContent className="w-[32rem] max-w-[calc(100vw-2rem)] p-6">
              <DialogTitle>
                {t("settings.general.workspace.config-updated")}
              </DialogTitle>
              <div className="mt-4 flex flex-col gap-y-4">
                <div>
                  {t(
                    "settings.general.workspace.default-landing-page.default-view-changed-to-sql-editor"
                  )}
                </div>
              </div>
              <div className="mt-4 flex items-center justify-end gap-x-2">
                <Button variant="outline" onClick={() => setShowModal(false)}>
                  {t("common.ok")}
                </Button>
                <Button onClick={goToSQLEditor}>
                  {t(
                    "settings.general.workspace.default-landing-page.go-to-sql-editor"
                  )}
                </Button>
              </div>
            </DialogContent>
          </Dialog>
        )}
      </div>
    );
  }
);
