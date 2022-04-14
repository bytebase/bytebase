import { defineStore } from "pinia";
import axios from "axios";
import {
  Activity,
  Inbox,
  InboxId,
  InboxPatch,
  InboxState,
  InboxSummary,
  PrincipalId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "@/types";
import { useActivityStore } from "./activity";

function convert(inbox: ResourceObject, includedList: ResourceObject[]): Inbox {
  const activityId = (inbox.relationships!.activity.data as ResourceIdentifier)
    .id;
  let activity: Activity = unknown("ACTIVITY") as Activity;
  activity.id = parseInt(activityId);

  const activityStore = useActivityStore();
  for (const item of includedList || []) {
    if (item.type == "activity" && item.id == activityId) {
      activity = activityStore.convert(item, includedList);
    }
  }

  return {
    ...(inbox.attributes as Omit<Inbox, "id">),
    id: parseInt(inbox.id),
    activity,
  };
}

export const useInboxStore = defineStore("inbox", {
  state: (): InboxState => ({
    inboxListByUser: new Map(),
    inboxSummaryByUser: new Map(),
  }),
  actions: {
    getInboxListByUser(userId: PrincipalId): Inbox[] {
      return this.inboxListByUser.get(userId) || [];
    },
    getInboxSummaryByUser(userId: PrincipalId): InboxSummary {
      return (
        this.inboxSummaryByUser.get(userId) || {
          hasUnread: false,
          hasUnreadError: false,
        }
      );
    },
    setInboxListByUser({
      userId,
      inboxList,
    }: {
      userId: PrincipalId;
      inboxList: Inbox[];
    }) {
      this.inboxListByUser.set(userId, inboxList);
    },
    setInboxSummaryByUser({
      userId,
      inboxSummary,
    }: {
      userId: PrincipalId;
      inboxSummary: InboxSummary;
    }) {
      this.inboxSummaryByUser.set(userId, inboxSummary);
    },
    updateInboxById({ inboxId, inbox }: { inboxId: InboxId; inbox: Inbox }) {
      for (const [_, inboxList] of this.inboxListByUser) {
        const i = inboxList.findIndex((item: Inbox) => item.id == inboxId);
        if (i >= 0) {
          inboxList[i] = inbox;
        }
      }
    },
    async fetchInboxListByUser({
      userId,
      readCreatedAfterTs,
    }: {
      userId: PrincipalId;
      readCreatedAfterTs?: number;
    }) {
      let url = `/api/inbox/user/${userId}`;
      if (readCreatedAfterTs) {
        url += `?created=${readCreatedAfterTs}`;
      }
      const data = (await axios.get(url)).data;
      const inboxList = data.data.map((inbox: ResourceObject) => {
        return convert(inbox, data.included);
      });

      this.setInboxListByUser({ userId, inboxList });
      return inboxList;
    },
    async fetchInboxSummaryByUser(userId: PrincipalId) {
      const inboxSummary = (
        await axios.get(`/api/inbox/user/${userId}/summary`)
      ).data;

      this.setInboxSummaryByUser({ userId, inboxSummary });
      return inboxSummary;
    },
    async patchInbox({
      inboxId,
      inboxPatch,
    }: {
      inboxId: InboxId;
      inboxPatch: InboxPatch;
    }) {
      const data = (
        await axios.patch(`/api/inbox/${inboxId}`, {
          data: {
            type: "inboxPatch",
            attributes: inboxPatch,
          },
        })
      ).data;
      const updatedInbox = convert(data.data, data.included);

      this.updateInboxById({ inboxId, inbox: updatedInbox });

      return updatedInbox;
    },
  },
});
