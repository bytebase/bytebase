<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="activityList"
    :show-header="true"
    :row-clickable="false"
    :left-bordered="true"
    :right-bordered="true"
  >
    <template #body="{ rowData: activity }">
      <BBTableCell :left-padding="4" class="w-8">
        <div class="flex flew-row space-x-1">
          <span
            v-if="activity.level != `INFO`"
            class="w-5 h-5 flex items-center justify-center rounded-full select-none"
          >
            <template v-if="activity.level === `WARN`">
              <heroicons-outline:information-circle class="text-warning" />
            </template>
            <template v-else-if="activity.level === `ERROR`">
              <heroicons-outline:exclamation-circle class="text-error" />
            </template>
          </span>
          <span class="capitalize">{{ activityName(activity.type) }}</span>
          <ActivityTypeLink :activity="activity" />
        </div>
      </BBTableCell>
      <BBTableCell class="w-96">
        <div class="flex flex-row space-x-1">
          <span>{{ activity.comment }}</span>
          <ActivityCommentLink :activity="activity" />
        </div>
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ humanizeTs(activity.createdTs) }}
      </BBTableCell>
      <BBTableCell class="w-8">
        <div class="flex flex-row items-center">
          <BBAvatar :size="'SMALL'" :username="activity.creator.name" />
          <span class="ml-2">{{ activity.creator.name }}</span>
        </div>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { BBTableColumn } from "../bbkit/types";
import { Activity, activityName } from "../types";
import ActivityTypeLink from "./ActivityTable/ActivityTypeLink.vue";
import ActivityCommentLink from "./ActivityTable/ActivityCommentLink.vue";

export default {
  name: "ActivityTable",
  components: { ActivityTypeLink, ActivityCommentLink },
  props: {
    activityList: {
      required: true,
      type: Object as PropType<Activity[]>,
    },
  },
  setup() {
    const { t } = useI18n();

    const COLUMN_LIST = computed((): BBTableColumn[] => [
      {
        title: t("activity.name"),
      },
      {
        title: t("activity.comment"),
      },
      {
        title: t("activity.created"),
      },
      {
        title: t("activity.invoker"),
      },
    ]);

    return {
      activityName,
      COLUMN_LIST,
    };
  },
};
</script>
