<template>
  <div
    class="divide-y divide-block-border border border-block-border rounded-sm"
  >
    <div class="flex py-2 px-4 justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <!-- <template v-if="projectHook.type.startsWith('GITLAB')">
          <img class="h-6 w-auto" src="../assets/gitlab-logo.svg" />
        </template> -->
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ projectHook.name }}
        </h3>
      </div>
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="viewProjectHook"
      >
        View
      </button>
    </div>
    <div class="border-t border-block-border">
      <dl class="divide-y divide-block-border">
        <div class="grid grid-cols-5 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">URL</dt>
          <dd class="py-0.5 flex text-sm text-main col-span-4">
            {{ projectHook.url }}
          </dd>
        </div>
        <div class="grid grid-cols-5 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">Events</dt>
          <dd class="py-0.5 flex text-sm text-main col-span-4">
            {{ projectHook.activityList.join(", ") }}
          </dd>
        </div>
        <div class="grid grid-cols-5 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">Created by</dt>
          <dd class="py-0.5 flex items-center text-sm text-main col-span-4">
            <div class="flex flex-row items-center space-x-2 mr-1">
              <div class="flex flex-row items-center space-x-1">
                <PrincipalAvatar
                  :principal="projectHook.creator"
                  :size="'SMALL'"
                />
                <router-link
                  :to="`/u/${projectHook.creator.id}`"
                  class="normal-link"
                  >{{ projectHook.creator.name }}
                </router-link>
              </div>
            </div>
            at {{ humanizeTs(projectHook.createdTs) }}
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, PropType } from "vue";
import { useRouter } from "vue-router";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import { ProjectHook, redirectURL } from "../types";
import { projectHookSlug } from "../utils";

interface LocalState {}

export default {
  name: "ProjectWebhookCard",
  components: { PrincipalAvatar },
  props: {
    projectHook: {
      required: true,
      type: Object as PropType<ProjectHook>,
    },
  },
  setup(props, ctx) {
    const router = useRouter();

    const state = reactive<LocalState>({});

    const viewProjectHook = () => {
      router.push({
        name: "workspace.project.hook.detail",
        params: {
          projectHookSlug: projectHookSlug(props.projectHook),
        },
      });
    };

    return {
      state,
      redirectURL,
      viewProjectHook,
    };
  },
};
</script>
