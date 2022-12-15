import { defineStore } from "pinia";
import { computed, Ref, unref, watchEffect } from "vue";
import axios from "axios";
import {
  DatabaseId,
  EnvironmentId,
  MaybeRef,
  PolicyState,
  ResourceIdentifier,
  ResourceObject,
  ResourceType,
  unknown,
  UNKNOWN_ID,
} from "@/types";
import {
  PipelineApprovalPolicyPayload,
  Policy,
  PolicyResourceType,
  PolicyType,
  PolicyUpsert,
  SensitiveDataPolicyPayload,
} from "@/types/policy";
import { getPrincipalFromIncludedList } from "./principal";
import { useEnvironmentStore } from "./environment";
import { useCurrentUser } from "./auth";

function convertEnvironment(
  policy: ResourceObject,
  includedList: ResourceObject[]
) {
  // The `environment` relationship cannot retire now.
  // But for database-level policies it will be null.
  // In order not to break the typings, we will fallback to <<Unknown Environment>>
  // for database-level policies here.
  let environment = unknown("ENVIRONMENT");
  const data = policy.relationships?.environment?.data as
    | ResourceIdentifier
    | undefined;
  if (data) {
    const environmentId = data.id;
    environment.id = parseInt(environmentId);

    const environmentStore = useEnvironmentStore();
    for (const item of includedList || []) {
      if (item.type == "environment" && data.id == item.id) {
        environment = environmentStore.convert(item, includedList);
      }
    }
  }
  return environment;
}

function convert(
  policy: ResourceObject,
  includedList: ResourceObject[]
): Policy {
  const environment = convertEnvironment(policy, includedList);

  const result = {
    ...(policy.attributes as Omit<
      Policy,
      "id" | "environment" | "payload" | "creator" | "updater"
    >),
    id: parseInt(policy.id),
    creator: getPrincipalFromIncludedList(
      policy.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      policy.relationships!.updater.data,
      includedList
    ),
    environment,
    payload: JSON.parse((policy.attributes.payload as string) || "{}"),
  };
  if (result.type === "bb.policy.pipeline-approval") {
    const payload = result.payload as PipelineApprovalPolicyPayload;
    if (!payload.assigneeGroupList) {
      // Assign an empty array as fallback
      payload.assigneeGroupList = [];
    }
  }
  if (result.type === "bb.policy.sensitive-data") {
    const payload = result.payload as SensitiveDataPolicyPayload;
    if (!payload.sensitiveDataList) {
      // The array might be null, fill it with empty array to fallback.
      payload.sensitiveDataList = [];
    }
  }

  return result;
}

export const usePolicyStore = defineStore("policy", {
  state: (): PolicyState => ({
    policyMapByEnvironmentId: new Map(),
    policyMapByDatabaseId: new Map(),
    policyMapByResourceType: new Map(),
  }),
  actions: {
    getPolicyByEnvironmentIdAndType(
      environmentId: EnvironmentId,
      type: PolicyType
    ): Policy | undefined {
      const map = this.policyMapByEnvironmentId.get(environmentId);
      if (map) {
        return map.get(type);
      }
      return undefined;
    },
    setPolicyByEnvironmentId({
      environmentId,
      policy,
    }: {
      environmentId: EnvironmentId;
      policy: Policy;
    }) {
      const map = this.policyMapByEnvironmentId.get(environmentId);
      if (map) {
        map.set(policy.type, policy);
      } else {
        this.policyMapByEnvironmentId.set(
          environmentId,
          new Map([[policy.type, policy]])
        );
      }
    },
    async fetchPolicyListByType(type: PolicyType): Promise<Policy[]> {
      const data: { data: ResourceObject[]; included: ResourceObject[] } = (
        await axios.get(`/api/policy?type=${type}`)
      ).data;

      return data.data.map((d) => convert(d, data.included));
    },
    async fetchPolicyListByTypeAndResourceType(
      type: PolicyType,
      resourceType: ResourceType
    ): Promise<Policy[]> {
      const data: { data: ResourceObject[]; included: ResourceObject[] } = (
        await axios.get(
          `/api/policy?type=${type}&resourceType=${resourceType.toLowerCase()}`
        )
      ).data;

      return data.data.map((d) => convert(d, data.included));
    },
    async fetchPolicyByEnvironmentAndType({
      environmentId,
      type,
    }: {
      environmentId: EnvironmentId;
      type: PolicyType;
    }): Promise<Policy> {
      const data = (
        await axios.get(`/api/policy/environment/${environmentId}?type=${type}`)
      ).data;
      const policy = convert(data.data, data.included);
      this.setPolicyByEnvironmentId({ environmentId, policy });

      return policy;
    },
    async upsertPolicyByEnvironmentAndType({
      environmentId,
      type,
      policyUpsert,
    }: {
      environmentId: EnvironmentId;
      type: PolicyType;
      policyUpsert: PolicyUpsert;
    }): Promise<Policy> {
      const data = (
        await axios.patch(
          `/api/policy/environment/${environmentId}?type=${type}`,
          {
            data: {
              type: "policyUpsert",
              attributes: {
                rowStatus: policyUpsert.rowStatus,
                payload: policyUpsert.payload
                  ? JSON.stringify(policyUpsert.payload)
                  : undefined,
              },
            },
          }
        )
      ).data;
      const policy = convert(data.data, data.included);

      this.setPolicyByEnvironmentId({ environmentId, policy });

      return policy;
    },
    async deletePolicyByEnvironmentAndType({
      environmentId,
      type,
    }: {
      environmentId: EnvironmentId;
      type: PolicyType;
    }) {
      await axios.delete(
        `/api/policy/environment/${environmentId}?type=${type}`
      );
    },

    getPolicyByDatabaseIdAndType(
      databaseId: DatabaseId,
      type: PolicyType
    ): Policy | undefined {
      const map = this.policyMapByDatabaseId.get(databaseId);
      if (map) {
        return map.get(type);
      }
      return undefined;
    },
    setPolicyByDatabaseId({
      databaseId,
      policy,
    }: {
      databaseId: DatabaseId;
      policy: Policy;
    }) {
      const map = this.policyMapByDatabaseId.get(databaseId);
      if (map) {
        map.set(policy.type, policy);
      } else {
        this.policyMapByDatabaseId.set(
          databaseId,
          new Map([[policy.type, policy]])
        );
      }
    },
    async fetchPolicyByDatabaseAndType({
      databaseId,
      type,
    }: {
      databaseId: DatabaseId;
      type: PolicyType;
    }): Promise<Policy> {
      const data = (
        await axios.get(`/api/policy/database/${databaseId}?type=${type}`)
      ).data;
      const policy = convert(data.data, data.included);
      this.setPolicyByDatabaseId({ databaseId, policy });

      return policy;
    },
    async upsertPolicyByDatabaseAndType({
      databaseId,
      type,
      policyUpsert,
    }: {
      databaseId: DatabaseId;
      type: PolicyType;
      policyUpsert: PolicyUpsert;
    }): Promise<Policy> {
      const data = (
        await axios.patch(`/api/policy/database/${databaseId}?type=${type}`, {
          data: {
            type: "policyUpsert",
            attributes: {
              rowStatus: policyUpsert.rowStatus,
              payload: policyUpsert.payload
                ? JSON.stringify(policyUpsert.payload)
                : undefined,
            },
          },
        })
      ).data;
      const policy = convert(data.data, data.included);

      this.setPolicyByDatabaseId({ databaseId, policy });

      return policy;
    },
    async deletePolicyByDatabaseAndType({
      databaseId,
      type,
    }: {
      databaseId: DatabaseId;
      type: PolicyType;
    }) {
      await axios.delete(`/api/policy/database/${databaseId}?type=${type}`);
      // Remove it from local store.
      const map = this.policyMapByDatabaseId.get(databaseId);
      if (map) {
        if (map.has(type)) {
          map.delete(type);
        }
      }
    },

    getPolicyListByResourceTypeAndPolicyType(
      resourceType: PolicyResourceType,
      policyType: PolicyType
    ) {
      const map = this.policyMapByResourceType.get(resourceType);
      if (!map) return [];
      return map.get(policyType) ?? [];
    },
    setPolicyListByResourceTypeAndPolicyType(
      resourceType: PolicyResourceType,
      policyType: PolicyType,
      policyList: Policy[]
    ) {
      const map = this.policyMapByResourceType.get(resourceType);
      if (map) {
        map.set(policyType, policyList);
      } else {
        this.policyMapByResourceType.set(
          resourceType,
          new Map([[policyType, policyList]])
        );
      }
    },
    async fetchPolicyListByResourceTypeAndPolicyType(
      resourceType: PolicyResourceType,
      policyType: PolicyType
    ) {
      const url = `/api/policy?resourceType=${resourceType}&type=${policyType}`;
      const data: { data: ResourceObject[]; included: ResourceObject[] } = (
        await axios.get(url)
      ).data;

      const policyList = data.data.map((d) => convert(d, data.included));
      this.setPolicyListByResourceTypeAndPolicyType(
        resourceType,
        policyType,
        policyList
      );

      return policyList;
    },
  },
});

export const usePolicyByEnvironmentAndType = (
  params: Ref<{ environmentId: EnvironmentId; type: PolicyType }>
) => {
  const store = usePolicyStore();
  const currentUser = useCurrentUser();
  watchEffect(() => {
    if (currentUser.value.id === UNKNOWN_ID) return;

    store.fetchPolicyByEnvironmentAndType(params.value);
  });

  return computed(() =>
    store.getPolicyByEnvironmentIdAndType(
      params.value.environmentId,
      params.value.type
    )
  );
};

export const usePolicyByDatabaseAndType = (
  params: MaybeRef<{ databaseId: DatabaseId; type: PolicyType }>
) => {
  const store = usePolicyStore();
  const currentUser = useCurrentUser();
  watchEffect(() => {
    if (currentUser.value.id === UNKNOWN_ID) return;
    store.fetchPolicyByDatabaseAndType(unref(params));
  });

  return computed(() => {
    const { databaseId, type } = unref(params);
    return store.getPolicyByDatabaseIdAndType(databaseId, type);
  });
};

export const usePolicyListByResourceTypeAndPolicyType = (
  params: MaybeRef<{ resourceType: PolicyResourceType; policyType: PolicyType }>
) => {
  const store = usePolicyStore();
  const currentUser = useCurrentUser();
  watchEffect(() => {
    if (currentUser.value.id === UNKNOWN_ID) return;
    const { resourceType, policyType } = unref(params);

    store.fetchPolicyListByResourceTypeAndPolicyType(resourceType, policyType);
  });

  return computed(() => {
    const { resourceType, policyType } = unref(params);
    return store.getPolicyListByResourceTypeAndPolicyType(
      resourceType,
      policyType
    );
  });
};
