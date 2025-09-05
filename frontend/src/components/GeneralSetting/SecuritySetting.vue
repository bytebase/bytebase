<template>
  <div id="security" class="py-6 lg:flex space-y-4 lg:space-y-0">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>

    <div class="flex-1 lg:px-4 space-y-7">
      <div>
        <div class="flex items-center gap-x-2">
          <Switch
            v-model:value="state.enableWatermark"
            :text="true"
            :disabled="!allowEdit || !hasWatermarkFeature"
          />
          <span class="font-medium">
            {{ $t("settings.general.workspace.watermark.enable") }}
          </span>
          <FeatureBadge :feature="PlanFeature.FEATURE_WATERMARK" />
        </div>
        <div class="mt-1 mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.watermark.description") }}
        </div>
      </div>
      <div>
        <div class="flex items-center gap-x-2">
          <Switch
            v-model:value="state.enableDataExport"
            :text="true"
            :disabled="!allowEdit"
          />
          <span class="font-medium">
            {{ $t("settings.general.workspace.data-export.enable") }}
          </span>
        </div>
        <div class="mt-1 mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.data-export.description") }}
        </div>
      </div>
      <RestrictIssueCreationConfigure
        ref="restrictIssueCreationConfigureRef"
        :resource="''"
        :allow-edit="allowEdit"
        text-class="font-medium"
      />
      <MaximumSQLResultSizeSetting
        ref="maximumSQLResultSizeSettingRef"
        :allow-edit="allowEdit"
      />
      <QueryDataPolicySetting ref="queryDataPolicySettingRef" />
      <MaximumRoleExpirationSetting
        ref="maximumRoleExpirationSettingRef"
        :allow-edit="allowEdit"
      />
      <DomainRestrictionSetting
        ref="domainRestrictionSettingRef"
        :allow-edit="allowEdit"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { computed, reactive, ref } from "vue";
import { Switch } from "@/components/v2";
import {
  featureToRef,
  usePolicyByParentAndType,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import {
  QueryDataPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  Setting_SettingName,
  ValueSchema as SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";
import DomainRestrictionSetting from "./DomainRestrictionSetting.vue";
import MaximumRoleExpirationSetting from "./MaximumRoleExpirationSetting.vue";
import MaximumSQLResultSizeSetting from "./MaximumSQLResultSizeSetting.vue";
import QueryDataPolicySetting from "./QueryDataPolicySetting.vue";
import RestrictIssueCreationConfigure from "./RestrictIssueCreationConfigure.vue";

interface LocalState {
  enableWatermark: boolean;
  enableDataExport: boolean;
}

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const policyV1Store = usePolicyV1Store();
const hasWatermarkFeature = featureToRef(PlanFeature.FEATURE_WATERMARK);

const domainRestrictionSettingRef =
  ref<InstanceType<typeof DomainRestrictionSetting>>();
const maximumRoleExpirationSettingRef =
  ref<InstanceType<typeof MaximumRoleExpirationSetting>>();
const maximumSQLResultSizeSettingRef =
  ref<InstanceType<typeof MaximumSQLResultSizeSetting>>();
const restrictIssueCreationConfigureRef =
  ref<InstanceType<typeof RestrictIssueCreationConfigure>>();
const queryDataPolicySettingRef =
  ref<InstanceType<typeof QueryDataPolicySetting>>();

const settingRefList = computed(() => [
  domainRestrictionSettingRef,
  maximumRoleExpirationSettingRef,
  maximumSQLResultSizeSettingRef,
  restrictIssueCreationConfigureRef,
  queryDataPolicySettingRef,
]);

const { policy: queryDataPolicy } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: "",
    policyType: PolicyType.DATA_QUERY,
  }))
);

const policyPayload = computed(() => {
  return queryDataPolicy.value?.policy?.case === "queryDataPolicy"
    ? queryDataPolicy.value.policy.value
    : create(QueryDataPolicySchema);
});

const getInitialState = (): LocalState => {
  const watermarkSetting = settingV1Store.getSettingByName(
    Setting_SettingName.WATERMARK
  );
  let enableWatermark = false;
  if (watermarkSetting?.value?.value?.case === "stringValue") {
    enableWatermark = watermarkSetting.value.value.value === "1";
  }
  return {
    enableWatermark,
    enableDataExport: !policyPayload.value.disableExport,
  };
};

const state = reactive<LocalState>({
  ...getInitialState(),
});

const isDirty = computed(() => {
  return (
    !isEqual(state, getInitialState()) ||
    settingRefList.value.some((settingRef) => settingRef.value?.isDirty)
  );
});

const handleDataExportToggle = async () => {
  await policyV1Store.upsertPolicy({
    parentPath: "",
    policy: {
      type: PolicyType.DATA_QUERY,
      resourceType: PolicyResourceType.WORKSPACE,
      policy: {
        case: "queryDataPolicy",
        value: create(QueryDataPolicySchema, {
          ...policyPayload.value,
          disableExport: !state.enableDataExport,
        }),
      },
    },
  });
};

const handleWatermarkToggle = async () => {
  const value = state.enableWatermark ? "1" : "0";
  await settingV1Store.upsertSetting({
    name: Setting_SettingName.WATERMARK,
    value: create(SettingValueSchema, {
      value: {
        case: "stringValue",
        value: value,
      },
    }),
  });
};

const onUpdate = async () => {
  for (const settingRef of settingRefList.value) {
    if (settingRef.value?.isDirty) {
      await settingRef.value.update();
    }
  }
  if (state.enableWatermark !== getInitialState().enableWatermark) {
    await handleWatermarkToggle();
  }
  if (state.enableDataExport !== getInitialState().enableDataExport) {
    await handleDataExportToggle();
  }
};

defineExpose({
  isDirty,
  update: onUpdate,
  title: props.title,
  revert: () => {
    Object.assign(state, getInitialState());
    for (const settingRef of settingRefList.value) {
      settingRef.value?.revert();
    }
  },
});
</script>
