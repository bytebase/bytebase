<template>
  <NDropdown
    v-if="options.length > 0"
    trigger="click"
    :options="options"
    v-bind="dropdownProps"
    @select="handleAction"
  >
    <heroicons-outline:dots-horizontal
      v-show="!transparent"
      class="bb-overlay-stack-ignore-esc w-6 h-6 p-1 rounded outline-none hover:bg-link-hover"
      :class="[secondary ? '' : 'border border-gray-300 bg-white shadow']"
    />
  </NDropdown>
</template>

<script lang="ts" setup>
import {
  type DropdownOption,
  NDropdown,
  useDialog,
  DropdownProps,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useWorkSheetStore, pushNotification, useTabStore } from "@/store";
import {
  Worksheet,
  Worksheet_Visibility,
} from "@/types/proto/v1/worksheet_service";
import { isWorksheetWritableV1 } from "@/utils";
import { useSheetContext, type SheetViewMode } from "../";

const props = defineProps<{
  view: SheetViewMode;
  sheet: Worksheet;
  dropdownProps?: DropdownProps;
  secondary?: boolean;
  transparent?: boolean;
}>();

const { t } = useI18n();
const worksheetV1Store = useWorkSheetStore();
const dialog = useDialog();
const { events } = useSheetContext();

const options = computed(() => {
  const options: DropdownOption[] = [];
  const { sheet, view } = props;

  if (sheet.starred) {
    options.push({
      key: "unstar",
      label: t("sheet.unstar"),
    });
  } else {
    options.push({
      key: "star",
      label: t("sheet.star"),
    });
  }

  const canWriteSheet = isWorksheetWritableV1(sheet);
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
        try {
          dialogInstance.loading = true;
          await worksheetV1Store.deleteSheetByName(sheet.name);
          events.emit("refresh", { views: ["my", "shared", "starred"] });
          turnSheetToUnsavedTab(sheet);
        } finally {
          dialogInstance.destroy();
          dialogInstance.loading = false;
        }
      },
      onNegativeClick() {
        dialogInstance.destroy();
      },
      negativeText: t("common.cancel"),
      positiveText: t("common.confirm"),
      showIcon: true,
    });
  } else if (key === "star" || key === "unstar") {
    await worksheetV1Store.upsertSheetOrganizer({
      worksheet: sheet.name,
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
        await worksheetV1Store.createSheet(
          Worksheet.fromPartial({
            title: sheet.title,
            project: sheet.project,
            content: sheet.content,
            database: sheet.database,
            visibility: Worksheet_Visibility.VISIBILITY_PRIVATE,
          })
        );
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

const turnSheetToUnsavedTab = (sheet: Worksheet) => {
  const tabStore = useTabStore();
  const tab = tabStore.tabList.find((tab) => tab.sheetName === sheet.name);
  if (tab) {
    tab.sheetName = undefined;
    tab.isSaved = false;
  }
};
</script>
