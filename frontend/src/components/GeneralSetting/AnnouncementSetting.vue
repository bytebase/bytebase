<template>
  <div id="announcement" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center gap-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
        <FeatureBadge :feature="PlanFeature.FEATURE_DASHBOARD_ANNOUNCEMENT" />
      </div>

      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{
          $t("settings.general.workspace.announcement.admin-or-dba-can-edit")
        }}
      </span>
    </div>
    <div class="flex-1 lg:px-5">
      <div class="mt-5 lg:mt-0">
        <label class="flex items-center gap-x-2">
          <span class="font-medium">{{
            $t(
              "settings.general.workspace.announcement-alert-level.description"
            )
          }}</span>
        </label>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <div class="flex flex-wrap py-2 radio-set-row gap-4">
              <AnnouncementLevelSelect
                v-model:level="state.level"
                :allow-edit="allowEdit && hasAnnouncementFeature"
              />
            </div>
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{
              $t(
                "settings.general.workspace.announcement.admin-or-dba-can-edit"
              )
            }}
          </span>
        </NTooltip>

        <label class="flex items-center mt-2 gap-x-2">
          <span class="font-medium"
            >{{ $t("settings.general.workspace.announcement-text.self") }}
          </span>
        </label>
        <div class="mb-3 text-sm text-gray-400">
          {{ $t("settings.general.workspace.announcement-text.description") }}
        </div>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <NInput
              v-model:value="state.text"
              class="mb-3 w-full"
              :placeholder="
                $t('settings.general.workspace.announcement-text.placeholder')
              "
              :disabled="!allowEdit || !hasAnnouncementFeature"
            />
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{
              $t(
                "settings.general.workspace.announcement.admin-or-dba-can-edit"
              )
            }}
          </span>
        </NTooltip>

        <label class="flex items-center py-2 gap-x-2">
          <span class="font-medium">{{
            $t("settings.general.workspace.extra-link.self")
          }}</span>
        </label>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <NInput
              v-model:value="state.link"
              class="mb-5 w-full"
              :placeholder="
                $t('settings.general.workspace.extra-link.placeholder')
              "
              :disabled="!allowEdit || !hasAnnouncementFeature"
            />
          </template>
          <span class="text-sm text-gray-400 -translate-y-2">
            {{
              $t(
                "settings.general.workspace.announcement.admin-or-dba-can-edit"
              )
            }}
          </span>
        </NTooltip>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { NInput, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { AnnouncementLevelSelect } from "@/components/v2";
import { featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import type { Announcement } from "@/types/proto-es/v1/setting_service_pb";
import {
  Announcement_AlertLevel,
  AnnouncementSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { FeatureBadge } from "../FeatureGuard";

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const hasAnnouncementFeature = featureToRef(
  PlanFeature.FEATURE_DASHBOARD_ANNOUNCEMENT
);

const rawAnnouncement = computed(
  () =>
    settingV1Store.workspaceProfileSetting?.announcement ??
    create(AnnouncementSchema, {
      level: Announcement_AlertLevel.INFO,
    })
);

const state = reactive<Announcement>(cloneDeep(rawAnnouncement.value));

const allowSave = computed((): boolean => {
  return !isEqual(rawAnnouncement.value, state);
});

const updateAnnouncementSetting = async () => {
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      announcement: { ...state },
    },
    updateMask: create(FieldMaskSchema, {
      paths: ["value.workspace_profile.announcement"],
    }),
  });
};

defineExpose({
  isDirty: allowSave,
  title: props.title,
  update: updateAnnouncementSetting,
  revert: () => {
    Object.assign(state, cloneDeep(rawAnnouncement.value));
  },
});
</script>
