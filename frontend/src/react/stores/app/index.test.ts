import { create as createProto } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ReactShellBridgeEvent } from "@/react/shell-bridge";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { ActuatorInfoSchema } from "@/types/proto-es/v1/actuator_service_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BindingSchema,
  IamPolicySchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import {
  DatabaseChangeMode,
  EnvironmentSetting_EnvironmentSchema,
  EnvironmentSettingSchema,
  SettingSchema,
  SettingValueSchema,
  WorkspaceProfileSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  PlanFeature,
  PlanType,
  SubscriptionSchema,
} from "@/types/proto-es/v1/subscription_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import {
  storageKeyIntroState,
  storageKeyRecentProjects,
  storageKeyRecentVisit,
} from "@/utils/storage-keys";
import { createAppStore } from ".";

const mocks = vi.hoisted(() => ({
  localStorage: (() => {
    const storage = new Map<string, string>();
    const localStorage = {
      clear: vi.fn(() => storage.clear()),
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      removeItem: vi.fn((key: string) => storage.delete(key)),
      setItem: vi.fn((key: string, value: string) => {
        storage.set(key, value);
      }),
    };
    vi.stubGlobal("localStorage", localStorage);
    return localStorage;
  })(),
  getCurrentUser: vi.fn(),
  logout: vi.fn(),
  getActuatorInfo: vi.fn(),
  getWorkspace: vi.fn(),
  getIamPolicy: vi.fn(),
  listRoles: vi.fn(),
  getSubscription: vi.fn(),
  uploadLicense: vi.fn(),
  getSetting: vi.fn(),
  getProject: vi.fn(),
  getProjectIamPolicy: vi.fn(),
  batchGetProjects: vi.fn(),
  searchProjects: vi.fn(),
  createProject: vi.fn(),
  getInstance: vi.fn(),
}));

vi.mock("@/connect", () => ({
  actuatorServiceClientConnect: {
    getActuatorInfo: mocks.getActuatorInfo,
  },
  authServiceClientConnect: {
    logout: mocks.logout,
  },
  projectServiceClientConnect: {
    getProject: mocks.getProject,
    getIamPolicy: mocks.getProjectIamPolicy,
    batchGetProjects: mocks.batchGetProjects,
    searchProjects: mocks.searchProjects,
    createProject: mocks.createProject,
  },
  instanceServiceClientConnect: {
    getInstance: mocks.getInstance,
  },
  roleServiceClientConnect: {
    listRoles: mocks.listRoles,
  },
  settingServiceClientConnect: {
    getSetting: mocks.getSetting,
  },
  subscriptionServiceClientConnect: {
    getSubscription: mocks.getSubscription,
    uploadLicense: mocks.uploadLicense,
  },
  userServiceClientConnect: {
    getCurrentUser: mocks.getCurrentUser,
  },
  workspaceServiceClientConnect: {
    getWorkspace: mocks.getWorkspace,
    getIamPolicy: mocks.getIamPolicy,
  },
}));

const user = createProto(UserSchema, {
  name: "users/alice@example.com",
  email: "alice@example.com",
  groups: ["groups/dba"],
  workspace: "workspaces/default",
});

const projectA = createProto(ProjectSchema, {
  name: "projects/a",
  title: "A",
  state: State.ACTIVE,
});

const projectB = createProto(ProjectSchema, {
  name: "projects/b",
  title: "B",
  state: State.ACTIVE,
});

const timestampSeconds = (seconds: number) => ({
  seconds: BigInt(seconds),
  nanos: 0,
});

beforeEach(() => {
  vi.clearAllMocks();
  localStorage.clear();
});

describe("useAppStore", () => {
  test("combines app state slices behind one bounded store", () => {
    const store = createAppStore();
    const state = store.getState();

    expect(state.loadCurrentUser).toBeTypeOf("function");
    expect(state.loadWorkspace).toBeTypeOf("function");
    expect(state.hasWorkspacePermission).toBeTypeOf("function");
    expect(state.fetchProject).toBeTypeOf("function");
    expect(state.fetchInstance).toBeTypeOf("function");
    expect(state.notify).toBeTypeOf("function");
    expect(state.recordRecentVisit).toBeTypeOf("function");
    expect(state.removeRecentVisit).toBeTypeOf("function");
  });

  test("deduplicates project fetches and caches the result", async () => {
    mocks.getProject.mockResolvedValue(projectA);
    const store = createAppStore();

    const [first, second] = await Promise.all([
      store.getState().fetchProject(projectA.name),
      store.getState().fetchProject(projectA.name),
    ]);

    expect(first).toBe(projectA);
    expect(second).toBe(projectA);
    expect(mocks.getProject).toHaveBeenCalledTimes(1);
    expect(store.getState().projectsByName[projectA.name]).toBe(projectA);
  });

  test("falls back to individual project fetches when batch fetch fails", async () => {
    mocks.batchGetProjects.mockRejectedValue(new Error("batch failed"));
    mocks.getProject.mockImplementation(({ name }: { name: string }) => {
      return Promise.resolve(name === projectA.name ? projectA : projectB);
    });
    const store = createAppStore();

    const projects = await store
      .getState()
      .batchFetchProjects([projectA.name, projectB.name]);

    expect(projects).toEqual([projectA, projectB]);
    expect(mocks.batchGetProjects).toHaveBeenCalledTimes(1);
    expect(mocks.getProject).toHaveBeenCalledTimes(2);
  });

  test("writes created projects into the entity cache", async () => {
    mocks.createProject.mockResolvedValue(projectA);
    const store = createAppStore();

    const created = await store.getState().createProject("A", "a");

    expect(created).toBe(projectA);
    expect(store.getState().projectsByName[projectA.name]).toBe(projectA);
  });

  test("checks workspace permission through user, groups, and roles", () => {
    const store = createAppStore();
    store.setState({
      currentUser: user,
      roles: [
        createProto(RoleSchema, {
          name: "roles/admin",
          permissions: ["bb.projects.create"],
        }),
        createProto(RoleSchema, {
          name: "roles/viewer",
          permissions: ["bb.projects.get"],
        }),
      ],
      workspacePolicy: createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/admin",
            members: ["group:dba"],
          }),
          createProto(BindingSchema, {
            role: "roles/viewer",
            members: ["user:allUsers"],
          }),
        ],
      }),
    });

    expect(store.getState().hasWorkspacePermission("bb.projects.create")).toBe(
      true
    );
    expect(store.getState().hasWorkspacePermission("bb.settings.set")).toBe(
      false
    );
  });

  test("ignores expired IAM bindings when checking workspace permission", () => {
    const store = createAppStore();
    const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    store.setState({
      currentUser: user,
      roles: [
        createProto(RoleSchema, {
          name: "roles/admin",
          permissions: ["bb.projects.create"],
        }),
      ],
      workspacePolicy: createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/admin",
            members: ["group:dba"],
            condition: createProto(ExprSchema, {
              expression: `request.time < timestamp("${yesterday}")`,
            }),
          }),
        ],
      }),
    });

    expect(store.getState().hasWorkspacePermission("bb.projects.create")).toBe(
      false
    );
  });

  test("derives app features from the workspace profile setting", async () => {
    mocks.getSetting.mockResolvedValue(
      createProto(SettingSchema, {
        value: createProto(SettingValueSchema, {
          value: {
            case: "workspaceProfile",
            value: createProto(WorkspaceProfileSettingSchema, {
              databaseChangeMode: DatabaseChangeMode.EDITOR,
            }),
          },
        }),
      })
    );
    const store = createAppStore();

    await store.getState().loadWorkspaceProfile();

    expect(store.getState().appFeatures["bb.feature.hide-quick-start"]).toBe(
      true
    );
  });

  test("uses pipeline app feature defaults before profile overrides", () => {
    const store = createAppStore();

    expect(
      store.getState().appFeatures["bb.feature.database-change-mode"]
    ).toBe(DatabaseChangeMode.PIPELINE);
    expect(store.getState().appFeatures["bb.feature.hide-quick-start"]).toBe(
      false
    );
    expect(store.getState().appFeatures["bb.feature.hide-trial"]).toBe(false);
  });

  test("derives actuator state for React consumers", () => {
    const store = createAppStore();
    store.setState({
      serverInfo: createProto(ActuatorInfoSchema, {
        workspace: "workspaces/default",
        version: "3.10.2",
        externalUrl: "",
        saas: true,
        activatedInstanceCount: 2,
        totalInstanceCount: 3,
        userCountInIam: 7,
      }),
    });

    expect(store.getState().isSaaSMode()).toBe(true);
    expect(store.getState().workspaceResourceName()).toBe("workspaces/default");
    expect(store.getState().needConfigureExternalUrl()).toBe(true);
    expect(store.getState().changelogURL()).toBe(
      "https://docs.bytebase.com/changelog/bytebase-3-10-2/"
    );
    expect(store.getState().activatedInstanceCount()).toBe(2);
    expect(store.getState().totalInstanceCount()).toBe(3);
    expect(store.getState().userCountInIam()).toBe(7);
  });

  test("derives subscription limits and feature state", () => {
    const store = createAppStore();
    store.setState({
      subscription: createProto(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        seats: -1,
        instances: -1,
        activeInstances: -1,
        expiresTime: timestampSeconds(
          Math.floor(Date.now() / 1000) + 60 * 60 * 24 * 30
        ),
      }),
    });

    expect(store.getState().currentPlan()).toBe(PlanType.ENTERPRISE);
    expect(store.getState().isFreePlan()).toBe(false);
    expect(store.getState().isExpired()).toBe(false);
    expect(store.getState().showTrial()).toBe(false);
    expect(store.getState().instanceCountLimit()).toBe(Number.MAX_VALUE);
    expect(store.getState().userCountLimit()).toBe(Number.MAX_VALUE);
    expect(store.getState().instanceLicenseCount()).toBe(Number.MAX_VALUE);
    expect(store.getState().hasUnifiedInstanceLicense()).toBe(true);
    expect(store.getState().hasFeature(PlanFeature.FEATURE_DATA_MASKING)).toBe(
      true
    );
    expect(
      store.getState().hasInstanceFeature(
        PlanFeature.FEATURE_DATA_MASKING,
        createProto(InstanceSchema, {
          name: "instances/prod",
          activation: false,
        })
      )
    ).toBe(true);
    expect(
      store.getState().instanceMissingLicense(
        PlanFeature.FEATURE_DATA_MASKING,
        createProto(InstanceSchema, {
          name: "instances/prod",
          activation: false,
        })
      )
    ).toBe(false);

    store.setState({
      subscription: createProto(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        instances: 50,
        activeInstances: 20,
        expiresTime: timestampSeconds(
          Math.floor(Date.now() / 1000) + 60 * 60 * 24 * 30
        ),
      }),
    });
    expect(store.getState().hasUnifiedInstanceLicense()).toBe(false);
    expect(
      store.getState().hasInstanceFeature(
        PlanFeature.FEATURE_DATA_MASKING,
        createProto(InstanceSchema, {
          name: "instances/prod",
          activation: false,
        })
      )
    ).toBe(false);
    expect(
      store.getState().instanceMissingLicense(
        PlanFeature.FEATURE_DATA_MASKING,
        createProto(InstanceSchema, {
          name: "instances/prod",
          activation: false,
        })
      )
    ).toBe(true);
  });

  test("treats expired subscriptions as feature unavailable", () => {
    const store = createAppStore();
    store.setState({
      subscription: createProto(SubscriptionSchema, {
        plan: PlanType.ENTERPRISE,
        expiresTime: timestampSeconds(
          Math.floor(Date.now() / 1000) - 60 * 60 * 24
        ),
      }),
    });

    expect(store.getState().isExpired()).toBe(true);
    expect(store.getState().hasFeature(PlanFeature.FEATURE_DATA_MASKING)).toBe(
      false
    );
  });

  test("refreshes subscription state after external mutations", async () => {
    mocks.getSubscription.mockResolvedValue(
      createProto(SubscriptionSchema, {
        plan: PlanType.TEAM,
        seats: 12,
      })
    );
    const store = createAppStore();
    store.setState({
      subscription: createProto(SubscriptionSchema, {
        plan: PlanType.FREE,
        seats: 0,
      }),
    });

    const subscription = await store.getState().refreshSubscription();

    expect(subscription?.plan).toBe(PlanType.TEAM);
    expect(store.getState().currentPlan()).toBe(PlanType.TEAM);
    expect(store.getState().userCountLimit()).toBe(12);
  });

  test("loads environment settings into React state", async () => {
    mocks.getSetting.mockResolvedValue(
      createProto(SettingSchema, {
        value: createProto(SettingValueSchema, {
          value: {
            case: "environment",
            value: createProto(EnvironmentSettingSchema, {
              environments: [
                createProto(EnvironmentSetting_EnvironmentSchema, {
                  id: "dev",
                  title: "Development",
                  color: "#00aa00",
                }),
                createProto(EnvironmentSetting_EnvironmentSchema, {
                  id: "prod",
                  title: "Production",
                  tags: { protected: "protected" },
                }),
              ],
            }),
          },
        }),
      })
    );
    const store = createAppStore();

    const environments = await store.getState().loadEnvironmentList();

    expect(environments.map((env) => env.name)).toEqual([
      "environments/dev",
      "environments/prod",
    ]);
    expect(store.getState().environmentList[1]).toMatchObject({
      id: "prod",
      order: 1,
      tags: { protected: "protected" },
    });
  });

  test("refreshes environment settings after cache warm-up", async () => {
    mocks.getSetting
      .mockResolvedValueOnce(
        createProto(SettingSchema, {
          value: createProto(SettingValueSchema, {
            value: {
              case: "environment",
              value: createProto(EnvironmentSettingSchema, {
                environments: [
                  createProto(EnvironmentSetting_EnvironmentSchema, {
                    id: "dev",
                    title: "Development",
                  }),
                ],
              }),
            },
          }),
        })
      )
      .mockResolvedValueOnce(
        createProto(SettingSchema, {
          value: createProto(SettingValueSchema, {
            value: {
              case: "environment",
              value: createProto(EnvironmentSettingSchema, {
                environments: [
                  createProto(EnvironmentSetting_EnvironmentSchema, {
                    id: "prod",
                    title: "Production",
                  }),
                ],
              }),
            },
          }),
        })
      );
    const store = createAppStore();

    await store.getState().loadEnvironmentList();
    await store.getState().loadEnvironmentList();
    expect(mocks.getSetting).toHaveBeenCalledTimes(1);

    const environments = await store.getState().refreshEnvironmentList();

    expect(mocks.getSetting).toHaveBeenCalledTimes(2);
    expect(environments.map((env) => env.name)).toEqual(["environments/prod"]);
  });

  test("loads project IAM policy and checks project permissions", async () => {
    mocks.getCurrentUser.mockResolvedValue(user);
    mocks.getActuatorInfo.mockResolvedValue({
      workspace: "workspaces/default",
    });
    mocks.getWorkspace.mockResolvedValue({ name: "workspaces/default" });
    mocks.listRoles.mockResolvedValue({
      roles: [
        createProto(RoleSchema, {
          name: "roles/projectDeveloper",
          permissions: ["bb.projects.update"],
        }),
      ],
    });
    mocks.getIamPolicy.mockResolvedValue(createProto(IamPolicySchema, {}));
    mocks.getProjectIamPolicy.mockResolvedValue(
      createProto(IamPolicySchema, {
        bindings: [
          createProto(BindingSchema, {
            role: "roles/projectDeveloper",
            members: ["user:alice@example.com"],
          }),
        ],
      })
    );
    const store = createAppStore();

    await store.getState().loadProjectIamPolicy(projectA.name);

    expect(mocks.getProjectIamPolicy).toHaveBeenCalledTimes(1);
    expect(
      store.getState().hasProjectPermission(projectA, "bb.projects.update")
    ).toBe(true);
    expect(
      store.getState().hasProjectPermission(projectA, "bb.projects.delete")
    ).toBe(false);
  });

  test("caches instances by resource name", async () => {
    const instance = createProto(InstanceSchema, {
      name: "instances/prod",
      title: "Prod",
    });
    mocks.getInstance.mockResolvedValue(instance);
    const store = createAppStore();

    const [first, second] = await Promise.all([
      store.getState().fetchInstance(instance.name),
      store.getState().fetchInstance(instance.name),
    ]);

    expect(first).toBe(instance);
    expect(second).toBe(instance);
    expect(mocks.getInstance).toHaveBeenCalledTimes(1);
    expect(store.getState().instancesByName[instance.name]).toBe(instance);
  });

  test("notify() routes to the React toast manager", async () => {
    const { toastManager } = await import("@/react/lib/toast");
    const addSpy = vi.spyOn(toastManager, "add");
    const store = createAppStore();

    store.getState().notify({
      module: "bytebase",
      style: "SUCCESS",
      title: "Saved",
    });

    expect(addSpy).toHaveBeenCalledTimes(1);
    expect(addSpy.mock.calls[0][0]).toMatchObject({
      title: "Saved",
      type: "success",
    });
    addSpy.mockRestore();
  });

  test("keeps user-scoped preferences in localStorage", () => {
    const store = createAppStore();
    const listener = vi.fn();
    store.setState({ currentUser: user });
    window.addEventListener(ReactShellBridgeEvent.quickstartReset, listener);

    store.getState().setRecentProject(projectA.name);
    store.getState().setRecentProject(projectB.name);
    store.getState().recordRecentVisit("/projects/a?tab=1");
    store.getState().recordRecentVisit("/projects/a?tab=2");
    store.getState().removeRecentVisit("/missing");
    store.getState().resetQuickstartProgress();

    expect(
      JSON.parse(localStorage.getItem(storageKeyRecentProjects(user.email))!)
    ).toEqual([projectB.name, projectA.name]);
    expect(
      JSON.parse(localStorage.getItem(storageKeyRecentVisit(user.email))!)
    ).toEqual(["/projects/a?tab=2"]);
    expect(
      JSON.parse(localStorage.getItem(storageKeyIntroState(user.email))!)
    ).toMatchObject({
      hidden: false,
      "project.visit": false,
      "data.query": false,
    });
    expect(listener).toHaveBeenCalledWith(
      expect.objectContaining({
        detail: expect.objectContaining({
          keys: expect.arrayContaining(["hidden", "data.query"]),
        }),
      })
    );
    window.removeEventListener(ReactShellBridgeEvent.quickstartReset, listener);
  });
});
