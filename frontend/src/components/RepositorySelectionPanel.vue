<template>
  <BBAttention :style="'WARN'" :description="attentionText" />
  <div class="mt-4 space-y-2">
    <div class="flex justify-between items-center">
      <button class="btn-icon" @click.prevent="refreshRepositoryList">
        <heroicons-outline:refresh class="w-6 h-6" />
      </button>
      <BBTableSearch
        :placeholder="$t('repository.select-repository-search')"
        @change-text="(text) => changeSearchText(text)"
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
              {{ repository.fullPath }}
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
import { useStore } from "vuex";
import { reactive, computed, onMounted } from "vue";
import {
  ExternalRepositoryInfo,
  OAuthToken,
  ProjectRepositoryConfig,
} from "../types";

interface LocalState {
  repositoryList: ExternalRepositoryInfo[];
  searchText: string;
}

const props = defineProps<{ config: ProjectRepositoryConfig }>();

const emit = defineEmits<{
  (event: "next"): void;
  (event: "set-token", payload: OAuthToken): void;
  (event: "set-repository", payload: ExternalRepositoryInfo): void;
}>();

const store = useStore();
const state = reactive<LocalState>({
  repositoryList: [],
  searchText: "",
});

onMounted(() => {
  prepareRepositoryList();
});

const prepareRepositoryList = () => {
  if (props.config.vcs.type == "GITLAB_SELF_HOST") {
    store
      .dispatch("oauth/exchangeVCSToken", {
        code: props.config.code,
        vcsId: props.config.vcs.id,
      })
      .then((token: OAuthToken) => {
        emit("set-token", token);
        store
          .dispatch("gitlab/fetchProjectList", {
            vcs: props.config.vcs,
            token: token,
          })
          .then((list) => {
            state.repositoryList = list;
          });
      });
  }
};

const refreshRepositoryList = () => {
  if (props.config.vcs.type == "GITLAB_SELF_HOST") {
    store
      .dispatch("gitlab/fetchProjectList", {
        vcs: props.config.vcs,
        token: props.config.token,
      })
      .then((list) => {
        state.repositoryList = list;
      });
  }
};

const repositoryList = computed(() => {
  if (state.searchText == "") {
    return state.repositoryList;
  }
  return state.repositoryList.filter((repository: ExternalRepositoryInfo) => {
    return repository.fullPath.toLowerCase().includes(state.searchText);
  });
});

const attentionText = computed((): string => {
  if (props.config.vcs.type == "GITLAB_SELF_HOST") {
    return "repository.select-repository-attention-gitlab";
  }
  return "";
});

const selectRepository = (repository: ExternalRepositoryInfo) => {
  emit("set-repository", repository);
  emit("next");
};

const changeSearchText = (searchText: string) => {
  state.searchText = searchText;
};
</script>
