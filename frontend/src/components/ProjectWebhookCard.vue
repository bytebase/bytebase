<template>
  <div
    class="divide-y divide-block-border border border-block-border rounded-sm"
  >
    <div class="flex py-2 px-4 justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
        <template v-if="projectWebhook.type == 'bb.plugin.webhook.slack'">
          <img class="h-5 w-5" src="../assets/slack-logo.png" />
        </template>
        <template
          v-else-if="projectWebhook.type == 'bb.plugin.webhook.discord'"
        >
          <img class="h-5 w-5" src="../assets/discord-logo.svg" />
        </template>
        <template v-else-if="projectWebhook.type == 'bb.plugin.webhook.teams'">
          <img class="h-5 w-5" src="../assets/teams-logo.svg" />
        </template>
        <template
          v-else-if="projectWebhook.type == 'bb.plugin.webhook.dingtalk'"
        >
          <img class="h-5 w-5" src="../assets/dingtalk-logo.png" />
        </template>
        <template v-else-if="projectWebhook.type == 'bb.plugin.webhook.feishu'">
          <img class="h-5 w-5" src="../assets/feishu-logo.png" />
        </template>
        <template v-else-if="projectWebhook.type == 'bb.plugin.webhook.wecom'">
          <img class="h-5 w-5" src="../assets/wecom-logo.png" />
        </template>
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ projectWebhook.name }}
        </h3>
      </div>
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="viewProjectWebhook"
      >
        {{ $t("common.view") }}
      </button>
    </div>
    <div class="border-t border-block-border">
      <dl class="divide-y divide-block-border">
        <div class="grid grid-cols-5 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">URL</dt>
          <dd class="py-0.5 flex text-sm text-main col-span-4">
            {{ projectWebhook.url }}
          </dd>
        </div>
        <div class="grid grid-cols-5 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("project.webhook.triggering-activity") }}
          </dt>
          <dd class="py-0.5 flex text-sm text-main col-span-4">
            {{ activityListStr }}
          </dd>
        </div>
        <div class="grid grid-cols-5 gap-4 px-4 py-2 items-center">
          <dt class="text-sm font-medium text-control-light">
            {{ $t("project.webhook.last-updated-by") }}
          </dt>
          <dd class="py-0.5 flex items-center text-sm text-main col-span-4">
            <div class="flex flex-row items-center space-x-2 mr-1">
              <div class="flex flex-row items-center space-x-1">
                <PrincipalAvatar
                  :principal="projectWebhook.updater"
                  :size="'SMALL'"
                />
                <router-link
                  :to="`/u/${projectWebhook.updater.id}`"
                  class="normal-link"
                  >{{ projectWebhook.updater.name }}
                </router-link>
              </div>
            </div>
            at {{ humanizeTs(projectWebhook.updatedTs) }}
          </dd>
        </div>
      </dl>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, PropType, computed, defineComponent } from "vue";
import { useRouter } from "vue-router";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  ProjectWebhook,
  PROJECT_HOOK_ACTIVITY_ITEM_LIST,
  redirectUrl,
} from "../types";
import { projectWebhookSlug } from "../utils";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default defineComponent({
  name: "ProjectWebhookCard",
  components: { PrincipalAvatar },
  props: {
    projectWebhook: {
      required: true,
      type: Object as PropType<ProjectWebhook>,
    },
  },
  setup(props) {
    const router = useRouter();

    const state = reactive<LocalState>({});

    const viewProjectWebhook = () => {
      router.push({
        name: "workspace.project.hook.detail",
        params: {
          projectWebhookSlug: projectWebhookSlug(props.projectWebhook),
        },
      });
    };

    const activityListStr = computed(() => {
      const list = props.projectWebhook.activityList.map((activity) => {
        for (const item of PROJECT_HOOK_ACTIVITY_ITEM_LIST()) {
          if (item.activity == activity) {
            return item.title;
          }
        }
        return activity;
      });

      return list.join(", ");
    });

    return {
      state,
      redirectUrl,
      viewProjectWebhook,
      activityListStr,
    };
  },
});
</script>
