<template>
  <div class="flex flex-col gap-y-4">
    <div class="flex items-center justify-end">
      <PermissionGuardWrapper
        v-slot="slotProps"
        :project="project"
        :permissions="['bb.projects.update']"
      >
        <NButton
          type="primary"
          :disabled="slotProps.disabled"
          @click.prevent="addProjectWebhook"
        >
          <template #icon>
            <PlusIcon class="h-4 w-4" />
          </template>
          {{ $t("project.webhook.add-a-webhook") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>
    <NDataTable
      :data="project.webhooks"
      :columns="columnList"
      :striped="true"
      :row-key="(webhook: Webhook) => webhook.name"
      :bordered="true"
    />
  </div>
</template>

<script setup lang="tsx">
import { PlusIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NButton, NDataTable, NTag } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import WebhookTypeIcon from "@/components/Project/WebhookTypeIcon.vue";
import {
  PROJECT_V1_ROUTE_WEBHOOK_CREATE,
  PROJECT_V1_ROUTE_WEBHOOK_DETAIL,
} from "@/router/dashboard/projectV1";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { projectWebhookV1ActivityItemList } from "@/types";
import type { Webhook } from "@/types/proto-es/v1/project_service_pb";
import { Activity_Type } from "@/types/proto-es/v1/project_service_pb";
import { projectWebhookV1Slug } from "@/utils";

const props = defineProps<{
  projectId: string;
  allowEdit: boolean;
}>();

const router = useRouter();
const { t } = useI18n();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

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
      ellipsis: true,
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
          return Activity_Type[activity] || `ACTIVITY_${activity}`;
        });

        return (
          <div class="flex flex-wrap gap-2">
            {list.map((title) => (
              <NTag size="small">{title}</NTag>
            ))}
          </div>
        );
      },
    },
    {
      key: "actions",
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
            { default: () => t("common.view") }
          )
        ),
    },
  ];
});
</script>
