<template>
  <div
    class="divide-y divide-block-border border border-block-border rounded-sm"
  >
    <div class="flex py-2 px-4 justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <img class="h-5 w-5" :src="`/src/assets/${logo}`" />
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
            {{ activityListStr }}
          </dd>
        </div>
        <div class="grid grid-cols-5 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            Last updated by
          </dt>
          <dd class="py-0.5 flex items-center text-sm text-main col-span-4">
            <div class="flex flex-row items-center space-x-2 mr-1">
              <div class="flex flex-row items-center space-x-1">
                <PrincipalAvatar
                  :principal="projectHook.updater"
                  :size="'SMALL'"
                />
                <router-link
                  :to="`/u/${projectHook.updater.id}`"
                  class="normal-link"
                  >{{ projectHook.updater.name }}
                </router-link>
              </div>
            </div>
            at {{ humanizeTs(projectHook.updatedTs) }}
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, PropType, computed } from "vue";
import { useRouter } from "vue-router";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  ProjectHook,
  PROJECT_HOOK_TYPE_ITEM_LIST,
  PROJECT_HOOK_EVENT_ITEM_LIST,
  redirectURL,
} from "../types";
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

    const logo = computed(() => {
      for (const item of PROJECT_HOOK_TYPE_ITEM_LIST) {
        if (item.type == props.projectHook.type) {
          return item.logo;
        }
      }

      return "";
    });

    const activityListStr = computed(() => {
      const list = props.projectHook.activityList.map((activity) => {
        for (const item of PROJECT_HOOK_EVENT_ITEM_LIST) {
          if (item.event == activity) {
            return item.title;
          }
        }
        return activity;
      });

      return list.join(", ");
    });

    return {
      state,
      redirectURL,
      viewProjectHook,
      logo,
      activityListStr,
    };
  },
};
</script>
