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
          :username="getUser(inbox.activity)?.title"
          :size="'SMALL'"
          :class="inbox.activity?.comment ? '' : '-mt-0.5'"
        />
        <div class="flex-1 space-y-1">
          <div class="flex w-full justify-between space-x-2">
            <h3
              class="text-sm font-base text-control-light flex flex-row items-center whitespace-nowrap"
            >
              <template v-if="showCreator(inbox.activity)">
                <router-link
                  :to="`/u/${getUserId(inbox.activity)}`"
                  class="mr-1 font-medium text-main hover:underline"
                  @click.stop
                  >{{ getUser(inbox.activity)?.title }}</router-link
                >
              </template>
              <span> {{ actionSentence(inbox.activity) }}</span>
              <template
                v-if="inbox.activity?.level == LogEntity_Level.LEVEL_WARNING"
              >
                <heroicons-outline:exclamation
                  class="ml-1 h-6 w-6 text-warning"
                />
              </template>
              <template
                v-else-if="inbox.activity?.level == LogEntity_Level.LEVEL_ERROR"
              >
                <heroicons-outline:exclamation-circle
                  class="ml-1 w-6 h-6 text-error"
                />
              </template>
            </h3>
            <p class="text-sm text-control whitespace-nowrap">
              {{
                humanizeTs((inbox.activity?.createTime?.getTime() ?? 0) / 1000)
              }}
            </p>
          </div>
          <div v-if="inbox.activity?.comment" class="text-sm text-control">
            {{ inbox.activity.comment }}
          </div>
        </div>
      </div>
    </li>
  </ul>
  <div v-else class="text-center text-control-light">No items</div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { isEmpty } from "lodash-es";
import { PropType } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useInboxV1Store, useActivityV1Store } from "@/store";
import { useUserStore } from "@/store";
import { InboxMessage_Status } from "@/types/proto/v1/inbox_service";
import { InboxMessage } from "@/types/proto/v1/inbox_service";
import {
  LogEntity,
  LogEntity_Action,
  LogEntity_Level,
} from "@/types/proto/v1/logging_service";
import { extractUserResourceName, extractUserUID } from "@/utils";
import PrincipalAvatar from "../components/PrincipalAvatar.vue";
import {
  ActivityIssueCommentCreatePayload,
  ActivityIssueCreatePayload,
  ActivityIssueFieldUpdatePayload,
  ActivityIssueStatusUpdatePayload,
  ActivityTaskStatusUpdatePayload,
  ActivityTaskStatementUpdatePayload,
  ActivityTaskEarliestAllowedTimeUpdatePayload,
} from "../types";
import { issueActivityActionSentence } from "../utils";

defineProps({
  inboxList: {
    required: true,
    type: Object as PropType<InboxMessage[]>,
  },
});

const { t } = useI18n();
const inboxV1Store = useInboxV1Store();
const activityV1Store = useActivityV1Store();
const router = useRouter();

const getUser = (activity: LogEntity | undefined) => {
  const email = extractUserResourceName(activity?.creator ?? "");
  return useUserStore().getUserByEmail(email);
};

const getUserId = (activity: LogEntity | undefined) => {
  const username = getUser(activity)?.name ?? "";
  return extractUserUID(username);
};

const actionLink = (activity: LogEntity | undefined): string => {
  if (!activity) {
    return "";
  }
  if (activity.resource.startsWith("issues")) {
    return `/issue/${activityV1Store.getResourceId(activity)}`;
  } else if (
    activity.action == LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE
  ) {
    const payload = JSON.parse(
      activity.payload
    ) as ActivityTaskStatusUpdatePayload;
    return `/issue/${activityV1Store.getResourceId(activity)}?task=${
      payload.taskId
    }`;
  }

  return "";
};

const showCreator = (activity: LogEntity | undefined): boolean => {
  return (
    activity?.resource.startsWith("issues") ||
    activity?.action ==
      LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE ||
    activity?.action ==
      LogEntity_Action.ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE
  );
};

const actionSentence = (activity: LogEntity | undefined): string => {
  if (activity?.resource.startsWith("issues")) {
    const [tid, params] = issueActivityActionSentence(activity);
    const actionStr = t(tid, params);
    switch (activity.action) {
      case LogEntity_Action.ACTION_ISSUE_CREATE: {
        const payload = JSON.parse(
          activity.payload
        ) as ActivityIssueCreatePayload;
        return `${actionStr} - '${payload?.issueName || ""}'`;
      }
      case LogEntity_Action.ACTION_ISSUE_COMMENT_CREATE: {
        const payload = JSON.parse(
          activity.payload
        ) as ActivityIssueCommentCreatePayload;
        return `${actionStr} - '${payload?.issueName || ""}'`;
      }
      case LogEntity_Action.ACTION_ISSUE_FIELD_UPDATE: {
        const payload = JSON.parse(
          activity.payload
        ) as ActivityIssueFieldUpdatePayload;
        return `${actionStr} - '${payload?.issueName || ""}'`;
      }
      case LogEntity_Action.ACTION_ISSUE_STATUS_UPDATE: {
        const payload = JSON.parse(
          activity.payload
        ) as ActivityIssueStatusUpdatePayload;
        return `${actionStr} - '${payload?.issueName || ""}'`;
      }
    }
    return actionStr;
  }
  switch (activity?.action) {
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATUS_UPDATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityTaskStatusUpdatePayload;
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
    case LogEntity_Action.ACTION_PIPELINE_TASK_STATEMENT_UPDATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityTaskStatementUpdatePayload;
      return t("activity.sentence.changed-from-to", {
        name: "SQL",
        oldValue: payload.oldStatement,
        newValue: payload.newStatement,
      });
    }
    case LogEntity_Action.ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityTaskEarliestAllowedTimeUpdatePayload;
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

const clickInbox = (inbox: InboxMessage) => {
  if (inbox.status == InboxMessage_Status.STATUS_UNREAD) {
    inbox.status = InboxMessage_Status.STATUS_READ;
    inboxV1Store.patchInbox(inbox).then(() => {
      inboxV1Store.updateInboxSummary({
        unread: -1,
        unreadError:
          inbox.activity?.level === LogEntity_Level.LEVEL_ERROR ? -1 : 0,
      });
    });
  }
  const link = actionLink(inbox.activity);
  if (!isEmpty(link)) {
    router.push(link);
  }
};
</script>
