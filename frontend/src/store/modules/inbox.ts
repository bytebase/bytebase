import { defineStore } from "pinia";
import axios from "axios";
import {
  Inbox,
  InboxId,
  InboxPatch,
  InboxState,
  InboxSummary,
  PrincipalId,
  ResourceObject,
} from "@/types";
import { useActivityV1Store } from "./v1";

async function convert(raw: ResourceObject): Promise<Inbox> {
  const inbox: Inbox = {
    ...(raw.attributes as Omit<Inbox, "id">),
    id: parseInt(raw.id),
  };

  try {
    const activity = await useActivityV1Store().fetchActivityByUID(
      inbox.activityId
    );
    inbox.activity = activity;
  } finally {
    return inbox;
  }
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
      const { data } = (await axios.get(url)).data;
      const inboxList = await Promise.all(
        (data as ResourceObject[]).map((inbox: ResourceObject) => {
          return convert(inbox);
        })
      );

      this.setInboxListByUser({
        userId,
        inboxList,
      });
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
      const updatedInbox = await convert(data.data);
      this.updateInboxById({ inboxId, inbox: updatedInbox });

      return updatedInbox;
    },
  },
});
