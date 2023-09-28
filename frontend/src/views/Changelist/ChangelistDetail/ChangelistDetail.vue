<template>
  <div class="flex flex-col gap-y-4 px-4 relative">
    <NavBar />

    <ChangeTable
      v-model:selected="selectedChanges"
      :changes="state.changes"
      :reorder-mode="reorderMode"
      @remove-change="handleRemoveChange($event)"
      @select-change="handleSelectChange($event)"
    />

    <AddChangePanel />

    <div
      v-if="state.isUpdating"
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
  </div>
</template>

<script lang="ts" setup>
import { useTitle } from "@vueuse/core";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useChangelistStore } from "@/store";
import {
  Changelist_Change as Change,
  Changelist,
} from "@/types/proto/v1/changelist_service";
import { getChangelistChangeSourceType } from "@/utils";
import AddChangePanel from "./AddChangePanel";
import BranchDetailPanel from "./BranchDetailPanel";
import ChangeHistoryDetailPanel from "./ChangeHistoryDetailPanel";
import ChangeTable from "./ChangeTable";
import NavBar from "./NavBar";
import { provideChangelistDetailContext } from "./context";

const { t } = useI18n();
const { changelist, reorderMode, selectedChanges } =
  provideChangelistDetailContext();

const state = reactive({
  changes: [] as Change[],
  isUpdating: false,
  detailChangeHistoryName: undefined as string | undefined,
  detailBranchName: undefined as string | undefined,
  detailRawSQLSheetName: undefined as string | undefined,
});

const handleRemoveChange = async (change: Change) => {
  const changes = [...changelist.value.changes];
  const index = changes.indexOf(change);
  if (index < 0) {
    return;
  }
  changes.splice(index, 1);
  const patch = {
    ...Changelist.fromPartial(changelist.value),
    changes,
  };
  try {
    state.isUpdating = true;
    await useChangelistStore().patchChangelist(patch, ["changes"]);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.isUpdating = false;
  }
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
    alert("tbd:" + change.sheet);
  }
};

const documentTitle = computed(() => {
  return changelist.value.description;
});
useTitle(documentTitle);

watch(
  () => changelist.value.changes,
  (changes) => {
    state.changes = [...changes];
  },
  { immediate: true }
);
</script>
