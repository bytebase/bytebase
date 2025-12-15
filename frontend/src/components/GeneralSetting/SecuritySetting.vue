<template>
  <div id="security" class="py-6 lg:flex gap-y-4 lg:gap-y-0">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center gap-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
      </div>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>

    <div class="flex-1 lg:px-4 flex flex-col gap-y-6">
      <div>
        <div class="flex items-center gap-x-2">
          <Switch
            v-model:value="enableWatermark"
            :text="true"
            :disabled="!allowEdit || !hasWatermarkFeature"
          />
          <span class="font-medium">
            {{ $t("settings.general.workspace.watermark.enable") }}
          </span>
          <FeatureBadge :feature="PlanFeature.FEATURE_WATERMARK" />
        </div>
        <div class="mt-1 text-sm text-gray-400">
          {{ $t("settings.general.workspace.watermark.description") }}
        </div>
      </div>
      <QueryDataPolicySetting
        ref="queryDataPolicySettingRef"
        resource=""
      />
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
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { computed, ref } from "vue";
import { Switch } from "@/components/v2";
import { featureToRef, useSettingV1Store } from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";
import DomainRestrictionSetting from "./DomainRestrictionSetting.vue";
import MaximumRoleExpirationSetting from "./MaximumRoleExpirationSetting.vue";
import QueryDataPolicySetting from "./QueryDataPolicySetting.vue";

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const hasWatermarkFeature = featureToRef(PlanFeature.FEATURE_WATERMARK);

const domainRestrictionSettingRef =
  ref<InstanceType<typeof DomainRestrictionSetting>>();
const maximumRoleExpirationSettingRef =
  ref<InstanceType<typeof MaximumRoleExpirationSetting>>();
const queryDataPolicySettingRef =
  ref<InstanceType<typeof QueryDataPolicySetting>>();

const settingRefList = computed(() => [
  domainRestrictionSettingRef,
  maximumRoleExpirationSettingRef,
  queryDataPolicySettingRef,
]);

const initEnableWatermark = computed(() => {
  return settingV1Store.workspaceProfileSetting?.watermark ?? false;
});

const enableWatermark = ref(initEnableWatermark.value);

const isDirty = computed(() => {
  return (
    enableWatermark.value !== initEnableWatermark.value ||
    settingRefList.value.some((settingRef) => settingRef.value?.isDirty)
  );
});

const handleWatermarkToggle = async () => {
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      watermark: enableWatermark.value,
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.watermark"],
    }),
  });
};

const onUpdate = async () => {
  for (const settingRef of settingRefList.value) {
    if (settingRef.value?.isDirty) {
      await settingRef.value.update();
    }
  }
  if (enableWatermark.value !== initEnableWatermark.value) {
    await handleWatermarkToggle();
  }
};

defineExpose({
  isDirty,
  update: onUpdate,
  title: props.title,
  revert: () => {
    enableWatermark.value = initEnableWatermark.value;
    for (const settingRef of settingRefList.value) {
      settingRef.value?.revert();
    }
  },
});
</script>
