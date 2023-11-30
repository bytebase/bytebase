<template>
  <div class="flex flex-col items-stretch gap-y-4 relative">
    <NavBar />

    <ChangeTable
      :changes="state.changes"
      :reorder-mode="reorderMode"
      @remove-change="handleRemoveChange($event)"
      @select-change="handleSelectChange($event)"
      @reorder-move="(row, delta) => handleReorderMove(row, delta)"
    />

    <div class="flex flex-row">
      <DeleteChangelistButton />
    </div>

    <AddChangePanel />

    <div
      v-if="isUpdating"
      class="absolute inset-0 bg-white/50 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>

    <ChangeHistoryDetailPanel
      :change-history-name="state.detailChangeHistoryName"
      @close="state.detailChangeHistoryName = undefined"
    />
    <BranchDetailPanel
      :branch-name="state.detailBranchName"
      @close="state.detailBranchName = undefined"
    />
    <RawSQLPanel
      :sheet-name="state.detailRawSQLSheetName"
      @close="state.detailRawSQLSheetName = undefined"
    />

    <ApplyToDatabasePanel />
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { pushNotification, useChangelistStore } from "@/store";
import {
  Changelist_Change as Change,
  Changelist,
} from "@/types/proto/v1/changelist_service";
import { getChangelistChangeSourceType } from "@/utils";
import AddChangePanel from "./AddChangePanel";
import ApplyToDatabasePanel from "./ApplyToDatabasePanel";
import BranchDetailPanel from "./BranchDetailPanel";
import ChangeHistoryDetailPanel from "./ChangeHistoryDetailPanel";
import ChangeTable from "./ChangeTable";
import DeleteChangelistButton from "./DeleteChangelistButton.vue";
import NavBar from "./NavBar";
import RawSQLPanel from "./RawSQLPanel";
import { provideChangelistDetailContext } from "./context";

const { t } = useI18n();
const route = useRoute();
const { changelist, reorderMode, isUpdating, events } =
  provideChangelistDetailContext();

const state = reactive({
  changes: [] as Change[],
  detailChangeHistoryName: undefined as string | undefined,
  detailBranchName: undefined as string | undefined,
  detailRawSQLSheetName: undefined as string | undefined,
});

const patchChanges = async (changes: Change[]) => {
  const patch = {
    ...Changelist.fromPartial(changelist.value),
    changes,
  };
  try {
    isUpdating.value = true;
    await useChangelistStore().patchChangelist(patch, ["changes"]);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    isUpdating.value = false;
  }
};

const handleRemoveChange = async (change: Change) => {
  const changes = [...changelist.value.changes];
  const index = changes.indexOf(change);
  if (index < 0) {
    return;
  }
  changes.splice(index, 1);
  await patchChanges(changes);
};
const handleSelectChange = async (change: Change) => {
  const sourceType = getChangelistChangeSourceType(change);
  if (sourceType === "CHANGE_HISTORY") {
    state.detailChangeHistoryName = change.source;
  }
  if (sourceType === "BRANCH") {
    state.detailBranchName = change.source;
  }
  if (sourceType === "RAW_SQL") {
    state.detailRawSQLSheetName = change.sheet;
  }
};
const handleReorderMove = (row: number, delta: -1 | 1) => {
  const target = row + delta;
  const { changes } = state;
  if (target < 0 || target >= changes.length) {
    return;
  }
  const temp = changes[row];
  changes[row] = changes[target];
  changes[target] = temp;
};

const documentTitle = computed(() => {
  if (route.name !== "workspace.project.changelist.detail") {
    return undefined;
  }
  return changelist.value.description;
});
watch(
  documentTitle,
  (title) => {
    if (title) {
      document.title = title;
    }
  },
  { immediate: true }
);

useEmitteryEventListener(events, "reorder-cancel", () => {
  state.changes = [...changelist.value.changes];
  reorderMode.value = false;
});
useEmitteryEventListener(events, "reorder-confirm", async () => {
  const changes = [...state.changes];
  await patchChanges(changes);
  reorderMode.value = false;
});

watch(
  () => changelist.value.changes,
  (changes) => {
    state.changes = [...changes];
  },
  { immediate: true }
);
</script>
