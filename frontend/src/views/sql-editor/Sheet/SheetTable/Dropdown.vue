<template>
  <NDropdown
    v-if="options.length > 0"
    trigger="click"
    :options="options"
    @select="handleAction"
  >
    <heroicons-outline:dots-horizontal
      class="w-6 h-auto border border-gray-300 bg-white p-1 rounded outline-none shadow"
    />
  </NDropdown>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { type DropdownOption, NDropdown, useDialog } from "naive-ui";

import type { Sheet, SheetCreate, SheetOrganizerUpsert } from "@/types";
import type { SheetViewMode } from "../types";
import { getDefaultSheetPayloadWithSource, isSheetWritable } from "@/utils";
import { useSheetStore } from "@/store";

const props = defineProps<{
  view: SheetViewMode;
  sheet: Sheet;
}>();

const emit = defineEmits<{
  (event: "refresh"): void;
}>();

const { t } = useI18n();
const sheetStore = useSheetStore();
const dialog = useDialog();

const options = computed(() => {
  const options: DropdownOption[] = [];
  const { sheet, view } = props;

  if (sheet.starred) {
    options.push({
      key: "unstar",
      label: t("common.unstar"),
    });
  } else {
    options.push({
      key: "star",
      label: t("common.star"),
    });
  }

  const canDeleteSheet = isSheetWritable(sheet);
  if (view === "my") {
    if (canDeleteSheet) {
      options.push({
        key: "delete",
        label: t("common.delete"),
      });
    }
  } else if (view === "shared") {
    if (canDeleteSheet) {
      options.push({
        key: "delete",
        label: t("common.delete"),
      });
    }

    options.push({
      key: "duplicate",
      label: t("common.duplicate"),
    });
  }

  return options;
});

const handleAction = async (key: string) => {
  const { sheet } = props;
  if (key === "delete") {
    const dialogInstance = dialog.create({
      title: t("sheet.hint-tips.confirm-to-delete-this-sheet"),
      type: "info",
      async onPositiveClick() {
        await sheetStore.deleteSheetById(sheet.id);
        dialogInstance.destroy();
      },
      onNegativeClick() {
        dialogInstance.destroy();
      },
      negativeText: t("common.cancel"),
      positiveText: t("common.confirm"),
      showIcon: true,
    });
  } else if (key === "star" || key === "unstar") {
    const sheetOrganizerUpsert: SheetOrganizerUpsert = {
      sheeId: sheet.id,
    };

    if (key === "star") {
      sheetOrganizerUpsert.starred = true;
    } else if (key === "unstar") {
      sheetOrganizerUpsert.starred = false;
    }

    await sheetStore.upsertSheetOrganizer(sheetOrganizerUpsert);
    emit("refresh");
  } else if (key === "duplicate") {
    const dialogInstance = dialog.create({
      title: t("sheet.hint-tips.confirm-to-duplicate-sheet"),
      type: "info",
      async onPositiveClick() {
        const sheetCreate: SheetCreate = {
          projectId: sheet.projectId,
          name: sheet.name,
          statement: sheet.statement,
          visibility: "PRIVATE",
          payload: getDefaultSheetPayloadWithSource("BYTEBASE"),
          source: "BYTEBASE",
        };
        if (sheet.databaseId) {
          sheetCreate.databaseId = sheet.databaseId;
        }
        await sheetStore.createSheet(sheetCreate);
        dialogInstance.destroy();
      },
      onNegativeClick() {
        dialogInstance.destroy();
      },
      negativeText: t("common.cancel"),
      positiveText: t("common.confirm"),
      showIcon: true,
    });
  }
};
</script>
