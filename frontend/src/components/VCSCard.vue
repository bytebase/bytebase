<template>
  <div class="flex flex-row space-x-2">
    <template v-if="vcs.type.startsWith('GITLAB')">
      <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
    </template>
    <h3 class="text-lg leading-6 font-medium text-gray-900">
      {{ vcs.name }}
    </h3>
  </div>
  <div class="mt-5 border-t border-gray-200">
    <dl class="divide-y divide-gray-200">
      <div class="py-4 sm:py-5 sm:grid sm:grid-cols-3 sm:gap-4">
        <dt class="text-sm font-medium text-gray-500">Redirect URL</dt>
        <dd class="mt-1 flex text-sm text-gray-900 sm:mt-0 sm:col-span-2">
          <span class="flex-grow">{{ redirectURL(vcs) }}</span>
        </dd>
      </div>
      <div class="py-4 sm:grid sm:py-5 sm:grid-cols-3 sm:gap-4">
        <dt class="text-sm font-medium text-gray-500">Instance URL</dt>
        <dd class="mt-1 flex text-sm text-gray-900 sm:mt-0 sm:col-span-2">
          <span class="flex-grow">{{ vcs.instanceURL }}</span>
        </dd>
      </div>
      <div class="py-4 sm:grid sm:py-5 sm:grid-cols-3 sm:gap-4">
        <dt class="text-sm font-medium text-gray-500">API URL</dt>
        <dd class="mt-1 flex text-sm text-gray-900 sm:mt-0 sm:col-span-2">
          <span class="flex-grow">{{ vcs.apiURL }}</span>
        </dd>
      </div>
      <div class="py-4 sm:grid sm:py-5 sm:grid-cols-3 sm:gap-4">
        <dt class="text-sm font-medium text-gray-500">Application ID</dt>
        <dd class="mt-1 flex text-sm text-gray-900 sm:mt-0 sm:col-span-2">
          <span class="flex-grow">{{ vcs.applicationId }}</span>
        </dd>
      </div>
      <div class="py-4 sm:grid sm:py-5 sm:grid-cols-3 sm:gap-4">
        <dt class="text-sm font-medium text-gray-500">Secret</dt>
        <dd class="mt-1 flex text-sm text-gray-900 sm:mt-0 sm:col-span-2">
          <span class="flex-grow">{{ vcs.secret }}</span>
        </dd>
      </div>
      <div class="py-4 sm:grid sm:py-5 sm:grid-cols-3 sm:gap-4">
        <dt class="text-sm font-medium text-gray-500">OAuth Token</dt>
        <dd class="mt-1 flex text-sm text-gray-900 sm:mt-0 sm:col-span-2">
          <span class="flex-grow">{{ vcs.accessToken }}</span>
          <span v-if="!vcs.accessToken" class="ml-4 flex-shrink-0">
            <button
              type="button"
              class="
                bg-white
                rounded-md
                font-medium
                text-indigo-600
                hover:text-indigo-500
                focus:outline-none
                focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500
              "
              @click.prevent="linkVCS(vcs)"
            >
              <template v-if="vcs.type.startsWith('GITLAB')">
                Link this GitLab instance
              </template>
            </button>
          </span>
        </dd>
      </div>
    </dl>
  </div>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { VCS, redirectURL } from "../types";
import { randomString } from "../utils";

interface LocalState {}

export default {
  name: "VCSCard",
  components: {},
  props: {
    vcs: {
      required: true,
      type: Object as PropType<VCS>,
    },
  },
  setup(props, ctx) {
    const state = reactive<LocalState>({});

    const linkVCS = (vcs: VCS) => {
      const state = randomString(40);
      window.open(
        `${vcs.instanceURL}/oauth/authorize?client_id=${
          vcs.applicationId
        }&redirect_uri=${encodeURIComponent(
          redirectURL(vcs)
        )}&response_type=code&state=${state}`,
        "_blank"
      );
    };

    return {
      state,
      redirectURL,
      linkVCS,
    };
  },
};
</script>
