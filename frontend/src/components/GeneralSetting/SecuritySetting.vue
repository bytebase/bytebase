<template>
  <div id="security" class="py-6 lg:flex space-y-4 lg:space-y-0">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.security") }}
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
            :value="watermarkEnabled"
            :text="true"
            :disabled="!allowEdit || !hasWatermarkFeature"
            @update:value="handleWatermarkToggle"
          />
          <span class="textlabel">
            {{ $t("settings.general.workspace.watermark.enable") }}
          </span>
          <FeatureBadge feature="bb.feature.watermark" />
        </div>
        <div class="mt-1 mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.watermark.description") }}
        </div>
      </div>
      <div>
        <div class="flex items-center gap-x-2">
          <Switch
            :value="dataExportEnable"
            :text="true"
            :disabled="!allowEdit || !hasAccessControlFeature"
            @update:value="handleDataExportToggle"
          />
          <span class="textlabel">
            {{ $t("settings.general.workspace.data-export.enable") }}
          </span>
          <FeatureBadge feature="bb.feature.access-control" />
        </div>
        <div class="mt-1 mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.watermark.description") }}
        </div>
      </div>
      <RestrictIssueCreationConfigure :resource="''" :allow-edit="allowEdit" />
      <MaximumSQLResultSizeSetting :allow-edit="allowEdit" />
      <MaximumRoleExpirationSetting :allow-edit="allowEdit" />
      <DomainRestrictionSetting :allow-edit="allowEdit" />
    </div>
  </div>

  <FeatureModal
    :open="!!state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { Switch } from "@/components/v2";
import {
  featureToRef,
  pushNotification,
  useSettingV1Store,
  usePolicyV1Store,
  usePolicyByParentAndType,
} from "@/store";
import type { FeatureType } from "@/types";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { FeatureBadge, FeatureModal } from "../FeatureGuard";
import DomainRestrictionSetting from "./DomainRestrictionSetting.vue";
import MaximumRoleExpirationSetting from "./MaximumRoleExpirationSetting.vue";
import MaximumSQLResultSizeSetting from "./MaximumSQLResultSizeSetting.vue";
import RestrictIssueCreationConfigure from "./RestrictIssueCreationConfigure.vue";

interface LocalState {
  featureNameForModal?: FeatureType;
}

defineProps<{
  allowEdit: boolean;
}>();

const state = reactive<LocalState>({});
const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const policyV1Store = usePolicyV1Store();
const hasWatermarkFeature = featureToRef("bb.feature.branding");
const hasAccessControlFeature = featureToRef("bb.feature.access-control");

const exportDataPolicy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: "",
    policyType: PolicyType.DATA_EXPORT,
  }))
);

const watermarkEnabled = computed((): boolean => {
  return (
    settingV1Store.getSettingByName("bb.workspace.watermark")?.value
      ?.stringValue === "1"
  );
});

const dataExportEnable = computed(() => {
  return !exportDataPolicy.value?.exportDataPolicy?.disable;
});

const handleDataExportToggle = async (on: boolean) => {
  await policyV1Store.upsertPolicy({
    parentPath: "",
    policy: {
      type: PolicyType.DATA_EXPORT,
      resourceType: PolicyResourceType.WORKSPACE,
      exportDataPolicy: {
        disable: !on,
      },
    },
  });

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const handleWatermarkToggle = async (on: boolean) => {
  if (!hasWatermarkFeature.value && on) {
    state.featureNameForModal = "bb.feature.watermark";
    return;
  }
  const value = on ? "1" : "0";
  await settingV1Store.upsertSetting({
    name: "bb.workspace.watermark",
    value: {
      stringValue: value,
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.watermark.update-success"),
  });
};
</script>
