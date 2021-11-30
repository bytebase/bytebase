import axios from "axios";
import {
  Activity,
  Inbox,
  InboxID,
  InboxPatch,
  InboxState,
  InboxSummary,
  PrincipalID,
  ResourceIdentifier,
  ResourceObject,
  unknown,
} from "../../types";

function convert(
  inbox: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Inbox {
  const activityID = (inbox.relationships!.activity.data as ResourceIdentifier)
    .id;
  let activity: Activity = unknown("ACTIVITY") as Activity;
  activity.id = parseInt(activityID);

  for (const item of includedList || []) {
    if (item.type == "activity" && item.id == activityID) {
      activity = rootGetters["activity/convert"](item, includedList);
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
    (userID: PrincipalID): Inbox[] => {
      return state.inboxListByUser.get(userID) || [];
    },

  inboxSummaryByUser:
    (state: InboxState) =>
    (userID: PrincipalID): InboxSummary => {
      return (
        state.inboxSummaryByUser.get(userID) || {
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
      userID,
      readCreatedAfterTs,
    }: { userID: PrincipalID; readCreatedAfterTs?: number }
  ) {
    let url = `/api/inbox?user=${userID}`;
    if (readCreatedAfterTs) {
      url += `&created=${readCreatedAfterTs}`;
    }
    const data = (await axios.get(url)).data;
    const inboxList = data.data.map((inbox: ResourceObject) => {
      return convert(inbox, data.included, rootGetters);
    });

    commit("setInboxListByUser", { userID, inboxList });
    return inboxList;
  },

  async fetchInboxSummaryByUser({ commit }: any, userID: PrincipalID) {
    const inboxSummary = (await axios.get(`/api/inbox/summary?user=${userID}`))
      .data;

    commit("setInboxSummaryByUser", { userID, inboxSummary });
    return inboxSummary;
  },

  async patchInbox(
    { commit, rootGetters }: any,
    { inboxID, inboxPatch }: { inboxID: InboxID; inboxPatch: InboxPatch }
  ) {
    const data = (
      await axios.patch(`/api/inbox/${inboxID}`, {
        data: {
          type: "inboxPatch",
          attributes: inboxPatch,
        },
      })
    ).data;
    const updatedInbox = convert(data.data, data.included, rootGetters);

    commit("updateInboxByID", { inboxID, inbox: updatedInbox });

    return updatedInbox;
  },
};

const mutations = {
  setInboxListByUser(
    state: InboxState,
    {
      userID,
      inboxList,
    }: {
      userID: PrincipalID;
      inboxList: Inbox[];
    }
  ) {
    state.inboxListByUser.set(userID, inboxList);
  },

  setInboxSummaryByUser(
    state: InboxState,
    {
      userID,
      inboxSummary,
    }: {
      userID: PrincipalID;
      inboxSummary: InboxSummary;
    }
  ) {
    state.inboxSummaryByUser.set(userID, inboxSummary);
  },

  updateInboxByID(
    state: InboxState,
    {
      inboxID,
      inbox,
    }: {
      inboxID: InboxID;
      inbox: Inbox;
    }
  ) {
    for (const [_, inboxList] of state.inboxListByUser) {
      const i = inboxList.findIndex((item: Inbox) => item.id == inboxID);
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
