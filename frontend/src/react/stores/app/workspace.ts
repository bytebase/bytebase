import { create as createProto } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import semver from "semver";
import {
  actuatorServiceClientConnect,
  authServiceClientConnect,
  settingServiceClientConnect,
  subscriptionServiceClientConnect,
  userServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/connect";
import {
  settingNamePrefix,
  workspaceNamePrefix,
} from "@/react/lib/resourceName";
import {
  broadcastWorkspaceSwitch,
  workspaceSwitchChannel,
} from "@/store/workspaceSwitchChannel";
import { defaultAppProfile } from "@/types/appProfile";
import {
  hasFeature as checkFeature,
  hasInstanceFeature as checkInstanceFeature,
  getMinimumRequiredPlan,
  instanceLimitFeature,
  PLANS,
} from "@/types/plan";
import { SwitchWorkspaceRequestSchema } from "@/types/proto-es/v1/auth_service_pb";
import {
  DatabaseChangeMode,
  type EnvironmentSetting_Environment,
  EnvironmentSetting_EnvironmentSchema,
  GetSettingRequestSchema,
  Setting_SettingName,
  WorkspaceProfileSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  GetSubscriptionRequestSchema,
  PlanType,
  UploadLicenseRequestSchema,
} from "@/types/proto-es/v1/subscription_service_pb";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
} from "@/types/timestamp";
import type { Environment } from "@/types/v1/environment";
import { formatAbsoluteDateTime } from "@/utils/datetime";
import type { AppSliceCreator, WorkspaceSlice } from "./types";

// Listen on the shared cross-tab channel (see store/workspaceSwitchChannel.ts).
// Using `addEventListener` rather than `onmessage = ...` allows the Vue-side
// store to register its own handler on the same object, and source-object
// exclusion correctly suppresses both handlers when a post originates from
// this tab (e.g. the OAuth2 consent page's in-place switch).
workspaceSwitchChannel.addEventListener("message", () => {
  window.location.href = "/";
});

const workspaceProfileSettingName = `${settingNamePrefix}${
  Setting_SettingName[Setting_SettingName.WORKSPACE_PROFILE]
}`;
const environmentSettingName = `${settingNamePrefix}${
  Setting_SettingName[Setting_SettingName.ENVIRONMENT]
}`;
const externalUrlPlaceholder =
  "https://docs.bytebase.com/get-started/self-host/external-url";
const trialingDays = 14;

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

function environmentNameFromId(id: string) {
  return `environments/${id}`;
}

function convertEnvironmentList(
  environments: EnvironmentSetting_Environment[]
): Environment[] {
  return environments.map<Environment>((env, i) => ({
    ...createProto(EnvironmentSetting_EnvironmentSchema, {
      name: environmentNameFromId(env.id),
      id: env.id,
      title: env.title,
      color: env.color,
      tags: env.tags,
    }),
    order: i,
  }));
}

function isSelfHostLicense() {
  return import.meta.env.MODE.toLowerCase() !== "release-aws";
}

export const createWorkspaceSlice: AppSliceCreator<WorkspaceSlice> = (
  set,
  get
) => ({
  serverInfoTs: 0,
  workspaceList: [],
  environmentList: [],
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
    // Always re-fetch currentUser to get the latest auth context.
    // The cached currentUser may be from before login (undefined or stale).
    const user = await userServiceClientConnect
      .getCurrentUser({})
      .catch(() => undefined);
    if (user) {
      set({ currentUser: user });
    }
    const name =
      user?.workspace ||
      get().currentUser?.workspace ||
      get().serverInfo?.workspace ||
      `${workspaceNamePrefix}-`;
    // Return cached workspace if it matches the current auth context.
    const existing = get().workspace;
    if (existing?.name === name) return existing;
    const pending = get().workspaceRequest;
    if (pending) return pending;
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

  loadWorkspaceList: async () => {
    const resp = await workspaceServiceClientConnect.listWorkspaces({});
    set({ workspaceList: resp.workspaces });
    return resp.workspaces;
  },

  switchWorkspace: async (workspaceName: string) => {
    await authServiceClientConnect.switchWorkspace(
      createProto(SwitchWorkspaceRequestSchema, {
        workspace: workspaceName,
        web: true,
      })
    );
    // Notify other tabs to reload with the new workspace.
    broadcastWorkspaceSwitch(workspaceName);
    // Full-reload to the landing page to reset all frontend state.
    window.location.href = "/";
  },

  loadWorkspaceProfile: async (force = false) => {
    const existing = get().workspaceProfile;
    if (!force && existing) return existing;
    const pending = get().workspaceProfileRequest;
    if (!force && pending) return pending;
    // request is captured so .then handlers can identity-check against
    // the latest in-flight request before writing — protects against an
    // older in-flight request resolving after a newer forced reload.
    const request = settingServiceClientConnect
      .getSetting(
        createProto(GetSettingRequestSchema, {
          name: workspaceProfileSettingName,
        })
      )
      .then((setting) => {
        if (get().workspaceProfileRequest !== request) {
          return get().workspaceProfile;
        }
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
        if (get().workspaceProfileRequest === request) {
          set({ workspaceProfileRequest: undefined });
        }
        return undefined;
      });
    set({ workspaceProfileRequest: request });
    return request;
  },

  loadEnvironmentList: async (force = false) => {
    const existing = get().environmentList;
    if (!force && existing.length > 0) return existing;
    const pending = get().environmentRequest;
    if (pending) return pending;
    const request = settingServiceClientConnect
      .getSetting(
        createProto(GetSettingRequestSchema, {
          name: environmentSettingName,
        })
      )
      .then((setting) => {
        const settingValue = setting.value?.value;
        const environments =
          settingValue?.case === "environment"
            ? convertEnvironmentList(settingValue.value.environments)
            : [];
        set({ environmentList: environments, environmentRequest: undefined });
        return environments;
      })
      .catch(() => {
        set({ environmentRequest: undefined });
        return [];
      });
    set({ environmentRequest: request });
    return request;
  },

  refreshEnvironmentList: async () => get().loadEnvironmentList(true),

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

  refreshSubscription: async () => {
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

  currentPlan: () => {
    return get().subscription?.plan ?? PlanType.FREE;
  },

  isFreePlan: () => get().currentPlan() === PlanType.FREE,

  isTrialing: () => Boolean(get().subscription?.trialing),

  isExpired: () => {
    const subscription = get().subscription;
    if (!subscription?.expiresTime || get().isFreePlan()) {
      return false;
    }
    return dayjs(
      getDateForPbTimestampProtoEs(subscription.expiresTime)
    ).isBefore(new Date());
  },

  daysBeforeExpire: () => {
    const subscription = get().subscription;
    if (!subscription?.expiresTime || get().isFreePlan()) {
      return -1;
    }
    return Math.max(
      dayjs(getDateForPbTimestampProtoEs(subscription.expiresTime)).diff(
        new Date(),
        "day"
      ),
      0
    );
  },

  trialingDays: () => trialingDays,

  showTrial: () => {
    if (!isSelfHostLicense()) {
      return false;
    }
    return !get().subscription || get().isFreePlan();
  },

  expireAt: () => {
    const subscription = get().subscription;
    if (!subscription?.expiresTime || get().isFreePlan()) {
      return "";
    }
    return formatAbsoluteDateTime(
      getTimeForPbTimestampProtoEs(subscription.expiresTime)
    );
  },

  instanceCountLimit: () => {
    const subscription = get().subscription;
    const licenseLimit = subscription?.instances ?? 0;
    if (licenseLimit > 0) {
      return licenseLimit;
    }
    const planLimit =
      PLANS.find((plan) => plan.type === get().currentPlan())
        ?.maximumInstanceCount ?? 0;
    if (planLimit < 0) {
      return licenseLimit > 0 ? licenseLimit : Number.MAX_VALUE;
    }
    return planLimit;
  },

  userCountLimit: () => {
    let limit =
      PLANS.find((plan) => plan.type === get().currentPlan())
        ?.maximumSeatCount ?? 0;
    if (limit < 0) {
      limit = Number.MAX_VALUE;
    }
    const seats = get().subscription?.seats ?? 0;
    if (seats < 0) {
      return Number.MAX_VALUE;
    }
    if (seats === 0) {
      return limit;
    }
    return seats;
  },

  instanceLicenseCount: () => {
    const count = get().subscription?.activeInstances ?? 0;
    return count < 0 ? Number.MAX_VALUE : count;
  },

  hasUnifiedInstanceLicense: () => {
    return get().instanceCountLimit() <= get().instanceLicenseCount();
  },

  hasFeature: (feature) => {
    if (get().isExpired()) {
      return false;
    }
    return checkFeature(get().currentPlan(), feature);
  },

  hasInstanceFeature: (feature, instance) => {
    const plan = get().currentPlan();
    if (plan === PlanType.FREE) {
      return get().hasFeature(feature);
    }
    if (!instance || !instanceLimitFeature.has(feature)) {
      return get().hasFeature(feature);
    }
    return checkInstanceFeature(
      plan,
      feature,
      get().hasUnifiedInstanceLicense() || instance.activation
    );
  },

  instanceMissingLicense: (feature, instance) => {
    if (!instanceLimitFeature.has(feature) || !instance) {
      return false;
    }
    if (get().hasUnifiedInstanceLicense()) {
      return false;
    }
    return get().hasFeature(feature) && !instance.activation;
  },

  getMinimumRequiredPlan,

  isSaaSMode: () => get().serverInfo?.saas ?? false,

  workspaceResourceName: () => get().serverInfo?.workspace ?? "",

  externalUrl: () => get().serverInfo?.externalUrl ?? "",

  needConfigureExternalUrl: () => {
    const serverInfo = get().serverInfo;
    if (!serverInfo) return false;
    const url = serverInfo.externalUrl ?? "";
    return url === "" || url === externalUrlPlaceholder;
  },

  version: () => get().serverInfo?.version ?? "",

  changelogURL: () => {
    const version = semver.valid(get().serverInfo?.version);
    if (!version) return "";
    return `https://docs.bytebase.com/changelog/bytebase-${version
      .split(".")
      .join("-")}/`;
  },

  activatedInstanceCount: () => get().serverInfo?.activatedInstanceCount ?? 0,

  totalInstanceCount: () => get().serverInfo?.totalInstanceCount ?? 0,

  userCountInIam: () => get().serverInfo?.userCountInIam ?? 0,
});
