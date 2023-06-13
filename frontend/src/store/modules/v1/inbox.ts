import { defineStore } from "pinia";
import { reactive } from "vue";
import { InboxMessage, InboxSummary } from "@/types/proto/v1/inbox_service";
import { inboxServiceClient } from "@/grpcweb";
import { useActivityV1Store } from "./activity";
import { InboxV1 } from "@/types";

export const useInboxV1Store = defineStore("inbox_v1", () => {
  let inboxMessageList = reactive<InboxV1[]>([]);
  let inboxSummary = reactive<InboxSummary>({
    hasUnread: false,
    hasUnreadError: false,
  });

  const composeActivity = async (
    inboxMessage: InboxMessage
  ): Promise<InboxV1 | undefined> => {
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
    const resp = await inboxServiceClient.listInbox({});

    const list = (
      await Promise.all(resp.inboxMessages.map(composeActivity))
    ).filter((inbox) => inbox !== undefined) as InboxV1[];
    inboxMessageList = list;
    return inboxMessageList;
  };

  const fetchInboxSummary = async () => {
    const summary = await inboxServiceClient.getInboxSummary({});
    inboxSummary = summary;
    return inboxSummary;
  };

  const patchInbox = async (inboxMessage: Partial<InboxMessage>) => {
    const inbox = await inboxServiceClient.updateInbox({
      inboxMessage,
      updateMask: ["status"],
    });

    const inboxV1 = await composeActivity(inbox);
    if (!inboxV1) {
      return;
    }

    const index = inboxMessageList.findIndex((i) => i.name === inboxV1.name);

    inboxMessageList = [
      ...inboxMessageList.slice(0, index),
      inboxV1,
      ...inboxMessageList.slice(0, index + 1),
    ];
    return inboxV1;
  };

  return {
    inboxSummary,
    inboxMessageList,
    fetchInboxList,
    fetchInboxSummary,
    patchInbox,
  };
});
