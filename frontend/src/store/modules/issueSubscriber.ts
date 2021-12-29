import axios from "axios";
import {
  PrincipalId,
  IssueSubscriber,
  IssueSubscriberState,
  ResourceObject,
  Principal,
  IssueId,
} from "../../types";

function convert(
  issueSubscriber: ResourceObject,
  rootGetters: any
): IssueSubscriber {
  return {
    issueId: issueSubscriber.attributes.issueId as IssueId,
    subscriber: issueSubscriber.attributes.subscriber as Principal,
  };
}

const state: () => IssueSubscriberState = () => ({
  subscriberListByIssue: new Map(),
});

const getters = {
  subscriberListByIssue:
    (state: IssueSubscriberState) =>
    (issueId: IssueId): IssueSubscriber[] => {
      return state.subscriberListByIssue.get(issueId) || [];
    },
};

const actions = {
  async fetchSubscriberListByIssue(
    { commit, rootGetters }: any,
    issueId: IssueId
  ) {
    const subscriberList = (
      await axios.get(`/api/issue/${issueId}/subscriber`)
    ).data.data.map((issueSubscriber: ResourceObject) => {
      return convert(issueSubscriber, rootGetters);
    });
    commit("setSubscriberListByIssueId", {
      issueId,
      subscriberList,
    });
    return subscriberList;
  },

  async createSubscriber(
    { commit, rootGetters }: any,
    {
      issueId,
      subscriberId,
    }: {
      issueId: IssueId;
      subscriberId: PrincipalId;
    }
  ) {
    const createdIssueSubscriber = convert(
      (
        await axios.post(`/api/issue/${issueId}/subscriber`, {
          data: {
            type: "issueSubscriber",
            attributes: {
              subscriberId,
            },
          },
        })
      ).data.data,
      rootGetters
    );

    commit("upsertSubsriberListByIssueId", {
      issueId,
      subscriber: createdIssueSubscriber,
    });

    return createdIssueSubscriber;
  },

  async deleteSubscriber(
    { commit }: any,
    {
      issueId,
      subscriberId,
    }: {
      issueId: IssueId;
      subscriberId: PrincipalId;
    }
  ) {
    await axios.delete(`/api/issue/${issueId}/subscriber/${subscriberId}`);

    commit("deleteIssueSubscriberByIssueId", { issueId, subscriberId });
  },
};

const mutations = {
  setSubscriberListByIssueId(
    state: IssueSubscriberState,
    {
      issueId,
      subscriberList,
    }: {
      issueId: IssueId;
      subscriberList: IssueSubscriber[];
    }
  ) {
    state.subscriberListByIssue.set(issueId, subscriberList);
  },

  upsertSubsriberListByIssueId(
    state: IssueSubscriberState,
    {
      issueId,
      subscriber,
    }: {
      issueId: IssueId;
      subscriber: IssueSubscriber;
    }
  ) {
    const list = state.subscriberListByIssue.get(issueId);
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
      state.subscriberListByIssue.set(issueId, [subscriber]);
    }
  },

  deleteIssueSubscriberByIssueId(
    state: IssueSubscriberState,
    {
      issueId,
      subscriberId,
    }: {
      issueId: IssueId;
      subscriberId: PrincipalId;
    }
  ) {
    const list = state.subscriberListByIssue.get(issueId);
    if (list) {
      const i = list.findIndex(
        (item: IssueSubscriber) => item.subscriber.id == subscriberId
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
