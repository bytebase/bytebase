import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { pullAt, uniq } from "lodash-es";
import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { reviewConfigServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { policyNamePrefix } from "@/store/modules/v1/common";
import type {
  ComposedDatabase,
  MaybeRef,
  SchemaPolicyRule,
  SQLReviewPolicy,
} from "@/types";
import {
  PolicyType,
  TagPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { ReviewConfig } from "@/types/proto-es/v1/review_config_service_pb";
import {
  DeleteReviewConfigRequestSchema,
  GetReviewConfigRequestSchema,
  ListReviewConfigsRequestSchema,
  ReviewConfigSchema,
  UpdateReviewConfigRequestSchema,
} from "@/types/proto-es/v1/review_config_service_pb";
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
          policy: {
            case: "tagPolicy",
            value: create(TagPolicySchema, {
              tags: {
                [reviewConfigTagName]: configName,
              },
            }),
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
  return {
    id: reviewConfig.name,
    name: reviewConfig.title,
    resources: reviewConfig.resources,
    ruleList: reviewConfig.rules, // Use proto rules directly
    enforce: reviewConfig.enabled,
  };
};

interface SQLReviewState {
  reviewPolicyList: SQLReviewPolicy[];
}

const getTagPolicyName = (resourcePath: string): string => {
  return `${resourcePath}/${policyNamePrefix}tag`;
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
      const request = create(DeleteReviewConfigRequestSchema, {
        name: targetPolicy.id,
      });
      await reviewConfigServiceClientConnect.deleteReviewConfig(request);

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
      const patch: ReviewConfig = create(ReviewConfigSchema, {
        name: id,
      });
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
        patch.rules = ruleList; // Use proto rules directly
      }

      const request = create(UpdateReviewConfigRequestSchema, {
        reviewConfig: patch,
        updateMask: { paths: updateMask },
        allowMissing: true,
      });
      const updated =
        await reviewConfigServiceClientConnect.updateReviewConfig(request);

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
      const request = create(ListReviewConfigsRequestSchema, {});
      const { reviewConfigs } =
        await reviewConfigServiceClientConnect.listReviewConfigs(request);

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
      const request = create(GetReviewConfigRequestSchema, { name });
      const reviewConfig =
        await reviewConfigServiceClientConnect.getReviewConfig(request, {
          contextValues: createContextValues().set(silentContextKey, silent),
        });
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
      const sqlReviewName =
        policy.policy?.case === "tagPolicy"
          ? policy.policy.value.tags[reviewConfigTagName]
          : undefined;
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

    const { effectiveEnvironment } = unref(database)!;
    if (effectiveEnvironment) {
      await store.getOrFetchReviewPolicyByResource(effectiveEnvironment);
    }
  });

  return computed(() => {
    if (!unref(database)) return undefined;
    const reviewForProject = store.getReviewPolicyByResouce(
      unref(database)!.project
    );
    if (reviewForProject && reviewForProject.enforce) {
      return reviewForProject;
    }
    const { effectiveEnvironment } = unref(database)!;
    if (effectiveEnvironment) {
      return store.getReviewPolicyByResouce(effectiveEnvironment);
    }
    return undefined;
  });
};
