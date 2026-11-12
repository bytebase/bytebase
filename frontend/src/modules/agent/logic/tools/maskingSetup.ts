import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { projectServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import i18n from "@/lib/i18n";
import { useAppStore } from "@/stores/app";
import type { AppStoreState } from "@/stores/app/types";
import { projectWideBindings } from "@/stores/app/utils";
import type { Permission } from "@/types/iam/permission";
import {
  getMinimumRequiredPlan,
  hasFeature as planHasFeature,
} from "@/types/plan";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { GetIamPolicyRequestSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  PlanFeature,
  PlanType,
  type Subscription,
} from "@/types/proto-es/v1/subscription_service_pb";
import { getTimeForPbTimestampProtoEs } from "@/types/timestamp";

type PermissionStatus = "allowed" | "denied" | "unknown" | "needs-scope";
type FeatureStatus = "available" | "unavailable" | "unknown";
interface FeatureRequirement {
  feature: string;
  featureTitle: string;
  minimumPlan: string;
  minimumPlanTitle: string;
  reason?:
    | "subscription-unverified"
    | "subscription-expired"
    | "plan-does-not-include-feature"
    | "instance-license-required"
    | "scope-unverified"
    | "feature-unavailable";
}
type SetupStepId =
  | "semantic-types"
  | "data-classification"
  | "global-masking"
  | "database-catalog"
  | "masking-exemptions";

interface PermissionEvidence {
  status: PermissionStatus;
  scope: string;
  requiredPermissions: Permission[];
  missingPermissions: Permission[];
}

interface SetupStep {
  id: SetupStepId;
  readAccess: PermissionEvidence;
  configure: PermissionEvidence;
  additionalRequirements?: (
    | { source: "console-ui"; permission: Permission }
    | (PermissionEvidence & {
        source: "backend";
        permission: Permission;
        condition: "when-policy-missing";
      })
  )[];
}

function permissionEvidence(
  requiredPermissions: Permission[],
  scope: string | undefined,
  scopeKind: string,
  known: boolean,
  check: (permission: Permission) => boolean,
  verify: (permission: Permission) => boolean
): PermissionEvidence {
  const missingPermissions = known
    ? requiredPermissions.filter((permission) => !check(permission))
    : [];
  return {
    status: !scope
      ? "needs-scope"
      : !known
        ? "unknown"
        : missingPermissions.length
          ? "denied"
          : requiredPermissions.every(verify)
            ? "allowed"
            : "unknown",
    scope: scope ?? scopeKind,
    requiredPermissions,
    missingPermissions,
  };
}

export async function getMaskingSetup({
  workspaceName,
  projectName,
  databaseName,
  database,
}: {
  workspaceName: string;
  projectName?: string;
  databaseName?: string;
  // Freshly read evidence, not an entry from the UI's database cache.
  database?: Database;
}): Promise<{
  source: "frontend-permission-check";
  features: {
    masking: FeatureStatus;
    classification: FeatureStatus;
    catalogMasking: FeatureStatus;
  };
  subscription:
    | { status: "available"; plan: string; planTitle: string; expired: boolean }
    | { status: "unknown" };
  featureRequirements: Record<
    "masking" | "classification" | "catalogMasking",
    FeatureRequirement
  >;
  steps: SetupStep[];
  scopeAccess?: {
    project: PermissionEvidence;
    databasePicker: PermissionEvidence;
    projects: PermissionEvidence;
  };
}> {
  const before = useAppStore.getState();
  const initialUserName = before.currentUser?.name;
  const previousRoles = before.roles;
  const previousPolicy = before.workspacePolicy;
  let workspaceReady = false;
  try {
    const user = await before.fetchCurrentUser(true);
    await before.loadWorkspacePermissionState();
    const after = useAppStore.getState();
    // These caches change only after successful reads; the loader swallows errors.
    workspaceReady =
      Boolean(user?.name && after.currentUser === user) &&
      after.roles !== previousRoles &&
      Array.isArray(user?.groups) &&
      Boolean(
        after.workspacePolicy && after.workspacePolicy !== previousPolicy
      );
  } catch {
    workspaceReady = false;
  }
  const userName = initialUserName ?? useAppStore.getState().currentUser?.name;
  const sameContext = () => {
    const state = useAppStore.getState();
    return Boolean(
      userName &&
        state.currentUser?.name === userName &&
        state.serverInfo?.workspace === workspaceName
    );
  };
  workspaceReady &&= sameContext();

  let verifiedSubscription: Subscription | undefined;
  if (sameContext()) {
    try {
      verifiedSubscription = await useAppStore.getState().refreshSubscription();
    } catch {
      verifiedSubscription = undefined;
    }
  }

  let project: Project | undefined;
  let projectReady = false;
  if (
    projectName &&
    /^projects\/[A-Za-z0-9_-]+$/.test(projectName) &&
    workspaceReady
  ) {
    try {
      const loaded = await useAppStore
        .getState()
        .getOrFetchProjectByName(projectName, true);
      if (loaded.name === projectName) {
        project = loaded;
        const policy = await projectServiceClientConnect.getIamPolicy(
          create(GetIamPolicyRequestSchema, { resource: projectName }),
          { contextValues: createContextValues().set(silentContextKey, true) }
        );
        if (sameContext()) {
          useAppStore.setState((state) => ({
            projectPoliciesByName: {
              ...state.projectPoliciesByName,
              [projectName]: policy,
            },
          }));
          projectReady = true;
        }
      }
    } catch {
      projectReady = false;
    }
  }

  const state = useAppStore.getState();
  const subscriptionKnown = Boolean(
    verifiedSubscription &&
      state.subscription === verifiedSubscription &&
      [PlanType.FREE, PlanType.TEAM, PlanType.ENTERPRISE].includes(
        verifiedSubscription.plan
      ) &&
      sameContext()
  );
  const currentPlan = subscriptionKnown
    ? verifiedSubscription?.plan
    : undefined;
  // Expired licenses retain their expiration when the effective plan becomes Community.
  const expiresAt = getTimeForPbTimestampProtoEs(
    verifiedSubscription?.expiresTime
  );
  const expired =
    subscriptionKnown && expiresAt !== undefined && expiresAt < Date.now();
  const planTitle = (plan: PlanType) =>
    i18n.t(`subscription.plan.${PlanType[plan].toLowerCase()}.title`);
  workspaceReady &&= sameContext();
  projectReady &&= workspaceReady;
  const rolesByName = new Map(state.roles.map((role) => [role.name, role]));
  const verifiedPermissions = (
    policy: AppStoreState["workspacePolicy"],
    skipAllUsers: boolean
  ) =>
    new Set(
      state.currentUser
        ? projectWideBindings(policy, state.currentUser, {
            skipAllUsers,
          }).flatMap(
            (binding) => rolesByName.get(binding.role)?.permissions ?? []
          )
        : []
    );
  // Coarse getters accept conditions that only the server can fully evaluate.
  const workspaceVerified = verifiedPermissions(
    state.workspacePolicy,
    state.isSaaSMode()
  );
  const projectVerified = verifiedPermissions(
    projectName ? state.projectPoliciesByName[projectName] : undefined,
    false
  );
  const verifyProjectPermission = (permission: Permission) =>
    workspaceVerified.has(permission) || projectVerified.has(permission);
  const workspacePermission = (permissions: Permission[]) =>
    permissionEvidence(
      permissions,
      workspaceName,
      "workspace",
      workspaceReady,
      state.hasWorkspacePermission,
      (permission) => workspaceVerified.has(permission)
    );
  const projectPermission = (permissions: Permission[]) =>
    permissionEvidence(
      permissions,
      projectName,
      "project",
      projectReady,
      (permission) =>
        Boolean(project && state.hasProjectPermission(project, permission)),
      verifyProjectPermission
    );
  const databaseScopeKnown = Boolean(
    databaseName &&
      projectName &&
      /^(?:projects\/[A-Za-z0-9_-]+\/)?instances\/[A-Za-z0-9_-]+\/databases\/[^/]+$/.test(
        databaseName
      ) &&
      database?.name === databaseName &&
      database.project === projectName &&
      (!databaseName.startsWith("projects/") ||
        databaseName.startsWith(`${projectName}/`))
  );
  const databasePermission = (permissions: Permission[]) =>
    permissionEvidence(
      permissions,
      databaseName,
      "database",
      projectReady && databaseScopeKnown,
      (permission) =>
        Boolean(project && state.hasProjectPermission(project, permission)),
      verifyProjectPermission
    );
  const feature = (feature: PlanFeature): FeatureStatus =>
    !subscriptionKnown || !sameContext()
      ? "unknown"
      : !expired && state.hasFeature(feature)
        ? "available"
        : "unavailable";
  const instance = databaseScopeKnown ? database?.instanceResource : undefined;
  const masking = feature(PlanFeature.FEATURE_DATA_MASKING);
  const classification = feature(PlanFeature.FEATURE_DATA_CLASSIFICATION);
  const catalogMasking: FeatureStatus =
    masking !== "available"
      ? masking
      : !instance
        ? "unknown"
        : state.hasInstanceFeature(PlanFeature.FEATURE_DATA_MASKING, instance)
          ? "available"
          : "unavailable";
  const requirement = (
    feature: PlanFeature,
    status: FeatureStatus,
    instanceLicenseRequired = false
  ): FeatureRequirement => {
    const minimumPlan = getMinimumRequiredPlan(feature);
    const reason: FeatureRequirement["reason"] =
      status === "available"
        ? undefined
        : !subscriptionKnown
          ? "subscription-unverified"
          : expired
            ? "subscription-expired"
            : currentPlan !== undefined && !planHasFeature(currentPlan, feature)
              ? "plan-does-not-include-feature"
              : instanceLicenseRequired
                ? "instance-license-required"
                : status === "unknown"
                  ? "scope-unverified"
                  : "feature-unavailable";
    return {
      feature: PlanFeature[feature],
      featureTitle: i18n.t(
        `dynamic.subscription.features.${PlanFeature[feature]}.title`
      ),
      minimumPlan: PlanType[minimumPlan],
      minimumPlanTitle: planTitle(minimumPlan),
      ...(reason ? { reason } : {}),
    };
  };

  return {
    source: "frontend-permission-check",
    subscription:
      subscriptionKnown && currentPlan !== undefined
        ? {
            status: "available",
            plan: PlanType[currentPlan],
            planTitle: planTitle(currentPlan),
            expired,
          }
        : { status: "unknown" },
    featureRequirements: {
      masking: requirement(PlanFeature.FEATURE_DATA_MASKING, masking),
      classification: requirement(
        PlanFeature.FEATURE_DATA_CLASSIFICATION,
        classification
      ),
      catalogMasking: requirement(
        PlanFeature.FEATURE_DATA_MASKING,
        catalogMasking,
        Boolean(
          masking === "available" &&
            instance &&
            state.instanceMissingLicense(
              PlanFeature.FEATURE_DATA_MASKING,
              instance
            )
        )
      ),
    },
    scopeAccess: {
      project: projectPermission(["bb.projects.get"]),
      databasePicker: projectPermission([
        "bb.projects.get",
        "bb.databases.list",
      ]),
      projects: workspacePermission(["bb.projects.list"]),
    },
    features: {
      masking,
      classification,
      catalogMasking,
    },
    steps: [
      {
        id: "semantic-types",
        readAccess: workspacePermission(["bb.settings.get"]),
        configure: workspacePermission([
          "bb.settings.set",
          "bb.policies.update",
        ]),
        additionalRequirements: [
          { source: "console-ui", permission: "bb.policies.update" },
        ],
      },
      {
        id: "data-classification",
        readAccess: workspacePermission(["bb.settings.get"]),
        configure: workspacePermission(["bb.settings.set"]),
      },
      {
        id: "global-masking",
        readAccess: workspacePermission(["bb.policies.getMaskingRulePolicy"]),
        configure: workspacePermission(["bb.policies.updateMaskingRulePolicy"]),
        additionalRequirements: [
          {
            ...workspacePermission(["bb.policies.createMaskingRulePolicy"]),
            source: "backend",
            permission: "bb.policies.createMaskingRulePolicy",
            condition: "when-policy-missing",
          },
        ],
      },
      {
        id: "database-catalog",
        readAccess: databasePermission([
          "bb.projects.get",
          "bb.databases.get",
          "bb.databaseCatalogs.get",
        ]),
        configure: databasePermission(["bb.databaseCatalogs.update"]),
      },
      {
        id: "masking-exemptions",
        readAccess: projectPermission([
          "bb.projects.get",
          "bb.databases.get",
          "bb.policies.getMaskingExemptionPolicy",
        ]),
        configure: projectPermission([
          "bb.policies.createMaskingExemptionPolicy",
          "bb.policies.updateMaskingExemptionPolicy",
          "bb.policies.getMaskingExemptionPolicy",
          "bb.databases.list",
          "bb.databaseCatalogs.get",
        ]),
      },
    ],
  };
}
