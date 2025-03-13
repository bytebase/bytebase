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
        </div>

        <PagedDatabaseTable
          mode="ALL_SHORT"
          :single-selection="true"
          :custom-click="true"
          :filter="filter"
          :parent="projectName"
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
import { PagedDatabaseTable } from "@/components/v2/Model/DatabaseV1Table";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { useDatabaseV1Store, useProjectByName } from "@/store";
import {
  instanceNamePrefix,
  environmentNamePrefix,
} from "@/store/modules/v1/common";
import { isValidDatabaseName } from "@/types";
import type { SearchParams, SearchScope } from "@/utils";
import { generateIssueTitle, extractProjectResourceName } from "@/utils";
import { useCommonSearchScopeOptions } from "../AdvancedSearch/useCommonSearchScopeOptions";
import { DrawerContent } from "../v2";

type LocalState = {
  selectedDatabaseName?: string;
  params: SearchParams;
};

const props = defineProps<{
  projectName: string;
}>();

useProjectByName(props.projectName);

const emit = defineEmits(["dismiss"]);

const router = useRouter();
const databaseV1Store = useDatabaseV1Store();

const readonlyScopes = computed((): SearchScope[] => [
  {
    id: "project",
    value: extractProjectResourceName(props.projectName),
    readonly: true,
  },
]);
const state = reactive<LocalState>({
  params: {
    query: "",
    scopes: [...readonlyScopes.value],
  },
});

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  ["environment", "instance", "label"]
);

const selectedInstance = computed(() => {
  const instanceId = state.params.scopes.find(
    (scope) => scope.id === "instance"
  )?.value;
  if (!instanceId) {
    return;
  }
  return `${instanceNamePrefix}${instanceId}`;
});

const selectedEnvironment = computed(() => {
  const environmentId = state.params.scopes.find(
    (scope) => scope.id === "environment"
  )?.value;
  if (!environmentId) {
    return;
  }
  return `${environmentNamePrefix}${environmentId}`;
});

const selectedLabels = computed(() => {
  return state.params.scopes
    .filter((scope) => scope.id === "label")
    .map((scope) => scope.value);
});

const filter = computed(() => ({
  instance: selectedInstance.value,
  environment: selectedEnvironment.value,
  query: state.params.query,
  labels: selectedLabels.value,
}));

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

  const selectedDatabase = databaseV1Store.getDatabaseByName(
    state.selectedDatabaseName
  );
  if (!isValidDatabaseName(selectedDatabase.name)) {
    return;
  }

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
