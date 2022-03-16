<template>
  <BBAttention :style="'WARN'" :description="attentionText" />
  <div class="mt-4 space-y-2">
    <div class="flex justify-between items-center">
      <button class="btn-icon" @click.prevent="prepareRepositoryList">
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
import { useStore } from "vuex";
import { reactive, computed, PropType, watchEffect } from "vue";
import { ExternalRepositoryInfo, ProjectRepositoryConfig } from "../types";

interface LocalState {
  repositoryList: ExternalRepositoryInfo[];
  searchText: string;
}

export default {
  name: "RepositorySelectionPanel",
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepositoryConfig>,
    },
  },
  emits: ["next"],
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      repositoryList: [],
      searchText: "",
    });

    const prepareRepositoryList = () => {
      if (props.config.vcs.type == "GITLAB_SELF_HOST") {
        store
          .dispatch("gitlab/fetchProjectList", {
            vcs: props.config.vcs,
            token: props.config.token.accessToken,
          })
          .then((list) => {
            state.repositoryList = list;
          });
      } else if (props.config.vcs.type == "GITHUB_DOT_COM") {
        store
          .dispatch("github/fetchProjectList", {
            vcs: props.config.vcs,
            token: props.config.token.accessToken,
          })
          .then((list) => {
            state.repositoryList = list;
          });
      }
    };

    watchEffect(prepareRepositoryList);

    const repositoryList = computed(() => {
      if (state.searchText == "") {
        return state.repositoryList;
      }
      return state.repositoryList.filter(
        (repository: ExternalRepositoryInfo) => {
          return repository.fullPath.toLowerCase().includes(state.searchText);
        }
      );
    });

    const attentionText = computed((): string => {
      if (props.config.vcs.type == "GITLAB_SELF_HOST") {
        return "repository.select-repository-attention-gitlab";
      }
      return "";
    });

    const selectRepository = (repository: ExternalRepositoryInfo) => {
      props.config.repositoryInfo = repository;
      emit("next");
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      state,
      selectRepository,
      repositoryList,
      attentionText,
      changeSearchText,
      prepareRepositoryList,
    };
  },
};
</script>
