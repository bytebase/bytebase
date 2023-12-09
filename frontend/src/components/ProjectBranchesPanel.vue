<template>
  <div class="space-y-4 w-full overflow-x-auto">
    <div class="flex flex-row justify-end items-center gap-x-2 pt-0.5">
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
import { useBranchListByProject } from "@/store/modules/branch";
import { getProjectAndBranchId } from "@/store/modules/v1/common";
import { ComposedProject } from "@/types";
import { Branch } from "@/types/proto/v1/branch_service";
import { projectV1Slug } from "@/utils";

const props = defineProps<{
  project: ComposedProject;
}>();

interface LocalState {
  searchKeyword: string;
}

const router = useRouter();
const { branchList, ready } = useBranchListByProject(
  computed(() => props.project.name)
);
const state = reactive<LocalState>({
  searchKeyword: "",
});

const filteredBranches = computed(() => {
  return orderBy(branchList.value, "updateTime", "desc").filter((branch) => {
    return state.searchKeyword
      ? branch.branchId.includes(state.searchKeyword)
      : true;
  });
});

const handleCreateBranch = () => {
  router.push({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: projectV1Slug(props.project),
      branchName: "new",
    },
  });
};

const handleBranchClick = async (branch: Branch) => {
  const [_, branchId] = getProjectAndBranchId(branch.name);
  router.push({
    name: "workspace.project.branch.detail",
    params: {
      projectSlug: projectV1Slug(props.project),
      branchName: `${branchId}`,
    },
  });
};
</script>
