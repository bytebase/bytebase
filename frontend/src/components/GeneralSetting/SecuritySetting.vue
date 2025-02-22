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
            v-model:value="state.enableDataExport"
            :text="true"
            :disabled="!allowEdit || !hasAccessControlFeature"
          />
          <span class="textlabel">
            {{ $t("settings.general.workspace.data-export.enable") }}
          </span>
          <FeatureBadge feature="bb.feature.access-control" />
        </div>
        <div class="mt-1 mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.data-export.description") }}
        </div>
      </div>
      <RestrictIssueCreationConfigure
        ref="restrictIssueCreationConfigureRef"
        :resource="''"
        :allow-edit="allowEdit"
      />
      <MaximumSQLResultSizeSetting
        ref="maximumSQLResultSizeSettingRef"
        :allow-edit="allowEdit"
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

  <FeatureModal
    :open="!!state.featureNameForModal"
    :feature="state.featureNameForModal"
    @cancel="state.featureNameForModal = undefined"
  />
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import { computed, reactive, ref } from "vue";
import { Switch } from "@/components/v2";
import {
  featureToRef,
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
  enableWatermark: boolean;
  enableDataExport: boolean;
}

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const policyV1Store = usePolicyV1Store();
const hasWatermarkFeature = featureToRef("bb.feature.branding");
const hasAccessControlFeature = featureToRef("bb.feature.access-control");

const domainRestrictionSettingRef =
  ref<InstanceType<typeof DomainRestrictionSetting>>();
const maximumRoleExpirationSettingRef =
  ref<InstanceType<typeof MaximumRoleExpirationSetting>>();
const maximumSQLResultSizeSettingRef =
  ref<InstanceType<typeof MaximumSQLResultSizeSetting>>();
const restrictIssueCreationConfigureRef =
  ref<InstanceType<typeof RestrictIssueCreationConfigure>>();

const { policy: exportDataPolicy } = usePolicyByParentAndType(
  computed(() => ({
    parentPath: "",
    policyType: PolicyType.DATA_EXPORT,
  }))
);

const getInitialState = (): LocalState => {
  return {
    enableWatermark:
      settingV1Store.getSettingByName("bb.workspace.watermark")?.value
        ?.stringValue === "1",
    enableDataExport: !exportDataPolicy.value?.exportDataPolicy?.disable,
  };
};

const state = reactive<LocalState>({
  ...getInitialState(),
});

const isDirty = computed(() => {
  return (
    !isEqual(state, getInitialState()) ||
    domainRestrictionSettingRef.value?.isDirty ||
    restrictIssueCreationConfigureRef.value?.isDirty ||
    maximumSQLResultSizeSettingRef.value?.isDirty ||
    maximumRoleExpirationSettingRef.value?.isDirty
  );
});

const handleDataExportToggle = async () => {
  await policyV1Store.upsertPolicy({
    parentPath: "",
    policy: {
      type: PolicyType.DATA_EXPORT,
      resourceType: PolicyResourceType.WORKSPACE,
      exportDataPolicy: {
        disable: !state.enableDataExport,
      },
    },
  });
};

const handleWatermarkToggle = async () => {
  const value = state.enableWatermark ? "1" : "0";
  await settingV1Store.upsertSetting({
    name: "bb.workspace.watermark",
    value: {
      stringValue: value,
    },
  });
};

const onUpdate = async () => {
  if (domainRestrictionSettingRef.value?.isDirty) {
    await domainRestrictionSettingRef.value.update();
  }
  if (restrictIssueCreationConfigureRef.value?.isDirty) {
    await restrictIssueCreationConfigureRef.value.update();
  }
  if (maximumSQLResultSizeSettingRef.value?.isDirty) {
    await maximumSQLResultSizeSettingRef.value.update();
  }
  if (maximumRoleExpirationSettingRef.value?.isDirty) {
    await maximumRoleExpirationSettingRef.value.update();
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
});
</script>
