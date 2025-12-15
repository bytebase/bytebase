<template>
  <div class="flex-1 flex flex-col gap-y-6">
    <div>
        <div class="flex items-center gap-x-2">
          <Switch
            :value="!disableExport"
            :text="true"
            :disabled="!allowEdit"
            @update:value="(val: boolean) => disableExport = !val"
          />
          <span class="font-medium">
            {{ $t("settings.general.workspace.data-export.enable") }}
          </span>
          <NTooltip v-if="scope">
            <template #trigger>
              <CircleQuestionMarkIcon class="w-4 textinfolabel" />
            </template>
            <span>
              {{ $t("settings.general.workspace.query-data-policy.tooltip", { scope }) }}
            </span>
          </NTooltip>
        </div>
        <div class="mt-1 text-sm text-gray-400">
          {{ $t("settings.general.workspace.data-export.description") }}
        </div>
      </div>
      <MaximumSQLResultSizeSetting
        ref="maximumSQLResultSizeSettingRef"
        :allow-edit="allowEdit"
        :resource="resource"
        :policy="policyPayload"
        :tooltip="scope ? $t('settings.general.workspace.query-data-policy.tooltip', { scope }) : ''"
      />
      <div>
    <p class="font-medium flex flex-row justify-start items-center">
      <span class="mr-2">
        {{ $t("settings.general.workspace.query-data-policy.timeout.self") }}
      </span>
      <FeatureBadge :feature="PlanFeature.FEATURE_QUERY_POLICY" />
      <NTooltip v-if="scope">
        <template #trigger>
          <CircleQuestionMarkIcon class="w-4 textinfolabel" />
        </template>
        <span>
          {{ $t("settings.general.workspace.query-data-policy.tooltip", { scope }) }}
        </span>
      </NTooltip>
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
      <NInputNumber
        :value="maxQueryTimeInseconds"
        :disabled="!allowEdit"
        class="w-60"
        :min="0"
        :precision="0"
        @update:value="handleInput"
      >
        <template #suffix>{{
          $t("settings.general.workspace.query-data-policy.seconds")
        }}</template>
      </NInputNumber>
    </div>
  </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { DurationSchema } from "@bufbuild/protobuf/wkt";
import { CircleQuestionMarkIcon } from "lucide-vue-next";
import { NInputNumber, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { Switch } from "@/components/v2";
import { t } from "@/plugins/i18n";
import {
  featureToRef,
  usePolicyByParentAndType,
  usePolicyV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  PolicyResourceType,
  PolicyType,
  QueryDataPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { FeatureBadge } from "../FeatureGuard";
import MaximumSQLResultSizeSetting from "./MaximumSQLResultSizeSetting.vue";

const props = defineProps<{
  resource: string;
}>();

const policyV1Store = usePolicyV1Store();
const hasQueryPolicyFeature = featureToRef(PlanFeature.FEATURE_QUERY_POLICY);

const scope = computed(() => {
  if (props.resource.startsWith(projectNamePrefix)) {
    return t("settings.general.workspace.query-data-policy.workspace-scope");
  }
  if (props.resource === "") {
    return t("settings.general.workspace.query-data-policy.project-scope");
  }
  return "";
});

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

const maximumSQLResultSizeSettingRef =
  ref<InstanceType<typeof MaximumSQLResultSizeSetting>>();

const revert = () => {
  maxQueryTimeInseconds.value = initialMaxQueryTimeInseconds.value;
  disableExport.value = policyPayload.value.disableExport;
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

const allowEdit = computed(
  () =>
    hasWorkspacePermissionV2("bb.policies.update") &&
    hasQueryPolicyFeature.value
);

const isDirty = computed(() => {
  return (
    maxQueryTimeInseconds.value !== initialMaxQueryTimeInseconds.value ||
    disableExport.value !== policyPayload.value.disableExport ||
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
