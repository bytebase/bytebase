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
} from "../../types";
import { useActivityStore } from "../pinia-modules";

function convert(
  inbox: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Inbox {
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

const state: () => InboxState = () => ({
  inboxListByUser: new Map(),
  inboxSummaryByUser: new Map(),
});

const getters = {
  inboxListByUser:
    (state: InboxState) =>
    (userId: PrincipalId): Inbox[] => {
      return state.inboxListByUser.get(userId) || [];
    },

  inboxSummaryByUser:
    (state: InboxState) =>
    (userId: PrincipalId): InboxSummary => {
      return (
        state.inboxSummaryByUser.get(userId) || {
          hasUnread: false,
          hasUnreadError: false,
        }
      );
    },
};

const actions = {
  async fetchInboxListByUser(
    { commit, rootGetters }: any,
    {
      userId,
      readCreatedAfterTs,
    }: { userId: PrincipalId; readCreatedAfterTs?: number }
  ) {
    let url = `/api/inbox/user/${userId}`;
    if (readCreatedAfterTs) {
      url += `?created=${readCreatedAfterTs}`;
    }
    const data = (await axios.get(url)).data;
    const inboxList = data.data.map((inbox: ResourceObject) => {
      return convert(inbox, data.included, rootGetters);
    });

    commit("setInboxListByUser", { userId, inboxList });
    return inboxList;
  },

  async fetchInboxSummaryByUser({ commit }: any, userId: PrincipalId) {
    const inboxSummary = (await axios.get(`/api/inbox/user/${userId}/summary`))
      .data;

    commit("setInboxSummaryByUser", { userId, inboxSummary });
    return inboxSummary;
  },

  async patchInbox(
    { commit, rootGetters }: any,
    { inboxId, inboxPatch }: { inboxId: InboxId; inboxPatch: InboxPatch }
  ) {
    const data = (
      await axios.patch(`/api/inbox/${inboxId}`, {
        data: {
          type: "inboxPatch",
          attributes: inboxPatch,
        },
      })
    ).data;
    const updatedInbox = convert(data.data, data.included, rootGetters);

    commit("updateInboxById", { inboxId, inbox: updatedInbox });

    return updatedInbox;
  },
};

const mutations = {
  setInboxListByUser(
    state: InboxState,
    {
      userId,
      inboxList,
    }: {
      userId: PrincipalId;
      inboxList: Inbox[];
    }
  ) {
    state.inboxListByUser.set(userId, inboxList);
  },

  setInboxSummaryByUser(
    state: InboxState,
    {
      userId,
      inboxSummary,
    }: {
      userId: PrincipalId;
      inboxSummary: InboxSummary;
    }
  ) {
    state.inboxSummaryByUser.set(userId, inboxSummary);
  },

  updateInboxById(
    state: InboxState,
    {
      inboxId,
      inbox,
    }: {
      inboxId: InboxId;
      inbox: Inbox;
    }
  ) {
    for (const [_, inboxList] of state.inboxListByUser) {
      const i = inboxList.findIndex((item: Inbox) => item.id == inboxId);
      if (i >= 0) {
        inboxList[i] = inbox;
      }
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
