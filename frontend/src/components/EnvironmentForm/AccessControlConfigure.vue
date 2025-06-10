<template>
  <div class="flex flex-col gap-y-2">
    <div class="font-medium flex items-center space-x-2">
      <label>
        {{ $t("environment.access-control.title") }}
      </label>
      // TODO(d): fix this feature control.
      <FeatureBadge :feature="PlanLimitConfig_Feature.QUERY_DATASOURCE_RESTRICTION" />
    </div>
    <div>
      <div class="w-full inline-flex items-center gap-x-2">
        <Switch
          v-model:value="state.disableCopyDataPolicy.active"
          :text="true"
          :disabled="!allowUpdatePolicy || !hasAccessControlFeature"
        />
        <span class="textlabel">{{
          $t("environment.access-control.disable-copy-data-from-sql-editor")
        }}</span>
      </div>
      <div class="">
        <div class="w-full inline-flex items-center gap-x-2">
          <Switch
            :value="adminDataSourceQueruRestrictionEnabled"
            :text="true"
            :disabled="!allowUpdatePolicy || !hasAccessControlFeature"
            @update:value="switchDataSourceQueryPolicyEnabled"
          />
          <span class="textlabel">{{
            $t("environment.access-control.restrict-admin-connection.self")
          }}</span>
        </div>
        <div v-if="adminDataSourceQueruRestrictionEnabled" class="ml-12">
          <NRadioGroup
            v-model:value="
              state.dataSourceQueryPolicy.adminDataSourceRestriction
            "
            :disabled="!allowUpdatePolicy || !hasAccessControlFeature"
          >
            <NRadio
              class="w-full"
              :value="DataSourceQueryPolicy_Restriction.DISALLOW"
            >
              {{
                $t(
                  "environment.access-control.restrict-admin-connection.disallow"
                )
              }}
            </NRadio>
            <NRadio
              class="w-full"
              :value="DataSourceQueryPolicy_Restriction.FALLBACK"
            >
              {{
                $t(
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
    <div class="font-medium flex items-center space-x-2">
      <label>
        {{ $t("environment.statement-execution.title") }}
      </label>
    </div>
    <div>
      <div class="w-full inline-flex items-center gap-x-2">
        <Switch
          v-model:value="state.dataSourceQueryPolicy.disallowDdl"
          :text="true"
          :disabled="!allowUpdatePolicy"
        />
        <span class="textlabel">
          {{ $t("environment.statement-execution.disallow-ddl") }}
        </span>
      </div>
      <div class="w-full inline-flex items-center gap-x-2">
        <Switch
          v-model:value="state.dataSourceQueryPolicy.disallowDml"
          :text="true"
          :disabled="!allowUpdatePolicy"
        />
        <span class="textlabel">
          {{ $t("environment.statement-execution.disallow-dml") }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { hasFeature, usePolicyV1Store } from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import {
  DataSourceQueryPolicy,
  DataSourceQueryPolicy_Restriction,
  DisableCopyDataPolicy,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { PlanLimitConfig_Feature } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { cloneDeep, isEqual } from "lodash-es";
import { NRadio, NRadioGroup } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { FeatureBadge } from "../FeatureGuard";
import { Switch } from "../v2";

interface LocalState {
  disableCopyDataPolicy: DisableCopyDataPolicy;
  dataSourceQueryPolicy: DataSourceQueryPolicy;
}

const props = defineProps<{
  resource: string;
  allowEdit: boolean;
}>();

const policyStore = usePolicyV1Store();

const getInitialState = (): LocalState => {
  return {
    disableCopyDataPolicy: cloneDeep(
      policyStore.getPolicyByParentAndType({
        parentPath: props.resource,
        policyType: PolicyType.DISABLE_COPY_DATA,
      })?.disableCopyDataPolicy ?? DisableCopyDataPolicy.fromPartial({})
    ),
    dataSourceQueryPolicy: cloneDeep(
      policyStore.getPolicyByParentAndType({
        parentPath: props.resource,
        policyType: PolicyType.DATA_SOURCE_QUERY,
      })?.dataSourceQueryPolicy ?? DataSourceQueryPolicy.fromPartial({})
    ),
  };
};

const state = reactive<LocalState>(getInitialState());

watchEffect(async () => {
  await Promise.all([
    policyStore.getOrFetchPolicyByParentAndType({
      parentPath: props.resource,
      policyType: PolicyType.DISABLE_COPY_DATA,
    }),
    policyStore.getOrFetchPolicyByParentAndType({
      parentPath: props.resource,
      policyType: PolicyType.DATA_SOURCE_QUERY,
    }),
  ]);

  Object.assign(state, getInitialState());
});

const adminDataSourceQueruRestrictionEnabled = computed(() => {
  return (
    state.dataSourceQueryPolicy.adminDataSourceRestriction &&
    [
      DataSourceQueryPolicy_Restriction.DISALLOW,
      DataSourceQueryPolicy_Restriction.FALLBACK,
    ].includes(state.dataSourceQueryPolicy.adminDataSourceRestriction)
  );
});

const hasAccessControlFeature = computed(() =>
  hasFeature(PlanLimitConfig_Feature.QUERY_DATASOURCE_RESTRICTION)
);

const allowUpdatePolicy = computed(() => {
  return props.allowEdit && hasWorkspacePermissionV2("bb.policies.update");
});

const updateDisableCopyDataPolicy = async () => {
  await policyStore.upsertPolicy({
    parentPath: props.resource,
    policy: {
      type: PolicyType.DISABLE_COPY_DATA,
      disableCopyDataPolicy: {
        ...state.disableCopyDataPolicy,
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
      dataSourceQueryPolicy: DataSourceQueryPolicy.fromPartial({
        ...state.dataSourceQueryPolicy,
      }),
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
    if (
      !isEqual(state.disableCopyDataPolicy, initialState.disableCopyDataPolicy)
    ) {
      await updateDisableCopyDataPolicy();
    }
  },
  revert: () => {
    Object.assign(state, getInitialState());
  },
});
</script>
