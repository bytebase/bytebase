<template>
  <div class="flex flex-col gap-y-4">
    <div class="flex flex-col gap-y-2 max-w-max">
      <div
        v-if="changes.length === 0"
        class="text-control-placeholder text-sm leading-[28px]"
      >
        {{
          $t("changelist.add-change.branch.select-at-least-one-branch-below")
        }}
      </div>
      <BranchChangeItem
        v-for="change in changes"
        :key="change.source"
        :change="change"
        :branch="branchForChange(change)"
        @click-item="handleClickChange($event)"
        @remove-item="handleRemoveChange($event)"
      />
    </div>
    <div class="flex flex-row items-center justify-between py-0.5">
      <div class="flex flex-row items-center justify-start gap-x-2">
        <DatabaseSelect
          v-model:database="state.databaseUID"
          :project="project.uid"
        />
      </div>
      <div class="flex flex-row items-center justify-end gap-x-2">
        <SearchBox
          v-model:value="state.keyword"
          :placeholder="$t('common.filter-by-name')"
        />
      </div>
    </div>

    <BranchTable
      v-model:selected="selectedBranchList"
      :branch-list="filteredBranchList"
      :is-fetching="!ready"
      :keyword="state.keyword"
      @click-item="state.detailBranchName = $event.name"
    />
    <BranchDetailPanel
      :branch-name="state.detailBranchName"
      @close="state.detailBranchName = undefined"
    />
  </div>
</template>

<script setup lang="ts">
import { first, orderBy } from "lodash-es";
import { computed, reactive, watch } from "vue";
import { DatabaseSelect } from "@/components/v2";
import { useDatabaseV1Store, useLocalSheetStore } from "@/store";
import { useBranchList, useBranchStore } from "@/store/modules/branch";
import { UNKNOWN_ID } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import { keyBy } from "@/utils";
import BranchDetailPanel from "../../BranchDetailPanel";
import { useChangelistDetailContext } from "../../context";
import { useAddChangeContext } from "../context";
import BranchChangeItem from "./BranchChangeItem.vue";
import BranchTable from "./BranchTable";

type LocalState = {
  keyword: string;
  databaseUID: string | undefined;
  branchList: Branch[];
  detailBranchName: string | undefined;
};

const state = reactive<LocalState>({
  keyword: "",
  databaseUID: undefined,
  branchList: [],
  detailBranchName: undefined,
});

const { project } = useChangelistDetailContext();
const { changesFromBranch: changes } = useAddChangeContext();
const { branchList, ready } = useBranchList();
const localSheetStore = useLocalSheetStore();

const database = computed(() => {
  const uid = state.databaseUID;
  if (!uid || uid === String(UNKNOWN_ID)) return undefined;
  return useDatabaseV1Store().getDatabaseByUID(uid);
});

const filteredBranchList = computed(() => {
  const db = database.value;
  if (!db) {
    return [];
  }
  const branchListFilterByDatabase = branchList.value.filter((branch) => {
    return branch.baselineDatabase === db.name;
  });
  let list = orderBy(
    branchListFilterByDatabase,
    (branch) => branch.updateTime,
    "desc"
  );

  const kw = state.keyword.trim().toLowerCase();
  if (kw) {
    list = list.filter((branch) => branch.title.toLowerCase().includes(kw));
  }

  return list;
});

const selectedBranchList = computed<string[]>({
  get() {
    return changes.value.map((change) => {
      return change.source;
    });
  },
  set(selected) {
    const existedChangesByBranchName = keyBy(
      changes.value,
      (change) => change.source
    );
    const updatedChanges: Change[] = [];
    for (let i = 0; i < selected.length; i++) {
      const name = selected[i];
      const existedChange = existedChangesByBranchName.get(name);
      if (existedChange) {
        updatedChanges.push(existedChange);
      } else {
        const uid = localSheetStore.nextUID();
        const sheet = localSheetStore.createLocalSheet(
          `${project.value.name}/sheets/${uid}`
        );
        updatedChanges.push({
          sheet: sheet.name,
          source: name,
        });
      }
    }
    changes.value = updatedChanges;
  },
});

const handleRemoveChange = (change: Change) => {
  const index = changes.value.findIndex((c) => c.source === change.source);
  if (index >= 0) {
    changes.value.splice(index, 1);
  }
};

const handleClickChange = async (change: Change) => {
  const branchName = change.source;
  const branch = await useBranchStore().fetchBranchByName(
    branchName,
    true /* useCache */
  );
  const database = useDatabaseV1Store().getDatabaseByName(
    branch.baselineDatabase
  );
  state.databaseUID = database.uid;
  state.detailBranchName = branchName;
};

const branchForChange = (change: Change) => {
  return branchList.value.find((br) => br.name === change.source);
};

// Select the first database automatically
watch(
  () => project.value.name,
  (project) => {
    const databaseList = useDatabaseV1Store().databaseListByProject(project);
    state.databaseUID = first(databaseList)?.uid;
  },
  { immediate: true }
);
</script>
