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
import { PropType } from "@vue/runtime-core";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  ActionIssueCommentCreatePayload,
  ActionIssueCreatePayload,
  ActionIssueFieldUpdatePayload,
  ActionIssueStatusUpdatePayload,
  ActionTaskStatusUpdatePayload,
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

    const actionLink = (activity: Activity): string => {
      if (activity.actionType.startsWith("bb.issue.")) {
        return `/issue/${activity.containerId}`;
      } else if (activity.actionType == "bb.pipeline.task.status.update") {
        const payload = activity.payload as ActionTaskStatusUpdatePayload;
        return `/issue/${activity.containerId}?task=${payload.taskId}`;
      }

      return "";
    };

    const showCreator = (activity: Activity): boolean => {
      return activity.actionType.startsWith("bb.issue.");
    };

    const actionSentence = (activity: Activity): string => {
      if (activity.actionType.startsWith("bb.issue.")) {
        const actionStr = issueActivityActionSentence(activity);
        switch (activity.actionType) {
          case "bb.issue.create": {
            const payload = activity.payload as ActionIssueCreatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.comment.create": {
            const payload = activity.payload as ActionIssueCommentCreatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.field.update": {
            const payload = activity.payload as ActionIssueFieldUpdatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
          case "bb.issue.status.update": {
            const payload = activity.payload as ActionIssueStatusUpdatePayload;
            return `${actionStr} - '${payload?.issueName || ""}'`;
          }
        }
        return actionStr;
      } else if (activity.actionType == "bb.pipeline.task.status.update") {
        const payload = activity.payload as ActionTaskStatusUpdatePayload;
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
        store.dispatch("inbox/patchInbox", {
          inboxId: inbox.id,
          inboxPatch: {
            status: "READ",
          },
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
