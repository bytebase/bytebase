<template>
  <div class="flex flex-col items-stretch gap-y-4 overflow-x-hidden">
    <div class="flex flex-col gap-y-2 max-w-max">
      <div
        v-if="changes.length === 0"
        class="text-control-placeholder text-sm leading-[28px]"
      >
        {{
          $t(
            "changelist.add-change.changelog.select-at-least-one-changelog-below"
          )
        }}
      </div>
      <ChangelogChangeItem
        v-for="change in changes"
        :key="change.source"
        :change="change"
        @click-item="handleClickChange($event)"
        @remove-item="handleRemoveChange($event)"
      />
    </div>
    <div class="flex flex-row items-center justify-start gap-x-2">
      <DatabaseSelect
        v-model:database-name="state.databaseName"
        :project-name="project.name"
        style="max-width: max-content"
      />
      <NCheckboxGroup v-model:value="state.changelogTypes">
        <NCheckbox :value="Changelog_Type.MIGRATE">DDL</NCheckbox>
        <NCheckbox :value="Changelog_Type.DATA">DML</NCheckbox>
      </NCheckboxGroup>
    </div>
    <ChangelogDataTable
      v-model:selected-changelogs="selectedChangelogList"
      :changelogs="filteredChangelogList"
      :show-selection="true"
      :custom-click="true"
      @row-click="state.detailChangelogName = $event.name"
    />
    <ChangelogDetailPanel
      :changelog-name="state.detailChangelogName"
      @close="state.detailChangelogName = undefined"
    />
  </div>
</template>

<script setup lang="ts">
import { orderBy } from "lodash-es";
import { NCheckbox, NCheckboxGroup } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { ChangelogDataTable } from "@/components/Changelog";
import { DatabaseSelect } from "@/components/v2";
import {
  useChangelogStore,
  useDBSchemaV1Store,
  useDatabaseV1Store,
} from "@/store";
import {
  isValidDatabaseName,
  type AffectedTable,
  EmptyAffectedTable,
} from "@/types";
import {
  Changelist_Change,
  type Changelist_Change as Change,
} from "@/types/proto/v1/changelist_service";
import {
  Changelog,
  Changelog_Status,
  Changelog_Type,
} from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName, extractIssueUID } from "@/utils";
import ChangelogDetailPanel from "../../ChangelogDetailPanel";
import { useChangelistDetailContext } from "../../context";
import { useAddChangeContext } from "../context";
import ChangelogChangeItem from "./ChangelogChangeItem.vue";

type LocalState = {
  isLoading: boolean;
  databaseName: string | undefined;
  changelogList: Changelog[];
  affectedTable: AffectedTable;
  changelogTypes: Changelog_Type[];
  detailChangelogName: string | undefined;
};

const changelogStore = useChangelogStore();
const databaseStore = useDatabaseV1Store();
const { project } = useChangelistDetailContext();
const { changesFromChangelog: changes } = useAddChangeContext();

const state = reactive<LocalState>({
  isLoading: false,
  databaseName: undefined,
  changelogList: [],
  affectedTable: EmptyAffectedTable,
  changelogTypes: [Changelog_Type.DATA, Changelog_Type.MIGRATE],
  detailChangelogName: undefined,
});

const database = computed(() => {
  const name = state.databaseName;
  if (!isValidDatabaseName(name)) return undefined;
  return databaseStore.getDatabaseByName(name);
});

const filteredChangelogList = computed(() => {
  return state.changelogList.filter((changelog) => {
    return (
      state.changelogTypes.includes(changelog.type) &&
      changelog.status === Changelog_Status.DONE
    );
  });
});

const selectedChangelogList = computed<string[]>({
  get() {
    return changes.value.map((change) => {
      return change.source;
    });
  },
  set(selected) {
    const changelogs: Changelog[] = [];
    for (let i = 0; i < selected.length; i++) {
      const name = selected[i];
      const changelog = changelogStore.getChangelogByName(name);
      if (changelog) {
        changelogs.push(changelog);
      }
    }

    const updatedChanges = orderBy(
      changelogs,
      [(c) => parseInt(extractIssueUID(c.issue), 10)],
      ["asc"]
    ).map<Change>((changelog) =>
      Changelist_Change.fromPartial({
        sheet: changelog.statementSheet,
        source: changelog.name,
      })
    );
    changes.value = updatedChanges;
  },
});

const fetchChangelogList = async () => {
  const db = database.value;
  if (!db) {
    state.changelogList = [];
    return;
  }

  state.isLoading = true;
  const name = db.name;
  await useDBSchemaV1Store().getOrFetchDatabaseMetadata({
    database: name,
    skipCache: false,
    silent: true,
  });
  const changelogList =
    await changelogStore.getOrFetchChangelogListOfDatabase(name);
  // Check if the state is still valid
  if (name === database.value?.name) {
    state.changelogList = changelogList;
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
  const changelogName = change.source;
  const database = databaseStore.getDatabaseByName(
    extractDatabaseResourceName(changelogName).database
  );
  state.databaseName = database.name;
  state.detailChangelogName = changelogName;
};

watch(() => database.value?.name, fetchChangelogList, { immediate: true });

watch(
  () => state.changelogList,
  () => {
    state.affectedTable = EmptyAffectedTable;
  },
  { immediate: true }
);
</script>
