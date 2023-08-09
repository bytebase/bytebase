<template>
  <div class="w-full px-4 py-1 pt-2">
    <div class="w-full flex flex-row justify-between items-center">
      <div class="flex items-center gap-x-4">
        <NInputGroup>
          <ProjectSelect
            :project="state.filterParams.project?.uid ?? String(UNKNOWN_ID)"
            :include-all="true"
            @update:project="changeProjectId"
          />
          <InstanceSelect
            :instance="state.filterParams.instance?.uid ?? String(UNKNOWN_ID)"
            :include-all="true"
            @update:instance="changeInstanceId"
          />
          <DatabaseSelect
            :database="state.filterParams.database?.uid ?? String(UNKNOWN_ID)"
            :instance="state.filterParams.instance?.uid"
            :project="state.filterParams.project?.uid"
            :include-all="true"
            @update:database="changeDatabaseId"
          />
        </NInputGroup>
        <NButton
          v-if="filterIssueId !== String(UNKNOWN_ID)"
          @click="clearFilterIssueId"
        >
          <span>#{{ filterIssueId }}</span>
          <heroicons:x-mark class="w-4 h-4 ml-1 -mr-1.5" />
        </NButton>
      </div>
      <div>
        <NButton @click="handleRequestExportClick">
          <FeatureBadge
            feature="bb.feature.access-control"
            custom-class="mr-2"
          />
          {{ $t("quick-action.request-export") }}
        </NButton>
      </div>
    </div>
    <div class="w-full mt-4">
      <ExportRecordTable :export-records="filterExportRecords" />
    </div>
  </div>

  <RequestExportPanel
    v-if="state.showRequestExportPanel"
    @close="state.showRequestExportPanel = false"
  />

  <FeatureModal
    feature="bb.feature.access-control"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NButton, NInputGroup } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useRoute, useRouter } from "vue-router";
import RequestExportPanel from "@/components/Issue/panel/RequestExportPanel/index.vue";
import { ProjectSelect, InstanceSelect, DatabaseSelect } from "@/components/v2";
import {
  featureToRef,
  useCurrentUserV1,
  useDatabaseV1Store,
  useInstanceV1Store,
  useProjectIamPolicyStore,
  useProjectV1ListByCurrentUser,
  useProjectV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { convertFromExpr } from "@/utils/issue/cel";
import ExportRecordTable from "./ExportRecordTable.vue";
import { FilterParams, ExportRecord } from "./types";

interface LocalState {
  filterParams: FilterParams;
  exportRecords: ExportRecord[];
  showRequestExportPanel: boolean;
  showFeatureModal: boolean;
}

const issueDescriptionRegexp = /^#(\d+)$/;

const route = useRoute();
const router = useRouter();
const currentUser = useCurrentUserV1();
const projectIamPolicyStore = useProjectIamPolicyStore();
const databaseStore = useDatabaseV1Store();
const { projectList } = useProjectV1ListByCurrentUser();
const state = reactive<LocalState>({
  filterParams: {
    project: undefined,
    instance: undefined,
    database: undefined,
  },
  exportRecords: [],
  showRequestExportPanel: false,
  showFeatureModal: false,
});
const hasDataAccessControlFeature = featureToRef("bb.feature.access-control");

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

    if (state.filterParams.project) {
      if (record.database.project !== state.filterParams.project.name) {
        return false;
      }
    }
    if (state.filterParams.instance) {
      if (record.database.instance !== state.filterParams.instance.name) {
        return false;
      }
    }
    if (state.filterParams.database) {
      if (record.database.name !== state.filterParams.database.name) {
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
      projectNameList
    );

  const tempExportRecords: ExportRecord[] = [];
  for (const iamPolicy of iamPolicyList) {
    const bindings = iamPolicy.bindings.filter(
      (binding) =>
        binding.role === "roles/EXPORTER" &&
        binding.members.includes(`user:${currentUser.value.email}`)
    );
    for (const binding of bindings) {
      if (!binding.parsedExpr?.expr) {
        continue;
      }

      const conditionExpr = convertFromExpr(binding.parsedExpr.expr);
      const databaseResource = head(conditionExpr.databaseResources);
      if (databaseResource) {
        const description = binding.condition?.description || "";
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

const changeProjectId = (id: string | undefined) => {
  if (id && id !== String(UNKNOWN_ID)) {
    const project = useProjectV1Store().getProjectByUID(id);
    state.filterParams.project = project;
  } else {
    state.filterParams.project = undefined;
  }
};

const changeInstanceId = (uid: string | undefined) => {
  if (uid && uid !== String(UNKNOWN_ID)) {
    const instance = useInstanceV1Store().getInstanceByUID(uid);
    state.filterParams.instance = instance;
  } else {
    state.filterParams.instance = undefined;
  }
};

const changeDatabaseId = (uid: string | undefined) => {
  if (uid && uid !== String(UNKNOWN_ID)) {
    const database = useDatabaseV1Store().getDatabaseByUID(uid);
    state.filterParams.database = database;
  } else {
    state.filterParams.database = undefined;
  }
};

const clearFilterIssueId = () => {
  router.replace({
    ...route,
    hash: "",
  });
};
</script>
