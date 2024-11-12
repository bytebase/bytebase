import { pullAt, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { reviewConfigServiceClient } from "@/grpcweb";
import { policyNamePrefix } from "@/store/modules/v1/common";
import type {
  SchemaPolicyRule,
  SQLReviewPolicy,
  MaybeRef,
  ComposedDatabase,
} from "@/types";
import {
  PolicyType,
  policyTypeToJSON,
} from "@/types/proto/v1/org_policy_service";
import { ReviewConfig } from "@/types/proto/v1/review_config_service";
import { usePolicyV1Store } from "./v1/policy";

const reviewConfigTagName = "bb.tag.review_config";

const upsertReviewConfigTag = async (
  resources: string[],
  configName: string
) => {
  const policyStore = usePolicyV1Store();
  await Promise.all(
    resources.map(async (resourcePath) => {
      await policyStore.upsertPolicy({
        parentPath: resourcePath,
        policy: {
          name: getTagPolicyName(resourcePath),
          type: PolicyType.TAG,
          tagPolicy: {
            tags: {
              [reviewConfigTagName]: configName,
            },
          },
        },
      });
    })
  );
};

const removeReviewConfigTag = async (resources: string[]) => {
  const policyStore = usePolicyV1Store();
  await Promise.all(
    resources.map((resource) =>
      policyStore.deletePolicy(getTagPolicyName(resource))
    )
  );
};

const convertToSQLReviewPolicy = (
  reviewConfig: ReviewConfig
): SQLReviewPolicy | undefined => {
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

  return {
    id: reviewConfig.name,
    name: reviewConfig.title,
    resources: reviewConfig.resources,
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
    removeResourceForReview(map: Map<string, string[]>) {
      for (const [configId, resources] of map.entries()) {
        const reviewPolicy = this.reviewPolicyList.find(
          (r) => r.id === configId
        );
        if (!reviewPolicy) {
          continue;
        }
        reviewPolicy.resources = reviewPolicy.resources.filter(
          (resource) => !resources.includes(resource)
        );
      }
    },
    async upsertReviewConfigTag({
      oldResources,
      newResources,
      review,
    }: {
      oldResources: string[];
      newResources: string[];
      review: string;
    }) {
      await removeReviewConfigTag(uniq(oldResources));
      await upsertReviewConfigTag(uniq(newResources), review);

      const reviewPolicy = this.reviewPolicyList.find((r) => r.id === review);
      if (reviewPolicy) {
        reviewPolicy.resources = newResources;
      }
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

      await removeReviewConfigTag(targetPolicy.resources);

      pullAt(this.reviewPolicyList, index);
    },
    async upsertReviewPolicy({
      id,
      title,
      enforce,
      ruleList,
      resources,
    }: {
      id: string;
      title?: string;
      enforce?: boolean;
      ruleList?: SchemaPolicyRule[];
      resources?: string[];
    }) {
      const patch: Partial<ReviewConfig> = {
        name: id,
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
        updateMask.push("rules");
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
        allowMissing: true,
      });

      if (resources) {
        await this.upsertReviewConfigTag({
          oldResources: [],
          newResources: resources,
          review: updated.name,
        });
        updated.resources = resources;
      }

      const reviewPolicy = convertToSQLReviewPolicy(updated);
      if (!reviewPolicy) {
        throw new Error(`invalid review config ${JSON.stringify(updated)}`);
      }
      this.setReviewPolicy(reviewPolicy);
      return reviewPolicy;
    },
    getReviewPolicyByName(name: string) {
      return this.reviewPolicyList.find((g) => g.id === name);
    },
    getReviewPolicyByResouce(
      resourcePath: string
    ): SQLReviewPolicy | undefined {
      return this.reviewPolicyList.find((policy) => {
        return policy.resources.find((resource) => resource === resourcePath);
      });
    },

    async fetchReviewPolicyList(): Promise<SQLReviewPolicy[]> {
      const { reviewConfigs } =
        await reviewConfigServiceClient.listReviewConfigs({});

      const reviewPolicyList: SQLReviewPolicy[] = [];
      for (const config of reviewConfigs) {
        const reviewPolicy = convertToSQLReviewPolicy(config);
        if (reviewPolicy) {
          reviewPolicyList.push(reviewPolicy);
        }
      }
      this.reviewPolicyList = reviewPolicyList;
      return reviewPolicyList;
    },
    async fetchReviewPolicyByName({
      name,
      silent = false,
    }: {
      name: string;
      silent?: boolean;
    }) {
      const reviewConfig = await reviewConfigServiceClient.getReviewConfig(
        {
          name,
        },
        { silent }
      );
      if (!reviewConfig) {
        return;
      }
      const reviewPolicy = convertToSQLReviewPolicy(reviewConfig);
      if (reviewPolicy) {
        this.setReviewPolicy(reviewPolicy);
      }
      return reviewPolicy;
    },
    async getOrFetchReviewPolicyByName(name: string) {
      const policy = this.getReviewPolicyByName(name);
      if (policy) {
        return policy;
      }

      const reviewPolicy = await this.fetchReviewPolicyByName({
        name,
      });
      return reviewPolicy;
    },
    async getOrFetchReviewPolicyByResource(
      resourcePath: string
    ): Promise<SQLReviewPolicy | undefined> {
      const cached = this.getReviewPolicyByResouce(resourcePath);
      if (cached) {
        return cached;
      }

      const policyStore = usePolicyV1Store();
      const policy = await policyStore.getOrFetchPolicyByName(
        getTagPolicyName(resourcePath)
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

export const useReviewPolicyByResource = (
  resourcePath: MaybeRef<string | undefined>
) => {
  const store = useSQLReviewStore();
  watchEffect(() => {
    if (!unref(resourcePath)) return;
    store.getOrFetchReviewPolicyByResource(unref(resourcePath)!);
  });

  return computed(() => {
    if (!unref(resourcePath)) return undefined;
    return store.getReviewPolicyByResouce(unref(resourcePath)!);
  });
};

export const useReviewPolicyForDatabase = (
  database: MaybeRef<ComposedDatabase | undefined>
) => {
  const store = useSQLReviewStore();

  watchEffect(async () => {
    if (!unref(database)) return;

    const reviewForProject = await store.getOrFetchReviewPolicyByResource(
      unref(database)!.project
    );
    if (reviewForProject) {
      return;
    }

    await store.getOrFetchReviewPolicyByResource(
      unref(database)!.effectiveEnvironment
    );
  });

  return computed(() => {
    if (!unref(database)) return undefined;
    const reviewForProject = store.getReviewPolicyByResouce(
      unref(database)!.project
    );
    if (reviewForProject && reviewForProject.enforce) {
      return reviewForProject;
    }
    return store.getReviewPolicyByResouce(
      unref(database)!.effectiveEnvironment
    );
  });
};
