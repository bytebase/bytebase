<template>
  <div id="announcement" class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ title }}
        </h1>
        <FeatureBadge feature="bb.feature.announcement" />
      </div>

      <span v-if="!allowEdit" class="text-sm text-gray-400">
        {{
          $t("settings.general.workspace.announcement.admin-or-dba-can-edit")
        }}
      </span>
    </div>
    <div class="flex-1 lg:px-5">
      <div class="mb-7 mt-5 lg:mt-0">
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
import { cloneDeep, isEqual } from "lodash-es";
import { NInput, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { AnnouncementLevelSelect } from "@/components/v2";
import { featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import { Announcement } from "@/types/proto/v1/setting_service";
import { Announcement_AlertLevel } from "@/types/proto/v1/setting_service";
import { FeatureBadge } from "../FeatureGuard";

const props = defineProps<{
  title: string;
  allowEdit: boolean;
}>();

const settingV1Store = useSettingV1Store();
const hasAnnouncementFeature = featureToRef("bb.feature.announcement");

const rawAnnouncement = computed(() =>
  cloneDeep(
    settingV1Store.workspaceProfileSetting?.announcement ??
      Announcement.fromPartial({
        level: Announcement_AlertLevel.ALERT_LEVEL_INFO,
      })
  )
);

const state = reactive<Announcement>(rawAnnouncement.value);

const allowSave = computed((): boolean => {
  return !isEqual(rawAnnouncement.value, state);
});

const updateAnnouncementSetting = async () => {
  await settingV1Store.updateWorkspaceProfile({
    payload: {
      announcement: { ...state },
    },
    updateMask: ["value.workspace_profile_setting_value.announcement"],
  });
};

defineExpose({
  isDirty: allowSave,
  title: props.title,
  update: updateAnnouncementSetting,
  revert: () => {
    Object.assign(state, rawAnnouncement.value);
  },
});
</script>
