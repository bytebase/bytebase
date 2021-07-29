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
              <template v-if="inbox.level == 'WARNING'">
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
              <template v-else-if="inbox.level == 'ERROR'">
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
import { PropType } from "@vue/runtime-core";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  ActionIssueCommentCreatePayload,
  ActionIssueCreatePayload,
  ActionIssueFieldUpdatePayload,
  ActionIssueStatusUpdatePayload,
  ActionMemberActivateDeactivatePayload,
  ActionMemberCreatePayload,
  ActionMemberRoleUpdatePayload,
  ActionTaskStatusUpdatePayload,
  Activity,
  Inbox,
  Principal,
} from "../types";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { isEmpty } from "lodash";
import { issueActivityActionSentence, roleName } from "../utils";

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
      } else if (activity.actionType.startsWith("bb.member.")) {
        return `/setting/member`;
      }

      return "";
    };

    const showCreator = (activity: Activity): boolean => {
      return (
        activity.actionType.startsWith("bb.issue.") ||
        activity.actionType.startsWith("bb.member.")
      );
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
      } else if (activity.actionType.startsWith("bb.member.")) {
        switch (activity.actionType) {
          case "bb.member.create": {
            const payload = activity.payload as ActionMemberCreatePayload;
            if (payload.principalId == activity.creator.id) {
              return `(${payload.principalEmail}) joined as a ${roleName(
                payload.role
              )}`;
            }
            var verb = "added";
            if (payload.memberStatus == "INVITED") {
              verb = "invited";
            }
            return `${verb} ${payload.principalName} (${
              payload.principalEmail
            }) as a ${roleName(payload.role)}`;
          }
          case "bb.member.role.update": {
            const payload = activity.payload as ActionMemberRoleUpdatePayload;
            return `changed ${payload.principalName} (${
              payload.principalEmail
            }) from ${roleName(payload.oldRole)} to ${roleName(
              payload.newRole
            )}`;
          }
          case "bb.member.activate": {
            const payload =
              activity.payload as ActionMemberActivateDeactivatePayload;
            return `activated ${roleName(payload.role)} ${
              payload.principalName
            } (${payload.principalEmail})`;
          }
          case "bb.member.deactivate": {
            const payload =
              activity.payload as ActionMemberActivateDeactivatePayload;
            return `deactivated ${roleName(payload.role)} ${
              payload.principalName
            } (${payload.principalEmail})`;
          }
        }
        return "";
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
