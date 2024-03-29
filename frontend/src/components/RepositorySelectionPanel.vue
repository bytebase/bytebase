<template>
  <BBAttention type="warning" :description="attentionText" />
  <div class="mt-4 space-y-2">
    <div class="flex justify-between items-center">
      <button class="btn-icon" @click.prevent="refreshRepositoryList">
        <heroicons-outline:refresh class="w-6 h-6" />
      </button>
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
          <div class="flex items-center px-4 py-3">
            <div class="min-w-0 flex-1 flex items-center">
              {{ repository.fullpath }}
            </div>
            <div>
              <!-- Heroicon name: solid/chevron-right -->
              <heroicons-solid:chevron-right class="h-5 w-5 text-control" />
            </div>
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts">
export default { name: "RepositorySelectionPanel" };
</script>

<script setup lang="ts">
import { reactive, computed, onMounted } from "vue";
import type { ExternalRepositoryInfo, ProjectRepositoryConfig } from "@/types";
import { pushNotification, useCurrentUserV1, useVCSV1Store } from "@/store";
import type { SearchVCSProviderProjectsResponse_Project } from "@/types/proto/v1/vcs_provider_service";
import { VCSProvider_Type } from "@/types/proto/v1/vcs_provider_service";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  repositoryList: SearchVCSProviderProjectsResponse_Project[];
  searchText: string;
}

const props = defineProps<{ config: ProjectRepositoryConfig }>();

const emit = defineEmits<{
  (event: "next"): void;
  (event: "set-repository", payload: ExternalRepositoryInfo): void;
}>();

const currentUser = useCurrentUserV1();
const vcsV1Store = useVCSV1Store();
const state = reactive<LocalState>({
  repositoryList: [],
  searchText: "",
});

onMounted(() => {
  prepareRepositoryList();
});

const prepareRepositoryList = () => {
  refreshRepositoryList();
};

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

  const projects = await vcsV1Store.listVCSExternalProjects(
    props.config.vcs.name
  );
  state.repositoryList = projects;
};

const repositoryList = computed(() => {
  if (state.searchText == "") {
    return state.repositoryList;
  }
  return state.repositoryList.filter(
    (repository: SearchVCSProviderProjectsResponse_Project) => {
      return repository.fullpath.toLowerCase().includes(state.searchText);
    }
  );
});

const attentionText = computed((): string => {
  if (props.config.vcs.type === VCSProvider_Type.GITLAB) {
    return "repository.select-repository-attention-gitlab";
  } else if (props.config.vcs.type === VCSProvider_Type.GITHUB) {
    return "repository.select-repository-attention-github";
  } else if (props.config.vcs.type === VCSProvider_Type.BITBUCKET) {
    return "repository.select-repository-attention-bitbucket";
  } else if (props.config.vcs.type === VCSProvider_Type.AZURE_DEVOPS) {
    return "repository.select-repository-attention-azure-devops";
  }
  return "";
});

const selectRepository = (
  repository: SearchVCSProviderProjectsResponse_Project
) => {
  emit("set-repository", {
    externalId: repository.id,
    name: repository.title,
    fullPath: repository.fullpath,
    webUrl: repository.webUrl,
  });
  emit("next");
};
</script>
