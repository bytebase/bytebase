<template>
  <ArchiveBanner v-if="release.state === State.DELETED" />
  <div class="w-full flex flex-row items-center justify-between gap-x-4">
    <div class="flex-1 p-0.5 overflow-hidden">
      <h1 class="text-xl font-medium truncate">{{ releaseName }}</h1>
    </div>
    <div class="flex items-center justify-end gap-x-3">
      <ApplyToDatabaseButton v-if="release.state === State.ACTIVE" />
      <NDropdown
        v-if="dropdownOptions.length > 0"
        trigger="click"
        :options="dropdownOptions"
        @select="handleDropdownSelect"
      >
        <NButton size="small" quaternary class="px-1!">
          <template #icon>
            <EllipsisVerticalIcon class="w-4 h-4" />
          </template>
        </NButton>
      </NDropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown, useDialog } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { useReleaseStore } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { useReleaseDetailContext } from "../context";
import ApplyToDatabaseButton from "./ApplyToDatabaseButton.vue";

const { release } = useReleaseDetailContext();
const { t } = useI18n();
const dialog = useDialog();
const releaseStore = useReleaseStore();

const releaseName = computed(() => {
  const parts = release.value.name.split("/");
  return parts[parts.length - 1] || release.value.name;
});

const dropdownOptions = computed((): DropdownOption[] => {
  if (release.value.state === State.ACTIVE) {
    return [
      {
        key: "abandon",
        label: t("common.abandon"),
      },
    ];
  } else if (release.value.state === State.DELETED) {
    return [
      {
        key: "restore",
        label: t("common.restore"),
      },
    ];
  }
  return [];
});

const handleDropdownSelect = async (key: string) => {
  if (key === "abandon") {
    dialog.warning({
      title: t("bbkit.confirm-button.sure-to-abandon"),
      content: t("bbkit.confirm-button.can-undo"),
      negativeText: t("common.cancel"),
      positiveText: t("common.confirm"),
      onPositiveClick: async () => {
        await releaseStore.deleteRelease(release.value.name);
      },
    });
  } else if (key === "restore") {
    await releaseStore.undeleteRelease(release.value.name);
  }
};
</script>
