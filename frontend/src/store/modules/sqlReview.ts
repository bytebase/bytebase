import { pullAt } from "lodash-es";
import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { reviewConfigServiceClient } from "@/grpcweb";
import {
  policyNamePrefix,
  environmentNamePrefix,
} from "@/store/modules/v1/common";
import type {
  SchemaPolicyRule,
  SQLReviewPolicy,
  IdType,
  MaybeRef,
} from "@/types";
import type { Environment } from "@/types/proto/v1/environment_service";
import {
  PolicyType,
  policyTypeToJSON,
} from "@/types/proto/v1/org_policy_service";
import { ReviewConfig } from "@/types/proto/v1/review_config_service";
import { useEnvironmentV1Store } from "./v1/environment";
import { usePolicyV1Store } from "./v1/policy";

const reviewConfigTagName = "bb.tag.review_config";

const convertToSQLReviewPolicy = async (
  reviewConfig: ReviewConfig
): Promise<SQLReviewPolicy | undefined> => {
  const environmentName = reviewConfig.resources.find((resource) =>
    resource.startsWith(environmentNamePrefix)
  );
  if (!environmentName) {
    return;
  }

  const ruleList: SchemaPolicyRule[] = [];
  for (const r of reviewConfig.rules) {
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

  const environment =
    await useEnvironmentV1Store().getOrFetchEnvironmentByName(environmentName);

  return {
    id: reviewConfig.name,
    name: reviewConfig.title,
    environment,
    ruleList,
    enforce: reviewConfig.enabled,
  };
};

interface SQLReviewState {
  reviewPolicyList: SQLReviewPolicy[];
}

const getTagPolicyName = (environmentPath: string): string => {
  return `${environmentPath}/${policyNamePrefix}${policyTypeToJSON(
    PolicyType.TAG
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
      reviewPolicyId: string | undefined
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
    async createReviewPolicy({
      id,
      title,
      environmentPath,
      ruleList,
    }: {
      id: string;
      title: string;
      environmentPath: string;
      ruleList: SchemaPolicyRule[];
    }) {
      const reviewConfig = await reviewConfigServiceClient.createReviewConfig({
        reviewConfig: {
          name: id,
          title,
          rules: ruleList.map((r) => {
            return {
              type: r.type as string,
              level: r.level,
              engine: r.engine,
              comment: r.comment,
              payload: r.payload ? JSON.stringify(r.payload) : "{}",
            };
          }),
          enabled: true,
        },
      });

      const policyStore = usePolicyV1Store();
      await policyStore.upsertPolicy({
        updateMask: ["payload"],
        parentPath: environmentPath,
        policy: {
          name: getTagPolicyName(environmentPath),
          type: PolicyType.TAG,
          tagPolicy: {
            tags: {
              [reviewConfigTagName]: reviewConfig.name,
            },
          },
        },
      });

      reviewConfig.resources = [environmentPath];
      const reviewPolicy = await convertToSQLReviewPolicy(reviewConfig);
      if (!reviewPolicy) {
        throw new Error(
          `invalid review config ${JSON.stringify(reviewConfig)}`
        );
      }

      this.setReviewPolicy(reviewPolicy);
    },
    async removeReviewPolicy(id: string) {
      const index = this.reviewPolicyList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }

      const targetPolicy = this.reviewPolicyList[index];
      await reviewConfigServiceClient.deleteReviewConfig({
        name: targetPolicy.id,
      });

      if (!targetPolicy.environment) {
        return;
      }

      // TODO(ed): for now, we can just simply delete the tag policy for environment.
      const policyStore = usePolicyV1Store();
      await policyStore.deletePolicy(
        getTagPolicyName(targetPolicy.environment.name)
      );

      pullAt(this.reviewPolicyList, index);
    },
    async updateReviewPolicy({
      id,
      title,
      enforce,
      ruleList,
    }: {
      id: string;
      title?: string;
      enforce?: boolean;
      ruleList?: SchemaPolicyRule[];
    }) {
      const index = this.reviewPolicyList.findIndex((g) => g.id === id);
      if (index < 0) {
        return;
      }

      const targetPolicy = this.reviewPolicyList[index];

      const patch: Partial<ReviewConfig> = {
        name: targetPolicy.id,
      };
      const updateMask: string[] = [];
      if (enforce !== undefined) {
        updateMask.push("enabled");
        patch.enabled = enforce;
      }
      if (title) {
        updateMask.push("title");
        patch.title = title;
      }
      if (ruleList) {
        updateMask.push("payload");
        patch.rules = ruleList.map((r) => {
          return {
            type: r.type as string,
            level: r.level,
            engine: r.engine,
            comment: r.comment,
            payload: r.payload ? JSON.stringify(r.payload) : "{}",
          };
        });
      }

      const updated = await reviewConfigServiceClient.updateReviewConfig({
        reviewConfig: patch,
        updateMask,
      });
      const reviewPolicy = await convertToSQLReviewPolicy(updated);
      if (reviewPolicy) {
        this.setReviewPolicy(reviewPolicy);
      }
    },
    getReviewPolicyByEnvironmentName(
      name: string
    ): SQLReviewPolicy | undefined {
      return this.reviewPolicyList.find((g) => g.environment.name === name);
    },

    async fetchReviewPolicyList(): Promise<SQLReviewPolicy[]> {
      const { reviewConfigs } =
        await reviewConfigServiceClient.listReviewConfigs({});

      const reviewPolicyList: SQLReviewPolicy[] = [];
      for (const config of reviewConfigs) {
        const reviewPolicy = await convertToSQLReviewPolicy(config);
        if (reviewPolicy) {
          reviewPolicyList.push(reviewPolicy);
        }
      }
      this.reviewPolicyList = reviewPolicyList;
      return reviewPolicyList;
    },
    async getOrFetchReviewPolicyByName(name: string) {
      const policy = this.reviewPolicyList.find((g) => g.id === name);
      if (policy) {
        return policy;
      }

      const reviewConfig = await reviewConfigServiceClient.getReviewConfig({
        name,
      });
      if (!reviewConfig) {
        return;
      }
      const reviewPolicy = await convertToSQLReviewPolicy(reviewConfig);
      if (reviewPolicy) {
        this.setReviewPolicy(reviewPolicy);
      }
      return reviewPolicy;
    },
    async getOrFetchReviewPolicyByEnvironmentName(
      name: string
    ): Promise<SQLReviewPolicy | undefined> {
      const cached = this.getReviewPolicyByEnvironmentName(name);
      if (cached) {
        return cached;
      }

      const environmentV1Store = useEnvironmentV1Store();
      const environment = await environmentV1Store.getOrFetchEnvironmentByName(
        name,
        true /* silent */
      );
      const policyStore = usePolicyV1Store();
      const policy = await policyStore.getOrFetchPolicyByName(
        getTagPolicyName(environment.name)
      );

      if (!policy) {
        return;
      }
      const sqlReviewName = policy.tagPolicy?.tags[reviewConfigTagName];
      if (!sqlReviewName) {
        return;
      }

      return this.getOrFetchReviewPolicyByName(sqlReviewName);
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
