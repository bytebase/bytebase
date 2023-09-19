import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";
import { defineStore } from "pinia";
import { reactive } from "vue";
import { inboxServiceClient } from "@/grpcweb";
import { InboxMessage, InboxSummary } from "@/types/proto/v1/inbox_service";

dayjs.extend(utc);

export const useInboxV1Store = defineStore("inbox_v1", () => {
  const inboxMessageList = reactive<InboxMessage[]>([]);
  const inboxSummary = reactive<InboxSummary>({
    unread: 0,
    unreadError: 0,
  });

  const fetchInboxList = async (readCreatedAfterTs: number) => {
    if (inboxMessageList.length > 0) {
      return inboxMessageList;
    }
    const resp = await inboxServiceClient.listInbox({
      filter: `create_time >= ${dayjs(readCreatedAfterTs).utc().format()}`,
    });

    inboxMessageList.splice(0, inboxMessageList.length);
    for (const inbox of resp.inboxMessages) {
      inboxMessageList.push(inbox);
    }

    return inboxMessageList;
  };

  const fetchInboxSummary = async () => {
    const summary = await inboxServiceClient.getInboxSummary({});
    inboxSummary.unread = summary.unread;
    inboxSummary.unreadError = summary.unreadError;
    return inboxSummary;
  };

  const updateInboxSummary = (summary: InboxSummary) => {
    inboxSummary.unread += summary.unread;
    inboxSummary.unreadError += summary.unreadError;
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
    updateInboxSummary,
    patchInbox,
  };
});
