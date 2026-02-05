<template>
  <div class="flex-1 flex flex-col gap-y-6">
    <div class="w-full inline-flex items-center gap-x-2">
      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="[
          'bb.policies.update'
        ]"
      >
        <Switch
          :value="!state.disableExport"
          :text="true"
          :disabled="slotProps.disabled || !hasQueryPolicyFeature"
          @update:value="(val: boolean) => state.disableExport = !val"
        />
      </PermissionGuardWrapper>
      <span class="font-medium">
        {{ $t("settings.general.workspace.data-export") }}
      </span>
      <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
    </div>
    <div class="w-full inline-flex items-center gap-x-2">
      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="[
          'bb.policies.update'
        ]"
      >
        <Switch
          v-model:value="state.disableCopyData"
          :text="true"
          :disabled="slotProps.disabled || !hasRestrictCopyingDataFeature"
        />
      </PermissionGuardWrapper>
      <span class="textlabel">
        {{ $t("settings.general.workspace.disable-copy-data") }}
      </span>
      <FeatureBadge :feature="PlanFeature.FEATURE_RESTRICT_COPYING_DATA" />
    </div>
    <div>
      <div class="w-full inline-flex items-center gap-x-2">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="[
            'bb.policies.update'
          ]"
        >
          <Switch
            v-model:value="state.allowAdminDataSource"
            :text="true"
            :disabled="slotProps.disabled || !hasQueryPolicyFeature"
          />
        </PermissionGuardWrapper>
        <span class="textlabel">
          {{ t("settings.general.workspace.allow-admin-data-source.self") }}
        </span>
        <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
      </div>
      <span class="mt-1 text-sm text-gray-400">
        {{ t("settings.general.workspace.allow-admin-data-source.description") }}
      </span>
    </div>
    <MaximumSQLResultSizeSetting
      ref="maximumSQLResultSizeSettingRef"
      :resource="resource"
      :policy="policyPayload"
    />
    <div>
      <p class="font-medium flex flex-row justify-start items-center">
        <span class="mr-2">
          {{ $t("settings.general.workspace.query-data-policy.timeout.self") }}
        </span>
        <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
      </p>
      <p class="text-sm text-gray-400 mt-1">
        {{
          $t("settings.general.workspace.query-data-policy.timeout.description")
        }}
        <span class="font-semibold! textinfolabel">
          {{ $t("settings.general.workspace.no-limit") }}
        </span>
      </p>
      <div class="mt-3 w-full flex flex-row justify-start items-center gap-4">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="[
            'bb.settings.setWorkspaceProfile'
          ]"
        >
          <NInputNumber
            :value="maxQueryTimeInseconds"
            :disabled="!hasQueryPolicyFeature || slotProps.disabled"
            class="w-60"
            :min="0"
            :precision="0"
            @update:value="handleInput"
          >
            <template #suffix>{{
              $t("settings.general.workspace.query-data-policy.seconds")
            }}</template>
          </NInputNumber>
        </PermissionGuardWrapper>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import { NInputNumber } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { Switch } from "@/components/v2";
import { t } from "@/plugins/i18n";
import {
  featureToRef,
  usePolicyByParentAndType,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";
import MaximumSQLResultSizeSetting from "./MaximumSQLResultSizeSetting.vue";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";

interface LocalState {
  disableExport: boolean;
  disableCopyData: boolean;
  allowAdminDataSource: boolean;
}

const props = defineProps<{
  resource: string;
}>();

const policyV1Store = usePolicyV1Store();
const settingV1Store = useSettingV1Store();

const hasQueryPolicyFeature = featureToRef(PlanFeature.FEATURE_QUERY_POLICY);
const hasRestrictCopyingDataFeature = featureToRef(
  PlanFeature.FEATURE_RESTRICT_COPYING_DATA
);

const { ready } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.resource,
    policyType: PolicyType.DATA_QUERY,
  }))
);

const policyPayload = computed(() => {
  return policyV1Store.getQueryDataPolicyByParent(props.resource);
});

const getInitialState = (): LocalState => {
  return {
    disableExport: policyPayload.value.disableExport,
    disableCopyData: policyPayload.value.disableCopyData,
    allowAdminDataSource: policyPayload.value.allowAdminDataSource,
  };
};

const getInitTimeInseconds = () => {
  return Number(settingV1Store.workspaceProfile.queryTimeout?.seconds ?? 0)
}

const state = reactive<LocalState>(getInitialState());
const maxQueryTimeInseconds = ref<number>(
  getInitTimeInseconds()
)

const maximumSQLResultSizeSettingRef =
  ref<InstanceType<typeof MaximumSQLResultSizeSetting>>();

const revert = () => {
  Object.assign(state, getInitialState());
  maxQueryTimeInseconds.value = getInitTimeInseconds()
  maximumSQLResultSizeSettingRef.value?.revert();
};

watch(
  () => ready.value,
  (ready) => {
    if (ready) {
      revert();
    }
  }
);

const isDirty = computed(() => {
  return (
    !isEqual(state, getInitialState()) ||
    getInitTimeInseconds() !== maxQueryTimeInseconds.value ||
    maximumSQLResultSizeSettingRef.value?.isDirty
  );
});

const updateChange = async () => {
  if (maximumSQLResultSizeSettingRef.value?.isDirty) {
    await maximumSQLResultSizeSettingRef.value.update();
  }

  if (getInitTimeInseconds() !== maxQueryTimeInseconds.value) {
    await settingV1Store.updateWorkspaceProfile({
      payload: {
        queryTimeout: create(DurationSchema, {
            seconds: BigInt(maxQueryTimeInseconds.value),
          })
      },
      updateMask: create(FieldMaskSchema, {
        paths: ["value.workspace_profile.query_timeout"],
      }),
    });
  }
  await policyV1Store.upsertPolicy({
    parentPath: props.resource,
    policy: {
      type: PolicyType.DATA_QUERY,
      resourceType: PolicyResourceType.WORKSPACE,
      policy: {
        case: "queryDataPolicy",
        value: create(QueryDataPolicySchema, {
          ...policyPayload.value,
          disableExport: state.disableExport,
          disableCopyData: state.disableCopyData,
          allowAdminDataSource: state.allowAdminDataSource,
        }),
      },
    },
  });
};

const handleInput = (value: number | null) => {
  if (value === null) return;
  if (value === undefined) return;
  maxQueryTimeInseconds.value = value;
};

defineExpose({
  isDirty,
  update: updateChange,
  revert,
});
</script>
