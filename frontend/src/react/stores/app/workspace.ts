import { create as createProto } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { createContextValues } from "@connectrpc/connect";
import dayjs from "dayjs";
import semver from "semver";
import {
  actuatorServiceClientConnect,
  authServiceClientConnect,
  settingServiceClientConnect,
  subscriptionServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import {
  settingNamePrefix,
  workspaceNamePrefix,
} from "@/react/lib/resourceName";
import { WORKSPACE_ROUTE_LANDING } from "@/react/router";
import { resolvePath } from "@/react/router/navigation";
import { getEnvironmentId } from "@/store/modules/v1/common";
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
  type DataClassificationSetting_DataClassificationConfig,
  type EnvironmentSetting_Environment,
  EnvironmentSetting_EnvironmentSchema,
  EnvironmentSettingSchema,
  GetSettingRequestSchema,
  Setting_SettingName,
  SettingSchema,
  SettingValueSchema,
  UpdateSettingRequestSchema,
  WorkspaceProfileSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  CancelPurchaseRequestSchema,
  CreatePurchaseRequestSchema,
  GetPaymentInfoRequestSchema,
  GetSubscriptionRequestSchema,
  ListPurchasePlansRequestSchema,
  PlanType,
  UpdatePurchaseRequestSchema,
  UploadLicenseRequestSchema,
  VerifyCheckoutSessionRequestSchema,
} from "@/types/proto-es/v1/subscription_service_pb";
import {
  UpdateWorkspaceRequestSchema,
  type Workspace,
} from "@/types/proto-es/v1/workspace_service_pb";
import {
  getDateForPbTimestampProtoEs,
  getTimeForPbTimestampProtoEs,
} from "@/types/timestamp";
import type { Environment } from "@/types/v1/environment";
import {
  nullEnvironment as createNullEnvironment,
  unknownEnvironment as createUnknownEnvironment,
  isValidEnvironmentName,
  NULL_ENVIRONMENT_NAME,
} from "@/types/v1/environment";
import { colorToHex, hexToColor } from "@/utils";
import { formatAbsoluteDateTime } from "@/utils/datetime";
import type { AppSliceCreator, WorkspaceSlice } from "./types";

let workspaceSwitchListenerRegistered = false;

const workspaceProfileSettingName = `${settingNamePrefix}${
  Setting_SettingName[Setting_SettingName.WORKSPACE_PROFILE]
}`;
const environmentSettingName = `${settingNamePrefix}${
  Setting_SettingName[Setting_SettingName.ENVIRONMENT]
}`;
const externalUrlPlaceholder =
  "https://docs.bytebase.com/get-started/self-host/external-url";
const trialingDays = 14;

// Stable empty profile so `getWorkspaceProfile()` can return a non-null value
// without allocating a fresh object each call (which would break Zustand's
// selector identity check). Mirrors the Pinia getter's default.
const EMPTY_WORKSPACE_PROFILE = createProto(WorkspaceProfileSettingSchema, {});

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
      tags: env.tags,
    }),
    color: env.color ? colorToHex(env.color) : "",
    order: i,
  }));
}

function isSelfHostLicense() {
  return import.meta.env.MODE.toLowerCase() !== "release-aws";
}

function convertEnvironmentsToSetting(
  environments: Environment[]
): EnvironmentSetting_Environment[] {
  return environments.map((env) =>
    createProto(EnvironmentSetting_EnvironmentSchema, {
      name: env.name,
      id: env.id,
      title: env.title,
      color: env.color ? hexToColor(env.color) : undefined,
      tags: env.tags,
    })
  );
}

export const createWorkspaceSlice: AppSliceCreator<WorkspaceSlice> = (
  set,
  get
) => {
  // Listen on the shared cross-tab channel (see store/workspaceSwitchChannel.ts).
  // Using `addEventListener` rather than `onmessage = ...` allows the Vue-side
  // store to register its own handler on the same object, and source-object
  // exclusion correctly suppresses both handlers when a post originates from
  // this tab (e.g. the OAuth2 consent page's in-place switch).
  if (!workspaceSwitchListenerRegistered) {
    workspaceSwitchListenerRegistered = true;
    workspaceSwitchChannel.addEventListener("message", (event) => {
      get().recordRecentVisit(
        resolvePath(WORKSPACE_ROUTE_LANDING),
        typeof event.data === "string" ? event.data : undefined
      );
      window.location.href = "/";
    });
  }

  const unknownEnvironment = createUnknownEnvironment();
  const nullEnvironment = createNullEnvironment();
  const environmentFallbacksByName = new Map<string, Environment>();

  const getEnvironmentFallback = (name: string, id: string) => {
    const existing = environmentFallbacksByName.get(name);
    if (existing) return existing;
    const environment = { ...unknownEnvironment, id, name, title: id };
    environmentFallbacksByName.set(name, environment);
    return environment;
  };

  // Persist the ENVIRONMENT setting and re-derive the cached environment list
  // from the server's response (mirrors the legacy Pinia env store).
  const writeEnvironmentSetting = async (
    environments: EnvironmentSetting_Environment[]
  ): Promise<Environment[]> => {
    const response = await get().upsertSetting({
      name: Setting_SettingName.ENVIRONMENT,
      value: createProto(SettingValueSchema, {
        value: {
          case: "environment",
          value: createProto(EnvironmentSettingSchema, { environments }),
        },
      }),
    });
    const next =
      response.value?.value?.case === "environment"
        ? convertEnvironmentList(response.value.value.value.environments)
        : [];
    set({ environmentList: next });
    return next;
  };

  return {
    serverInfoTs: 0,
    workspaceList: [],
    environmentList: [],
    settingsByName: {},
    settingRequests: {},
    appFeatures: defaultAppProfile().features,
    purchasePlans: [],

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
      // Resolve the authenticated user via the cached loader. It returns the
      // existing currentUser when present (login / fetchCurrentUser refresh it
      // on the auth transition), so we must NOT mint a fresh currentUser object
      // here: a raw getCurrentUser() response is a new reference every call,
      // which thrashes effects keyed on `currentUser` (e.g. AuthGate's mount
      // gate) into an unmount/remount fetch loop.
      const user = await get().loadCurrentUser();
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

    updateWorkspace: async (workspace: Workspace, updateMask: string[]) => {
      const updated = await workspaceServiceClientConnect.updateWorkspace(
        createProto(UpdateWorkspaceRequestSchema, {
          workspace,
          updateMask: createProto(FieldMaskSchema, { paths: updateMask }),
        })
      );
      set((state) => ({
        workspace:
          state.workspace?.name === updated.name ? updated : state.workspace,
        workspaceList: state.workspaceList.map((ws) =>
          ws.name === updated.name ? updated : ws
        ),
      }));
      return updated;
    },

    switchWorkspace: async (workspaceName: string, redirect = true) => {
      await authServiceClientConnect.switchWorkspace(
        createProto(SwitchWorkspaceRequestSchema, {
          workspace: workspaceName,
          web: true,
        })
      );
      get().recordRecentVisit(
        resolvePath(WORKSPACE_ROUTE_LANDING),
        workspaceName
      );
      // Notify other tabs to reload with the new workspace.
      broadcastWorkspaceSwitch(workspaceName);
      if (redirect) {
        // Full-reload to the landing page to reset all frontend state.
        window.location.href = "/";
      }
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

    // Mirrors the Pinia `useEnvironmentV1Store().getEnvironmentByName`: maps
    // a resource name to its environment, falling back to a synthesized
    // entry (id-as-title) for unknown names when `fallback` is set.
    getEnvironmentByName: (name, fallback = true) => {
      if (name === NULL_ENVIRONMENT_NAME) {
        return nullEnvironment;
      }
      const id = getEnvironmentId(name);
      if (!id) {
        return unknownEnvironment;
      }
      const environment =
        get().environmentList.find((e) => e.id === id) ?? unknownEnvironment;
      if (!isValidEnvironmentName(environment.name) && fallback) {
        return getEnvironmentFallback(name, id);
      }
      return environment;
    },

    // Mirrors the Pinia `useSettingV1Store`: general-purpose setting cache
    // keyed by resource name (`settings/{Setting_SettingName}`). Used for AI /
    // workspace-profile / etc.
    getSettingByName: (name) => {
      const resourceName = `${settingNamePrefix}${Setting_SettingName[name]}`;
      return get().settingsByName[resourceName];
    },

    setSettingByName: (setting) => {
      set((state) => ({
        settingsByName: {
          ...state.settingsByName,
          [setting.name]: setting,
        },
      }));
    },

    upsertSetting: async ({
      name,
      value,
      validateOnly = false,
      updateMask,
    }) => {
      const response = await settingServiceClientConnect.updateSetting(
        createProto(UpdateSettingRequestSchema, {
          setting: createProto(SettingSchema, {
            name: `${settingNamePrefix}${Setting_SettingName[name]}`,
            value,
          }),
          validateOnly,
          allowMissing: true,
          updateMask,
        })
      );
      get().setSettingByName(response);
      return response;
    },

    getOrFetchSettingByName: async (name, silent = false) => {
      const resourceName = `${settingNamePrefix}${Setting_SettingName[name]}`;
      const cached = get().settingsByName[resourceName];
      if (cached) return cached;
      const pending = get().settingRequests[resourceName];
      if (pending) return pending;

      const request = settingServiceClientConnect
        .getSetting(
          createProto(GetSettingRequestSchema, { name: resourceName }),
          {
            contextValues: createContextValues().set(silentContextKey, silent),
          }
        )
        .then((response) => {
          set((state) => {
            const { [resourceName]: _, ...settingRequests } =
              state.settingRequests;
            return {
              settingsByName: {
                ...state.settingsByName,
                [response.name]: response,
              },
              settingRequests,
            };
          });
          return response;
        })
        .catch(() => {
          set((state) => {
            const { [resourceName]: _, ...settingRequests } =
              state.settingRequests;
            return { settingRequests };
          });
          return undefined;
        });
      set((state) => ({
        settingRequests: {
          ...state.settingRequests,
          [resourceName]: request,
        },
      }));
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

    activeVcsUserCount: () => get().serverInfo?.activeVcsUserCount ?? 0,

    activeUserCount: () => get().serverInfo?.activatedUserCount ?? 0,

    enableOnboarding: () =>
      get().activeUserCount() === 1 && !get().isSaaSMode(),

    quickStartEnabled: () => {
      if (get().appFeatures["bb.feature.hide-quick-start"]) {
        return false;
      }
      if (!get().serverInfo?.enableSample) {
        return false;
      }
      return get().activeUserCount() <= 1;
    },

    setupSample: async () => {
      await actuatorServiceClientConnect.setupSample({});
    },

    // Alias for the legacy Pinia `actuatorStore.fetchServerInfo(workspace?)`.
    fetchServerInfo: async (workspaceResourceName) => {
      const info = await actuatorServiceClientConnect.getActuatorInfo({
        name: workspaceResourceName ?? get().serverInfo?.workspace ?? "",
      });
      set({ serverInfo: info, serverInfoTs: Date.now() });
      return info;
    },

    // Always returns a profile (never undefined), mirroring the Pinia
    // `useSettingV1Store().workspaceProfile` getter, so consumers can read
    // fields without null checks. Reactive via Zustand's selector re-run.
    getWorkspaceProfile: () =>
      get().workspaceProfile ?? EMPTY_WORKSPACE_PROFILE,

    classification: () => {
      const setting =
        get().settingsByName[
          `${settingNamePrefix}${
            Setting_SettingName[Setting_SettingName.DATA_CLASSIFICATION]
          }`
        ];
      const value = setting?.value?.value;
      if (value?.case === "dataClassification") {
        return value.value.configs;
      }
      return [] as DataClassificationSetting_DataClassificationConfig[];
    },

    getProjectClassification: (classificationId) =>
      get()
        .classification()
        .find((config) => config.id === classificationId),

    updateWorkspaceProfile: async ({ payload, updateMask }) => {
      const base =
        get().workspaceProfile ??
        createProto(WorkspaceProfileSettingSchema, {});
      const profile = { ...base, ...payload };
      await get().upsertSetting({
        name: Setting_SettingName.WORKSPACE_PROFILE,
        value: createProto(SettingValueSchema, {
          value: { case: "workspaceProfile", value: profile },
        }),
        updateMask,
      });
      set({
        workspaceProfile: profile,
        appFeatures: appFeaturesFromDatabaseChangeMode(
          profile.databaseChangeMode
        ),
      });
      // Refresh the latest server info (mirrors the Pinia store).
      await get().fetchServerInfo(get().workspaceResourceName());
    },

    fetchEnvironments: async (force = false) => {
      await get().loadEnvironmentList(force);
    },

    createEnvironment: async (environment) => {
      const next = await writeEnvironmentSetting([
        ...convertEnvironmentsToSetting(get().environmentList),
        createProto(EnvironmentSetting_EnvironmentSchema, {
          name: "",
          id: environment.id ?? "",
          title: environment.title ?? "",
          color: environment.color ? hexToColor(environment.color) : undefined,
          tags: environment.tags ?? {},
        }),
      ]);
      const created = next.find((e) => e.id === (environment.id ?? ""));
      if (!created) {
        throw new Error(`environment with id ${environment.id} not found`);
      }
      return created;
    },

    updateEnvironment: async (update) => {
      const next = await writeEnvironmentSetting(
        convertEnvironmentsToSetting(
          get().environmentList.map((environment) =>
            environment.id === update.id
              ? {
                  ...environment,
                  title: update.title ?? environment.title,
                  color: update.color ?? environment.color,
                  tags: update.tags ?? environment.tags,
                  order: update.order ?? environment.order,
                }
              : environment
          )
        )
      );
      const updated = next.find((e) => e.id === update.id);
      if (!updated) {
        throw new Error(`environment with id ${update.id} not found`);
      }
      return updated;
    },

    deleteEnvironment: async (name) => {
      const id = getEnvironmentId(name);
      await writeEnvironmentSetting(
        convertEnvironmentsToSetting(
          get().environmentList.filter((environment) => environment.id !== id)
        )
      );
    },

    reorderEnvironmentList: async (orderedEnvironmentList) =>
      writeEnvironmentSetting(
        convertEnvironmentsToSetting(orderedEnvironmentList)
      ),

    setSubscription: (subscription) => {
      set({ subscription });
    },

    hasSplitInstanceLicense: () =>
      !get().isFreePlan() && !get().hasUnifiedInstanceLicense(),

    pollSubscriptionUntil: async (predicate, options = {}) => {
      const { timeoutMs = 60_000, intervalMs = 2_000, signal } = options;
      const deadline = Date.now() + timeoutMs;
      while (Date.now() < deadline) {
        if (signal?.aborted) return undefined;
        const sub = await subscriptionServiceClientConnect
          .getSubscription(createProto(GetSubscriptionRequestSchema, {}))
          .catch(() => undefined);
        if (signal?.aborted) return undefined;
        if (sub && predicate(sub)) {
          get().setSubscription(sub);
          return sub;
        }
        await new Promise((r) => setTimeout(r, intervalMs));
      }
      return undefined;
    },

    createPurchase: async (plan, interval, seats) => {
      const response = await subscriptionServiceClientConnect.createPurchase(
        createProto(CreatePurchaseRequestSchema, { plan, interval, seats })
      );
      return response.paymentUrl;
    },

    updatePurchase: async (plan, interval, seats, etag) => {
      const response = await subscriptionServiceClientConnect.updatePurchase(
        createProto(UpdatePurchaseRequestSchema, {
          plan,
          interval,
          seats,
          etag,
        })
      );
      return response.paymentUrl;
    },

    cancelPurchase: async (feedback, comment) => {
      await subscriptionServiceClientConnect.cancelPurchase(
        createProto(CancelPurchaseRequestSchema, { feedback, comment })
      );
      await get().refreshSubscription();
    },

    fetchPaymentInfo: async () => {
      try {
        const info = await subscriptionServiceClientConnect.getPaymentInfo(
          createProto(GetPaymentInfoRequestSchema, {})
        );
        set({ paymentInfo: info });
        return info;
      } catch (e) {
        console.error(e);
        return undefined;
      }
    },

    verifyCheckoutSession: async (sessionId) => {
      const response =
        await subscriptionServiceClientConnect.verifyCheckoutSession(
          createProto(VerifyCheckoutSessionRequestSchema, { sessionId })
        );
      return response.status;
    },

    fetchPurchasePlans: async () => {
      try {
        const response =
          await subscriptionServiceClientConnect.listPurchasePlans(
            createProto(ListPurchasePlansRequestSchema, {})
          );
        set({ purchasePlans: response.plans });
        return response.plans;
      } catch (e) {
        console.error(e);
        return undefined;
      }
    },
  };
};
