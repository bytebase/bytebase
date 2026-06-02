import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { pullAt, uniq } from "lodash-es";
import { create } from "zustand";
import { reviewConfigServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { useAppStore } from "@/react/stores/app";
import type { SchemaPolicyRule, SQLReviewPolicy } from "@/types";
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

const reviewConfigTagName = "bb.tag.review_config";

// `${resourcePath}/policies/tag` — PolicyType[TAG] lowercased.
const getTagPolicyName = (resourcePath: string): string =>
  `${resourcePath}/policies/${PolicyType[PolicyType.TAG].toLowerCase()}`;

const upsertReviewConfigTag = async (
  resources: string[],
  configName: string
) => {
  await Promise.all(
    resources.map((resourcePath) =>
      useAppStore.getState().upsertPolicy({
        parentPath: resourcePath,
        policy: {
          type: PolicyType.TAG,
          policy: {
            case: "tagPolicy",
            value: createProto(TagPolicySchema, {
              tags: { [reviewConfigTagName]: configName },
            }),
          },
        },
      })
    )
  );
};

const removeReviewConfigTag = async (resources: string[]) => {
  await Promise.all(
    resources.map((resource) =>
      useAppStore.getState().deletePolicy(getTagPolicyName(resource))
    )
  );
};

const convertToSQLReviewPolicy = (
  reviewConfig: ReviewConfig
): SQLReviewPolicy => ({
  id: reviewConfig.name,
  name: reviewConfig.title,
  resources: reviewConfig.resources,
  ruleList: reviewConfig.rules,
  enforce: reviewConfig.enabled,
});

export type SQLReviewState = {
  reviewPolicyList: SQLReviewPolicy[];
  setReviewPolicy: (reviewPolicy: SQLReviewPolicy) => void;
  removeResourceForReview: (map: Map<string, string[]>) => void;
  upsertReviewConfigTag: (params: {
    oldResources: string[];
    newResources: string[];
    review: string;
  }) => Promise<void>;
  removeReviewPolicy: (id: string) => Promise<void>;
  upsertReviewPolicy: (params: {
    id: string;
    title?: string;
    enforce?: boolean;
    ruleList?: SchemaPolicyRule[];
    resources?: string[];
  }) => Promise<SQLReviewPolicy>;
  getReviewPolicyByName: (name: string) => SQLReviewPolicy | undefined;
  getReviewPolicyByResouce: (
    resourcePath: string
  ) => SQLReviewPolicy | undefined;
  fetchReviewPolicyList: () => Promise<SQLReviewPolicy[]>;
  fetchReviewPolicyByName: (params: {
    name: string;
    silent?: boolean;
  }) => Promise<SQLReviewPolicy | undefined>;
  getOrFetchReviewPolicyByName: (
    name: string,
    silent: boolean
  ) => Promise<SQLReviewPolicy | undefined>;
  getOrFetchReviewPolicyByResource: (
    resourcePath: string,
    silent: boolean
  ) => Promise<SQLReviewPolicy | undefined>;
};

// Standalone Zustand port of the legacy Pinia `useSQLReviewStore`. Tag-policy
// reads/writes go through the app store's policy slice (no Pinia dependency).
// The Vue composables (useSQLReviewPolicyList etc.) had no consumers and are
// dropped.
export const useSQLReviewStore = create<SQLReviewState>()((set, get) => ({
  reviewPolicyList: [],

  setReviewPolicy: (reviewPolicy) => {
    set((state) => {
      const index = state.reviewPolicyList.findIndex(
        (r) => r.id === reviewPolicy.id
      );
      if (index < 0) {
        return { reviewPolicyList: [...state.reviewPolicyList, reviewPolicy] };
      }
      const reviewPolicyList = [...state.reviewPolicyList];
      reviewPolicyList[index] = {
        ...reviewPolicyList[index],
        ...reviewPolicy,
      };
      return { reviewPolicyList };
    });
  },

  removeResourceForReview: (map) => {
    set((state) => ({
      reviewPolicyList: state.reviewPolicyList.map((reviewPolicy) => {
        const toRemove = map.get(reviewPolicy.id);
        if (!toRemove) {
          return reviewPolicy;
        }
        return {
          ...reviewPolicy,
          resources: reviewPolicy.resources.filter(
            (resource) => !toRemove.includes(resource)
          ),
        };
      }),
    }));
  },

  upsertReviewConfigTag: async ({ oldResources, newResources, review }) => {
    await removeReviewConfigTag(uniq(oldResources));
    await upsertReviewConfigTag(uniq(newResources), review);
    set((state) => ({
      reviewPolicyList: state.reviewPolicyList.map((r) =>
        r.id === review ? { ...r, resources: newResources } : r
      ),
    }));
  },

  removeReviewPolicy: async (id) => {
    const index = get().reviewPolicyList.findIndex((g) => g.id === id);
    if (index < 0) {
      return;
    }
    await reviewConfigServiceClientConnect.deleteReviewConfig(
      createProto(DeleteReviewConfigRequestSchema, { name: id })
    );
    set((state) => {
      const reviewPolicyList = [...state.reviewPolicyList];
      pullAt(reviewPolicyList, index);
      return { reviewPolicyList };
    });
  },

  upsertReviewPolicy: async ({ id, title, enforce, ruleList, resources }) => {
    const patch = createProto(ReviewConfigSchema, { name: id });
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
      patch.rules = ruleList;
    }

    const updated = await reviewConfigServiceClientConnect.updateReviewConfig(
      createProto(UpdateReviewConfigRequestSchema, {
        reviewConfig: patch,
        updateMask: { paths: updateMask },
        allowMissing: true,
      })
    );

    if (resources) {
      await get().upsertReviewConfigTag({
        oldResources: [],
        newResources: resources,
        review: updated.name,
      });
      updated.resources = resources;
    }

    const reviewPolicy = convertToSQLReviewPolicy(updated);
    get().setReviewPolicy(reviewPolicy);
    return reviewPolicy;
  },

  getReviewPolicyByName: (name) =>
    get().reviewPolicyList.find((g) => g.id === name),

  getReviewPolicyByResouce: (resourcePath) =>
    get().reviewPolicyList.find((policy) =>
      policy.resources.find((resource) => resource === resourcePath)
    ),

  fetchReviewPolicyList: async () => {
    const { reviewConfigs } =
      await reviewConfigServiceClientConnect.listReviewConfigs(
        createProto(ListReviewConfigsRequestSchema, {})
      );
    const reviewPolicyList = reviewConfigs.map(convertToSQLReviewPolicy);
    set({ reviewPolicyList });
    return reviewPolicyList;
  },

  fetchReviewPolicyByName: async ({ name, silent = false }) => {
    const reviewConfig = await reviewConfigServiceClientConnect.getReviewConfig(
      createProto(GetReviewConfigRequestSchema, { name }),
      { contextValues: createContextValues().set(silentContextKey, silent) }
    );
    if (!reviewConfig) {
      return undefined;
    }
    const reviewPolicy = convertToSQLReviewPolicy(reviewConfig);
    get().setReviewPolicy(reviewPolicy);
    return reviewPolicy;
  },

  getOrFetchReviewPolicyByName: async (name, silent) => {
    const policy = get().getReviewPolicyByName(name);
    if (policy) {
      return policy;
    }
    return get().fetchReviewPolicyByName({ name, silent });
  },

  getOrFetchReviewPolicyByResource: async (resourcePath, silent) => {
    const cached = get().getReviewPolicyByResouce(resourcePath);
    if (cached) {
      return cached;
    }
    const policy = await useAppStore
      .getState()
      .getOrFetchPolicyByName(getTagPolicyName(resourcePath));
    const sqlReviewName =
      policy?.policy?.case === "tagPolicy"
        ? policy.policy.value.tags[reviewConfigTagName]
        : undefined;
    if (!sqlReviewName) {
      return undefined;
    }
    return get().getOrFetchReviewPolicyByName(sqlReviewName, silent);
  },
}));
