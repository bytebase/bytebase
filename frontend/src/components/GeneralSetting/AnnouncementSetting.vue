<template>
  <div class="py-6 lg:flex">
    <div class="text-left lg:w-1/4">
      <div class="flex items-center space-x-2">
        <h1 class="text-2xl font-bold">
          {{ $t("settings.general.workspace.announcement.self") }}
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
        <label
          class="flex items-center gap-x-2"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
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
                v-model:level="state.announcement.level"
                :allow-edit="allowEdit"
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

        <label
          class="flex items-center mt-2 gap-x-2"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
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
              v-model:value="state.announcement.text"
              class="mb-3 w-full"
              :placeholder="
                $t('settings.general.workspace.announcement-text.placeholder')
              "
              :disabled="!allowEdit"
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

        <label
          class="flex items-center py-2 gap-x-2"
          :class="[allowEdit ? 'cursor-pointer' : 'cursor-not-allowed']"
        >
          <span class="font-medium">{{
            $t("settings.general.workspace.extra-link.self")
          }}</span>
        </label>
        <NTooltip placement="top-start" :disabled="allowEdit">
          <template #trigger>
            <NInput
              v-model:value="state.announcement.link"
              class="mb-5 w-full"
              :placeholder="
                $t('settings.general.workspace.extra-link.placeholder')
              "
              :disabled="!allowEdit"
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

        <div class="flex justify-end">
          <NButton
            type="primary"
            :disabled="!allowSave"
            @click.prevent="updateAnnouncementSetting"
          >
            {{ $t("common.update") }}
          </NButton>
        </div>
      </div>
    </div>

    <FeatureModal
      feature="bb.feature.announcement"
      :open="state.showFeatureModal"
      @cancel="state.showFeatureModal = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { AnnouncementLevelSelect } from "@/components/v2";
import { pushNotification, useCurrentUserV1, featureToRef } from "@/store";
import { useSettingV1Store } from "@/store/modules/v1/setting";
import {
  Announcement,
  Announcement_AlertLevel,
} from "@/types/proto/v1/setting_service";
import { hasWorkspacePermissionV1 } from "@/utils";

interface LocalState {
  announcement: Announcement;
  showFeatureModal: boolean;
}

const { t } = useI18n();
const settingV1Store = useSettingV1Store();
const currentUserV1 = useCurrentUserV1();
const hasAnnouncementSetting = featureToRef("bb.feature.announcement");

const defaultAnnouncement = function (): Announcement {
  return {
    level: Announcement_AlertLevel.ALERT_LEVEL_INFO,
    text: "",
    link: "",
  };
};

const state = reactive<LocalState>({
  announcement: defaultAnnouncement(),
  showFeatureModal: false,
});

watchEffect(() => {
  const announcement = settingV1Store.workspaceProfileSetting?.announcement;
  if (announcement) {
    state.announcement = cloneDeep(announcement);
  }
});

const allowEdit = computed((): boolean => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-announcement",
    currentUserV1.value.userRole
  );
});

const allowSave = computed((): boolean => {
  if (!allowEdit.value) {
    return false;
  }

  if (
    settingV1Store.workspaceProfileSetting?.announcement === undefined &&
    state.announcement.text === ""
  ) {
    return false;
  }

  return !isEqual(
    settingV1Store.workspaceProfileSetting?.announcement,
    state.announcement
  );
});

const updateAnnouncementSetting = async () => {
  if (!hasAnnouncementSetting.value) {
    state.showFeatureModal = true;
    return;
  }

  if (!allowSave.value) {
    return;
  }

  if (state.announcement.text === "" && state.announcement.link !== "") {
    state.announcement.link = "";
  }

  // remove announcement setting from store if both text and link are empty regardless of level.
  let announcement: Announcement | undefined = cloneDeep(state.announcement);
  if (announcement.text === "" && announcement.link === "") {
    announcement = undefined;
  }

  await settingV1Store.updateWorkspaceProfile({
    announcement: announcement,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("settings.general.workspace.announcement.update-success"),
  });

  const currentSetting: Announcement | undefined = cloneDeep(
    settingV1Store.workspaceProfileSetting?.announcement
  );
  if (currentSetting === undefined) {
    state.announcement = defaultAnnouncement();
  } else {
    state.announcement = cloneDeep(currentSetting);
  }
};
</script>
