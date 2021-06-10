<template>
  <div class="space-y-2 divide-y divide-block-border">
    <div class="flex justify-end items-center">
      <BBTableSearch
        :placeholder="'Search repository'"
        @change-text="(text) => changeSearchText(text)"
      />
    </div>
    <div class="bg-white shadow overflow-hidden sm:rounded-md">
      <ul class="divide-y divide-control-border">
        <li
          v-for="(repository, index) in repositoryList"
          :key="index"
          class="block hover:bg-control-bg-hover cursor-pointer"
          @click.prevent="selectRepository(repository)"
        >
          <div class="flex items-center px-4 py-4 sm:px-6">
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
import { reactive } from "@vue/reactivity";
import { useStore } from "vuex";
import { computed, PropType, watchEffect } from "@vue/runtime-core";
import { ProjectRepoConfig, Repository } from "../types";

interface LocalState {
  repositoryList: Repository[];
  searchText: string;
}

export default {
  name: "RepositorySelectionPanel",
  emits: ["next"],
  props: {
    config: {
      required: true,
      type: Object as PropType<ProjectRepoConfig>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      repositoryList: [],
      searchText: "",
    });

    const prepareRepositoryList = () => {
      store
        .dispatch("gitlab/fetchProjectList", {
          vcs: props.config.vcs,
          token: props.config.accessToken,
        })
        .then((list) => {
          state.repositoryList = list;
        });
    };

    watchEffect(prepareRepositoryList);

    const repositoryList = computed(() => {
      if (state.searchText == "") {
        return state.repositoryList;
      }
      return state.repositoryList.filter((repository: Repository) => {
        return repository.fullPath.toLowerCase().includes(state.searchText);
      });
    });

    const selectRepository = (repository: Repository) => {
      props.config.repository = repository;
      emit("next");
    };

    const changeSearchText = (searchText: string) => {
      state.searchText = searchText;
    };

    return {
      state,
      selectRepository,
      repositoryList,
      changeSearchText,
    };
  },
};
</script>
