<template>
  <DrawerContent class="max-w-[100vw]">
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>
          {{ $t("custom-approval.risk-rule.risk.namespace.data_export") }}
        </span>
      </div>
    </template>

    <div
      class="space-y-4 h-full w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div class="space-y-3">
        <div class="w-full flex items-center space-x-2">
          <AdvancedSearch
            v-model:params="state.params"
            :placeholder="$t('database.filter-database')"
            :scope-options="scopeOptions"
          />
          <DatabaseLabelFilter
            v-model:selected="state.selectedLabels"
            :database-list="rawDatabaseList"
            :placement="'left-start'"
          />
        </div>
        <DatabaseV1Table
          mode="ALL_SHORT"
          :database-list="filteredDatabaseList"
          :single-selection="true"
          :keyword="state.params.query.trim().toLowerCase()"
          @update:selected-databases="handleDatabasesSelectionChanged"
        />
      </div>
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div></div>

        <div class="flex items-center justify-end gap-x-3">
          <NButton @click.prevent="cancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!state.selectedDatabaseName"
            @click="navigateToIssuePage"
          >
            {{ $t("common.next") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import AdvancedSearch from "@/components/AdvancedSearch";
import DatabaseV1Table from "@/components/v2/Model/DatabaseV1Table";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useProjectByName } from "@/store";
import type { ComposedDatabase } from "@/types";
import { UNKNOWN_ID, DEFAULT_PROJECT_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import type { SearchParams } from "@/utils";
import {
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  generateIssueTitle,
  extractProjectResourceName,
} from "@/utils";
import { useCommonSearchScopeOptions } from "../AdvancedSearch/useCommonSearchScopeOptions";
import { DatabaseLabelFilter, DrawerContent } from "../v2";

type LocalState = {
  label: string;
  selectedDatabaseName?: string;
  selectedLabels: { key: string; value: string }[];
  params: SearchParams;
};

const props = defineProps({
  projectName: {
    type: String,
    required: true,
  },
});

const emit = defineEmits(["dismiss"]);

const router = useRouter();
const databaseV1Store = useDatabaseV1Store();

const state = reactive<LocalState>({
  label: "environment",
  selectedLabels: [],
  params: {
    query: "",
    scopes: [],
  },
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  ["environment", "instance"]
);

const { project: selectedProject } = useProjectByName(props.projectName);

const selectedInstance = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "instance")?.value ??
    `${UNKNOWN_ID}`
  );
});

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ??
    `${UNKNOWN_ID}`
  );
});

const rawDatabaseList = computed(() => {
  let list: ComposedDatabase[] = [];
  if (selectedProject.value) {
    list = databaseV1Store.databaseListByProject(selectedProject.value.name);
  } else {
    list = databaseV1Store.databaseListByUser;
  }
  list = list.filter(
    (db) => db.syncState == State.ACTIVE && db.project !== DEFAULT_PROJECT_NAME
  );
  return list;
});

const filteredDatabaseList = computed(() => {
  let list = [...rawDatabaseList.value];

  list = list.filter((db) => {
    if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
      return (
        extractEnvironmentResourceName(db.effectiveEnvironment) ===
        selectedEnvironment.value
      );
    }
    if (selectedInstance.value !== `${UNKNOWN_ID}`) {
      return (
        extractInstanceResourceName(db.instance) === selectedInstance.value
      );
    }
    return filterDatabaseV1ByKeyword(db, state.params.query.trim(), [
      "name",
      "environment",
      "instance",
    ]);
  });

  const labels = state.selectedLabels;
  if (labels.length > 0) {
    list = list.filter((db) => {
      return labels.some((kv) => db.labels[kv.key] === kv.value);
    });
  }

  return sortDatabaseV1List(list);
});

const handleDatabasesSelectionChanged = (
  selectedDatabaseNameList: Set<string>
): void => {
  if (selectedDatabaseNameList.size !== 1) {
    return;
  }
  state.selectedDatabaseName = Array.from(selectedDatabaseNameList)[0];
};

const navigateToIssuePage = async () => {
  if (!state.selectedDatabaseName) {
    return;
  }

  const selectedDatabase = filteredDatabaseList.value.find(
    (db) => db.name === state.selectedDatabaseName
  ) as ComposedDatabase;

  const project = selectedDatabase?.projectEntity;
  const issueType = "bb.issue.database.data.export";
  const query: Record<string, any> = {
    template: issueType,
    name: generateIssueTitle(issueType, [selectedDatabase.databaseName]),
    databaseList: selectedDatabase.name,
  };
  router.push({
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.name),
      issueSlug: "create",
    },
    query,
  });
};

const cancel = () => {
  emit("dismiss");
};
</script>
