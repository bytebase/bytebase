<template>
  <div class="flex flex-col items-stretch gap-y-4 overflow-x-hidden">
    <div class="flex flex-col gap-y-2 max-w-max">
      <div
        v-if="changes.length === 0"
        class="text-control-placeholder text-sm leading-[28px]"
      >
        {{
          $t(
            "changelist.add-change.change-history.select-at-least-one-history-below"
          )
        }}
      </div>
      <ChangeHistoryChangeItem
        v-for="change in changes"
        :key="change.source"
        :change="change"
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
        <AffectedTableSelect
          v-model:affected-table="state.affectedTable"
          :change-history-list="state.changeHistoryList"
          style="width: 12rem"
        />
        <NCheckboxGroup v-model:value="state.changeHistoryTypes">
          <NCheckbox :value="ChangeHistory_Type.MIGRATE">DDL</NCheckbox>
          <NCheckbox :value="ChangeHistory_Type.DATA">DML</NCheckbox>
        </NCheckboxGroup>
      </div>
      <div class="flex flex-row items-center justify-end gap-x-2">
        <SearchBox
          v-model:value="state.keyword"
          :placeholder="$t('common.filter-by-name')"
        />
      </div>
    </div>
    <ChangeHistoryTable
      v-model:selected="selectedChangeHistoryList"
      :change-history-list="filteredChangeHistoryList"
      :is-fetching="state.isLoading"
      :keyword="state.keyword"
      class="w-full"
      @click-item="state.detailChangeHistoryName = $event.name"
    />

    <ChangeHistoryDetailPanel
      :change-history-name="state.detailChangeHistoryName"
      @close="state.detailChangeHistoryName = undefined"
    />
  </div>
</template>

<script setup lang="ts">
import { first, isEqual } from "lodash-es";
import { NCheckbox, NCheckboxGroup } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { AffectedTableSelect } from "@/components/ChangeHistory";
import { DatabaseSelect, SearchBox } from "@/components/v2";
import {
  useChangeHistoryStore,
  useDBSchemaV1Store,
  useDatabaseV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { AffectedTable, EmptyAffectedTable } from "@/types/changeHistory";
import { Changelist_Change as Change } from "@/types/proto/v1/changelist_service";
import {
  ChangeHistory,
  ChangeHistory_Status,
  ChangeHistory_Type,
} from "@/types/proto/v1/database_service";
import {
  extractDatabaseResourceName,
  getAffectedTablesOfChangeHistory,
  keyBy,
} from "@/utils";
import ChangeHistoryDetailPanel from "../../ChangeHistoryDetailPanel";
import { useChangelistDetailContext } from "../../context";
import { useAddChangeContext } from "../context";
import ChangeHistoryChangeItem from "./ChangeHistoryChangeItem.vue";
import ChangeHistoryTable from "./ChangeHistoryTable";
import { semanticChangeHistoryType } from "./utils";

type LocalState = {
  isLoading: boolean;
  keyword: string;
  databaseUID: string | undefined;
  changeHistoryList: ChangeHistory[];
  affectedTable: AffectedTable;
  changeHistoryTypes: ChangeHistory_Type[];
  detailChangeHistoryName: string | undefined;
};

const { project } = useChangelistDetailContext();
const { changesFromChangeHistory: changes } = useAddChangeContext();

const state = reactive<LocalState>({
  isLoading: false,
  keyword: "",
  databaseUID: undefined,
  changeHistoryList: [],
  affectedTable: EmptyAffectedTable,
  changeHistoryTypes: [ChangeHistory_Type.DATA, ChangeHistory_Type.MIGRATE],
  detailChangeHistoryName: undefined,
});

const database = computed(() => {
  const uid = state.databaseUID;
  if (!uid || uid === String(UNKNOWN_ID)) return undefined;
  return useDatabaseV1Store().getDatabaseByUID(uid);
});

const filteredChangeHistoryList = computed(() => {
  let list = [...state.changeHistoryList];
  const types = state.changeHistoryTypes;
  list = list.filter((changeHistory) => {
    const semanticType = semanticChangeHistoryType(changeHistory.type);
    return (
      types.includes(semanticType) &&
      changeHistory.status === ChangeHistory_Status.DONE
    );
  });

  const kw = state.keyword.trim().toLowerCase();
  if (kw) {
    list = list.filter((changeHistory) =>
      changeHistory.version.toLowerCase().includes(kw)
    );
  }
  const { affectedTable: table } = state;
  if (!isEqual(table, EmptyAffectedTable)) {
    list = list.filter((changeHistory) => {
      const affectedTables = getAffectedTablesOfChangeHistory(changeHistory);
      return affectedTables.find((item) => isEqual(item, table));
    });
  }

  return list;
});

const selectedChangeHistoryList = computed<string[]>({
  get() {
    return changes.value.map((change) => {
      return change.source;
    });
  },
  set(selected) {
    const existedChangesByChangeHistoryName = keyBy(
      changes.value,
      (change) => change.source
    );
    const updatedChanges: Change[] = [];
    for (let i = 0; i < selected.length; i++) {
      const name = selected[i];
      const changeHistory =
        useChangeHistoryStore().getChangeHistoryByName(name);
      if (changeHistory) {
        const existedChange = existedChangesByChangeHistoryName.get(name);
        if (existedChange) {
          updatedChanges.push(existedChange);
        } else {
          updatedChanges.push({
            sheet: changeHistory.statementSheet,
            source: changeHistory.name,
          });
        }
      }
    }
    changes.value = updatedChanges;
  },
});

const fetchChangeHistoryList = async () => {
  const db = database.value;
  if (!db) {
    state.changeHistoryList = [];
    return;
  }

  state.isLoading = true;
  const name = db.name;
  await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
    database: name,
    skipCache: false,
    silent: true,
  });
  const changeHistoryList =
    await useChangeHistoryStore().getOrFetchChangeHistoryListOfDatabase(name);
  // Check if the state is still valid
  if (name === database.value?.name) {
    state.changeHistoryList = changeHistoryList;
  }
  state.isLoading = false;
};

const handleRemoveChange = (change: Change) => {
  const index = changes.value.findIndex((c) => c.source === change.source);
  if (index >= 0) {
    changes.value.splice(index, 1);
  }
};

const handleClickChange = (change: Change) => {
  const changeHistoryName = change.source;
  const database = useDatabaseV1Store().getDatabaseByName(
    extractDatabaseResourceName(changeHistoryName).full
  );
  state.databaseUID = database.uid;
  state.detailChangeHistoryName = changeHistoryName;
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

watch(() => database.value?.uid, fetchChangeHistoryList, { immediate: true });

watch(
  () => state.changeHistoryList,
  (options) => {
    state.affectedTable = EmptyAffectedTable;
  },
  { immediate: true }
);
</script>
