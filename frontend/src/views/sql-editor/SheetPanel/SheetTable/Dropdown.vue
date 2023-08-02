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

import { Sheet } from "@/types/proto/v1/sheet_service";
import { useSheetPanelContext, type SheetViewMode } from "../common";
import { extractProjectResourceName, isSheetWritableV1 } from "@/utils";
import { useSheetV1Store, pushNotification } from "@/store";
import {
  Sheet_Visibility,
  Sheet_Source,
  Sheet_Type,
} from "@/types/proto/v1/sheet_service";

const props = defineProps<{
  view: SheetViewMode;
  sheet: Sheet;
}>();

const { t } = useI18n();
const sheetV1Store = useSheetV1Store();
const dialog = useDialog();
const { events } = useSheetPanelContext();

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

  const canWriteSheet = isSheetWritableV1(sheet);
  if (canWriteSheet) {
    options.push({
      key: "delete",
      label: t("common.delete"),
    });
  }
  if (view === "shared") {
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
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      async onPositiveClick() {
        await sheetV1Store.deleteSheetByName(sheet.name);
        events.emit("refresh", { views: ["my", "shared", "starred"] });
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
    await sheetV1Store.upsertSheetOrganizer({
      sheet: sheet.name,
      starred: key === "star",
    });
    events.emit("refresh", { views: ["starred"] });
  } else if (key === "duplicate") {
    const dialogInstance = dialog.create({
      title: t("sheet.hint-tips.confirm-to-duplicate-sheet"),
      type: "info",
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      async onPositiveClick() {
        const project = extractProjectResourceName(sheet.name);
        await sheetV1Store.createSheet(`projects/${project}`, {
          title: sheet.title,
          content: sheet.content,
          database: sheet.database,
          visibility: Sheet_Visibility.VISIBILITY_PRIVATE,
          source: Sheet_Source.SOURCE_BYTEBASE,
          type: Sheet_Type.TYPE_SQL,
          payload: "{}",
        });
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("sheet.notifications.duplicate-success"),
        });
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
