<template>
  <div class="flex flex-col gap-y-2">
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
        <Switch
          v-model:value="state.queryDataPolicy.disableCopyData"
          :text="true"
          :disabled="!allowUpdatePolicy || !hasRestrictCopyingDataFeature"
        />
        <span class="textlabel">{{
          t("environment.access-control.disable-copy-data-from-sql-editor")
        }}</span>
        <FeatureBadge :feature="PlanFeature.FEATURE_RESTRICT_COPYING_DATA" />
      </div>
      <div class="">
        <div class="w-full inline-flex items-center gap-x-2">
          <Switch
            :value="adminDataSourceQueryRestrictionEnabled"
            :text="true"
            :disabled="!allowUpdatePolicy || !hasRestrictQueryDataSourceFeature"
            @update:value="switchDataSourceQueryPolicyEnabled"
          />
          <span class="textlabel">{{
            t("environment.access-control.restrict-admin-connection.self")
          }}</span>
          <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
        </div>
        <div v-if="adminDataSourceQueryRestrictionEnabled" class="ml-12">
          <NRadioGroup
            v-model:value="
              state.dataSourceQueryPolicy.adminDataSourceRestriction
            "
            :disabled="!allowUpdatePolicy || !hasRestrictQueryDataSourceFeature"
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
        </div>
      </div>
    </div>
  </div>
  <div
    v-if="resource.startsWith(environmentNamePrefix)"
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
        <Switch
          v-model:value="state.dataSourceQueryPolicy.disallowDdl"
          :text="true"
          :disabled="!allowUpdatePolicy || !hasRestrictQueryDataSourceFeature"
        />
        <span class="textlabel">
          {{ t("environment.statement-execution.disallow-ddl") }}
        </span>
      </div>
      <div class="w-full inline-flex items-center gap-x-2">
        <Switch
          v-model:value="state.dataSourceQueryPolicy.disallowDml"
          :text="true"
          :disabled="!allowUpdatePolicy || !hasRestrictQueryDataSourceFeature"
        />
        <span class="textlabel">
          {{ t("environment.statement-execution.disallow-dml") }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep, isEqual } from "lodash-es";
import { CircleQuestionMarkIcon } from "lucide-vue-next";
import { NRadio, NRadioGroup, NTooltip } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { hasFeature, usePolicyV1Store } from "@/store";
import {
  environmentNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import type {
  DataSourceQueryPolicy,
  QueryDataPolicy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  DataSourceQueryPolicy_Restriction,
  DataSourceQueryPolicySchema,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { FeatureBadge } from "../FeatureGuard";
import { Switch } from "../v2";

const { t } = useI18n();

interface LocalState {
  queryDataPolicy: QueryDataPolicy;
  dataSourceQueryPolicy: DataSourceQueryPolicy;
}

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const scope = computed(() => {
  if (props.resource.startsWith(projectNamePrefix)) {
    return t("settings.general.workspace.query-data-policy.environment-scope");
  }
  if (props.resource.startsWith(environmentNamePrefix)) {
    return t("settings.general.workspace.query-data-policy.project-scope");
  }
  return "";
});

const tooltip = computed(() => {
  if (!scope.value) {
    return "";
  }
  return t("settings.general.workspace.query-data-policy.tooltip", {
    scope: scope.value,
  });
});

const policyStore = usePolicyV1Store();

const getInitialState = (): LocalState => {
  return {
    queryDataPolicy: (() => {
      const policy = policyStore.getPolicyByParentAndType({
        parentPath: props.resource,
        policyType: PolicyType.DATA_QUERY,
      });
      if (policy?.policy.case === "queryDataPolicy") {
        return cloneDeep(policy.policy.value);
      }
      return createProto(QueryDataPolicySchema, {});
    })(),
    dataSourceQueryPolicy: (() => {
      const policy = policyStore.getPolicyByParentAndType({
        parentPath: props.resource,
        policyType: PolicyType.DATA_SOURCE_QUERY,
      });
      if (policy?.policy.case === "dataSourceQueryPolicy") {
        return cloneDeep(policy.policy.value);
      }
      return createProto(DataSourceQueryPolicySchema, {});
    })(),
  };
};

const state = reactive<LocalState>(getInitialState());

watchEffect(async () => {
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

const allowUpdatePolicy = computed(() => {
  return props.allowEdit && hasWorkspacePermissionV2("bb.policies.update");
});

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
        value: createProto(DataSourceQueryPolicySchema, {
          ...state.dataSourceQueryPolicy,
        }),
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
