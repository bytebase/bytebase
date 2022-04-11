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
              class="text-sm font-base text-control-light flex flex-row items-center whitespace-nowrap"
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
                <heroicons-outline:exclamation
                  class="ml-1 h-6 w-6 text-warning"
                />
              </template>
              <template v-else-if="inbox.activity.level == 'ERROR'">
                <heroicons-outline:exclamation-circle
                  class="ml-1 w-6 h-6 text-error"
                />
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
import { computed, defineComponent, PropType } from "vue";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  ActivityIssueCommentCreatePayload,
  ActivityIssueCreatePayload,
  ActivityIssueFieldUpdatePayload,
  ActivityIssueStatusUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  ActivityTaskStatementUpdatePayload,
  ActivityTaskEarliestAllowedTimeUpdatePayload,
  Activity,
  Inbox,
} from "../types";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { isEmpty } from "lodash-es";
import { issueActivityActionSentence } from "../utils";
import { useI18n } from "vue-i18n";
import dayjs from "dayjs";
import { useCurrentUser } from "@/store";

export default defineComponent({
  name: "InboxList",
  components: { PrincipalAvatar },
  props: {
    inboxList: {
      required: true,
      type: Object as PropType<Inbox[]>,
    },
  },
  setup() {
    const { t } = useI18n();
    const store = useStore();
    const router = useRouter();

    const currentUser = useCurrentUser();

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
      return (
        activity.type.startsWith("bb.issue.") ||
        activity.type == "bb.pipeline.task.statement.update" ||
        activity.type == "bb.pipeline.task.general.earliest-allowed-time.update"
      );
    };

    const actionSentence = (activity: Activity): string => {
      if (activity.type.startsWith("bb.issue.")) {
        const [tid, params] = issueActivityActionSentence(activity);
        const actionStr = t(tid, params);
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
      }
      switch (activity.type) {
        case "bb.pipeline.task.status.update": {
          const payload = activity.payload as ActivityTaskStatusUpdatePayload;
          let actionStr = t(`activity.sentence.changed`);
          switch (payload.newStatus) {
            case "PENDING": {
              if (payload.oldStatus == "RUNNING") {
                actionStr = t(`activity.sentence.canceled`);
              } else if (payload.oldStatus == "PENDING_APPROVAL") {
                actionStr = t(`activity.sentence.approved`);
              }
              break;
            }
            case "RUNNING": {
              actionStr = t(`activity.sentence.started`);
              break;
            }
            case "DONE": {
              actionStr = t(`activity.sentence.completed`);
              break;
            }
            case "FAILED": {
              actionStr = t(`activity.sentence.failed`);
              break;
            }
          }
          return `${t("activity.subject-prefix.task")} '${
            payload.taskName
          }' ${actionStr} - '${payload?.issueName || ""}'`;
        }
        case "bb.pipeline.task.statement.update": {
          const payload =
            activity.payload as ActivityTaskStatementUpdatePayload;
          return t("activity.sentence.changed-from-to", {
            name: "SQL",
            oldValue: payload.oldStatement,
            newValue: payload.newStatement,
          });
        }
        case "bb.pipeline.task.general.earliest-allowed-time.update": {
          const payload =
            activity.payload as ActivityTaskEarliestAllowedTimeUpdatePayload;
          const oldTs = payload.oldEarliestAllowedTs;
          const newTs = payload.newEarliestAllowedTs;

          return t("activity.sentence.changed-from-to", {
            name: "earliest allowed time",
            oldValue: oldTs
              ? dayjs(oldTs * 1000)
              : t("task.earliest-allowed-time-unset"),
            newValue: newTs
              ? dayjs(newTs * 1000)
              : t("task.earliest-allowed-time-unset"),
          });
        }
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
});
</script>
