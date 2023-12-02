<template>
  <div class="space-y-4 w-full overflow-x-auto">
    <div class="flex flex-row justify-end items-center gap-x-2">
      <SearchBox
        v-model:value="state.searchKeyword"
        :autofocus="true"
        :placeholder="$t('common.filter-by-name')"
      />
      <NButton type="primary" @click="handleCreateBranch">
        <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
        <span>{{ $t("database.new-branch") }}</span>
      </NButton>
    </div>

    <BranchDataTable
      class="border"
      :branches="filteredBranches"
      :hide-project-column="true"
      :ready="ready"
      @click="handleBranchClick"
    />
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import BranchDataTable from "@/components/Branch/BranchDataTable.vue";
import { useProjectV1Store } from "@/store";
import { useBranchList } from "@/store/modules/branch";
import { getProjectAndBranchId } from "@/store/modules/v1/common";
import { Branch } from "@/types/proto/v1/branch_service";
import { projectV1Slug } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

interface LocalState {
  searchKeyword: string;
}

const router = useRouter();
const projectV1Store = useProjectV1Store();
const { branchList, ready } = useBranchList(props.projectId);
const state = reactive<LocalState>({
  searchKeyword: "",
});

const project = computed(() => projectV1Store.getProjectByUID(props.projectId));

const filteredBranches = computed(() => {
  return orderBy(
    props.projectId
      ? branchList.value.filter((branch) =>
          branch.name.startsWith(project.value.name)
        )
      : branchList.value,
    "updateTime",
    "desc"
  ).filter((branch) => {
    return state.searchKeyword
      ? branch.title.includes(state.searchKeyword)
      : true;
  });
});

const handleCreateBranch = () => {
  router.push({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: projectV1Slug(project.value),
      branchName: "new",
    },
  });
};

const handleBranchClick = async (schemaDesign: Branch) => {
  const [_, branchId] = getProjectAndBranchId(schemaDesign.name);
  router.push({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: projectV1Slug(project.value),
      branchName: `${branchId}`,
    },
  });
};
</script>
