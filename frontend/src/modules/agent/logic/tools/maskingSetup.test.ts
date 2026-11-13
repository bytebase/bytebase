// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import i18n from "@/lib/i18n";
import type { AppStoreState } from "@/stores/app/types";
import { bindingMatchesUser } from "@/stores/app/utils";
import type { Permission } from "@/types/iam/permission";
import { hasFeature } from "@/types/plan";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { ActuatorInfoSchema } from "@/types/proto-es/v1/actuator_service_pb";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import {
  BindingSchema,
  IamPolicySchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import {
  PlanFeature,
  PlanType,
  SubscriptionSchema,
} from "@/types/proto-es/v1/subscription_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { getMaskingSetup } from "./maskingSetup";

const workspaceName = "workspaces/example";
const projectName = "projects/example";
const databaseName = "instances/primary/databases/app.v2";
const permissions: Permission[] = [
  "bb.settings.get",
  "bb.settings.set",
  "bb.policies.update",
  "bb.policies.getMaskingRulePolicy",
  "bb.policies.updateMaskingRulePolicy",
  "bb.policies.createMaskingRulePolicy",
  "bb.projects.get",
  "bb.databases.get",
  "bb.databases.list",
  "bb.databaseCatalogs.get",
  "bb.databaseCatalogs.update",
  "bb.policies.getMaskingExemptionPolicy",
  "bb.policies.createMaskingExemptionPolicy",
  "bb.policies.updateMaskingExemptionPolicy",
  "bb.accessGrants.create",
];

const mocks = vi.hoisted(() => ({
  state: undefined as unknown as AppStoreState,
  database: undefined as unknown as Database,
  getProject: vi.fn(),
  getIamPolicy: vi.fn(),
  fetchCurrentUser: vi.fn(),
  loadWorkspacePermissionState: vi.fn(),
  loadSubscription: vi.fn(),
  refreshSubscription: vi.fn(),
  getOrFetchProjectByName: vi.fn(),
  deniedWorkspace: new Set<string>(),
  deniedProject: new Set<string>(),
}));

vi.mock("@/api", () => ({
  projectServiceClientConnect: {
    getProject: mocks.getProject,
    getIamPolicy: mocks.getIamPolicy,
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => mocks.state,
    setState: (update: (state: AppStoreState) => Partial<AppStoreState>) => {
      Object.assign(mocks.state, update(mocks.state));
    },
  },
}));

function step(result: Awaited<ReturnType<typeof getMaskingSetup>>, id: string) {
  const found = result.steps.find((item: { id: string }) => item.id === id);
  expect(found, `expected ${id} setup evidence`).toBeDefined();
  if (!found) throw new Error(`missing ${id} setup evidence`);
  return found;
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.deniedWorkspace.clear();
  mocks.deniedProject.clear();
  const project = create(ProjectSchema, {
    name: projectName,
    state: State.ACTIVE,
    allowJustInTimeAccess: true,
  });
  mocks.database = create(DatabaseSchema$, {
    name: databaseName,
    project: projectName,
  });
  mocks.state = {
    serverInfo: create(ActuatorInfoSchema, { workspace: workspaceName }),
    currentUser: create(UserSchema, { name: "users/member@example.com" }),
    workspacePolicy: create(IamPolicySchema),
    projectPoliciesByName: {},
    roles: [],
    subscription: create(SubscriptionSchema, { plan: PlanType.ENTERPRISE }),
    instancesByName: {},
    databasesByName: {
      [databaseName]: create(DatabaseSchema$, {
        name: databaseName,
        project: projectName,
      }),
    },
    getOrFetchProjectByName: mocks.getOrFetchProjectByName,
    fetchCurrentUser: mocks.fetchCurrentUser,
    loadWorkspacePermissionState: mocks.loadWorkspacePermissionState,
    loadSubscription: mocks.loadSubscription,
    refreshSubscription: mocks.refreshSubscription,
    hasWorkspacePermission: (permission) =>
      !mocks.deniedWorkspace.has(permission),
    hasProjectPermission: (_project, permission) =>
      !mocks.deniedProject.has(permission),
    currentPlan: () => mocks.state.subscription?.plan ?? PlanType.FREE,
    isExpired: () => false,
    hasFeature: (feature) =>
      !mocks.state.isExpired() &&
      hasFeature(mocks.state.currentPlan(), feature),
    hasInstanceFeature: (_feature, instance) => Boolean(instance?.activation),
    instanceMissingLicense: (_feature, instance) =>
      Boolean(instance && !instance.activation),
    isSaaSMode: () => false,
  } as Partial<AppStoreState> as AppStoreState;
  mocks.loadWorkspacePermissionState.mockImplementation(async () => {
    mocks.state.roles = [
      create(RoleSchema, { name: "roles/example", permissions }),
    ];
    mocks.state.workspacePolicy = create(IamPolicySchema, {
      bindings: [
        { role: "roles/example", members: ["user:member@example.com"] },
      ],
    });
  });
  mocks.fetchCurrentUser.mockImplementation(
    async () => mocks.state.currentUser
  );
  mocks.getOrFetchProjectByName.mockResolvedValue(project);
  mocks.getProject.mockResolvedValue(project);
  mocks.getIamPolicy.mockResolvedValue(create(IamPolicySchema));
  mocks.loadSubscription.mockResolvedValue(undefined);
  mocks.refreshSubscription.mockImplementation(
    async () => mocks.state.subscription
  );
});

describe("getMaskingSetup", () => {
  test.each([
    { cached: false, fresh: true, state: State.ACTIVE, eligible: true },
    { cached: true, fresh: false, state: State.ACTIVE, eligible: false },
    { cached: true, fresh: true, state: State.DELETED, eligible: false },
  ])(
    "uses fresh project state $state and JIT flag $fresh instead of cached flag $cached",
    async ({ cached, fresh, state, eligible }) => {
      const cachedProject = create(ProjectSchema, {
        name: projectName,
        state: State.ACTIVE,
        allowJustInTimeAccess: cached,
      });
      mocks.state.projectsByName = { [projectName]: cachedProject };
      mocks.getOrFetchProjectByName.mockResolvedValue(cachedProject);
      mocks.getProject.mockResolvedValue(
        create(ProjectSchema, {
          name: projectName,
          state,
          allowJustInTimeAccess: fresh,
        })
      );

      const result = await getMaskingSetup({ workspaceName, projectName });

      expect(result.justInTime.enabled).toBe(state === State.ACTIVE && fresh);
      expect(result.project).toEqual({ name: projectName, state });
      expect(result.justInTime.eligible).toBe(eligible);
      expect(mocks.getProject).toHaveBeenCalledWith(
        expect.objectContaining({ name: projectName }),
        expect.objectContaining({ contextValues: expect.anything() })
      );
      expect(mocks.getOrFetchProjectByName).not.toHaveBeenCalled();
    }
  );

  test("keeps a permitted exemption preferred over an eligible Access Grant", async () => {
    const result = await getMaskingSetup({ workspaceName, projectName });

    expect(result.justInTime).toMatchObject({
      enabled: true,
      feature: "available",
      create: {
        status: "allowed",
        scope: projectName,
        requiredPermissions: ["bb.accessGrants.create"],
        missingPermissions: [],
      },
      eligible: true,
      exemptionFallback: false,
    });
    expect(result.justInTime.featureRequirement.feature).toBe("FEATURE_JIT");
    expect(mocks.getProject).toHaveBeenCalledWith(
      expect.objectContaining({ name: projectName }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(
      mocks.getProject.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
  });

  test("offers an Access Grant fallback when exemption configuration is denied", async () => {
    mocks.deniedProject.add("bb.policies.updateMaskingExemptionPolicy");

    const result = await getMaskingSetup({ workspaceName, projectName });

    expect(step(result, "masking-exemptions").configure.status).toBe("denied");
    expect(result.justInTime).toMatchObject({
      eligible: true,
      exemptionFallback: true,
    });
  });

  test("does not turn unverified exemption permission into a denied fallback", async () => {
    const refresh = mocks.loadWorkspacePermissionState.getMockImplementation()!;
    mocks.loadWorkspacePermissionState.mockImplementation(async () => {
      await refresh();
      mocks.state.workspacePolicy!.bindings[0].condition = create(ExprSchema, {
        expression: "false",
      });
      mocks.state.roles.push(
        create(RoleSchema, {
          name: "roles/access-grant-requester",
          permissions: ["bb.accessGrants.create"],
        })
      );
      mocks.state.workspacePolicy!.bindings.push(
        create(BindingSchema, {
          role: "roles/access-grant-requester",
          members: ["user:member@example.com"],
        })
      );
    });

    const result = await getMaskingSetup({ workspaceName, projectName });

    expect(step(result, "masking-exemptions").configure.status).toBe("unknown");
    expect(result.justInTime).toMatchObject({
      eligible: true,
      exemptionFallback: false,
    });
  });

  test("evaluates Access Grants independently of masking and instance licensing", async () => {
    mocks.state.hasFeature = (feature) => feature === PlanFeature.FEATURE_JIT;

    const result = await getMaskingSetup({ workspaceName, projectName });

    expect(result.features.masking).toBe("unavailable");
    expect(result.justInTime).toMatchObject({
      feature: "available",
      eligible: true,
    });
  });

  test.each([
    "disabled",
    "inactive",
    "create-denied",
    "feature-unavailable",
    "subscription-expired",
    "subscription-unknown",
    "permission-unknown",
    "project-unknown",
  ])("does not offer an Access Grant when %s", async (failure) => {
    if (failure === "disabled" || failure === "inactive") {
      mocks.getProject.mockResolvedValue(
        create(ProjectSchema, {
          name: projectName,
          state: failure === "inactive" ? State.DELETED : State.ACTIVE,
          allowJustInTimeAccess: failure !== "disabled",
        })
      );
    } else if (failure === "create-denied") {
      mocks.deniedProject.add("bb.accessGrants.create");
    } else if (failure === "feature-unavailable") {
      mocks.state.hasFeature = (feature) => feature !== PlanFeature.FEATURE_JIT;
    } else if (failure === "subscription-expired") {
      mocks.state.subscription!.expiresTime = create(TimestampSchema, {
        seconds: BigInt(Math.floor(Date.now() / 1000) - 60),
      });
    } else if (failure === "subscription-unknown") {
      mocks.refreshSubscription.mockResolvedValue(undefined);
    } else if (failure === "permission-unknown") {
      mocks.fetchCurrentUser.mockResolvedValue(undefined);
    } else {
      mocks.getIamPolicy.mockRejectedValue(new Error("Unavailable"));
    }
    mocks.deniedProject.add("bb.policies.updateMaskingExemptionPolicy");

    const result = await getMaskingSetup({ workspaceName, projectName });

    expect(result.justInTime.eligible).toBe(false);
    expect(result.justInTime.exemptionFallback).toBe(false);
  });

  test("requires project scope before offering an Access Grant", async () => {
    const result = await getMaskingSetup({ workspaceName });

    expect(result.justInTime).toMatchObject({
      create: { status: "needs-scope" },
      eligible: false,
      exemptionFallback: false,
    });
    expect(result.justInTime.enabled).toBeUndefined();
  });

  test.each([
    [PlanType.FREE, "Community"],
    [PlanType.TEAM, "Pro"],
  ])(
    "names the current plan and required plan for unavailable masking on %s",
    async (plan, title) => {
      await i18n.changeLanguage("en-US");
      mocks.state.subscription = create(SubscriptionSchema, { plan });

      const result = await getMaskingSetup({ workspaceName });

      expect(result.subscription).toEqual({
        status: "available",
        plan: PlanType[plan],
        planTitle: title,
        expired: false,
      });
      expect(result.features.masking).toBe("unavailable");
      expect(result.featureRequirements.masking).toEqual({
        feature: "FEATURE_DATA_MASKING",
        featureTitle: "Data masking",
        minimumPlan: "ENTERPRISE",
        minimumPlanTitle: "Enterprise",
        reason: "plan-does-not-include-feature",
      });
      expect(step(result, "global-masking").configure.status).toBe("allowed");
    }
  );

  test("distinguishes an expired subscription from a plan upgrade and instance licensing", async () => {
    mocks.state.subscription = create(SubscriptionSchema, {
      plan: PlanType.ENTERPRISE,
      expiresTime: { seconds: BigInt(Math.floor(Date.now() / 1000) - 60) },
    });
    mocks.database.instanceResource = create(InstanceResourceSchema, {
      name: "instances/primary",
      activation: true,
    });

    const result = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
      database: mocks.database,
    });

    expect(result.subscription).toMatchObject({
      status: "available",
      plan: "ENTERPRISE",
      expired: true,
    });
    expect(result.features.masking).toBe("unavailable");
    expect(result.features.catalogMasking).toBe("unavailable");
    expect(result.featureRequirements.masking.reason).toBe(
      "subscription-expired"
    );
    expect(result.featureRequirements.catalogMasking.reason).toBe(
      "subscription-expired"
    );
  });

  test("recognizes expiration after the server falls back to the Community plan", async () => {
    mocks.state.subscription = create(SubscriptionSchema, {
      plan: PlanType.FREE,
      expiresTime: { seconds: BigInt(Math.floor(Date.now() / 1000) - 60) },
    });

    const result = await getMaskingSetup({ workspaceName });

    expect(result.subscription).toMatchObject({
      status: "available",
      plan: "FREE",
      expired: true,
    });
    expect(result.features.masking).toBe("unavailable");
    expect(result.featureRequirements.masking.reason).toBe(
      "subscription-expired"
    );
  });

  test.each(["failed", "rejected"])(
    "keeps a %s subscription refresh unverified despite a cached plan",
    async (failure) => {
      if (failure === "failed")
        mocks.refreshSubscription.mockResolvedValue(undefined);
      else
        mocks.refreshSubscription.mockRejectedValue(new Error("Unavailable"));

      const result = await getMaskingSetup({ workspaceName });

      expect(result.subscription).toEqual({ status: "unknown" });
      expect(result.features.masking).toBe("unknown");
      expect(result.featureRequirements.masking.reason).toBe(
        "subscription-unverified"
      );
      expect(step(result, "global-masking").configure.status).toBe("allowed");
    }
  );

  test.each([
    [PlanType.FREE, PlanType.ENTERPRISE, "available"],
    [PlanType.ENTERPRISE, PlanType.FREE, "unavailable"],
  ])("refreshes cached plan %s to plan %s", async (cached, fresh, status) => {
    mocks.state.subscription = create(SubscriptionSchema, { plan: cached });
    mocks.refreshSubscription.mockImplementation(async () => {
      mocks.state.subscription = create(SubscriptionSchema, { plan: fresh });
      return mocks.state.subscription;
    });

    const result = await getMaskingSetup({ workspaceName });

    expect(mocks.refreshSubscription).toHaveBeenCalledOnce();
    expect(result.subscription).toMatchObject({
      status: "available",
      plan: PlanType[fresh],
    });
    expect(result.features.masking).toBe(status);
  });

  test("does not publish subscription details after the account changes during refresh", async () => {
    mocks.refreshSubscription.mockImplementation(async () => {
      mocks.state.currentUser = create(UserSchema, {
        name: "users/other@example.com",
      });
      return mocks.state.subscription;
    });

    const result = await getMaskingSetup({ workspaceName });

    expect(result.subscription).toEqual({ status: "unknown" });
    expect(result.features.masking).toBe("unknown");
    expect(result.featureRequirements.masking.reason).toBe(
      "subscription-unverified"
    );
  });

  test.each([0, 99])(
    "keeps unsupported subscription plan %s unverified",
    async (plan) => {
      mocks.state.subscription = create(SubscriptionSchema, {
        plan: plan as PlanType,
      });

      const result = await getMaskingSetup({ workspaceName });

      expect(result.subscription).toEqual({ status: "unknown" });
      expect(result.features.masking).toBe("unknown");
      expect(result.featureRequirements.masking.reason).toBe(
        "subscription-unverified"
      );
    }
  );

  test("distinguishes an unlicensed instance from an unavailable workspace plan", async () => {
    mocks.database.instanceResource = create(InstanceResourceSchema, {
      name: "instances/primary",
      activation: false,
    });

    const result = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
      database: mocks.database,
    });

    expect(result.features.masking).toBe("available");
    expect(result.features.catalogMasking).toBe("unavailable");
    expect(result.featureRequirements.masking.reason).toBeUndefined();
    expect(result.featureRequirements.catalogMasking.reason).toBe(
      "instance-license-required"
    );
  });

  test("distinguishes workspace read and configuration permissions and unselected scopes", async () => {
    const result = await getMaskingSetup({ workspaceName });

    expect(result.source).toBe("frontend-permission-check");
    expect(result.steps.map(({ id }: { id: string }) => id)).toEqual([
      "semantic-types",
      "data-classification",
      "global-masking",
      "database-catalog",
      "masking-exemptions",
    ]);
    expect(step(result, "semantic-types").readAccess).toEqual({
      status: "allowed",
      scope: workspaceName,
      requiredPermissions: ["bb.settings.get"],
      missingPermissions: [],
    });
    expect(
      step(result, "semantic-types").configure.requiredPermissions
    ).toEqual(["bb.settings.set", "bb.policies.update"]);
    expect(step(result, "database-catalog").readAccess.status).toBe(
      "needs-scope"
    );
    expect(step(result, "masking-exemptions").configure.status).toBe(
      "needs-scope"
    );
    expect(result.features).toEqual({
      masking: "available",
      classification: "available",
      catalogMasking: "unknown",
    });
    expect(JSON.stringify(result)).not.toContain("member@example.com");
  });

  test("lists exact missing management permissions without denying permitted reads", async () => {
    mocks.deniedWorkspace.add("bb.settings.set");
    mocks.deniedWorkspace.add("bb.policies.updateMaskingRulePolicy");

    const result = await getMaskingSetup({ workspaceName });

    expect(step(result, "semantic-types").readAccess.status).toBe("allowed");
    expect(step(result, "semantic-types").configure.status).toBe("denied");
    expect(step(result, "semantic-types").configure.missingPermissions).toEqual(
      ["bb.settings.set"]
    );
    expect(step(result, "global-masking").configure.missingPermissions).toEqual(
      ["bb.policies.updateMaskingRulePolicy"]
    );
    expect(
      step(result, "data-classification").configure.missingPermissions
    ).toEqual(["bb.settings.set"]);
  });

  test("identifies the semantic-types page's additional UI permission explicitly", async () => {
    mocks.deniedWorkspace.add("bb.policies.update");

    const result = await getMaskingSetup({ workspaceName });

    expect(step(result, "semantic-types").configure.missingPermissions).toEqual(
      ["bb.policies.update"]
    );
    expect(
      step(result, "semantic-types").additionalRequirements
    ).toContainEqual({
      source: "console-ui",
      permission: "bb.policies.update",
    });
  });

  test("reports initial policy creation separately from permitted updates", async () => {
    mocks.deniedWorkspace.add("bb.policies.createMaskingRulePolicy");

    const result = await getMaskingSetup({ workspaceName });

    expect(step(result, "global-masking").configure.status).toBe("allowed");
    expect(
      step(result, "global-masking").additionalRequirements
    ).toContainEqual({
      source: "backend",
      permission: "bb.policies.createMaskingRulePolicy",
      condition: "when-policy-missing",
      status: "denied",
      scope: workspaceName,
      requiredPermissions: ["bb.policies.createMaskingRulePolicy"],
      missingPermissions: ["bb.policies.createMaskingRulePolicy"],
    });
  });

  test.each([
    'request.time > timestamp("2999-01-01T00:00:00Z")',
    "false",
    'resource.database in ["instances/primary/databases/app"]',
  ])(
    "does not promise workspace permissions based only on %s",
    async (expression) => {
      const refresh =
        mocks.loadWorkspacePermissionState.getMockImplementation()!;
      mocks.loadWorkspacePermissionState.mockImplementation(async () => {
        await refresh();
        mocks.state.workspacePolicy!.bindings[0].condition = create(
          ExprSchema,
          { expression }
        );
      });

      const result = await getMaskingSetup({ workspaceName });

      expect(step(result, "semantic-types").readAccess.status).toBe("unknown");
      expect(
        step(result, "semantic-types").readAccess.missingPermissions
      ).toEqual([]);
      expect(step(result, "global-masking").configure.status).toBe("unknown");
    }
  );

  test.each(["", 'request.time < timestamp("2999-01-01T00:00:00Z")'])(
    "keeps an independent verified workspace grant allowed with condition %s",
    async (expression) => {
      const refresh =
        mocks.loadWorkspacePermissionState.getMockImplementation()!;
      mocks.loadWorkspacePermissionState.mockImplementation(async () => {
        await refresh();
        const binding = mocks.state.workspacePolicy!.bindings[0];
        binding.condition = create(ExprSchema, { expression: "false" });
        mocks.state.workspacePolicy!.bindings.push({
          ...binding,
          condition: create(ExprSchema, { expression }),
        });
      });

      const result = await getMaskingSetup({ workspaceName });

      expect(step(result, "semantic-types").readAccess.status).toBe("allowed");
      expect(step(result, "semantic-types").configure.status).toBe("allowed");
    }
  );

  test.each([false, true])(
    "verifies an independent project grant before promising scoped permissions: %s",
    async (independentGrant) => {
      const refresh =
        mocks.loadWorkspacePermissionState.getMockImplementation()!;
      mocks.loadWorkspacePermissionState.mockImplementation(async () => {
        await refresh();
        mocks.state.workspacePolicy = create(IamPolicySchema);
      });
      const policy = create(IamPolicySchema, {
        bindings: [
          {
            role: "roles/example",
            members: ["user:member@example.com"],
            condition: {
              expression: 'request.time > timestamp("2999-01-01T00:00:00Z")',
            },
          },
        ],
      });
      if (independentGrant)
        policy.bindings.push({ ...policy.bindings[0], condition: undefined });
      mocks.getIamPolicy.mockResolvedValue(policy);

      const result = await getMaskingSetup({
        workspaceName,
        projectName,
        databaseName,
        database: mocks.database,
      });

      expect(step(result, "database-catalog").configure.status).toBe(
        independentGrant ? "allowed" : "unknown"
      );
      expect(step(result, "masking-exemptions").readAccess.status).toBe(
        independentGrant ? "allowed" : "unknown"
      );
      expect(
        step(result, "masking-exemptions").readAccess.missingPermissions
      ).toEqual([]);
    }
  );

  test("does not treat unresolved current-user group membership as a permission denial", async () => {
    mocks.state.currentUser = {
      ...create(UserSchema, { name: "users/member@example.com" }),
      groups: undefined as never,
    };
    mocks.deniedWorkspace.add("bb.settings.get");

    const result = await getMaskingSetup({ workspaceName });

    expect(step(result, "semantic-types").readAccess.status).toBe("unknown");
    expect(
      step(result, "semantic-types").readAccess.missingPermissions
    ).toEqual([]);
  });

  test.each([true, false])(
    "uses fresh current-user group membership when group access becomes %s",
    async (isMember) => {
      const group = "groups/example";
      mocks.state.currentUser = create(UserSchema, {
        name: "users/member@example.com",
        groups: isMember ? [] : [group],
      });
      mocks.fetchCurrentUser.mockImplementation(async () => {
        const fresh = create(UserSchema, {
          name: "users/member@example.com",
          groups: isMember ? [group] : [],
        });
        mocks.state.currentUser = fresh;
        return fresh;
      });
      mocks.loadWorkspacePermissionState.mockImplementation(async () => {
        mocks.state.roles = [
          create(RoleSchema, { name: "roles/example", permissions }),
        ];
        mocks.state.workspacePolicy = create(IamPolicySchema, {
          bindings: [{ role: "roles/example", members: [group] }],
        });
      });
      mocks.state.hasWorkspacePermission = () =>
        Boolean(
          mocks.state.currentUser &&
            bindingMatchesUser(
              mocks.state.workspacePolicy,
              mocks.state.currentUser
            ).length
        );

      const result = await getMaskingSetup({ workspaceName });

      expect(mocks.fetchCurrentUser).toHaveBeenCalledWith(true);
      expect(step(result, "semantic-types").readAccess.status).toBe(
        isMember ? "allowed" : "denied"
      );
      expect(
        step(result, "semantic-types").readAccess.missingPermissions
      ).toEqual(isMember ? [] : ["bb.settings.get"]);
    }
  );

  test("returns unknown when user refresh fails despite fresh IAM and stale group access", async () => {
    mocks.fetchCurrentUser.mockResolvedValue(undefined);

    const result = await getMaskingSetup({ workspaceName, projectName });

    expect(mocks.loadWorkspacePermissionState).toHaveBeenCalledOnce();
    expect(step(result, "semantic-types").readAccess.status).toBe("unknown");
    expect(
      step(result, "semantic-types").readAccess.missingPermissions
    ).toEqual([]);
    expect(step(result, "masking-exemptions").readAccess.status).toBe(
      "unknown"
    );
    expect(mocks.getIamPolicy).not.toHaveBeenCalled();
  });

  test.each(["throw", "roles", "policy"])(
    "returns unknown on %s refresh failure despite stale permissions",
    async (failure) => {
      mocks.loadWorkspacePermissionState.mockImplementation(async () => {
        if (failure === "throw") throw new Error("unavailable");
        if (failure !== "roles") mocks.state.roles = [create(RoleSchema)];
        if (failure !== "policy")
          mocks.state.workspacePolicy = create(IamPolicySchema);
      });

      const result = await getMaskingSetup({ workspaceName, projectName });

      expect(step(result, "global-masking").readAccess.status).toBe("unknown");
      expect(step(result, "semantic-types").configure.status).toBe("unknown");
      expect(
        step(result, "semantic-types").configure.missingPermissions
      ).toEqual([]);
      expect(step(result, "masking-exemptions").configure.status).toBe(
        "unknown"
      );
    }
  );

  test("refreshes project IAM and reports database and exemption flow prerequisites at their scope", async () => {
    const fresh = create(IamPolicySchema, { etag: "fresh" });
    mocks.state.projectPoliciesByName[projectName] = create(IamPolicySchema, {
      etag: "stale",
    });
    mocks.getIamPolicy.mockResolvedValue(fresh);
    mocks.deniedProject.add("bb.databaseCatalogs.update");

    const result = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
      database: mocks.database,
    });

    expect(mocks.state.projectPoliciesByName[projectName]).toBe(fresh);
    expect(mocks.getIamPolicy).toHaveBeenCalledWith(
      expect.objectContaining({ resource: projectName }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(
      mocks.getIamPolicy.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
    expect(step(result, "database-catalog").readAccess.status).toBe("allowed");
    expect(step(result, "database-catalog").configure).toEqual({
      status: "denied",
      scope: databaseName,
      requiredPermissions: ["bb.databaseCatalogs.update"],
      missingPermissions: ["bb.databaseCatalogs.update"],
    });
    expect(
      step(result, "masking-exemptions").configure.requiredPermissions
    ).toEqual([
      "bb.policies.createMaskingExemptionPolicy",
      "bb.policies.updateMaskingExemptionPolicy",
      "bb.policies.getMaskingExemptionPolicy",
      "bb.databases.list",
      "bb.databaseCatalogs.get",
    ]);
  });

  test("does not turn a failed project refresh into a stale allow or deny verdict", async () => {
    mocks.state.projectPoliciesByName[projectName] = create(IamPolicySchema);
    mocks.getIamPolicy.mockRejectedValue(new Error("unavailable"));

    const result = await getMaskingSetup({ workspaceName, projectName });

    expect(step(result, "masking-exemptions").readAccess.status).toBe(
      "unknown"
    );
    expect(
      step(result, "masking-exemptions").configure.missingPermissions
    ).toEqual([]);
    expect(step(result, "global-masking").configure.status).toBe("allowed");
  });

  test("does not publish old-account IAM data or verdicts after the account changes", async () => {
    const previous = create(IamPolicySchema, { etag: "previous" });
    mocks.state.projectPoliciesByName[projectName] = previous;
    mocks.getIamPolicy.mockImplementation(async () => {
      mocks.state.currentUser = create(UserSchema, {
        name: "users/other@example.com",
      });
      return create(IamPolicySchema, { etag: "old-account-response" });
    });

    const result = await getMaskingSetup({ workspaceName, projectName });

    expect(mocks.state.projectPoliciesByName[projectName]).toBe(previous);
    expect(step(result, "global-masking").readAccess.status).toBe("unknown");
    expect(step(result, "masking-exemptions").readAccess.status).toBe(
      "unknown"
    );
    expect(result.features.masking).toBe("unknown");
  });

  test("keeps missing subscription evidence separate from permissions", async () => {
    mocks.state.subscription = undefined;

    const result = await getMaskingSetup({ workspaceName });

    expect(mocks.refreshSubscription).toHaveBeenCalledOnce();
    expect(result.features.masking).toBe("unknown");
    expect(result.features.classification).toBe("unknown");
    expect(step(result, "global-masking").configure.status).toBe("allowed");
  });

  test("reports unavailable features without inventing missing IAM permissions", async () => {
    mocks.state.hasFeature = () => false;

    const result = await getMaskingSetup({ workspaceName });

    expect(result.features.masking).toBe("unavailable");
    expect(result.features.classification).toBe("unavailable");
    expect(step(result, "global-masking").configure.missingPermissions).toEqual(
      []
    );
    expect(step(result, "global-masking").configure.status).toBe("allowed");
  });

  test("requires known database-instance entitlement rather than assuming the workspace feature applies", async () => {
    const withoutInstance = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
      database: mocks.database,
    });
    expect(withoutInstance.features.catalogMasking).toBe("unknown");
    mocks.database = create(DatabaseSchema$, {
      name: databaseName,
      project: projectName,
      instanceResource: create(InstanceResourceSchema, {
        name: "instances/primary",
        activation: false,
      }),
    });

    const withInstance = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
      database: mocks.database,
    });

    expect(withInstance.features.masking).toBe("available");
    expect(withInstance.features.catalogMasking).toBe("unavailable");
  });

  test("does not claim catalog permissions when the database belongs to another project", async () => {
    mocks.database = create(DatabaseSchema$, {
      name: databaseName,
      project: "projects/other",
    });

    const result = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
      database: mocks.database,
    });

    expect(step(result, "database-catalog").readAccess.status).toBe("unknown");
    expect(step(result, "database-catalog").configure.status).toBe("unknown");
    expect(result.features.catalogMasking).toBe("unknown");
  });

  test("uses fresh catalog eligibility when the database is absent from the UI cache", async () => {
    mocks.state.databasesByName = {};
    mocks.database.instanceResource = create(InstanceResourceSchema, {
      name: "instances/primary",
      activation: true,
    });

    const result = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
      database: mocks.database,
    });

    expect(result.features.catalogMasking).toBe("available");
    expect(step(result, "database-catalog").readAccess.status).toBe("allowed");
    expect(step(result, "database-catalog").configure.status).toBe("allowed");
    expect(mocks.state.databasesByName).toEqual({});
  });

  test.each([
    { cached: true, fresh: false, want: "unavailable" },
    { cached: false, fresh: true, want: "available" },
  ])(
    "uses fresh activation $fresh instead of cached activation $cached",
    async ({ cached, fresh, want }) => {
      mocks.state.databasesByName[databaseName].instanceResource = create(
        InstanceResourceSchema,
        { name: "instances/primary", activation: cached }
      );
      mocks.database.instanceResource = create(InstanceResourceSchema, {
        name: "instances/primary",
        activation: fresh,
      });

      const result = await getMaskingSetup({
        workspaceName,
        projectName,
        databaseName,
        database: mocks.database,
      });

      expect(result.features.catalogMasking).toBe(want);
      expect(step(result, "database-catalog").readAccess.status).toBe(
        "allowed"
      );
    }
  );

  test.each([
    {
      cached: "projects/other",
      fresh: projectName,
      want: "allowed",
      catalogMasking: "available",
    },
    {
      cached: projectName,
      fresh: "projects/other",
      want: "unknown",
      catalogMasking: "unknown",
    },
  ])(
    "uses fresh project ownership $fresh instead of cached ownership $cached",
    async ({ cached, fresh, want, catalogMasking }) => {
      mocks.state.databasesByName[databaseName].project = cached;
      mocks.state.databasesByName[databaseName].instanceResource = create(
        InstanceResourceSchema,
        { name: "instances/primary", activation: true }
      );
      mocks.database.project = fresh;
      mocks.database.instanceResource = create(InstanceResourceSchema, {
        name: "instances/primary",
        activation: true,
      });

      const result = await getMaskingSetup({
        workspaceName,
        projectName,
        databaseName,
        database: mocks.database,
      });

      expect(step(result, "database-catalog").readAccess.status).toBe(want);
      expect(step(result, "database-catalog").configure.status).toBe(want);
      expect(result.features.catalogMasking).toBe(catalogMasking);
    }
  );

  test("keeps catalog eligibility unknown when fresh database evidence is unavailable", async () => {
    mocks.state.databasesByName[databaseName].instanceResource = create(
      InstanceResourceSchema,
      { name: "instances/primary", activation: true }
    );

    const result = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
    });

    expect(result.features.catalogMasking).toBe("unknown");
    expect(step(result, "database-catalog").readAccess.status).toBe("unknown");
    expect(step(result, "database-catalog").configure.status).toBe("unknown");
  });

  test("rejects fresh database evidence for a different resource", async () => {
    mocks.database.name = "instances/secondary/databases/app.v2";

    const result = await getMaskingSetup({
      workspaceName,
      projectName,
      databaseName,
      database: mocks.database,
    });

    expect(step(result, "database-catalog").readAccess.status).toBe("unknown");
    expect(step(result, "database-catalog").configure.status).toBe("unknown");
    expect(result.features.catalogMasking).toBe("unknown");
  });
});
