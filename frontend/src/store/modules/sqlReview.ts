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
  RuleType,
  RuleLevel,
} from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { Environment } from "@/types/proto/v1/environment_service";
import {
  PolicyType,
  Policy,
  SQLReviewRuleLevel,
  policyTypeToJSON,
  SQLReviewPolicy as SQLReviewPolicyV1,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { useEnvironmentV1Store } from "./v1/environment";
import { usePolicyV1Store } from "./v1/policy";

const getEnvironmentById = async (
  environmentId: IdType
): Promise<Environment> => {
  const environmentV1Store = useEnvironmentV1Store();
  const environment = await environmentV1Store.getOrFetchEnvironmentByName(
    `${environmentNamePrefix}${environmentId}`
  );
  return environment;
};

const convertToSQLReviewPolicy = async (
  policy: Policy
): Promise<SQLReviewPolicy | undefined> => {
  if (policy.type !== PolicyType.SQL_REVIEW || !policy.sqlReviewPolicy) {
    return;
  }

  const ruleList = policy.sqlReviewPolicy.rules.map((r) => {
    let level = RuleLevel.DISABLED;
    switch (r.level) {
      case SQLReviewRuleLevel.WARNING:
        level = RuleLevel.WARNING;
        break;
      case SQLReviewRuleLevel.ERROR:
        level = RuleLevel.ERROR;
        break;
    }
    const rule: SchemaPolicyRule = {
      type: r.type as RuleType,
      level: level,
      comment: r.comment,
    };
    if (r.payload && r.payload !== "{}") {
      rule.payload = JSON.parse(r.payload);
    }
    return rule;
  });

  const environment = await getEnvironmentById(policy.resourceUid);

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
          let level = SQLReviewRuleLevel.DISABLED;
          switch (r.level) {
            case RuleLevel.WARNING:
              level = SQLReviewRuleLevel.WARNING;
              break;
            case RuleLevel.ERROR:
              level = SQLReviewRuleLevel.ERROR;
              break;
          }

          return {
            type: r.type as string,
            level,
            engine: Engine.ENGINE_UNSPECIFIED,
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
            let level = SQLReviewRuleLevel.DISABLED;
            switch (r.level) {
              case RuleLevel.WARNING:
                level = SQLReviewRuleLevel.WARNING;
                break;
              case RuleLevel.ERROR:
                level = SQLReviewRuleLevel.ERROR;
                break;
            }

            return {
              type: r.type as string,
              level,
              // TODO:
              engine: Engine.ENGINE_UNSPECIFIED,
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
    getReviewPolicyByEnvironmentUID(
      environmentUID: string
    ): SQLReviewPolicy | undefined {
      return this.reviewPolicyList.find(
        (g) => g.environment.uid == environmentUID
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
    async getOrFetchReviewPolicyByEnvironmentUID(
      environmentUID: string
    ): Promise<SQLReviewPolicy | undefined> {
      const environment = await getEnvironmentById(environmentUID);
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
  },
});

export const useSQLReviewPolicyList = () => {
  const store = useSQLReviewStore();

  watchEffect(() => {
    store.fetchReviewPolicyList();
  });

  return computed(() => store.reviewPolicyList);
};

export const useReviewPolicyByEnvironmentId = (
  environmentId: MaybeRef<string>
) => {
  const store = useSQLReviewStore();
  watchEffect(() => {
    store.getOrFetchReviewPolicyByEnvironmentUID(unref(environmentId));
  });

  return computed(() =>
    store.getReviewPolicyByEnvironmentUID(unref(environmentId))
  );
};
