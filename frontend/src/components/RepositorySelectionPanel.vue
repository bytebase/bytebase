<template>
  <BBAttention type="warning" :description="attentionText" />
  <div class="mt-4 space-y-2">
    <div class="flex justify-between items-center">
      <BBSpin v-if="state.loading" :size="20" />
      <NButton
        v-else
        quaternary
        circle
        :loading="state.loading"
        @click.prevent="refreshRepositoryList"
      >
        <RefreshCwIcon class="w-5 h-5" />
      </NButton>
      <SearchBox
        v-model:value="state.searchText"
        :placeholder="$t('repository.select-repository-search')"
      />
    </div>
    <div
      class="bg-white overflow-hidden rounded-sm border border-control-border"
    >
      <ul class="divide-y divide-control-border">
        <li
          v-for="(repository, index) in repositoryList"
          :key="index"
          class="block hover:bg-control-bg-hover cursor-pointer"
          @click.prevent="selectRepository(repository)"
        >
          <div class="flex items-center px-3 py-2">
            <div class="min-w-0 flex-1 flex items-center">
              {{ repository.fullPath }}
            </div>
            <ChevronRightIcon class="h-5 w-5 text-control" />
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { RefreshCwIcon, ChevronRightIcon } from "lucide-vue-next";
import { reactive, computed, onMounted } from "vue";
import BBSpin from "@/bbkit/BBSpin.vue";
import {
  pushNotification,
  useCurrentUserV1,
  useVCSProviderStore,
} from "@/store";
import type { ProjectRepositoryConfig } from "@/types";
import { VCSType } from "@/types/proto/v1/common";
import type { VCSRepository } from "@/types/proto/v1/vcs_provider_service";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  repositoryList: VCSRepository[];
  searchText: string;
  loading: boolean;
}

const props = defineProps<{ config: ProjectRepositoryConfig }>();

const emit = defineEmits<{
  (event: "next"): void;
  (event: "set-repository", payload: VCSRepository): void;
}>();

const currentUser = useCurrentUserV1();
const vcsV1Store = useVCSProviderStore();
const state = reactive<LocalState>({
  repositoryList: [],
  searchText: "",
  loading: true,
});

onMounted(() => {
  refreshRepositoryList();
});

const refreshRepositoryList = async () => {
  if (
    !hasWorkspacePermissionV2(
      currentUser.value,
      "bb.vcsProviders.searchProjects"
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Permission denied for searching projects",
    });
    return;
  }

  state.loading = true;
  const projects = await vcsV1Store.searchVCSProviderRepositories(
    props.config.vcs.name
  );
  state.repositoryList = projects;
  state.loading = false;
};

const repositoryList = computed(() => {
  if (state.searchText == "") {
    return state.repositoryList;
  }
  return state.repositoryList.filter((repository: VCSRepository) => {
    return repository.fullPath.toLowerCase().includes(state.searchText);
  });
});

const attentionText = computed((): string => {
  if (props.config.vcs.type === VCSType.GITLAB) {
    return "repository.select-repository-attention-gitlab";
  } else if (props.config.vcs.type === VCSType.GITHUB) {
    return "repository.select-repository-attention-github";
  } else if (props.config.vcs.type === VCSType.BITBUCKET) {
    return "repository.select-repository-attention-bitbucket";
  } else if (props.config.vcs.type === VCSType.AZURE_DEVOPS) {
    return "repository.select-repository-attention-azure-devops";
  }
  return "";
});

const selectRepository = (repository: VCSRepository) => {
  emit("set-repository", repository);
  emit("next");
};
</script>
