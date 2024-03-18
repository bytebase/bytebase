import { pullAt } from "lodash-es";
import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import {
  policyNamePrefix,
  environmentNamePrefix,
} from "@/store/modules/v1/common";
import {
  PolicyId,
  SchemaPolicyRule,
  SQLReviewPolicy,
  IdType,
  MaybeRef,
} from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import {
  PolicyType,
  Policy,
  policyTypeToJSON,
  SQLReviewPolicy as SQLReviewPolicyV1,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { extractEnvironmentResourceName } from "@/utils";
import { useEnvironmentV1Store } from "./v1/environment";
import { usePolicyV1Store } from "./v1/policy";

const convertToSQLReviewPolicy = async (
  policy: Policy
): Promise<SQLReviewPolicy | undefined> => {
  if (policy.type !== PolicyType.SQL_REVIEW || !policy.sqlReviewPolicy) {
    return;
  }

  const ruleList: SchemaPolicyRule[] = [];
  for (const r of policy.sqlReviewPolicy.rules) {
    const rule: SchemaPolicyRule = {
      type: r.type,
      level: r.level,
      engine: r.engine,
      comment: r.comment,
    };
    if (r.payload && r.payload !== "{}") {
      rule.payload = JSON.parse(r.payload);
    }
    ruleList.push(rule);
  }

  const environment = await useEnvironmentV1Store().getOrFetchEnvironmentByName(
    `${environmentNamePrefix}${extractEnvironmentResourceName(policy.name)}`
  );

  return {
    id: policy.name,
    name: policy.sqlReviewPolicy.name,
    environment,
    ruleList,
    enforce: policy.enforce,
  };
};

interface SQLReviewState {
  reviewPolicyList: SQLReviewPolicy[];
}

const getSQLReviewPolicyName = (environmentPath: string): string => {
  return `${environmentPath}/${policyNamePrefix}${policyTypeToJSON(
    PolicyType.SQL_REVIEW
  ).toLowerCase()}`;
};

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
        map.set(env.name, env);
        return map;
      }, new Map<IdType, Environment>());

      for (const reviewPolicy of this.reviewPolicyList) {
        if (reviewPolicy.id === reviewPolicyId || !reviewPolicy.environment) {
          continue;
        }
        if (envMap.has(reviewPolicy.environment.name)) {
          envMap.delete(reviewPolicy.environment.name);
        }
      }

      return [...envMap.values()];
    },
    async addReviewPolicy({
      name,
      environmentPath,
      ruleList,
    }: {
      name: string;
      environmentPath: string;
      ruleList: SchemaPolicyRule[];
    }) {
      const sqlReviewPolicy: SQLReviewPolicyV1 = {
        name,
        rules: ruleList.map((r) => {
          return {
            type: r.type as string,
            level: r.level,
            engine: r.engine,
            comment: r.comment,
            payload: r.payload ? JSON.stringify(r.payload) : "{}",
          };
        }),
      };

      const policyStore = usePolicyV1Store();
      const policy = await policyStore.createPolicy(environmentPath, {
        type: PolicyType.SQL_REVIEW,
        sqlReviewPolicy,
        inheritFromParent: true,
      });

      const reviewPolicy = await convertToSQLReviewPolicy(policy);
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
      const policyStore = usePolicyV1Store();
      await policyStore.deletePolicy(
        getSQLReviewPolicyName(targetPolicy.environment.name)
      );

      pullAt(this.reviewPolicyList, index);
    },
    async updateReviewPolicy({
      id,
      name,
      enforce,
      ruleList,
    }: {
      id: PolicyId;
      name?: string;
      enforce?: boolean;
      ruleList?: SchemaPolicyRule[];
    }) {
      const index = this.reviewPolicyList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }

      const targetPolicy = this.reviewPolicyList[index];
      const policyStore = usePolicyV1Store();

      const policy = await policyStore.getOrFetchPolicyByName(
        getSQLReviewPolicyName(targetPolicy.environment.name)
      );
      if (!policy) {
        return;
      }

      const updateMask: string[] = [];
      if (enforce !== undefined) {
        updateMask.push("enforce");
        policy.enforce = enforce;
      }
      if (name && ruleList) {
        updateMask.push("payload");
        policy.sqlReviewPolicy = {
          name,
          rules: ruleList.map((r) => {
            return {
              type: r.type as string,
              level: r.level,
              engine: r.engine,
              comment: r.comment,
              payload: r.payload ? JSON.stringify(r.payload) : "{}",
            };
          }),
        };
      }

      const updatedPolicy = await policyStore.updatePolicy(updateMask, policy);
      const reviewPolicy = await convertToSQLReviewPolicy(updatedPolicy);
      if (reviewPolicy) {
        this.setReviewPolicy(reviewPolicy);
      }
    },
    getReviewPolicyByEnvironmentName(
      name: string
    ): SQLReviewPolicy | undefined {
      return this.reviewPolicyList.find((g) => g.environment.name === name);
    },
    getReviewPolicyByEnvironmentUID(
      environmentId: string
    ): SQLReviewPolicy | undefined {
      return this.getReviewPolicyByEnvironmentName(
        `${environmentNamePrefix}${environmentId}`
      );
    },

    async fetchReviewPolicyList(): Promise<SQLReviewPolicy[]> {
      const policyStore = usePolicyV1Store();
      const policyList = await policyStore.fetchPolicies({
        resourceType: PolicyResourceType.ENVIRONMENT,
        policyType: PolicyType.SQL_REVIEW,
        showDeleted: true,
      });

      const reviewPolicyList: SQLReviewPolicy[] = [];
      for (const policy of policyList) {
        const reviewPolicy = await convertToSQLReviewPolicy(policy);
        if (reviewPolicy) {
          reviewPolicyList.push(reviewPolicy);
        }
      }
      this.reviewPolicyList = reviewPolicyList;
      return reviewPolicyList;
    },
    async getOrFetchReviewPolicyByEnvironmentName(
      name: string
    ): Promise<SQLReviewPolicy | undefined> {
      const environmentV1Store = useEnvironmentV1Store();
      const environment = await environmentV1Store.getOrFetchEnvironmentByName(
        name,
        true /* silent */
      );
      const policyStore = usePolicyV1Store();
      const policy = await policyStore.getOrFetchPolicyByName(
        getSQLReviewPolicyName(environment.name)
      );

      if (!policy) {
        return;
      }
      const reviewPolicy = await convertToSQLReviewPolicy(policy);
      if (reviewPolicy) {
        this.setReviewPolicy(reviewPolicy);
      }
      return reviewPolicy;
    },
    async getOrFetchReviewPolicyByEnvironmentUID(
      uid: string
    ): Promise<SQLReviewPolicy | undefined> {
      return this.getOrFetchReviewPolicyByEnvironmentName(
        `${environmentNamePrefix}${uid}`
      );
    },
  },
});

export const useSQLReviewPolicyList = () => {
  const store = useSQLReviewStore();

  watchEffect(() => {
    store.fetchReviewPolicyList();
  });

  return computed(() => store.reviewPolicyList);
};

export const useReviewPolicyByEnvironmentName = (
  name: MaybeRef<string | undefined>
) => {
  const store = useSQLReviewStore();
  watchEffect(() => {
    if (!unref(name)) return;
    store.getOrFetchReviewPolicyByEnvironmentName(unref(name)!);
  });

  return computed(() => {
    if (!unref(name)) return undefined;
    return store.getReviewPolicyByEnvironmentName(unref(name)!);
  });
};
