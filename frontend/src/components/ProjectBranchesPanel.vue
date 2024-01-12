<template>
  <div class="space-y-6 w-full overflow-x-auto">
    <div class="flex flex-row justify-end items-center gap-x-2 pt-0.5">
      <SearchBox
        v-model:value="state.searchKeyword"
        :autofocus="true"
        :placeholder="$t('common.filter-by-name')"
      />
      <NButton type="primary" @click="handleCreateBranch">
        <PlusIcon class="w-4 h-auto mr-0.5" />
        <span>{{ $t("database.new-branch") }}</span>
      </NButton>
    </div>
    <div class="w-full space-y-2">
      <p class="textlabel">{{ $t("branch.default-branches") }}</p>
      <BranchDataTable :branches="defaultBranches" :ready="ready" />
    </div>
    <div class="w-full space-y-2">
      <p class="textlabel">{{ $t("branch.your-branches") }}</p>
      <BranchDataTable
        :branches="currentUserBranches"
        :ready="ready"
        :show-parent-branch-column="true"
      />
    </div>
    <div class="w-full space-y-2">
      <p class="textlabel">{{ $t("branch.active-branches") }}</p>
      <BranchDataTable
        :branches="activeBranches"
        :ready="ready"
        :show-parent-branch-column="true"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { orderBy } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import BranchDataTable from "@/components/Branch/BranchDataTable/index.vue";
import { PROJECT_V1_BRANCHE_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentUserV1 } from "@/store";
import { useBranchListByProject } from "@/store/modules/branch";
import { userNamePrefix } from "@/store/modules/v1/common";
import { ComposedProject } from "@/types";

const props = defineProps<{
  project: ComposedProject;
}>();

interface LocalState {
  searchKeyword: string;
}

// BRANCH_DEFAULT_ACTIVE_DURATION is the duration that a branch is considered as active.
// Currently, it is set to 2 months.
const BRANCH_DEFAULT_ACTIVE_DURATION = 1000 * 60 * 60 * 24 * 30 * 2;

const router = useRouter();
const currentUser = useCurrentUserV1();
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

// defaultBranches are those branches that are parent branches.
const defaultBranches = computed(() => {
  return filteredBranches.value.filter((branch) => {
    return branch.parentBranch === "";
  });
});

// currentUserBranches are those branches that are created by the current user and are not parent branches.
const currentUserBranches = computed(() => {
  return filteredBranches.value.filter((branch) => {
    return (
      branch.parentBranch !== "" &&
      branch.creator === `${userNamePrefix}${currentUser.value.email}`
    );
  });
});

// activeBranches are those branches that are active in the last 2 months.
const activeBranches = computed(() => {
  return filteredBranches.value.filter((branch) => {
    return (
      branch.updateTime!.getTime() > Date.now() - BRANCH_DEFAULT_ACTIVE_DURATION
    );
  });
});

const handleCreateBranch = () => {
  router.push({
    name: PROJECT_V1_BRANCHE_DETAIL,
    params: {
      branchName: "new",
    },
  });
};
</script>
