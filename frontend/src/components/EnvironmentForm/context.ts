import Emittery from "emittery";
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { useDialog } from "naive-ui";
import type { InjectionKey, Ref } from "vue";
import { provide, inject, computed, ref, watch } from "vue";
import type { ResourceIdField } from "@/components/v2";
import type { FeatureType } from "@/types";
import { State } from "@/types/proto/v1/common";
import type {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import { PolicyType, type Policy } from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV2 } from "@/utils";

export type LocalState = {
  environment: Environment;
  rolloutPolicy: Policy;
  environmentTier: EnvironmentTier;
  policyChanged: boolean;
};

const KEY = Symbol(
  "bb.workspace.Environment-form"
) as InjectionKey<EnvironmentFormContext>;

export const provideEnvironmentFormContext = (baseContext: {
  create: Ref<boolean>;
  environment: Ref<Environment>;
  rolloutPolicy: Ref<Policy>;
  environmentTier: Ref<EnvironmentTier>;
}) => {
  const $d = useDialog();
  const events = new Emittery<{
    create: {
      environment: Partial<Environment>;
      rolloutPolicy: Policy;
      environmentTier: EnvironmentTier;
    };
    update: Environment;
    "update-policy": {
      environment: Environment;
      policyType: PolicyType;
      policy: Policy;
    };
    "update-access-control": undefined;
    "revert-access-control": undefined;
    "update-sql-review": undefined;
    "revert-sql-review": undefined;
    archive: Environment;
    restore: Environment;
    cancel: undefined;
  }>();
  const { create, environment, rolloutPolicy, environmentTier } = baseContext;
  const state = ref<LocalState>({
    environment: cloneDeep(environment.value),
    rolloutPolicy: cloneDeep(rolloutPolicy.value),
    environmentTier: environmentTier.value,
    policyChanged: false,
  });
  const missingFeature = ref<FeatureType | undefined>(undefined);
  const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

  const valueChanged = (
    field?:
      | "environment"
      | "approvalPolicy"
      | "rolloutPolicy"
      | "environmentTier"
  ): boolean => {
    switch (field) {
      case "environment":
        return !isEqual(environment.value, state.value.environment);
      case "rolloutPolicy":
        return !isEqual(rolloutPolicy.value, state.value.rolloutPolicy);
      case "environmentTier":
        return !isEqual(environmentTier.value, state.value.environmentTier);

      default:
        return (
          !isEqual(environment.value, state.value.environment) ||
          !isEqual(rolloutPolicy.value, state.value.rolloutPolicy) ||
          !isEqual(environmentTier.value, state.value.environmentTier) ||
          state.value.policyChanged
        );
    }
  };

  const allowCreate = computed(() => {
    return (
      !isEmpty(state.value.environment.title) &&
      resourceIdField.value?.resourceId &&
      resourceIdField.value?.isValidated
    );
  });

  const allowEdit = computed(() => {
    return (
      create.value ||
      (state.value.environment.state === State.ACTIVE &&
        hasWorkspacePermissionV2("bb.environments.update"))
    );
  });

  watch(environment, (cur) => {
    state.value.environment = cloneDeep(cur);
  });
  watch(rolloutPolicy, (cur) => {
    state.value.rolloutPolicy = cloneDeep(cur);
  });
  watch(environmentTier, (cur) => {
    state.value.environmentTier = cur;
  });

  const context = {
    ...baseContext,
    $d,
    events,
    state,
    missingFeature,
    resourceIdField,
    allowCreate,
    allowEdit,
    hasPermission: hasWorkspacePermissionV2,
    valueChanged,
  };
  provide(KEY, context);

  return context;
};

export const useEnvironmentFormContext = () => {
  return inject(KEY)!;
};

export type EnvironmentFormContext = ReturnType<
  typeof provideEnvironmentFormContext
>;
