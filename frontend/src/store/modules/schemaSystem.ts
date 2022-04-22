import { pullAt } from "lodash-es";
import {
  empty,
  unknown,
  Policy,
  PolicyId,
  RowStatus,
  EMPTY_ID,
  Environment,
  PolicyPatch,
  SchemaPolicyRule,
  PolicySchemaReviewPayload,
  DatabaseSchemaReviewPolicy,
} from "../../types";
import { defineStore } from "pinia";
import { usePolicyStore } from "./policy";

const convertToSchemaReviewPolicy = (
  policy: Policy
): DatabaseSchemaReviewPolicy | undefined => {
  if (policy.type !== "bb.policy.schema-review") {
    return;
  }
  const payload = policy.payload as PolicySchemaReviewPayload;
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

interface SchemaSystemState {
  reviewPolicyList: DatabaseSchemaReviewPolicy[];
}

export const useSchemaSystemStore = defineStore("schemaSystem", {
  state: (): SchemaSystemState => ({
    reviewPolicyList: [],
  }),
  actions: {
    setReviewPolicy(reviewPolicy: DatabaseSchemaReviewPolicy) {
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
      const payload: PolicySchemaReviewPayload = {
        name,
        ruleList: ruleList.map((r) => ({
          ...r,
          payload: r.payload ? JSON.stringify(r.payload) : "{}",
        })),
      };

      const policyStore = usePolicyStore();
      const policy = await policyStore.upsertPolicyByEnvironmentAndType({
        environmentId,
        type: "bb.policy.schema-review",
        policyUpsert: { payload },
      });

      const reviewPolicy = convertToSchemaReviewPolicy(policy);
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

      const policyStore = usePolicyStore();
      await policyStore.deletePolicyByIdAndType({
        id,
        type: "bb.policy.schema-review",
      });

      pullAt(this.reviewPolicyList, index);
    },
    async updateReviewPolicy({
      id,
      name,
      rowStatus,
      environmentId,
      ruleList,
    }: {
      id: PolicyId;
      name?: string;
      rowStatus?: RowStatus;
      environmentId?: number;
      ruleList?: SchemaPolicyRule[];
    }) {
      const index = this.reviewPolicyList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }

      const policyPatch: PolicyPatch = {};
      if (rowStatus) {
        policyPatch.rowStatus = rowStatus;
      }
      if (environmentId) {
        policyPatch.environmentId = environmentId;
      }

      if (name && ruleList) {
        const payload: PolicySchemaReviewPayload = {
          name,
          ruleList: ruleList.map((r) => ({
            ...r,
            payload: r.payload ? JSON.stringify(r.payload) : "{}",
          })),
        };
        policyPatch.payload = payload;
      }

      const policyStore = usePolicyStore();
      const policy = await policyStore.patchPolicyByIdAndType({
        id,
        type: "bb.policy.schema-review",
        policyPatch,
      });

      const reviewPolicy = convertToSchemaReviewPolicy(policy);
      this.reviewPolicyList = [
        ...this.reviewPolicyList.slice(0, index),
        {
          ...this.reviewPolicyList[index],
          ...reviewPolicy,
        },
        ...this.reviewPolicyList.slice(index + 1),
      ];
    },
    getReviewPolicyById(id: PolicyId): DatabaseSchemaReviewPolicy {
      if (id === EMPTY_ID) {
        return empty("SCHEMA_REVIEW") as DatabaseSchemaReviewPolicy;
      }

      return (
        this.reviewPolicyList.find((g) => g.id === id) ||
        (unknown("SCHEMA_REVIEW") as DatabaseSchemaReviewPolicy)
      );
    },

    async fetchReviewPolicyList(): Promise<DatabaseSchemaReviewPolicy[]> {
      const policyStore = usePolicyStore();
      const policyList = await policyStore.fetchPolicyByType(
        "bb.policy.schema-review"
      );

      const reviewPolicyList = policyList.reduce((list, policy) => {
        const reviewPolicy = convertToSchemaReviewPolicy(policy);
        if (reviewPolicy) {
          list.push(reviewPolicy);
        }
        return list;
      }, [] as DatabaseSchemaReviewPolicy[]);
      this.reviewPolicyList = reviewPolicyList;
      return reviewPolicyList;
    },
    async fetchReviewPolicyById(
      id: PolicyId
    ): Promise<DatabaseSchemaReviewPolicy | undefined> {
      const policyStore = usePolicyStore();
      const policy = await policyStore.fetchPolicyById(id);

      const reviewPolicy = convertToSchemaReviewPolicy(policy);
      if (reviewPolicy) {
        this.setReviewPolicy(reviewPolicy);
      }
      return reviewPolicy;
    },
  },
});
