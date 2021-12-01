<template>
  <ul v-if="inboxList.length > 0">
    <li
      v-for="(inbox, index) in inboxList"
      :key="index"
      class="p-3 hover:bg-control-bg-hover cursor-default"
      :class="actionLink(inbox.activity) ? 'cursor-pointer' : 'cursor-default'"
      @click.prevent="clickInbox(inbox)"
    >
      <div class="flex space-x-3">
        <PrincipalAvatar
          :principal="inbox.activity.creator"
          :size="'SMALL'"
          :class="inbox.activity.comment ? '' : '-mt-0.5'"
        />
        <div class="flex-1 space-y-1">
          <div class="flex w-full justify-between space-x-2">
            <h3
              class="
                text-sm
                font-base
                text-control-light
                flex flex-row
                items-center
                whitespace-nowrap
              "
            >
              <template v-if="showCreator(inbox.activity)">
                <router-link
                  :to="`/u/${inbox.activity.creator.id}`"
                  class="mr-1 font-medium text-main hover:underline"
                  >{{ inbox.activity.creator.name }}</router-link
                >
              </template>
              <span> {{ actionSentence(inbox.activity) }}</span>
              <template v-if="inbox.activity.level == 'WARN'">
                <svg
                  class="ml-1 h-6 w-6 text-warning"
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  aria-hidden="true"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
                  ></path>
                </svg>
              </template>
              <template v-else-if="inbox.activity.level == 'ERROR'">
                <svg
                  class="ml-1 w-6 h-6 text-error"
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
            </h3>
            <p class="text-sm text-control whitespace-nowrap">
              {{ humanizeTs(inbox.activity.createdTs) }}
            </p>
          </div>
          <div v-if="inbox.activity.comment" class="text-sm text-control">
            {{ inbox.activity.comment }}
          </div>
        </div>
      </div>
    </li>
  </ul>
  <div v-else class="text-center text-control-light">No items</div>
</template>

<script lang="ts">
import { computed, PropType } from "@vue/runtime-core";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  ActivityIssueCommentCreatePayload,
  ActivityIssueCreatePayload,
  ActivityIssueFieldUpdatePayload,
  ActivityIssueStatusUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  Activity,
  Inbox,
} from "../types";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { isEmpty } from "lodash";
import { issueActivityActionSentence } from "../utils";

export default {
  name: "InboxList",
  components: { PrincipalAvatar },
  props: {
    inboxList: {
      required: true,
      type: Object as PropType<Inbox[]>,
    },
  },
  setup() {
    const store = useStore();
    const router = useRouter();

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const actionLink = (activity: Activity): string => {
      if (activity.type.startsWith("bb.issue.")) {
        return `/issue/${activity.containerId}`;
      } else if (activity.type == "bb.pipeline.task.status.update") {
        const payload = activity.payload as ActivityTaskStatusUpdatePayload;
        return `/issue/${activity.containerId}?task=${payload.taskId}`;
      }

      return "";
    };

    const showCreator = (activity: Activity): boolean => {
      return activity.type.startsWith("bb.issue.");
    };

    const actionSentence = (activity: Activity): string => {
      if (activity.type.startsWith("bb.issue.")) {
        const actionStr = issueActivityActionSentence(activity);
        switch (activity.type) {
          case "bb.issue.create": {
            const payload = activity.payload as ActivityIssueCreatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.comment.create": {
            const payload =
              activity.payload as ActivityIssueCommentCreatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.field.update": {
            const payload = activity.payload as ActivityIssueFieldUpdatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.status.update": {
            const payload =
              activity.payload as ActivityIssueStatusUpdatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
        }
        return actionStr;
      } else if (activity.type == "bb.pipeline.task.status.update") {
        const payload = activity.payload as ActivityTaskStatusUpdatePayload;
        var actionStr = `changed`;
        switch (payload.newStatus) {
          case "PENDING": {
            if (payload.oldStatus == "RUNNING") {
              actionStr = `canceled`;
            } else if (payload.oldStatus == "PENDING_APPROVAL") {
              actionStr = `approved`;
            }
            break;
          }
          case "RUNNING": {
            actionStr = `started`;
            break;
          }
          case "DONE": {
            actionStr = `completed`;
            break;
          }
          case "FAILED": {
            actionStr = `failed`;
            break;
          }
        }
        return `Task '${payload.taskName}' ${actionStr} - '${
          payload?.issueName || ""
        }'`;
      }

      return "";
    };

    const clickInbox = (inbox: Inbox) => {
      if (inbox.status == "UNREAD") {
        store
          .dispatch("inbox/patchInbox", {
            inboxId: inbox.id,
            inboxPatch: {
              status: "READ",
            },
          })
          .then(() => {
            store.dispatch(
              "inbox/fetchInboxSummaryByUser",
              currentUser.value.id
            );
          });
      }
      const link = actionLink(inbox.activity);
      if (!isEmpty(link)) {
        router.push(link);
      }
    };

    return { actionLink, showCreator, actionSentence, clickInbox };
  },
};
</script>
