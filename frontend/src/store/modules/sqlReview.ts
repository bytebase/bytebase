import { pullAt } from "lodash-es";
import {
  empty,
  unknown,
  Policy,
  PolicyId,
  EnvironmentId,
  RowStatus,
  EMPTY_ID,
  Environment,
  PolicyUpsert,
  SchemaPolicyRule,
  SQLReviewPolicyPayload,
  SQLReviewPolicy,
} from "@/types";
import { defineStore } from "pinia";
import { usePolicyStore } from "./policy";

const convertToSQLReviewPolicy = (
  policy: Policy
): SQLReviewPolicy | undefined => {
  if (policy.type !== "bb.policy.sql-review") {
    return;
  }
  const payload = policy.payload as SQLReviewPolicyPayload;
  if (!Array.isArray(payload.ruleList) || !payload.name) {
    return;
  }

  const ruleList = payload.ruleList.map((r) => {
    const rule: SchemaPolicyRule = {
      type: r.type,
      level: r.level,
    };
    if (r.payload && r.payload !== "{}") {
      rule.payload = JSON.parse(r.payload);
    }
    return rule;
  });

  return {
    id: policy.id,
    creator: policy.creator,
    createdTs: policy.createdTs,
    updater: policy.updater,
    updatedTs: policy.updatedTs,
    rowStatus: policy.rowStatus,
    environment: policy.environment,
    name: payload.name,
    ruleList,
  };
};

interface SQLReviewState {
  reviewPolicyList: SQLReviewPolicy[];
}

export const useSQLReviewStore = defineStore("sqlReview", {
  state: (): SQLReviewState => ({
    reviewPolicyList: [],
  }),
  actions: {
    setReviewPolicy(reviewPolicy: SQLReviewPolicy) {
      const index = this.reviewPolicyList.findIndex(
        (r) => r.id === reviewPolicy.id
      );
      if (index < 0) {
        this.reviewPolicyList.push(reviewPolicy);
      } else {
        this.reviewPolicyList = [
          ...this.reviewPolicyList.slice(0, index),
          {
            ...this.reviewPolicyList[index],
            ...reviewPolicy,
          },
          ...this.reviewPolicyList.slice(index + 1),
        ];
      }
    },
    availableEnvironments(
      environmentList: Environment[],
      reviewPolicyId: PolicyId | undefined
    ): Environment[] {
      const envMap = environmentList.reduce((map, env) => {
        map.set(env.id, env);
        return map;
      }, new Map<number, Environment>());

      for (const reviewPolicy of this.reviewPolicyList) {
        if (
          reviewPolicy.id === reviewPolicyId ||
          !reviewPolicy.environment.id
        ) {
          continue;
        }
        if (envMap.has(reviewPolicy.environment.id)) {
          envMap.delete(reviewPolicy.environment.id);
        }
      }

      return [...envMap.values()];
    },
    async addReviewPolicy({
      name,
      environmentId,
      ruleList,
    }: {
      name: string;
      environmentId: number;
      ruleList: SchemaPolicyRule[];
    }) {
      const payload: SQLReviewPolicyPayload = {
        name,
        ruleList: ruleList.map((r) => ({
          ...r,
          payload: r.payload ? JSON.stringify(r.payload) : "{}",
        })),
      };

      const policyStore = usePolicyStore();
      const policy = await policyStore.upsertPolicyByEnvironmentAndType({
        environmentId,
        type: "bb.policy.sql-review",
        policyUpsert: { payload },
      });

      const reviewPolicy = convertToSQLReviewPolicy(policy);
      if (!reviewPolicy) {
        throw new Error(`invalid policy ${JSON.stringify(policy)}`);
      }

      this.setReviewPolicy(reviewPolicy);
    },
    async removeReviewPolicy(id: PolicyId) {
      const index = this.reviewPolicyList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }

      const targetPolicy = this.reviewPolicyList[index];
      const policyStore = usePolicyStore();
      await policyStore.deletePolicyByEnvironmentAndType({
        environmentId: targetPolicy.environment.id,
        type: "bb.policy.sql-review",
      });

      pullAt(this.reviewPolicyList, index);
    },
    async updateReviewPolicy({
      id,
      name,
      rowStatus,
      ruleList,
    }: {
      id: PolicyId;
      name?: string;
      rowStatus?: RowStatus;
      ruleList?: SchemaPolicyRule[];
    }) {
      const index = this.reviewPolicyList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }

      const targetPolicy = this.reviewPolicyList[index];

      const policyUpsert: PolicyUpsert = {};
      if (rowStatus) {
        policyUpsert.rowStatus = rowStatus;
      }
      if (name && ruleList) {
        const payload: SQLReviewPolicyPayload = {
          name,
          ruleList: ruleList.map((r) => ({
            ...r,
            payload: r.payload ? JSON.stringify(r.payload) : "{}",
          })),
        };
        policyUpsert.payload = payload;
      }

      const policyStore = usePolicyStore();
      const policy = await policyStore.upsertPolicyByEnvironmentAndType({
        environmentId: targetPolicy.environment.id,
        type: "bb.policy.sql-review",
        policyUpsert,
      });

      const reviewPolicy = convertToSQLReviewPolicy(policy);
      this.reviewPolicyList = [
        ...this.reviewPolicyList.slice(0, index),
        {
          ...this.reviewPolicyList[index],
          ...reviewPolicy,
        },
        ...this.reviewPolicyList.slice(index + 1),
      ];
    },
    getReviewPolicyByEnvironmentId(
      environmentId: EnvironmentId
    ): SQLReviewPolicy | undefined {
      if (environmentId === EMPTY_ID) {
        return empty("SQL_REVIEW") as SQLReviewPolicy;
      }

      return this.reviewPolicyList.find(
        (g) => g.environment.id === environmentId
      );
    },

    async fetchReviewPolicyList(): Promise<SQLReviewPolicy[]> {
      const policyStore = usePolicyStore();
      const policyList = await policyStore.fetchPolicyListByType(
        "bb.policy.sql-review"
      );

      const reviewPolicyList = policyList.reduce((list, policy) => {
        const reviewPolicy = convertToSQLReviewPolicy(policy);
        if (reviewPolicy) {
          list.push(reviewPolicy);
        }
        return list;
      }, [] as SQLReviewPolicy[]);
      this.reviewPolicyList = reviewPolicyList;
      return reviewPolicyList;
    },
    async fetchReviewPolicyByEnvironmentId(
      environmentId: EnvironmentId
    ): Promise<SQLReviewPolicy | undefined> {
      const policyStore = usePolicyStore();
      const policy = await policyStore.fetchPolicyByEnvironmentAndType({
        environmentId: environmentId,
        type: "bb.policy.sql-review",
      });

      if (!policy) {
        return;
      }
      const reviewPolicy = convertToSQLReviewPolicy(policy);
      if (reviewPolicy) {
        this.setReviewPolicy(reviewPolicy);
      }
      return reviewPolicy;
    },
  },
});
