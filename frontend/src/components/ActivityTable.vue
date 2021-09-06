<template>
  <BBTable
    :columnList="COLUMN_LIST"
    :dataSource="activityList"
    :showHeader="true"
    :rowClickable="false"
    :leftBordered="true"
    :rightBordered="true"
  >
    <template v-slot:body="{ rowData: activity }">
      <BBTableCell :leftPadding="4" class="w-4">
        <span
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
          <template v-if="activity.level === `INFO`">
            <svg
              class="text-info"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
          </template>
          <template v-else-if="activity.level === `WARN`">
            <svg
              class="text-warning"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
          </template>
          <template v-else-if="activity.level === `ERROR`">
            <svg
              class="text-error"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              ></path>
            </svg>
          </template>
        </span>
      </BBTableCell>
      <BBTableCell class="w-8">
        {{ activityName(activity.type) }}
        <template v-if="activityTypeLink(activity)">
          <a
            v-if="activityTypeLink(activity).external"
            :href="activityTypeLink(activity).path"
            target="_blank"
            class="normal-link"
          >
            {{ activityTypeLink(activity).title }}
          </a>
          <router-link
            v-else
            class="normal-link"
            :to="activityTypeLink(activity).path"
          >
            {{ activityTypeLink(activity).title }}
          </router-link>
        </template>
      </BBTableCell>
      <BBTableCell class="w-24">
        {{ activity.comment }}
        <template v-if="activityCommentLink(activity)">
          <a
            v-if="activityCommentLink(activity).external"
            :href="activityCommentLink(activity).path"
            target="_blank"
            class="normal-link"
          >
            {{ activityCommentLink(activity).title }}
          </a>
          <router-link
            v-else
            class="normal-link"
            :to="activityCommentLink(activity).path"
          >
            {{ activityCommentLink(activity).title }}
          </router-link>
        </template>
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
  ActivityProjectRepositoryPushPayload,
  Activity,
  ActivityProjectDatabaseTransferPayload,
} from "../types";
import { activityName, issueSlug } from "../utils";

type Link = {
  title: string;
  path: string;
  external: boolean;
};

const COLUMN_LIST: BBTableColumn[] = [
  {
    title: "",
  },
  {
    title: "Type",
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
  setup(props, ctx) {
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
