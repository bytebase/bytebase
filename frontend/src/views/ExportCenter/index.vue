<template>
  <div class="w-full px-4 py-6">
    <div
      class="w-full flex flex-col lg:flex-row items-start lg:items-center justify-between gap-2"
    >
      <div class="flex flex-1 max-w-full items-center gap-x-2">
        <AdvancedSearchBox
          v-model:params="state.params"
          :autofocus="false"
          :placeholder="''"
          :support-option-id-list="supportOptionIdList"
        />
        <NButton
          v-if="filterIssueId !== String(UNKNOWN_ID)"
          @click="clearFilterIssueId"
        >
          <span>#{{ filterIssueId }}</span>
          <heroicons:x-mark class="w-4 h-4 ml-1 -mr-1.5" />
        </NButton>
      </div>
      <NButton type="primary" @click="handleRequestExportClick">
        <FeatureBadge
          feature="bb.feature.access-control"
          custom-class="text-white pointer-events-none mr-2"
        />
        {{ $t("quick-action.request-export-data") }}
      </NButton>
    </div>
    <div class="w-full mt-4">
      <ExportRecordTable :export-records="filterExportRecords" />
    </div>
  </div>

  <RequestExportPanel
    v-if="state.showRequestExportPanel"
    :project-id="selectedProject?.uid"
    :database-id="selectedDatabase?.uid"
    :redirect-to-issue-page="true"
    :statement-only="true"
    @close="state.showRequestExportPanel = false"
  />

  <FeatureModal
    feature="bb.feature.access-control"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import RequestExportPanel from "@/components/Issue/panel/RequestExportPanel/index.vue";
import { getExpiredDateTime } from "@/components/ProjectMember/ProjectRoleTable/utils";
import {
  featureToRef,
  useCurrentUserV1,
  useDatabaseV1Store,
  useInstanceV1Store,
  useProjectIamPolicyStore,
  useProjectV1ListByCurrentUser,
  useProjectV1Store,
} from "@/store";
import { UNKNOWN_ID, PresetRoleType } from "@/types";
import { SearchParams, SearchScopeId } from "@/utils";
import { convertFromExpr } from "@/utils/issue/cel";
import ExportRecordTable from "./ExportRecordTable.vue";
import { ExportRecord } from "./types";

interface LocalState {
  exportRecords: ExportRecord[];
  showRequestExportPanel: boolean;
  showFeatureModal: boolean;
  params: SearchParams;
}

const issueDescriptionRegexp = /^#(\d+)$/;

const route = useRoute();
const router = useRouter();
const currentUser = useCurrentUserV1();
const projectIamPolicyStore = useProjectIamPolicyStore();
const databaseStore = useDatabaseV1Store();
const projectStore = useProjectV1Store();
const instanceStore = useInstanceV1Store();
const { projectList } = useProjectV1ListByCurrentUser();
const state = reactive<LocalState>({
  exportRecords: [],
  showRequestExportPanel: false,
  showFeatureModal: false,
  params: {
    query: "",
    scopes: [],
  },
});
const hasDataAccessControlFeature = featureToRef("bb.feature.access-control");

const selectedProject = computed(() => {
  const project = state.params.scopes.find(
    (scope) => scope.id === "project"
  )?.value;
  if (!project) {
    return;
  }
  return projectStore.getProjectByName(`projects/${project}`);
});

const selectedInstance = computed(() => {
  const instance = state.params.scopes.find(
    (scope) => scope.id === "instance"
  )?.value;
  if (!instance) {
    return;
  }
  return instanceStore.getInstanceByName(`instances/${instance}`);
});

const selectedDatabase = computed(() => {
  const database = state.params.scopes.find(
    (scope) => scope.id === "database"
  )?.value;
  if (!database) {
    return;
  }
  const uid = database.split("-").slice(-1)[0];
  return databaseStore.getDatabaseByUID(uid);
});

const filterIssueId = computed(() => {
  const hash = route.hash.replace(/^#+/g, "");
  const maybeIssueId = parseInt(hash, 10);
  if (!Number.isNaN(maybeIssueId) && maybeIssueId > 0) {
    return String(maybeIssueId);
  }
  return String(UNKNOWN_ID);
});

const filterExportRecords = computed(() => {
  return state.exportRecords.filter((record) => {
    if (filterIssueId.value !== String(UNKNOWN_ID)) {
      if (record.issueId !== filterIssueId.value) {
        return false;
      }
    }

    if (selectedProject.value) {
      if (record.database.project !== selectedProject.value.name) {
        return false;
      }
    }
    if (selectedInstance.value) {
      if (record.database.instance !== selectedInstance.value.name) {
        return false;
      }
    }
    if (selectedDatabase.value) {
      if (record.database.name !== selectedDatabase.value.name) {
        return false;
      }
    }
    return true;
  });
});

watchEffect(async () => {
  const projectNameList = projectList.value.map((project) => project.name);
  const iamPolicyList =
    await projectIamPolicyStore.batchGetOrFetchProjectIamPolicy(
      projectNameList,
      true /* skipCache */
    );

  const tempExportRecords: ExportRecord[] = [];
  for (const iamPolicy of iamPolicyList) {
    const bindings = iamPolicy.bindings.filter(
      (binding) =>
        binding.role === PresetRoleType.PROJECT_EXPORTER &&
        binding.members.includes(`user:${currentUser.value.email}`)
    );
    for (const binding of bindings) {
      if (!binding.parsedExpr?.expr) {
        continue;
      }
      // Skip the expired export record.
      const expiredDateTime = getExpiredDateTime(binding);
      if (
        expiredDateTime &&
        new Date().getTime() >= expiredDateTime.getTime()
      ) {
        continue;
      }
      const conditionExpr = convertFromExpr(binding.parsedExpr.expr);
      // Only show the export record with statement condition in export center.
      if (!conditionExpr.statement || conditionExpr.statement === "") {
        continue;
      }
      if (
        !conditionExpr.databaseResources ||
        conditionExpr.databaseResources.length !== 1
      ) {
        continue;
      }

      const databaseResource = conditionExpr.databaseResources[0];
      const description = binding.condition?.description || "";
      // TODO: Here issueDescription looks like "#{uid}" so we don't have any parent
      // project info here.
      // To migrate this, we need to DML the description and re-write the issue
      // id extraction logic below.
      const issueId = description.match(issueDescriptionRegexp)?.[1];
      const database = await databaseStore.getOrFetchDatabaseByName(
        databaseResource.databaseName
      );
      let statement = conditionExpr.statement || "";
      // NOTE: concat schema and table name to statement for table level export.
      // Maybe we need to move this into backend later.
      if (statement === "" && databaseResource.table) {
        const names = [];
        if (databaseResource.schema) {
          names.push(databaseResource.schema);
        }
        names.push(databaseResource.table);
        statement = `SELECT * FROM ${names.join(".")};`;
      }

      tempExportRecords.push({
        databaseResource,
        database,
        statement,
        expiration: conditionExpr.expiredTime || "",
        maxRowCount: conditionExpr.rowLimit || 0,
        exportFormat: (conditionExpr.exportFormat as any) || "JSON",
        issueId: issueId || String(UNKNOWN_ID),
      });
    }
  }
  state.exportRecords = tempExportRecords;
});

const handleRequestExportClick = () => {
  if (!hasDataAccessControlFeature.value) {
    state.showFeatureModal = true;
    return;
  }
  state.showRequestExportPanel = true;
};

const clearFilterIssueId = () => {
  router.replace({
    ...route,
    hash: "",
  });
};

const supportOptionIdList = computed((): SearchScopeId[] => [
  "project",
  "instance",
  "database",
]);
</script>
