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
import type { SheetCreate, SheetOrganizerUpsert } from "@/types";
import type { SheetViewMode } from "../types";
import { getDefaultSheetPayloadWithSource, isSheetWritableV1 } from "@/utils";
import { useSheetStore, useSheetV1Store, useProjectV1Store } from "@/store";
import {
  getProjectAndSheetId,
  getInstanceAndDatabaseId,
} from "@/store/modules/v1/common";

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

  const canDeleteSheet = isSheetWritableV1(sheet);
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
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      async onPositiveClick() {
        await useSheetV1Store().deleteSheetByName(sheet.name);
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
    const [_, uid] = getProjectAndSheetId(sheet.name);
    const sheetOrganizerUpsert: SheetOrganizerUpsert = {
      sheeId: uid,
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
      autoFocus: false,
      closable: false,
      maskClosable: false,
      closeOnEsc: false,
      async onPositiveClick() {
        const [projectId, _] = getProjectAndSheetId(sheet.name);
        const projectV1 = useProjectV1Store().getProjectByName(projectId);

        const sheetCreate: SheetCreate = {
          projectId: projectV1.uid,
          name: sheet.title,
          statement: new TextDecoder().decode(sheet.content),
          visibility: "PRIVATE",
          payload: getDefaultSheetPayloadWithSource("BYTEBASE"),
          source: "BYTEBASE",
        };
        if (sheet.database) {
          sheetCreate.databaseId = getInstanceAndDatabaseId(sheet.database)[1];
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
