<template>
  <div class="space-y-4">
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="gitprovider" class="textlabel"> Git provider </label>
        <template v-if="vcsType.startsWith('GITLAB')">
          <img class="h-4 w-auto" src="../assets/gitlab-logo.svg" />
        </template>
      </div>
      <input
        id="gitprovider"
        name="gitprovider"
        type="text"
        class="textfield mt-1 w-full"
        disabled="true"
        :value="vcsName"
      />
    </div>
    <div>
      <div class="flex flex-row space-x-2 items-center">
        <label for="repository" class="textlabel"> Repository </label>
        <button
          class="ml-1 btn-icon"
          @click.prevent="window.open(urlfy(repositoryInfo.webURL), '_blank')"
        >
          <svg
            class="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
            ></path>
          </svg>
        </button>
      </div>
      <input
        id="repository"
        name="repository"
        type="text"
        class="textfield mt-1 w-full"
        disabled="true"
        :value="repositoryInfo.fullPath"
      />
    </div>
    <div>
      <div class="textlabel">Base directory</div>
      <div class="mt-1 textinfolabel">
        The directory and all its sub-directories that Bytebase will observe the
        SQL file (.sql) change. If empty, then Bytebase will observe the entire
        repository.
      </div>
      <input
        id="basedirectory"
        name="basedirectory"
        type="text"
        class="textfield mt-2 w-full"
        v-model="repositoryConfig.baseDirectory"
      />
    </div>
    <div>
      <div class="textlabel">Branch</div>
      <div class="mt-1 textinfolabel">
        The branch where Bytebase will observe the SQL file (.sql) change. If
        empty, then Bytebase will observe the default branch (normally it's the
        'master' or 'main' branch).
      </div>
      <input
        id="branch"
        name="branch"
        type="text"
        class="textfield mt-2 w-full"
        placeholder="'Default branch (normally \'master\' or \'main\')'"
        v-model="repositoryConfig.branchFilter"
      />
      <div v-if="vcsType == 'GITLAB_SELF_HOST'" class="mt-2 textinfolabel">
        Tip: You can also use wildcard like 'feature/*'
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive } from "@vue/reactivity";
import { PropType } from "@vue/runtime-core";
import { ExternalRepositoryInfo, RepositoryConfig, VCSType } from "../types";

interface LocalState {}

export default {
  name: "RepositoryForm",
  props: {
    vcsType: {
      required: true,
      type: Object as PropType<VCSType>,
    },
    vcsName: {
      required: true,
      type: Object as PropType<String>,
    },
    repositoryInfo: {
      required: true,
      type: Object as PropType<ExternalRepositoryInfo>,
    },
    repositoryConfig: {
      required: true,
      type: Object as PropType<RepositoryConfig>,
    },
  },
  components: {},
  setup(props, { emit }) {
    const state = reactive<LocalState>({});

    return {
      state,
    };
  },
};
</script>
