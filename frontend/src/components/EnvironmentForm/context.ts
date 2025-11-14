import Emittery from "emittery";
import { cloneDeep, isEmpty, isEqual } from "lodash-es";
import { useDialog } from "naive-ui";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref, watch } from "vue";
import type { ResourceIdField } from "@/components/v2";
import {
  type Policy,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { Environment } from "@/types/v1/environment";
import { hasWorkspacePermissionV2 } from "@/utils";

export type LocalState = {
  environment: Environment;
  rolloutPolicy: Policy;
  policyChanged: boolean;
};

const KEY = Symbol(
  "bb.workspace.Environment-form"
) as InjectionKey<EnvironmentFormContext>;

export const provideEnvironmentFormContext = (baseContext: {
  create: Ref<boolean>;
  environment: Ref<Environment>;
  rolloutPolicy: Ref<Policy>;
}) => {
  const $d = useDialog();
  const events = new Emittery<{
    create: {
      environment: Partial<Environment>;
      rolloutPolicy: Policy;
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
    delete: Environment;
    cancel: undefined;
  }>();
  const { create, environment, rolloutPolicy } = baseContext;
  const state = ref<LocalState>({
    environment: cloneDeep(environment.value),
    rolloutPolicy: cloneDeep(rolloutPolicy.value),
    policyChanged: false,
  });
  const missingFeature = ref<PlanFeature | undefined>(undefined);
  const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

  const valueChanged = (
    field?: "environment" | "approvalPolicy" | "rolloutPolicy"
  ): boolean => {
    switch (field) {
      case "environment":
        return !isEqual(environment.value, state.value.environment);
      case "rolloutPolicy":
        return !isEqual(rolloutPolicy.value, state.value.rolloutPolicy);

      default:
        return (
          !isEqual(environment.value, state.value.environment) ||
          !isEqual(rolloutPolicy.value, state.value.rolloutPolicy) ||
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
    return create.value || hasWorkspacePermissionV2("bb.settings.set");
  });

  watch(environment, (cur) => {
    state.value.environment = cloneDeep(cur);
  });
  watch(rolloutPolicy, (cur) => {
    state.value.rolloutPolicy = cloneDeep(cur);
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
