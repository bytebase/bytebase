<template>
  <BBAttention :style="'WARN'" :description="attentionText" />
  <div class="mt-4 space-y-2">
    <div class="flex justify-between items-center">
      <button class="btn-icon" @click.prevent="prepareRepositoryList">
        <svg
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          ></path>
        </svg>
      </button>
      <BBTableSearch
        :placeholder="'Search repository'"
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
              <svg
                class="h-5 w-5 text-control"
                xmlns="http://www.w3.org/2000/svg"
                viewBox="0 0 20 20"
                fill="currentColor"
                aria-hidden="true"
              >
                <path
                  fill-rule="evenodd"
                  d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
                  clip-rule="evenodd"
                />
              </svg>
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
        return "Bytebase only lists GitLab projects granting you at least the 'Maintainer' role, which allows to configure the project webhook to observe the code push event.";
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
