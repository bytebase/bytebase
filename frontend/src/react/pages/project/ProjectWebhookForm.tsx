import { clone } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { EllipsisVertical, Info } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import { WebhookTypeIcon } from "@/react/components/WebhookTypeIcon";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import {
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
  PROJECT_V1_ROUTE_WEBHOOKS,
} from "@/router/dashboard/projectV1";
import { WORKSPACE_ROUTE_IM } from "@/router/dashboard/workspaceRoutes";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import {
  pushNotification,
  useActuatorV1Store,
  useProjectV1Store,
  useProjectWebhookV1Store,
  useSettingV1Store,
} from "@/store";
import {
  projectWebhookV1ActivityItemList,
  projectWebhookV1TypeItemList,
} from "@/types";
import type {
  Activity_Type,
  Project,
  Webhook,
} from "@/types/proto-es/v1/project_service_pb";
import { WebhookSchema } from "@/types/proto-es/v1/project_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { extractProjectWebhookID, hasWorkspacePermissionV2 } from "@/utils";

interface Props {
  allowEdit?: boolean;
  create: boolean;
  project: Project;
  webhook: Webhook;
}

export function ProjectWebhookForm({
  allowEdit = true,
  create,
  project,
  webhook,
}: Props) {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();
  const projectStore = useProjectV1Store();
  const projectWebhookV1Store = useProjectWebhookV1Store();
  const actuatorStore = useActuatorV1Store();

  const [state, setState] = useState<Webhook>(() =>
    clone(WebhookSchema, webhook)
  );
  const [loading, setLoading] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  // Sync state when webhook prop changes (e.g. navigating to a different webhook)
  useEffect(() => {
    setState(clone(WebhookSchema, webhook));
  }, [webhook]);

  // Fetch IM setting on mount
  useEffect(() => {
    settingStore.getOrFetchSettingByName(Setting_SettingName.APP_IM);
  }, [settingStore]);

  const externalUrl = useVueState(
    () => actuatorStore.serverInfo?.externalUrl ?? ""
  );

  const webhookTypeItemList = useMemo(() => projectWebhookV1TypeItemList(), []);
  const webhookActivityItemList = useMemo(
    () => projectWebhookV1ActivityItemList(),
    []
  );

  const selectedWebhook = useMemo(
    () => webhookTypeItemList.find((item) => item.type === state.type),
    [webhookTypeItemList, state.type]
  );

  const imSetting = useVueState(() => {
    const setting = settingStore.getSettingByName(Setting_SettingName.APP_IM);
    if (!setting?.value?.value) return undefined;
    const value = setting.value.value;
    if (value.case === "appIm") return value.value;
    return undefined;
  });

  const imApp = useMemo(() => {
    if (!selectedWebhook?.supportDirectMessage) return undefined;
    return imSetting?.settings.find(
      (setting) => setting.type === selectedWebhook.type
    );
  }, [imSetting, selectedWebhook]);

  const isPowerAutomateURL = useMemo(() => {
    try {
      const hostname = new URL(state.url).hostname.toLowerCase();
      return (
        hostname.endsWith(".powerplatform.com") ||
        hostname === "powerplatform.com" ||
        hostname.endsWith(".logic.azure.com") ||
        hostname === "logic.azure.com"
      );
    } catch {
      return false;
    }
  }, [state.url]);

  const webhookSupportDirectMessage = useMemo(
    () => selectedWebhook?.supportDirectMessage && !isPowerAutomateURL,
    [selectedWebhook, isPowerAutomateURL]
  );

  const activitySupportDirectMessage = useMemo(
    () =>
      state.notificationTypes.some(
        (event) =>
          webhookActivityItemList.find((item) => item.activity === event)
            ?.supportDirectMessage
      ),
    [state.notificationTypes, webhookActivityItemList]
  );

  const valueChanged = useMemo(
    () => !isEqual(webhook, state),
    [webhook, state]
  );

  const allowCreate = useMemo(
    () =>
      state.title.trim() !== "" &&
      state.url.trim() !== "" &&
      state.notificationTypes.length > 0,
    [state.title, state.url, state.notificationTypes]
  );

  const isEventOn = useCallback(
    (type: Activity_Type) => state.notificationTypes.includes(type),
    [state.notificationTypes]
  );

  const toggleEvent = useCallback((type: Activity_Type, on: boolean) => {
    setState((prev) => {
      const types = [...prev.notificationTypes];
      if (on) {
        if (!types.includes(type)) types.push(type);
      } else {
        const idx = types.indexOf(type);
        if (idx >= 0) types.splice(idx, 1);
      }
      types.sort();
      return clone(WebhookSchema, { ...prev, notificationTypes: types });
    });
  }, []);

  const updateField = useCallback(
    <K extends keyof Webhook>(field: K, value: Webhook[K]) => {
      setState((prev) => clone(WebhookSchema, { ...prev, [field]: value }));
    },
    []
  );

  const discardChanges = useCallback(() => {
    setState(clone(WebhookSchema, webhook));
  }, [webhook]);

  const cancel = useCallback(() => {
    router.push({ name: PROJECT_V1_ROUTE_WEBHOOKS });
  }, []);

  const withLoading = useCallback(
    async <T,>(fn: () => Promise<T>): Promise<T | void> => {
      setLoading(true);
      try {
        return await fn();
      } catch (error: unknown) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: (error as { message?: string })?.message ?? String(error),
        });
      } finally {
        setLoading(false);
      }
    },
    []
  );

  const createWebhook = useCallback(() => {
    withLoading(async () => {
      const updatedProject = await projectWebhookV1Store.createProjectWebhook(
        project.name,
        state
      );
      projectStore.updateProjectCache({ ...project, ...updatedProject });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.webhook.success-created-prompt", {
          name: state.title,
        }),
      });
      const createdWebhook = updatedProject.webhooks.find(
        (wh) =>
          wh.title === state.title &&
          wh.type === state.type &&
          wh.url === state.url
      );
      if (createdWebhook) {
        router.push({
          name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
          params: {
            webhookResourceId: extractProjectWebhookID(createdWebhook.name),
          },
        });
      }
    });
  }, [project, state, projectStore, projectWebhookV1Store, t, withLoading]);

  const updateWebhook = useCallback(() => {
    withLoading(async () => {
      const updateMask: string[] = [];
      if (state.title !== webhook.title) updateMask.push("title");
      if (state.url !== webhook.url) updateMask.push("url");
      if (state.directMessage !== webhook.directMessage)
        updateMask.push("direct_message");
      if (!isEqual(state.notificationTypes, webhook.notificationTypes))
        updateMask.push("notification_type");

      const updatedProject = await projectWebhookV1Store.updateProjectWebhook(
        state,
        updateMask
      );
      projectStore.updateProjectCache({ ...project, ...updatedProject });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.webhook.success-updated-prompt", {
          name: state.title,
        }),
      });
    });
  }, [
    project,
    state,
    webhook,
    projectStore,
    projectWebhookV1Store,
    t,
    withLoading,
  ]);

  const deleteWebhook = useCallback(() => {
    withLoading(async () => {
      const name = state.title;
      const updatedProject =
        await projectWebhookV1Store.deleteProjectWebhook(state);
      projectStore.updateProjectCache({ ...project, ...updatedProject });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.webhook.success-deleted-prompt", { name }),
      });
      cancel();
    });
  }, [
    project,
    state,
    projectStore,
    projectWebhookV1Store,
    t,
    withLoading,
    cancel,
  ]);

  const testWebhook = useCallback(() => {
    withLoading(async () => {
      const result = await projectWebhookV1Store.testProjectWebhook(
        project,
        state
      );
      if (result.error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("project.webhook.fail-tested-title"),
          description: result.error,
          manualHide: true,
        });
      } else {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("project.webhook.success-tested-prompt"),
        });
      }
    });
  }, [project, state, projectWebhookV1Store, t, withLoading]);

  const imSettingsUrl = useMemo(() => {
    return router.resolve({ name: WORKSPACE_ROUTE_IM }).fullPath;
  }, []);

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 mb-6 px-4">
        {/* Title */}
        {create ? (
          <>
            <div className="text-lg leading-6 font-medium text-main pt-4">
              {t("project.webhook.creation.title")}
            </div>
            <hr className="my-4" />
          </>
        ) : (
          <>
            <div className="flex flex-row justify-between items-center pt-4">
              <div className="flex flex-row gap-x-2 items-center">
                <WebhookTypeIcon type={webhook.type} className="h-6 w-6" />
                <h3 className="text-lg leading-6 font-medium text-main">
                  {webhook.title}
                </h3>
              </div>
              {allowEdit && (
                <DetailDropdown
                  loading={loading}
                  onDelete={() => setDeleteOpen(true)}
                />
              )}
            </div>
            <hr className="my-4" />
          </>
        )}

        {/* Missing external URL warning */}
        {!externalUrl && (
          <div className="mb-6 p-3 border border-red-300 bg-red-50 rounded text-sm text-red-800">
            <div className="font-medium">{t("banner.external-url")}</div>
            <div className="mt-1">
              {t("settings.general.workspace.external-url.description")}
            </div>
            {hasWorkspacePermissionV2("bb.settings.setWorkspaceProfile") && (
              <Button
                size="sm"
                className="mt-2"
                onClick={() =>
                  router.push({ name: SETTING_ROUTE_WORKSPACE_GENERAL })
                }
              >
                {t("common.configure-now")}
              </Button>
            )}
          </div>
        )}

        <div className="flex flex-col gap-y-4">
          {/* Destination type selector (create only) */}
          {create && (
            <div>
              <label className="font-medium text-main">
                {t("project.webhook.destination")}{" "}
                <span className="text-error">*</span>
              </label>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-7 mt-1">
                {webhookTypeItemList.map((item) => (
                  <div
                    key={item.type}
                    className={`flex justify-center px-2 py-4 rounded-sm border cursor-pointer hover:bg-control-bg-hover ${
                      state.type === item.type
                        ? "border-accent"
                        : "border-control-border"
                    }`}
                    onClick={() => updateField("type", item.type)}
                  >
                    <div className="flex flex-col items-center">
                      <WebhookTypeIcon type={item.type} className="h-10 w-10" />
                      <p className="mt-1 text-center text-sm font-medium">
                        {item.name}
                      </p>
                      <div className="mt-3">
                        <input
                          type="radio"
                          checked={state.type === item.type}
                          onChange={() => updateField("type", item.type)}
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Name */}
          <div>
            <label className="font-medium text-main">
              {t("common.name")} <span className="text-error">*</span>
            </label>
            <Input
              className="mt-1 w-full"
              placeholder={`${selectedWebhook?.name ?? "My"} Webhook`}
              value={state.title}
              onChange={(e) => updateField("title", e.target.value)}
              disabled={!allowEdit}
            />
          </div>

          {/* URL */}
          <div>
            <label className="font-medium text-main">
              {t("project.webhook.webhook-url")}{" "}
              <span className="text-error">*</span>
            </label>
            <div className="mt-1 text-sm text-control-light">
              {t("project.webhook.creation.desc", {
                destination: selectedWebhook?.name,
              })}{" "}
              <a
                href={selectedWebhook?.docUrl}
                target="_blank"
                rel="noreferrer"
                className="normal-link"
              >
                {t("project.webhook.creation.view-doc", {
                  destination: selectedWebhook?.name,
                })}
              </a>
              .
            </div>
            <Input
              className="mt-1 w-full"
              placeholder={selectedWebhook?.urlPlaceholder}
              value={state.url}
              onChange={(e) => updateField("url", e.target.value)}
              disabled={!allowEdit}
            />
          </div>

          {/* Triggering activities */}
          <div>
            <div className="text-md leading-6 font-medium text-main">
              {t("project.webhook.triggering-activity")}{" "}
              <span className="text-error">*</span>
            </div>
            <div className="flex flex-col gap-y-4 mt-2">
              {webhookActivityItemList.map((item) => (
                <div key={item.activity}>
                  <div className="flex items-center gap-x-1">
                    <label className="flex items-center gap-x-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={isEventOn(item.activity)}
                        onChange={(e) =>
                          toggleEvent(item.activity, e.target.checked)
                        }
                      />
                      <span className="text-sm">{item.title}</span>
                    </label>
                    {webhookSupportDirectMessage &&
                      item.supportDirectMessage && (
                        <Tooltip
                          content={t(
                            "project.webhook.activity-support-direct-message"
                          )}
                        >
                          <Info className="w-4 h-4 text-gray-500" />
                        </Tooltip>
                      )}
                  </div>
                  <div className="text-sm text-control-light ml-6">
                    {item.label}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Direct messages */}
          {webhookSupportDirectMessage && (
            <div>
              <div className="text-md leading-6 font-medium text-main">
                {t("project.webhook.direct-messages")}
              </div>
              <div className="my-2 p-3 border rounded text-sm">
                {imApp ? (
                  <span>{t("project.webhook.direct-messages-tip")}</span>
                ) : (
                  <span className="text-control-light">
                    {t("project.webhook.direct-messages-warning")}{" "}
                    <a
                      href={imSettingsUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="normal-link"
                    >
                      {t("common.configure-now")}
                    </a>
                  </span>
                )}
              </div>
              <span className="text-sm text-control-light">
                {t("project.webhook.direct-messages-description")}
                <ul className="list-disc pl-4">
                  {webhookActivityItemList
                    .filter((item) => item.supportDirectMessage)
                    .map((item) => (
                      <li key={item.activity}>{item.title}</li>
                    ))}
                </ul>
              </span>
              <div className="flex items-center mt-2">
                <label className="flex items-center gap-x-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={state.directMessage}
                    disabled={!activitySupportDirectMessage}
                    onChange={(e) =>
                      updateField("directMessage", e.target.checked)
                    }
                  />
                  <span className="text-sm">
                    {t("project.webhook.enable-direct-messages")}
                  </span>
                </label>
              </div>
            </div>
          )}

          {/* Test webhook */}
          <div className="mt-4">
            <Button
              variant="outline"
              disabled={!state.url || loading}
              onClick={testWebhook}
            >
              {t("project.webhook.test-webhook")}
            </Button>
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="w-full sticky bottom-0 z-10">
        <div className="w-full py-4 px-4 border-t border-block-border bg-white">
          <div className="flex justify-end">
            <div className="flex items-center gap-x-2">
              {create ? (
                <Button variant="outline" onClick={cancel}>
                  {t("common.cancel")}
                </Button>
              ) : (
                valueChanged && (
                  <Button variant="outline" onClick={discardChanges}>
                    {t("common.discard-changes")}
                  </Button>
                )
              )}
              {allowEdit &&
                (create ? (
                  <Button
                    disabled={!allowCreate || loading}
                    onClick={createWebhook}
                  >
                    {t("common.create")}
                  </Button>
                ) : (
                  <Button
                    disabled={
                      loading ||
                      !valueChanged ||
                      state.notificationTypes.length === 0
                    }
                    onClick={updateWebhook}
                  >
                    {t("common.update")}
                  </Button>
                ))}
            </div>
          </div>
        </div>
      </div>

      {/* Delete confirmation dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogTitle>
            {t("project.webhook.deletion.confirm-title", {
              title: state.title,
            })}
          </DialogTitle>
          <p className="text-sm text-control-light">
            {t("common.cannot-undo-this-action")}
          </p>
          <div className="flex justify-end gap-x-2 mt-4">
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={deleteWebhook}>
              {t("common.delete")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function DetailDropdown({
  loading,
  onDelete,
}: {
  loading: boolean;
  onDelete: () => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  return (
    <div className="relative">
      <button
        type="button"
        className="p-1 rounded hover:bg-gray-100"
        onClick={() => setOpen((v) => !v)}
      >
        <EllipsisVertical className="w-4 h-4" />
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute right-0 top-full z-20 mt-1 bg-white border rounded shadow-md min-w-[100px]">
            <button
              type="button"
              className="w-full text-left px-3 py-1.5 text-sm hover:bg-gray-50 text-error"
              disabled={loading}
              onClick={() => {
                setOpen(false);
                onDelete();
              }}
            >
              {t("common.delete")}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
