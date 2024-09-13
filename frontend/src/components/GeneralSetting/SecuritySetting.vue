<template>
  <div id="security" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.security") }}
        </h1>
        <FeatureBadge feature="bb.feature.watermark" />
      </div>
      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{ $t("settings.general.workspace.only-admin-can-edit") }}
      </span>
    </div>

    <div class="flex-1 lg:px-4">
      <div class="mb-7 mt-4 lg:mt-0">
        <div class="flex items-center gap-x-2">
          <Switch
            :value="watermarkEnabled"
            :text="true"
            :disabled="!allowEdit"
            @update:value="handleWatermarkToggle"
          />
          <span class="textlabel">
            {{ $t("settings.general.workspace.watermark.enable") }}
          </span>
        </div>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.watermark.description") }}
        </div>
      </div>
      <RestrictIssueCreationConfigure
        class="mb-7 mt-4 lg:mt-0"
        :resource="''"
        :allow-edit="allowEdit"
      />
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
import { featureToRef, pushNotification } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import type { FeatureType } from "@/types";
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
const hasWatermarkFeature = featureToRef("bb.feature.branding");

const watermarkEnabled = computed((): boolean => {
  return (
    settingV1Store.getSettingByName("bb.workspace.watermark")?.value
      ?.stringValue === "1"
  );
});

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
