<template>
  <div
    class="divide-y divide-block-border border border-block-border rounded-sm"
  >
    <div class="flex py-2 px-4 justify-between">
      <div class="flex flex-row space-x-2 items-center">
        <!-- This awkward code is author couldn't figure out proper way to use dynamic src under vite
                   https://github.com/vitejs/vite/issues/1265 -->
        <template v-if="projectWebhook.type === Webhook_Type.TYPE_SLACK">
          <img class="h-5 w-5" src="../assets/slack-logo.png" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_DISCORD">
          <img class="h-5 w-5" src="../assets/discord-logo.svg" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_TEAMS">
          <img class="h-5 w-5" src="../assets/teams-logo.svg" />
        </template>
        <template
          v-else-if="projectWebhook.type === Webhook_Type.TYPE_DINGTALK"
        >
          <img class="h-5 w-5" src="../assets/dingtalk-logo.png" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_FEISHU">
          <img class="h-5 w-5" src="../assets/feishu-logo.webp" />
        </template>
        <template v-else-if="projectWebhook.type === Webhook_Type.TYPE_WECOM">
          <img class="h-5 w-5" src="../assets/wecom-logo.png" />
        </template>
        <h3 class="text-lg leading-6 font-medium text-main">
          {{ projectWebhook.title }}
        </h3>
      </div>
      <NButton @click.prevent="viewProjectWebhook">
        {{ $t("common.view") }}
      </NButton>
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
      </dl>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { useRouter } from "vue-router";
import { PROJECT_V1_WEBHOOK_DETAIL } from "@/router/dashboard/projectV1";
import { projectWebhookV1ActivityItemList } from "@/types";
import {
  Webhook,
  Webhook_Type,
  activity_TypeToJSON,
} from "@/types/proto/v1/project_service";
import { projectWebhookV1Slug } from "@/utils";

const props = defineProps({
  projectWebhook: {
    required: true,
    type: Object as PropType<Webhook>,
  },
});
const router = useRouter();

const viewProjectWebhook = () => {
  router.push({
    name: PROJECT_V1_WEBHOOK_DETAIL,
    params: {
      projectWebhookSlug: projectWebhookV1Slug(props.projectWebhook),
    },
  });
};

const activityListStr = computed(() => {
  const wellknownActivityItemList = projectWebhookV1ActivityItemList();
  const list = props.projectWebhook.notificationTypes.map((activity) => {
    const item = wellknownActivityItemList.find(
      (item) => item.activity === activity
    );
    if (item) {
      return item.title;
    }
    return activity_TypeToJSON(activity);
  });

  return list.join(", ");
});
</script>
