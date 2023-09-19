<template>
  <BBTable
    :column-list="columnList"
    :data-source="activityList"
    :show-header="true"
    :row-clickable="false"
    :left-bordered="true"
    :right-bordered="true"
  >
    <template #body="{ rowData: activity }">
      <BBTableCell :left-padding="4" class="w-[20%]">
        <div class="flex flew-row space-x-1">
          <span
            v-if="activity.level != LogEntity_Level.LEVEL_INFO"
            class="w-5 h-5 flex items-center justify-center rounded-full select-none"
          >
            <template v-if="activity.level === LogEntity_Level.LEVEL_WARNING">
              <heroicons-outline:information-circle class="text-warning" />
            </template>
            <template
              v-else-if="activity.level === LogEntity_Level.LEVEL_ERROR"
            >
              <heroicons-outline:exclamation-circle class="text-error" />
            </template>
          </span>
          <span class="capitalize">{{ activityName(activity.action) }}</span>
          <ActivityTypeLink :activity="activity" />
        </div>
      </BBTableCell>
      <BBTableCell>
        <div class="flex flex-row space-x-1">
          <span>{{ activity.comment }}</span>
          <ActivityCommentLink :activity="activity" />
        </div>
      </BBTableCell>
      <BBTableCell class="w-[12%]">
        {{ humanizeTs((activity.createTime?.getTime() ?? 0) / 1000) }}
      </BBTableCell>
      <BBTableCell class="w-[12%]">
        <div class="flex flex-row items-center">
          <BBAvatar
            :size="'SMALL'"
            :username="getUser(activity)?.title"
            :email="getUser(activity)?.email"
          />
          <span class="ml-2">{{ getUser(activity)?.title }}</span>
        </div>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { useUserStore } from "@/store";
import { activityName } from "@/types";
import { LogEntity, LogEntity_Level } from "@/types/proto/v1/logging_service";
import { extractUserResourceName } from "@/utils";
import { BBTableColumn } from "../../bbkit/types";

defineProps({
  activityList: {
    type: Object as PropType<LogEntity[]>,
    required: true,
  },
});

const { t } = useI18n();

const columnList = computed((): BBTableColumn[] => [
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

const getUser = (activity: LogEntity) => {
  return useUserStore().getUserByEmail(
    extractUserResourceName(activity.creator)
  );
};
</script>
