import { defineStore } from "pinia";
import axios from "axios";
import {
  PrincipalId,
  IssueSubscriber,
  IssueSubscriberState,
  ResourceObject,
  IssueId,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  issueSubscriber: ResourceObject,
  includedList: ResourceObject[]
): IssueSubscriber {
  return {
    issueId: issueSubscriber.attributes.issueId as IssueId,
    subscriber: getPrincipalFromIncludedList(
      issueSubscriber.relationships!.subscriber.data,
      includedList
    ),
  };
}

export const useIssueSubscriberStore = defineStore("issueSubscriber", {
  state: (): IssueSubscriberState => ({
    subscriberList: new Map(),
  }),

  actions: {
    subscriberListByIssue(issueId: IssueId): IssueSubscriber[] {
      return this.subscriberList.get(issueId) || [];
    },
    async fetchSubscriberListByIssue(issueId: IssueId) {
      const data = (await axios.get(`/api/issue/${issueId}/subscriber`)).data;
      const subscriberList = data.data.map(
        (issueSubscriber: ResourceObject) => {
          return convert(issueSubscriber, data.included);
        }
      );
      this.setSubscriberListByIssueId({
        issueId,
        subscriberList,
      });
      return subscriberList;
    },

    async createSubscriber({
      issueId,
      subscriberId,
    }: {
      issueId: IssueId;
      subscriberId: PrincipalId;
    }) {
      const data = (
        await axios.post(`/api/issue/${issueId}/subscriber`, {
          data: {
            type: "issueSubscriber",
            attributes: {
              subscriberId,
            },
          },
        })
      ).data;
      const createdIssueSubscriber = convert(data.data, data.included);

      this.upsertSubsriberListByIssueId({
        issueId,
        subscriber: createdIssueSubscriber,
      });

      return createdIssueSubscriber;
    },

    async deleteSubscriber({
      issueId,
      subscriberId,
    }: {
      issueId: IssueId;
      subscriberId: PrincipalId;
    }) {
      await axios.delete(`/api/issue/${issueId}/subscriber/${subscriberId}`);

      this.deleteIssueSubscriberByIssueId({ issueId, subscriberId });
    },
    setSubscriberListByIssueId({
      issueId,
      subscriberList,
    }: {
      issueId: IssueId;
      subscriberList: IssueSubscriber[];
    }) {
      this.subscriberList.set(issueId, subscriberList);
    },

    upsertSubsriberListByIssueId({
      issueId,
      subscriber,
    }: {
      issueId: IssueId;
      subscriber: IssueSubscriber;
    }) {
      const list = this.subscriberList.get(issueId);
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
        this.subscriberList.set(issueId, [subscriber]);
      }
    },

    deleteIssueSubscriberByIssueId({
      issueId,
      subscriberId,
    }: {
      issueId: IssueId;
      subscriberId: PrincipalId;
    }) {
      const list = this.subscriberList.get(issueId);
      if (list) {
        const i = list.findIndex(
          (item: IssueSubscriber) => item.subscriber.id == subscriberId
        );
        if (i != -1) {
          list.splice(i, 1);
        }
      }
    },
  },
});
