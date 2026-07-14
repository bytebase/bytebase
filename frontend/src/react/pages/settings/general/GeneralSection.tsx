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
import {
  FormField,
  FormFieldGroup,
  FormSection,
} from "@/react/components/ui/form";
import { Input } from "@/react/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { useServerState } from "@/react/hooks/useAppState";
import {
  EXTERNAL_URL_PRODUCT_INTRO,
  useProductIntro,
} from "@/react/lib/productIntro";
import { router } from "@/react/router";
import { SQL_EDITOR_HOME_MODULE } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
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
    const refreshServerInfo = useAppStore((state) => state.refreshServerInfo);

    const { isSaaSMode, externalUrl, serverInfo } = useServerState();
    const externalUrlFromFlag = serverInfo?.externalUrlFromFlag ?? false;

    const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);

    const getInitialState = useCallback((): LocalState => {
      const mode = useAppStore
        .getState()
        .getWorkspaceProfile().databaseChangeMode;
      return {
        databaseChangeMode:
          mode === DatabaseChangeMode.PIPELINE ||
          mode === DatabaseChangeMode.EDITOR
            ? mode
            : DatabaseChangeMode.PIPELINE,
        externalUrl,
      };
    }, [externalUrl]);

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
        await useAppStore.getState().updateWorkspaceProfile({
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
        await useAppStore.getState().updateWorkspaceProfile({
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
    }, [state, getInitialState, refreshServerInfo]);

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

    useProductIntro({
      id: EXTERNAL_URL_PRODUCT_INTRO,
      title: t("settings.general.workspace.external-url.self"),
      description: t("settings.general.workspace.external-url.description"),
      disabled: isSaaSMode,
    });

    const goToSQLEditor = () => {
      router.push({ name: SQL_EDITOR_HOME_MODULE });
    };

    return (
      <>
        <FormSection title={title} style={{ paddingTop: 8 }}>
          <PermissionGuard
            permissions={["bb.settings.setWorkspaceProfile"]}
            display="block"
          >
            <FormFieldGroup>
              {/* Database change mode */}
              <FormField
                title={t(
                  "settings.general.workspace.default-landing-page.self"
                )}
                className="gap-y-4"
              >
                <RadioGroup
                  className="flex-col items-stretch gap-4"
                  value={String(state.databaseChangeMode)}
                  onValueChange={(value) =>
                    setState((s) => ({
                      ...s,
                      databaseChangeMode: Number(value),
                    }))
                  }
                >
                  <RadioGroupItem
                    value={String(DatabaseChangeMode.PIPELINE)}
                    disabled={!canEdit}
                    className="items-start gap-x-3"
                    contentClassName="flex flex-col gap-1"
                    radioClassName="mt-1"
                  >
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
                  </RadioGroupItem>
                  <RadioGroupItem
                    value={String(DatabaseChangeMode.EDITOR)}
                    disabled={!canEdit}
                    className="items-start gap-x-3"
                    contentClassName="flex flex-col gap-1"
                    radioClassName="mt-1"
                  >
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
                  </RadioGroupItem>
                </RadioGroup>
              </FormField>

              {/* External URL — hidden in SaaS/cloud mode */}
              {!isSaaSMode && (
                <FormField
                  title={t("settings.general.workspace.external-url.self")}
                  description={
                    <>
                      {t("settings.general.workspace.external-url.description")}{" "}
                      <LearnMoreLink
                        href="https://docs.bytebase.com/get-started/self-host/external-url?source=console"
                        className="text-accent"
                      />
                    </>
                  }
                >
                  {externalUrlFromFlag && (
                    <Alert
                      variant="info"
                      description={t(
                        "settings.general.workspace.external-url.cannot-edit-flag"
                      )}
                    />
                  )}
                  <Input
                    data-product-intro-target={EXTERNAL_URL_PRODUCT_INTRO}
                    value={state.externalUrl}
                    className="w-full"
                    disabled={!canEdit || externalUrlFromFlag}
                    onChange={(e) =>
                      setState((s) => ({ ...s, externalUrl: e.target.value }))
                    }
                  />
                </FormField>
              )}
            </FormFieldGroup>
          </PermissionGuard>
        </FormSection>

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
                <Button
                  appearance="outline"
                  onClick={() => setShowModal(false)}
                >
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
      </>
    );
  }
);
