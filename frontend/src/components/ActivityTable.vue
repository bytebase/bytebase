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
            class="
              w-5
              h-5
              flex
              items-center
              justify-center
              rounded-full
              select-none
            "
          >
            <template v-if="activity.level === `WARN`">
              <heroicons-outline:information-circle class="text-warning" />
            </template>
            <template v-else-if="activity.level === `ERROR`">
              <heroicons-outline:exclamation-circle class="text-error" />
            </template>
          </span>
          <span>{{ activityName(activity.type) }}</span>
          <template v-if="activityTypeLink(activity)">
            <a
              v-if="activityTypeLink(activity).external"
              :href="activityTypeLink(activity).path"
              target="_blank"
              class="normal-link flex flex-row items-center"
            >
              <span>{{ activityTypeLink(activity).title }}</span>
              <heroicons-outline:external-link class="w-4 h-4" />
            </a>
            <router-link
              v-else
              class="normal-link"
              :to="activityTypeLink(activity).path"
            >
              {{ activityTypeLink(activity).title }}
            </router-link>
          </template>
        </div>
      </BBTableCell>
      <BBTableCell class="w-96">
        <div class="flex flex-row space-x-1">
          <span>{{ activity.comment }}</span>
          <template v-if="activityCommentLink(activity)">
            <a
              v-if="activityCommentLink(activity).external"
              :href="activityCommentLink(activity).path"
              target="_blank"
              class="normal-link flex flex-row items-center"
            >
              <span>{{ activityCommentLink(activity).title }}</span>
              <heroicons-outline:external-link class="w-4 h-4" />
            </a>
            <router-link
              v-else
              class="normal-link"
              :to="activityCommentLink(activity).path"
            >
              {{ activityCommentLink(activity).title }}
            </router-link>
          </template>
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
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import {
  Activity,
  ActivityProjectRepositoryPushPayload,
  ActivityProjectDatabaseTransferPayload,
  activityName,
} from "../types";
import { issueSlug } from "../utils";

type Link = {
  title: string;
  path: string;
  external: boolean;
};

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "Name",
  },
  {
    title: "Comment",
  },
  {
    title: "Created",
  },
  {
    title: "Invoker",
  },
];

export default {
  name: "ActivityTable",
  components: {},
  props: {
    activityList: {
      required: true,
      type: Object as PropType<Activity[]>,
    },
  },
  setup() {
    const activityTypeLink = (activity: Activity): Link | undefined => {
      switch (activity.type) {
        case "bb.project.repository.push": {
          const payload =
            activity.payload as ActivityProjectRepositoryPushPayload;
          return {
            title: payload.pushEvent.fileCommit.id.substring(0, 7),
            path: payload.pushEvent.fileCommit.url,
            external: true,
          };
        }
        case "bb.project.database.transfer": {
          const payload =
            activity.payload as ActivityProjectDatabaseTransferPayload;
          return {
            title: payload.databaseName,
            path: `/db/${payload.databaseId}`,
            external: false,
          };
        }
      }
      return undefined;
    };

    const activityCommentLink = (activity: Activity): Link | undefined => {
      switch (activity.type) {
        case "bb.project.repository.push": {
          const payload =
            activity.payload as ActivityProjectRepositoryPushPayload;
          if (payload.issueId && payload.issueName) {
            return {
              title: `issue/${payload.issueId}`,
              path: `/issue/${issueSlug(payload.issueName!, payload.issueId!)}`,
              external: false,
            };
          }
          break;
        }
      }
      return undefined;
    };

    return {
      activityName,
      COLUMN_LIST,
      activityTypeLink,
      activityCommentLink,
    };
  },
};
</script>
