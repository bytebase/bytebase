<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>
          {{ $t("issue.data-export.title") }}
        </span>
      </div>
    </template>

    <div
      class="space-y-4 h-full w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div v-if="ready">
        <div class="space-y-3">
          <div class="w-full flex items-center space-x-2">
            <AdvancedSearchBox
              v-model:params="state.params"
              :autofocus="false"
              :placeholder="$t('database.filter-database')"
              :support-option-id-list="supportOptionIdList"
            />
            <DatabaseLabelFilter
              v-model:selected="state.selectedLabels"
              :database-list="rawDatabaseList"
              :placement="'left-start'"
            />
          </div>
          <DatabaseV1Table
            mode="ALL_SHORT"
            table-class="border"
            :custom-click="true"
            :database-list="selectableDatabaseList"
            :show-selection-column="true"
            :show-sql-editor-button="false"
            :show-placeholder="true"
            @select-database="handleSelectedDatabaseChange"
          >
            <template #selection="{ database }">
              <NRadio
                :checked="isDatabaseSelected(database as ComposedDatabase)"
                :value="(database as ComposedDatabase).uid"
                @click="
                  () =>
                    handleSelectedDatabaseChange(database as ComposedDatabase)
                "
              />
            </template>
          </DatabaseV1Table>
        </div>
      </div>
      <div
        v-if="!ready"
        class="w-full h-[20rem] flex items-center justify-center"
      >
        <BBSpin />
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
            :disabled="!state.selectedDatabaseUid"
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
import { NButton, NRadio } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import {
  useCurrentUserV1,
  useSearchDatabaseV1List,
  useDatabaseV1Store,
  useProjectV1Store,
} from "@/store";
import type { ComposedDatabase } from "@/types";
import { UNKNOWN_ID, DEFAULT_PROJECT_V1_NAME } from "@/types";
import { State } from "@/types/proto/v1/common";
import type { SearchScopeId, SearchParams } from "@/utils";
import {
  filterDatabaseV1ByKeyword,
  sortDatabaseV1List,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  generateIssueName,
  extractProjectResourceName,
} from "@/utils";
import { DatabaseLabelFilter, DatabaseV1Table, DrawerContent } from "../v2";

type LocalState = {
  label: string;
  selectedDatabaseUid?: string;
  selectedLabels: { key: string; value: string }[];
  params: SearchParams;
};

const props = defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});

const emit = defineEmits(["dismiss"]);

const router = useRouter();
const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const databaseV1Store = useDatabaseV1Store();

const state = reactive<LocalState>({
  label: "environment",
  selectedLabels: [],
  params: {
    query: "",
    scopes: [],
  },
});

const selectedProject = computed(() => {
  if (props.projectId) {
    return projectV1Store.getProjectByUID(props.projectId);
  }
  const filter = state.params.scopes.find(
    (scope) => scope.id === "project"
  )?.value;
  if (filter) {
    return projectV1Store.getProjectByName(`projects/${filter}`);
  }
  return undefined;
});

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

const { ready } = useSearchDatabaseV1List({
  filter: "instance = instances/-",
});

const rawDatabaseList = computed(() => {
  let list: ComposedDatabase[] = [];
  if (selectedProject.value) {
    list = databaseV1Store.databaseListByProject(selectedProject.value.name);
  } else {
    list = databaseV1Store.databaseListByUser(currentUserV1.value);
  }
  list = list.filter(
    (db) =>
      db.syncState == State.ACTIVE && db.project !== DEFAULT_PROJECT_V1_NAME
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
      "project",
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

const selectableDatabaseList = computed(() => {
  return filteredDatabaseList.value;
});

const handleSelectedDatabaseChange = (database: ComposedDatabase) => {
  state.selectedDatabaseUid = database.uid;
};

const isDatabaseSelected = (database: ComposedDatabase): boolean => {
  return state.selectedDatabaseUid === database.uid;
};

const navigateToIssuePage = async () => {
  if (!state.selectedDatabaseUid) {
    return;
  }

  const selectedDatabase = selectableDatabaseList.value.find(
    (db) => db.uid === state.selectedDatabaseUid
  ) as ComposedDatabase;

  const project = selectedDatabase?.projectEntity;
  const issueType = "bb.issue.database.data.export";
  const query: Record<string, any> = {
    template: issueType,
    name: generateIssueName(issueType, [selectedDatabase.databaseName]),
    project: project.uid,
    databaseList: state.selectedDatabaseUid,
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

const supportOptionIdList = computed((): SearchScopeId[] => {
  return ["project", "instance", "environment"];
});
</script>
