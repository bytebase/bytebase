<template>
  <div v-if="hasGetPermission" class="flex flex-col gap-y-2">
    <div class="font-medium flex items-center gap-x-2">
      <label>
        {{ t("environment.access-control.title") }}
      </label>
      <NTooltip v-if="tooltip">
        <template #trigger>
          <CircleQuestionMarkIcon class="w-4 textinfolabel" />
        </template>
        <span>
          {{ tooltip }}
        </span>
      </NTooltip>
    </div>
    <div>
      <div class="w-full inline-flex items-center gap-x-2">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :project="project"
          :permissions="[
            'bb.policies.update'
          ]"
        >
          <Switch
            v-model:value="state.queryDataPolicy.disableCopyData"
            :text="true"
            :disabled="slotProps.disabled || !hasRestrictCopyingDataFeature"
          />
        </PermissionGuardWrapper>
        <span class="textlabel">
          {{ t("environment.access-control.disable-copy-data-from-sql-editor") }}
        </span>
        <FeatureBadge :feature="PlanFeature.FEATURE_RESTRICT_COPYING_DATA" />
      </div>
      <div class="">
        <div class="w-full inline-flex items-center gap-x-2">
          <PermissionGuardWrapper
            v-slot="slotProps"
            :project="project"
            :permissions="[
              'bb.policies.update'
            ]"
          >
            <Switch
              :value="adminDataSourceQueryRestrictionEnabled"
              :text="true"
              :disabled="slotProps.disabled || !hasRestrictQueryDataSourceFeature"
              @update:value="switchDataSourceQueryPolicyEnabled"
            />
          </PermissionGuardWrapper>
          <span class="textlabel">{{
            t("environment.access-control.restrict-admin-connection.self")
          }}</span>
          <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
        </div>
        <div v-if="adminDataSourceQueryRestrictionEnabled" class="ml-12">
          <PermissionGuardWrapper
            v-slot="slotProps"
            :project="project"
            :permissions="[
              'bb.policies.update'
            ]"
          >
            <NRadioGroup
              v-model:value="
                state.dataSourceQueryPolicy.adminDataSourceRestriction
              "
              :disabled="slotProps.disabled || !hasRestrictQueryDataSourceFeature"
            >
              <NRadio
                class="w-full"
                :value="DataSourceQueryPolicy_Restriction.DISALLOW"
              >
                {{
                  t(
                    "environment.access-control.restrict-admin-connection.disallow"
                  )
                }}
              </NRadio>
              <NRadio
                class="w-full"
                :value="DataSourceQueryPolicy_Restriction.FALLBACK"
              >
                {{
                  t(
                    "environment.access-control.restrict-admin-connection.fallback"
                  )
                }}
              </NRadio>
            </NRadioGroup>
          </PermissionGuardWrapper>
        </div>
      </div>
    </div>
  </div>
  <div
    v-if="resource.startsWith(environmentNamePrefix) && hasGetPermission"
    class="flex flex-col gap-y-2"
  >
    <div class="font-medium flex items-center gap-x-2">
      <label>
        {{ t("environment.statement-execution.title") }}
      </label>
      <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
    </div>
    <div>
      <div class="w-full inline-flex items-center gap-x-2">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :project="project"
          :permissions="[
            'bb.policies.update'
          ]"
        >
          <Switch
            v-model:value="state.dataSourceQueryPolicy.disallowDdl"
            :text="true"
            :disabled="slotProps.disabled || !hasRestrictQueryDataSourceFeature"
          />
        </PermissionGuardWrapper>
        <span class="textlabel">
          {{ t("environment.statement-execution.disallow-ddl") }}
        </span>
      </div>
      <div class="w-full inline-flex items-center gap-x-2">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :project="project"
          :permissions="[
            'bb.policies.update'
          ]"
        >
          <Switch
            v-model:value="state.dataSourceQueryPolicy.disallowDml"
            :text="true"
            :disabled="slotProps.disabled || !hasRestrictQueryDataSourceFeature"
          />
        </PermissionGuardWrapper>
        <span class="textlabel">
          {{ t("environment.statement-execution.disallow-dml") }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep, isEqual } from "lodash-es";
import { CircleQuestionMarkIcon } from "lucide-vue-next";
import { NRadio, NRadioGroup, NTooltip } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { hasFeature, usePolicyV1Store, useProjectV1Store } from "@/store";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { isValidProjectName } from "@/types";
import type {
  DataSourceQueryPolicy,
  QueryDataPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  DataSourceQueryPolicy_Restriction,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";
import { FeatureBadge } from "../FeatureGuard";
import { Switch } from "../v2";

const { t } = useI18n();

interface LocalState {
  queryDataPolicy: QueryDataPolicy;
  dataSourceQueryPolicy: DataSourceQueryPolicy;
}

const props = defineProps<{
  resource: string;
}>();

const projectStore = useProjectV1Store();

const project = computed(() => {
  if (props.resource.startsWith(projectNamePrefix)) {
    const proj = projectStore.getProjectByName(props.resource);
    if (!isValidProjectName(proj.name)) {
      return undefined;
    }
    return proj;
  }
  return undefined;
});

const hasGetPermission = computed(() => {
  if (project.value) {
    return hasProjectPermissionV2(project.value, "bb.policies.get");
  }
  return hasWorkspacePermissionV2("bb.policies.get");
});

const tooltip = computed(() => {
  if (project.value) {
    return t("settings.general.workspace.query-data-policy.tooltip", {
      scope: t(
        "settings.general.workspace.query-data-policy.environment-scope"
      ),
    });
  }
  return t("settings.general.workspace.query-data-policy.tooltip", {
    scope: t("settings.general.workspace.query-data-policy.project-scope"),
  });
});

const policyStore = usePolicyV1Store();

const getInitialState = (): LocalState => {
  return {
    queryDataPolicy: (() => {
      const policy = policyStore.getQueryDataPolicyByParent(props.resource);
      return cloneDeep(policy);
    })(),
    dataSourceQueryPolicy: (() => {
      const policy = policyStore.getDataSourceQueryPolicyByParent(
        props.resource
      );
      return cloneDeep(policy);
    })(),
  };
};

const state = reactive<LocalState>(getInitialState());

watchEffect(async () => {
  if (!hasGetPermission.value) {
    return;
  }
  await Promise.all([
    policyStore.getOrFetchPolicyByParentAndType({
      parentPath: props.resource,
      policyType: PolicyType.DATA_QUERY,
    }),
    policyStore.getOrFetchPolicyByParentAndType({
      parentPath: props.resource,
      policyType: PolicyType.DATA_SOURCE_QUERY,
    }),
  ]);

  Object.assign(state, getInitialState());
});

const adminDataSourceQueryRestrictionEnabled = computed(() => {
  return (
    state.dataSourceQueryPolicy.adminDataSourceRestriction &&
    [
      DataSourceQueryPolicy_Restriction.DISALLOW,
      DataSourceQueryPolicy_Restriction.FALLBACK,
    ].includes(state.dataSourceQueryPolicy.adminDataSourceRestriction)
  );
});

const hasRestrictQueryDataSourceFeature = computed(() =>
  hasFeature(PlanFeature.FEATURE_QUERY_POLICY)
);

const hasRestrictCopyingDataFeature = computed(() =>
  hasFeature(PlanFeature.FEATURE_RESTRICT_COPYING_DATA)
);

const updateQueryDataPolicy = async () => {
  await policyStore.upsertPolicy({
    parentPath: props.resource,
    policy: {
      type: PolicyType.DATA_QUERY,
      policy: {
        case: "queryDataPolicy",
        value: {
          ...state.queryDataPolicy,
        },
      },
    },
  });
};

const switchDataSourceQueryPolicyEnabled = (on: boolean) => {
  state.dataSourceQueryPolicy.adminDataSourceRestriction = on
    ? DataSourceQueryPolicy_Restriction.DISALLOW
    : DataSourceQueryPolicy_Restriction.RESTRICTION_UNSPECIFIED;
};

const updateAdminDataSourceQueryRestrctionPolicy = async () => {
  await policyStore.upsertPolicy({
    parentPath: props.resource,
    policy: {
      type: PolicyType.DATA_SOURCE_QUERY,
      policy: {
        case: "dataSourceQueryPolicy",
        value: {
          ...state.dataSourceQueryPolicy,
        },
      },
    },
  });
};

defineExpose({
  isDirty: computed(() => !isEqual(state, getInitialState())),
  update: async () => {
    const initialState = getInitialState();
    if (
      !isEqual(state.dataSourceQueryPolicy, initialState.dataSourceQueryPolicy)
    ) {
      await updateAdminDataSourceQueryRestrctionPolicy();
    }
    if (!isEqual(state.queryDataPolicy, initialState.queryDataPolicy)) {
      await updateQueryDataPolicy();
    }
  },
  revert: () => {
    Object.assign(state, getInitialState());
  },
});
</script>
