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
        <div
          v-if="!create"
          class="ml-1 normal-link text-sm"
          @click.prevent="$emit('change-repository')"
        >
          Change
        </div>
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
        :disabled="!allowEdit"
        v-model="repositoryConfig.baseDirectory"
      />
    </div>
    <div>
      <div class="textlabel">Branch <span class="text-red-600">*</span></div>
      <div class="mt-1 textinfolabel">
        The branch where Bytebase observes the SQL file (.sql) change.
      </div>
      <input
        id="branch"
        name="branch"
        type="text"
        class="textfield mt-2 w-full"
        placeholder="e.g. master"
        :disabled="!allowEdit"
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
  emits: ["change-repository"],
  props: {
    allowEdit: {
      default: true,
      type: Boolean,
    },
    create: {
      type: Boolean,
      default: false,
    },
    vcsType: {
      required: true,
      type: String as PropType<VCSType>,
    },
    vcsName: {
      required: true,
      type: String,
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
