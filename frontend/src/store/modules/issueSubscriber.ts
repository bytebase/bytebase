import axios from "axios";
import {
  PrincipalID,
  IssueSubscriber,
  IssueSubscriberState,
  ResourceObject,
  Principal,
  IssueID,
} from "../../types";

function convert(
  issueSubscriber: ResourceObject,
  rootGetters: any
): IssueSubscriber {
  return {
    issueID: issueSubscriber.attributes.issueID as IssueID,
    subscriber: issueSubscriber.attributes.subscriber as Principal,
  };
}

const state: () => IssueSubscriberState = () => ({
  subscriberListByIssue: new Map(),
});

const getters = {
  subscriberListByIssue:
    (state: IssueSubscriberState) =>
    (issueID: IssueID): IssueSubscriber[] => {
      return state.subscriberListByIssue.get(issueID) || [];
    },
};

const actions = {
  async fetchSubscriberListByIssue(
    { commit, rootGetters }: any,
    issueID: IssueID
  ) {
    const subscriberList = (
      await axios.get(`/api/issue/${issueID}/subscriber`)
    ).data.data.map((issueSubscriber: ResourceObject) => {
      return convert(issueSubscriber, rootGetters);
    });
    commit("setSubscriberListByIssueID", {
      issueID,
      subscriberList,
    });
    return subscriberList;
  },

  async createSubscriber(
    { commit, rootGetters }: any,
    {
      issueID,
      subscriberID,
    }: {
      issueID: IssueID;
      subscriberID: PrincipalID;
    }
  ) {
    const createdIssueSubscriber = convert(
      (
        await axios.post(`/api/issue/${issueID}/subscriber`, {
          data: {
            type: "issueSubscriber",
            attributes: {
              subscriberID,
            },
          },
        })
      ).data.data,
      rootGetters
    );

    commit("upsertSubsriberListByIssueID", {
      issueID,
      subscriber: createdIssueSubscriber,
    });

    return createdIssueSubscriber;
  },

  async deleteSubscriber(
    { commit }: any,
    {
      issueID,
      subscriberID,
    }: {
      issueID: IssueID;
      subscriberID: PrincipalID;
    }
  ) {
    await axios.delete(`/api/issue/${issueID}/subscriber/${subscriberID}`);

    commit("deleteIssueSubscriberByIssueID", { issueID, subscriberID });
  },
};

const mutations = {
  setSubscriberListByIssueID(
    state: IssueSubscriberState,
    {
      issueID,
      subscriberList,
    }: {
      issueID: IssueID;
      subscriberList: IssueSubscriber[];
    }
  ) {
    state.subscriberListByIssue.set(issueID, subscriberList);
  },

  upsertSubsriberListByIssueID(
    state: IssueSubscriberState,
    {
      issueID,
      subscriber,
    }: {
      issueID: IssueID;
      subscriber: IssueSubscriber;
    }
  ) {
    const list = state.subscriberListByIssue.get(issueID);
    if (list) {
      const i = list.findIndex(
        (item: IssueSubscriber) =>
          item.subscriber.id == subscriber.subscriber.id
      );
      if (i != -1) {
        list[i] = subscriber;
      } else {
        list.push(subscriber);
      }
    } else {
      state.subscriberListByIssue.set(issueID, [subscriber]);
    }
  },

  deleteIssueSubscriberByIssueID(
    state: IssueSubscriberState,
    {
      issueID,
      subscriberID,
    }: {
      issueID: IssueID;
      subscriberID: PrincipalID;
    }
  ) {
    const list = state.subscriberListByIssue.get(issueID);
    if (list) {
      const i = list.findIndex(
        (item: IssueSubscriber) => item.subscriber.id == subscriberID
      );
      if (i != -1) {
        list.splice(i, 1);
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
