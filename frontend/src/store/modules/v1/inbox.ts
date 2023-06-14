import { defineStore } from "pinia";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { reactive } from "vue";
import { InboxMessage, InboxSummary } from "@/types/proto/v1/inbox_service";
import { inboxServiceClient } from "@/grpcweb";
import { useActivityV1Store } from "./activity";
import { ComposedInbox } from "@/types";

dayjs.extend(utc);

export const useInboxV1Store = defineStore("inbox_v1", () => {
  const inboxMessageList = reactive<ComposedInbox[]>([]);
  const inboxSummary = reactive<InboxSummary>({
    hasUnread: false,
    hasUnreadError: false,
  });

  const composeActivity = async (
    inboxMessage: InboxMessage
  ): Promise<ComposedInbox | undefined> => {
    try {
      const activity = await useActivityV1Store().fetchActivityByUID(
        inboxMessage.activityUid
      );
      if (!activity) {
        return;
      }
      return {
        ...inboxMessage,
        activity,
      };
    } catch {
      // nothing, we will skip inbox with undefined activity.
    }
    return;
  };

  const fetchInboxList = async (readCreatedAfterTs: number) => {
    const resp = await inboxServiceClient.listInbox({
      filter: `create_time >= ${dayjs(readCreatedAfterTs).utc().format()}`,
    });

    const list = await Promise.all(resp.inboxMessages.map(composeActivity));
    inboxMessageList.splice(0, inboxMessageList.length);
    for (const inbox of list) {
      if (inbox) {
        inboxMessageList.push(inbox);
      }
    }

    return inboxMessageList;
  };

  const fetchInboxSummary = async () => {
    const summary = await inboxServiceClient.getInboxSummary({});
    inboxSummary.hasUnread = summary.hasUnread;
    inboxSummary.hasUnreadError = summary.hasUnreadError;
    return inboxSummary;
  };

  const patchInbox = async (inboxMessage: Partial<InboxMessage>) => {
    const index = inboxMessageList.findIndex(
      (i) => i.name === inboxMessage.name
    );
    if (index < 0) {
      return;
    }
    const existed = inboxMessageList[index];

    const inbox = await inboxServiceClient.updateInbox({
      inboxMessage,
      updateMask: ["status"],
    });

    inboxMessageList[index] = {
      ...inbox,
      activity: existed.activity,
    };

    return inboxMessageList[index];
  };

  return {
    inboxSummary,
    inboxMessageList,
    fetchInboxList,
    fetchInboxSummary,
    patchInbox,
  };
});
