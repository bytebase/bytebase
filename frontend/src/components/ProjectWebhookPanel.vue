<template>
  <div class="space-y-4">
    <div v-if="allowEdit" class="flex items-center justify-end">
      <NButton
        type="primary"
        class="capitalize"
        @click.prevent="addProjectWebhook"
      >
        {{ $t("project.webhook.add-a-webhook") }}
      </NButton>
    </div>
    <NDataTable
      :data="project.webhooks"
      :columns="columnList"
      :striped="true"
      :bordered="true"
    />
  </div>
</template>

<script lang="ts" setup>
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import WebhookTypeIcon from "@/components/Project/WebhookTypeIcon.vue";
import {
  PROJECT_V1_ROUTE_WEBHOOK_CREATE,
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
} from "@/router/dashboard/projectV1";
import { projectWebhookV1ActivityItemList } from "@/types";
import type { Project, Webhook } from "@/types/proto/v1/project_service";
import { activity_TypeToJSON } from "@/types/proto/v1/project_service";
import { projectWebhookV1Slug } from "@/utils";

defineProps<{
  project: Project;
  allowEdit: boolean;
}>();
const router = useRouter();
const { t } = useI18n();

const addProjectWebhook = () => {
  router.push({
    name: PROJECT_V1_ROUTE_WEBHOOK_CREATE,
  });
};

const columnList = computed((): DataTableColumn<Webhook>[] => {
  return [
    {
      key: "title",
      title: t("common.name"),
      width: "15rem",
      resizable: true,
      render: (webhook) =>
        h("div", { class: "flex items-center gap-x-2" }, [
          h(WebhookTypeIcon, { type: webhook.type, class: "w-5 h-5" }),
          webhook.title,
        ]),
    },
    {
      key: "url",
      title: "URL",
      resizable: true,
      render: (webhook) => webhook.url,
    },
    {
      key: "triggering",
      title: t("project.webhook.triggering-activity"),
      resizable: true,
      render: (webhook) => {
        const wellknownActivityItemList = projectWebhookV1ActivityItemList();
        const list = webhook.notificationTypes.map((activity) => {
          const item = wellknownActivityItemList.find(
            (item) => item.activity === activity
          );
          if (item) {
            return item.title;
          }
          return activity_TypeToJSON(activity);
        });

        return list.join(", ");
      },
    },
    {
      key: "view",
      title: "",
      width: "5rem",
      render: (webhook) =>
        h(
          "div",
          { class: "flex justify-end" },
          h(
            NButton,
            {
              size: "small",
              onClick: () => {
                router.push({
                  name: PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
                  params: {
                    projectWebhookSlug: projectWebhookV1Slug(webhook),
                  },
                });
              },
            },
            t("common.view")
          )
        ),
    },
  ];
});
</script>
