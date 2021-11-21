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
      <BBTableCell :leftPadding="4" class="w-8">
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
          <span>{{ activityName(activity.type) }}</span>
          <template v-if="activityTypeLink(activity)">
            <a
              v-if="activityTypeLink(activity).external"
              :href="activityTypeLink(activity).path"
              target="_blank"
              class="normal-link flex flex-row items-center"
            >
              <span>{{ activityTypeLink(activity).title }}</span>
              <svg
                class="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                ></path>
              </svg>
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
              <svg
                class="w-4 h-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
                ></path>
              </svg>
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
            path: `/db/${payload.databaseID}`,
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
          if (payload.issueID && payload.issueName) {
            return {
              title: `issue/${payload.issueID}`,
              path: `/issue/${issueSlug(payload.issueName!, payload.issueID!)}`,
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
