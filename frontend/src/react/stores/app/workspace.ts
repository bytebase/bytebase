import { create as createProto } from "@bufbuild/protobuf";
import {
  actuatorServiceClientConnect,
  settingServiceClientConnect,
  subscriptionServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/connect";
import {
  settingNamePrefix,
  workspaceNamePrefix,
} from "@/react/lib/resourceName";
import { defaultAppProfile } from "@/types/appProfile";
import {
  DatabaseChangeMode,
  GetSettingRequestSchema,
  Setting_SettingName,
  WorkspaceProfileSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  GetSubscriptionRequestSchema,
  UploadLicenseRequestSchema,
} from "@/types/proto-es/v1/subscription_service_pb";
import type { AppSliceCreator, WorkspaceSlice } from "./types";

const workspaceProfileSettingName = `${settingNamePrefix}${
  Setting_SettingName[Setting_SettingName.WORKSPACE_PROFILE]
}`;

function appFeaturesFromDatabaseChangeMode(mode: DatabaseChangeMode) {
  const appFeatures = defaultAppProfile().features;
  appFeatures["bb.feature.database-change-mode"] =
    mode === DatabaseChangeMode.EDITOR
      ? DatabaseChangeMode.EDITOR
      : DatabaseChangeMode.PIPELINE;
  if (
    appFeatures["bb.feature.database-change-mode"] === DatabaseChangeMode.EDITOR
  ) {
    appFeatures["bb.feature.hide-quick-start"] = true;
    appFeatures["bb.feature.hide-trial"] = true;
  }
  return appFeatures;
}

export const createWorkspaceSlice: AppSliceCreator<WorkspaceSlice> = (
  set,
  get
) => ({
  serverInfoTs: 0,
  appFeatures: defaultAppProfile().features,

  loadServerInfo: async () => {
    const existing = get().serverInfo;
    if (existing) return existing;
    const pending = get().serverInfoRequest;
    if (pending) return pending;
    const request = actuatorServiceClientConnect
      .getActuatorInfo({ name: get().currentUser?.workspace ?? "" })
      .then((info) => {
        set({
          serverInfo: info,
          serverInfoRequest: undefined,
          serverInfoTs: Date.now(),
        });
        return info;
      })
      .catch(() => {
        set({ serverInfoRequest: undefined });
        return undefined;
      });
    set({ serverInfoRequest: request });
    return request;
  },

  refreshServerInfo: async () => {
    const info = await actuatorServiceClientConnect.getActuatorInfo({
      name: get().currentUser?.workspace ?? get().serverInfo?.workspace ?? "",
    });
    set({ serverInfo: info, serverInfoTs: Date.now() });
    return info;
  },

  loadWorkspace: async () => {
    const existing = get().workspace;
    if (existing) return existing;
    const pending = get().workspaceRequest;
    if (pending) return pending;
    await Promise.all([get().loadCurrentUser(), get().loadServerInfo()]);
    const name =
      get().serverInfo?.workspace ||
      get().currentUser?.workspace ||
      `${workspaceNamePrefix}-`;
    const request = workspaceServiceClientConnect
      .getWorkspace({ name })
      .then((workspace) => {
        set({ workspace, workspaceRequest: undefined });
        return workspace;
      })
      .catch(() => {
        set({ workspaceRequest: undefined });
        return undefined;
      });
    set({ workspaceRequest: request });
    return request;
  },

  loadWorkspaceProfile: async () => {
    const existing = get().workspaceProfile;
    if (existing) return existing;
    const pending = get().workspaceProfileRequest;
    if (pending) return pending;
    const request = settingServiceClientConnect
      .getSetting(
        createProto(GetSettingRequestSchema, {
          name: workspaceProfileSettingName,
        })
      )
      .then((setting) => {
        const settingValue = setting.value?.value;
        const profile =
          settingValue?.case === "workspaceProfile"
            ? settingValue.value
            : createProto(WorkspaceProfileSettingSchema, {});
        set({
          workspaceProfile: profile,
          workspaceProfileRequest: undefined,
          appFeatures: appFeaturesFromDatabaseChangeMode(
            profile.databaseChangeMode
          ),
        });
        return profile;
      })
      .catch(() => {
        set({ workspaceProfileRequest: undefined });
        return undefined;
      });
    set({ workspaceProfileRequest: request });
    return request;
  },

  loadSubscription: async () => {
    const existing = get().subscription;
    if (existing) return existing;
    const pending = get().subscriptionRequest;
    if (pending) return pending;
    const request = subscriptionServiceClientConnect
      .getSubscription(createProto(GetSubscriptionRequestSchema, {}))
      .then((subscription) => {
        set({ subscription, subscriptionRequest: undefined });
        return subscription;
      })
      .catch(() => {
        set({ subscriptionRequest: undefined });
        return undefined;
      });
    set({ subscriptionRequest: request });
    return request;
  },

  uploadLicense: async (license) => {
    const subscription = await subscriptionServiceClientConnect.uploadLicense(
      createProto(UploadLicenseRequestSchema, { license })
    );
    set({ subscription });
    return subscription;
  },
});
