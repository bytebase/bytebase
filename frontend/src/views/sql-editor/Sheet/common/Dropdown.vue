<template>
  <NDropdown
    v-if="options.length > 0"
    trigger="click"
    :options="options"
    v-bind="dropdownProps"
    @select="handleAction"
  >
    <NButton size="tiny" style="--n-padding: 0 4px" quaternary>
      <template #icon>
        <EllipsisIcon
          v-if="!unsaved"
          v-show="!transparent"
          class="bb-overlay-stack-ignore-esc"
        />
        <NTooltip v-if="unsaved" placement="right">
          <template #trigger>
            <DotIcon
              :stroke-width="12"
              class="text-accent w-4 h-4 focus:outline-0"
            />
          </template>
          <template #default>
            <span>{{ $t("sql-editor.tab.unsaved") }}</span>
          </template>
        </NTooltip>
      </template>
    </NButton>
  </NDropdown>
</template>

<script lang="ts" setup>
import { DotIcon, EllipsisIcon } from "lucide-vue-next";
import type { DropdownProps } from "naive-ui";
import {
  type DropdownOption,
  NDropdown,
  useDialog,
  NButton,
  NTooltip,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  useWorkSheetStore,
  pushNotification,
  useSQLEditorTabStore,
  useTabViewStateStore,
} from "@/store";
import {
  Worksheet,
  Worksheet_Visibility,
} from "@/types/proto/v1/worksheet_service";
import {
  isWorksheetWritableV1,
  getSheetStatement,
  defaultSQLEditorTab,
} from "@/utils";
import { useSheetContext, type SheetViewMode } from "../";
import { useSQLEditorContext } from "../../context";

const props = defineProps<{
  view: SheetViewMode;
  sheet: Worksheet;
  dropdownProps?: DropdownProps;
  secondary?: boolean;
  transparent?: boolean;
  unsaved?: boolean;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const { t } = useI18n();
const worksheetV1Store = useWorkSheetStore();
const dialog = useDialog();
const { events } = useSheetContext();
const { events: editorEvents } = useSQLEditorContext();
const { removeViewState } = useTabViewStateStore();

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
    options.push(
      {
        key: "delete",
        label: t("common.delete"),
      },
      {
        key: "rename",
        label: t("sql-editor.tab.context-menu.actions.rename"),
      }
    );
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
  const { sheet: worksheet } = props;
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
          await worksheetV1Store.deleteWorksheetByName(worksheet.name);
          events.emit("refresh", { views: ["my", "shared", "starred"] });
          turnSheetToUnsavedTab(worksheet);
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
    await worksheetV1Store.upsertWorksheetOrganizer({
      worksheet: worksheet.name,
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
        await worksheetV1Store.createWorksheet(
          Worksheet.fromPartial({
            title: worksheet.title,
            project: worksheet.project,
            content: worksheet.content,
            database: worksheet.database,
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
  } else if (key === "rename") {
    editorEvents.emit("save-sheet", {
      tab: {
        ...defaultSQLEditorTab(),
        worksheet: worksheet.name,
        title: worksheet.title,
        statement: getSheetStatement(worksheet),
      },
      editTitle: true,
      mask: ["title"],
    });
  }

  emit("dismiss");
};

const turnSheetToUnsavedTab = (sheet: Worksheet) => {
  const tabStore = useSQLEditorTabStore();
  const tab = tabStore.tabList.find((tab) => tab.worksheet === sheet.name);
  if (tab) {
    tabStore.removeTab(tab);
    removeViewState(tab.id);
  }
};
</script>
