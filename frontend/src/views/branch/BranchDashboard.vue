<template>
  <div class="space-y-3 pt-2 w-full overflow-x-auto">
    <div class="w-full px-4 flex flex-row justify-between items-center">
      <div>
        <ProjectSelect
          v-model:project="state.projectFilter"
          :include-all="true"
        />
      </div>
      <div class="flex flex-row justify-end items-center gap-x-2">
        <NInput
          v-model:value="state.searchKeyword"
          class="!w-36"
          clearable
          :placeholder="$t('common.filter-by-name')"
        />
        <NButton type="primary" @click="handleCreateBranch">
          <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
          <span>{{ $t("database.new-branch") }}</span>
        </NButton>
      </div>
    </div>

    <BranchDataTable
      :branches="filteredBranches"
      :ready="ready"
      @click="handleBranchClick"
    />
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import BranchDataTable from "@/components/Branch/BranchDataTable.vue";
import { ProjectSelect } from "@/components/v2";
import { useProjectV1Store, useDatabaseV1Store } from "@/store";
import { useSchemaDesignList } from "@/store/modules/schemaDesign";
import { getProjectAndSchemaDesignSheetId } from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "@/types";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { projectV1Slug } from "@/utils";

interface LocalState {
  searchKeyword: string;
  projectFilter: string;
}

const router = useRouter();
const { schemaDesignList, ready } = useSchemaDesignList();
const projectStore = useProjectV1Store();
const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  searchKeyword: "",
  projectFilter: String(UNKNOWN_ID),
});

const filteredBranches = computed(() => {
  return orderBy(schemaDesignList.value, "updateTime", "desc")
    .filter((branch) => {
      const [projectName] = getProjectAndSchemaDesignSheetId(branch.name);
      const project = projectStore.getProjectByName(`projects/${projectName}`);
      return (
        !state.projectFilter ||
        state.projectFilter === String(UNKNOWN_ID) ||
        project.uid === state.projectFilter
      );
    })
    .filter((branch) => {
      return state.searchKeyword
        ? branch.title.includes(state.searchKeyword)
        : true;
    });
});

const handleCreateBranch = () => {
  router.push({
    name: "workspace.branch.detail",
    params: {
      projectSlug: "-",
      branchName: "new",
    },
  });
};

const handleBranchClick = async (schemaDesign: SchemaDesign) => {
  const [_, sheetId] = getProjectAndSchemaDesignSheetId(schemaDesign.name);
  const baselineDatabase = databaseStore.getDatabaseByName(
    schemaDesign.baselineDatabase
  );
  console.log("handleBranchClick");
  router.push({
    name: "workspace.branch.detail",
    params: {
      projectSlug: projectV1Slug(baselineDatabase.projectEntity),
      branchName: sheetId,
    },
  });
};
</script>
