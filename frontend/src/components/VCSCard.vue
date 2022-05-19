<template>
  <div
    class="divide-y divide-block-border border border-block-border rounded-sm"
  >
    <div class="flex py-2 px-4 justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <template v-if="vcs.type.startsWith('GITLAB')">
          <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
        </template>
        <template v-if="vcs.type.startsWith('GITHUB')">
          <img class="h-6 w-auto" src="../assets/github-logo.svg" />
        </template>
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ vcs.name }}
        </h3>
      </div>
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="editVCS"
      >
        {{ $t("common.edit") }}
      </button>
    </div>
    <div class="border-t border-block-border">
      <dl class="divide-y divide-block-border">
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.instance") }} URL
          </dt>
          <dd class="mt-1 flex text-sm text-main col-span-2">
            {{ vcs.instanceUrl }}
          </dd>
        </div>
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.application") }} ID
          </dt>
          <dd class="mt-1 flex text-sm text-main col-span-2">
            {{ vcs.applicationId }}
          </dd>
        </div>
        <div class="grid grid-cols-4 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("common.created-at") }}
          </dt>
          <dd class="mt-1 flex text-sm text-main col-span-2">
            {{ humanizeTs(vcs.createdTs) }}
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, PropType, defineComponent } from "vue";
import { useRouter } from "vue-router";
import { VCS, redirectUrl } from "../types";
import { vcsSlug } from "../utils";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "VCSCard",
  components: {},
  props: {
    vcs: {
      required: true,
      type: Object as PropType<VCS>,
    },
  },
  setup(props) {
    const router = useRouter();

    const state = reactive<LocalState>({});

    const editVCS = () => {
      router.push({
        name: "setting.workspace.version-control.detail",
        params: {
          vcsSlug: vcsSlug(props.vcs),
        },
      });
    };

    return {
      state,
      redirectUrl,
      editVCS,
    };
  },
});
</script>
