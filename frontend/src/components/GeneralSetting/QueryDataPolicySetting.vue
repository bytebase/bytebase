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
            :value="!disableExport"
            :text="true"
            :disabled="slotProps.disabled || !hasQueryPolicyFeature"
            @update:value="(val: boolean) => disableExport = !val"
          />
        </PermissionGuardWrapper>
        <span class="font-medium">
          {{ $t("settings.general.workspace.data-export.enable") }}
        </span>
      </div>
    <div class="w-full inline-flex items-center gap-x-2">
        <PermissionGuardWrapper
          v-slot="slotProps"
          :permissions="[
            'bb.policies.update'
          ]"
        >
          <Switch
            v-model:value="disableCopyData"
            :text="true"
            :disabled="slotProps.disabled || !hasRestrictCopyingDataFeature"
          />
        </PermissionGuardWrapper>
        <span class="textlabel">
          {{ t("environment.access-control.disable-copy-data") }}
        </span>
        <FeatureBadge :feature="PlanFeature.FEATURE_RESTRICT_COPYING_DATA" />
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
            'bb.policies.update'
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
import { NInputNumber } from "naive-ui";
import { computed, ref, watch } from "vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { Switch } from "@/components/v2";
import { t } from "@/plugins/i18n";
import {
  featureToRef,
  usePolicyByParentAndType,
  usePolicyV1Store,
} from "@/store";
import {
  PolicyResourceType,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";
import MaximumSQLResultSizeSetting from "./MaximumSQLResultSizeSetting.vue";

const props = defineProps<{
  resource: string;
}>();

const policyV1Store = usePolicyV1Store();

const hasQueryPolicyFeature = featureToRef(PlanFeature.FEATURE_QUERY_POLICY);
const hasRestrictCopyingDataFeature = featureToRef(PlanFeature.FEATURE_RESTRICT_COPYING_DATA)

const { ready } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: props.resource,
    policyType: PolicyType.DATA_QUERY,
  }))
);

const policyPayload = computed(() => {
  return policyV1Store.getQueryDataPolicyByParent(props.resource);
});

const initialMaxQueryTimeInseconds = computed(() =>
  Number(policyPayload.value.timeout?.seconds ?? 0)
);

// limit in seconds.
const maxQueryTimeInseconds = ref<number>(initialMaxQueryTimeInseconds.value);
const disableExport = ref(policyPayload.value.disableExport);
const disableCopyData = ref(policyPayload.value.disableCopyData);

const maximumSQLResultSizeSettingRef =
  ref<InstanceType<typeof MaximumSQLResultSizeSetting>>();

const revert = () => {
  maxQueryTimeInseconds.value = initialMaxQueryTimeInseconds.value;
  disableExport.value = policyPayload.value.disableExport;
  disableCopyData.value = policyPayload.value.disableCopyData;
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
    maxQueryTimeInseconds.value !== initialMaxQueryTimeInseconds.value ||
    disableExport.value !== policyPayload.value.disableExport ||
    disableCopyData.value !== policyPayload.value.disableCopyData ||
    maximumSQLResultSizeSettingRef.value?.isDirty
  );
});

const updateChange = async () => {
  if (maximumSQLResultSizeSettingRef.value?.isDirty) {
    await maximumSQLResultSizeSettingRef.value.update();
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
          disableExport: disableExport.value,
          disableCopyData: disableCopyData.value,
          timeout: create(DurationSchema, {
            seconds: BigInt(maxQueryTimeInseconds.value),
          }),
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
