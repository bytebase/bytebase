import { clone } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { EllipsisVertical, Info } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { RouterLink } from "@/react/components/RouterLink";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/react/components/ui/alert-dialog";
import { Button, buttonVariants } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { Input } from "@/react/components/ui/input";
import { Tooltip } from "@/react/components/ui/tooltip";
import { WebhookTypeIcon } from "@/react/components/WebhookTypeIcon";
import { router } from "@/react/router";
import {
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
  PROJECT_V1_ROUTE_WEBHOOKS,
  SETTING_ROUTE_WORKSPACE_GENERAL,
  WORKSPACE_ROUTE_IM,
} from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import { pushNotification } from "@/store";
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
  const createProjectWebhook = useAppStore(
    (state) => state.createProjectWebhook
  );
  const updateProjectWebhook = useAppStore(
    (state) => state.updateProjectWebhook
  );
  const deleteProjectWebhook = useAppStore(
    (state) => state.deleteProjectWebhook
  );
  const testProjectWebhook = useAppStore((state) => state.testProjectWebhook);

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
    useAppStore.getState().getOrFetchSettingByName(Setting_SettingName.APP_IM);
  }, []);

  const externalUrl = useAppStore((s) => s.externalUrl());

  const webhookTypeItemList = useMemo(() => projectWebhookV1TypeItemList(), []);
  const webhookActivityItemList = useMemo(
    () => projectWebhookV1ActivityItemList(),
    []
  );

  const selectedWebhook = useMemo(
    () => webhookTypeItemList.find((item) => item.type === state.type),
    [webhookTypeItemList, state.type]
  );

  const settingsByName = useAppStore((s) => s.settingsByName);
  const imSetting = useMemo(() => {
    const setting = useAppStore
      .getState()
      .getSettingByName(Setting_SettingName.APP_IM);
    if (!setting?.value?.value) return undefined;
    const value = setting.value.value;
    if (value.case === "appIm") return value.value;
    return undefined;
  }, [settingsByName]);

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
      const updatedProject = await createProjectWebhook(project.name, state);
      useAppStore
        .getState()
        .updateProjectCache({ ...project, ...updatedProject });
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
  }, [project, state, createProjectWebhook, t, withLoading]);

  const updateWebhook = useCallback(() => {
    withLoading(async () => {
      const updateMask: string[] = [];
      if (state.title !== webhook.title) updateMask.push("title");
      if (state.url !== webhook.url) updateMask.push("url");
      if (state.directMessage !== webhook.directMessage)
        updateMask.push("direct_message");
      if (!isEqual(state.notificationTypes, webhook.notificationTypes))
        updateMask.push("notification_type");

      const updatedProject = await updateProjectWebhook(state, updateMask);
      useAppStore
        .getState()
        .updateProjectCache({ ...project, ...updatedProject });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.webhook.success-updated-prompt", {
          name: state.title,
        }),
      });
    });
  }, [project, state, webhook, updateProjectWebhook, t, withLoading]);

  const deleteWebhook = useCallback(() => {
    withLoading(async () => {
      const name = state.title;
      const updatedProject = await deleteProjectWebhook(state);
      useAppStore
        .getState()
        .updateProjectCache({ ...project, ...updatedProject });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("project.webhook.success-deleted-prompt", { name }),
      });
      cancel();
    });
  }, [project, state, deleteProjectWebhook, t, withLoading, cancel]);

  const testWebhook = useCallback(() => {
    withLoading(async () => {
      const result = await testProjectWebhook(project, state);
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
  }, [project, state, testProjectWebhook, t, withLoading]);

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 mb-6 px-4">
        {/* Title */}
        {create ? (
          <>
            <div className="text-lg leading-6 font-medium text-main pt-4">
              {t("project.webhook.self")}
            </div>
            <hr className="my-4" />
          </>
        ) : (
          <>
            <div className="flex flex-row justify-between items-center pt-4">
              <div className="flex flex-row gap-x-2 items-center">
                <WebhookTypeIcon type={webhook.type} className="size-6" />
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
          <div className="mb-6 p-3 border border-error/30 bg-error/5 rounded-xs text-sm text-error">
            <div className="font-medium">{t("banner.external-url")}</div>
            <div className="mt-1">
              {t("settings.general.workspace.external-url.description")}
            </div>
            {hasWorkspacePermissionV2("bb.settings.setWorkspaceProfile") && (
              <RouterLink
                to={{ name: SETTING_ROUTE_WORKSPACE_GENERAL }}
                className={buttonVariants({ size: "sm", className: "mt-2" })}
              >
                {t("common.configure-now")}
              </RouterLink>
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
                      <WebhookTypeIcon type={item.type} className="size-10" />
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
                      <Checkbox
                        checked={isEventOn(item.activity)}
                        onCheckedChange={(checked) =>
                          toggleEvent(item.activity, checked)
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
                          <Info className="size-4 text-control-light" />
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
              <div className="my-2 p-3 border rounded-xs text-sm">
                {imApp ? (
                  <span>{t("project.webhook.direct-messages-tip")}</span>
                ) : (
                  <span className="text-control-light">
                    {t("project.webhook.direct-messages-warning")}{" "}
                    <RouterLink
                      to={{ name: WORKSPACE_ROUTE_IM }}
                      target="_blank"
                      rel="noreferrer"
                      className="normal-link"
                    >
                      {t("common.configure-now")}
                    </RouterLink>
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
                  <Checkbox
                    checked={state.directMessage}
                    disabled={!activitySupportDirectMessage}
                    onCheckedChange={(checked) =>
                      updateField("directMessage", checked)
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
        <div className="w-full py-4 px-4 border-t border-block-border bg-background">
          <div className="flex justify-end items-center gap-x-4">
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

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("project.webhook.deletion.confirm-title", {
              title: state.title,
            })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("common.cannot-undo-this-action")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={deleteWebhook}>
              {t("common.delete")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
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

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="p-1 rounded-xs hover:bg-control-bg outline-hidden">
        <EllipsisVertical className="size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DropdownMenuItem
          className="text-error"
          disabled={loading}
          onClick={onDelete}
        >
          {t("common.delete")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
