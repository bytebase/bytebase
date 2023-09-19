<template>
  <BBAttention :style="'WARN'" :description="attentionText" />
  <div class="mt-4 space-y-2">
    <div class="flex justify-between items-center">
      <button class="btn-icon" @click.prevent="refreshRepositoryList">
        <heroicons-outline:refresh class="w-6 h-6" />
      </button>
      <BBTableSearch
        :placeholder="$t('repository.select-repository-search')"
        @change-text="(text: string) => changeSearchText(text)"
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
import { ExternalRepositoryInfo, ProjectRepositoryConfig } from "../types";
import { useVCSV1Store } from "@/store";
import {
  OAuthToken,
  ExternalVersionControl_Type,
  SearchExternalVersionControlProjectsResponse_Project,
} from "@/types/proto/v1/externalvs_service";

interface LocalState {
  repositoryList: SearchExternalVersionControlProjectsResponse_Project[];
  searchText: string;
}

const props = defineProps<{ config: ProjectRepositoryConfig }>();

const emit = defineEmits<{
  (event: "next"): void;
  (event: "set-token", payload: OAuthToken): void;
  (event: "set-repository", payload: ExternalRepositoryInfo): void;
}>();

const vcsV1Store = useVCSV1Store();
const state = reactive<LocalState>({
  repositoryList: [],
  searchText: "",
});

onMounted(() => {
  prepareRepositoryList();
});

const prepareRepositoryList = () => {
  vcsV1Store
    .exchangeToken({
      vcsName: props.config.vcs.name,
      code: props.config.code,
    })
    .then((token: OAuthToken) => {
      emit("set-token", token);
      vcsV1Store
        .listVCSExternalProjects(
          props.config.vcs.name,
          token.accessToken,
          token.refreshToken
        )
        .then((list) => {
          state.repositoryList = list;
        });
    });
};

const refreshRepositoryList = () => {
  vcsV1Store
    .listVCSExternalProjects(
      props.config.vcs.name,
      props.config.token.accessToken,
      props.config.token.refreshToken
    )
    .then((list) => {
      state.repositoryList = list;
    });
};

const repositoryList = computed(() => {
  if (state.searchText == "") {
    return state.repositoryList;
  }
  return state.repositoryList.filter(
    (repository: SearchExternalVersionControlProjectsResponse_Project) => {
      return repository.fullpath.toLowerCase().includes(state.searchText);
    }
  );
});

const attentionText = computed((): string => {
  if (props.config.vcs.type === ExternalVersionControl_Type.GITLAB) {
    return "repository.select-repository-attention-gitlab";
  } else if (props.config.vcs.type === ExternalVersionControl_Type.GITHUB) {
    return "repository.select-repository-attention-github";
  } else if (props.config.vcs.type === ExternalVersionControl_Type.BITBUCKET) {
    return "repository.select-repository-attention-bitbucket";
  } else if (
    props.config.vcs.type === ExternalVersionControl_Type.AZURE_DEVOPS
  ) {
    return "repository.select-repository-attention-azure-devops";
  }
  return "";
});

const selectRepository = (
  repository: SearchExternalVersionControlProjectsResponse_Project
) => {
  emit("set-repository", {
    externalId: repository.id,
    name: repository.title,
    fullPath: repository.fullpath,
    webUrl: repository.webUrl,
  });
  emit("next");
};

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};
</script>
