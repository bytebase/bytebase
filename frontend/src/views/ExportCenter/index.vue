<template>
  <div class="w-full px-4 py-1 pt-2">
    <div class="w-full flex flex-row justify-between items-center">
      <div>
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
      </div>
      <div>
        <NButton @click="createExportDataIssue">
          {{ $t("quick-action.request-export") }}
        </NButton>
      </div>
    </div>
    <div class="w-full mt-4">
      <ExportRecordTable :export-records="filterExportRecords" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { head } from "lodash-es";
import { NButton, NInputGroup } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { UNKNOWN_ID } from "@/types";
import { FilterParams, ExportRecord } from "./types";
import {
  useCurrentUserV1,
  useDatabaseV1Store,
  useInstanceV1Store,
  useProjectIamPolicyStore,
  useProjectV1ListByCurrentUser,
  useProjectV1Store,
} from "@/store";
import { ProjectSelect, InstanceSelect, DatabaseSelect } from "@/components/v2";
import { convertFromSimpleExpr } from "@/utils/issue/cel";
import ExportRecordTable from "./ExportRecordTable.vue";

interface LocalState {
  filterParams: FilterParams;
  exportRecords: ExportRecord[];
}

const issueDescriptionRegexp = /^#(\d+)$/;

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
});

const filterExportRecords = computed(() => {
  return state.exportRecords.filter((record) => {
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

      const conditionExpr = convertFromSimpleExpr(binding.parsedExpr.expr);
      const databaseResource = head(conditionExpr.databaseResources);
      if (databaseResource) {
        const description = binding.condition?.description || "";
        const issueId = description.match(issueDescriptionRegexp)?.[1];
        const database = await databaseStore.getOrFetchDatabaseByName(
          databaseResource.databaseName
        );
        tempExportRecords.push({
          database,
          expiration: conditionExpr.expiredTime || "",
          statement: conditionExpr.statement || "",
          maxRowCount: conditionExpr.rowLimit || 0,
          exportFormat: (conditionExpr.exportFormat as any) || "JSON",
          issueId: issueId || String(UNKNOWN_ID),
        });
      }
    }
  }
  state.exportRecords = tempExportRecords;
});

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

const createExportDataIssue = () => {
  const routeInfo = {
    name: "workspace.issue.detail",
    params: {
      issueSlug: "new",
    },
    query: {
      template: "bb.issue.grant.request",
      role: "EXPORTER",
      name: "New grant exporter request",
    },
  };
  router.push(routeInfo);
};
</script>
