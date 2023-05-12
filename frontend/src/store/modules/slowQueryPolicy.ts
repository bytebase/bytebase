import { defineStore } from "pinia";
import { computed, ref, watchEffect } from "vue";
import { UNKNOWN_ID } from "@/types";
import { useCurrentUser } from "./auth";
import { usePolicyV1Store } from "./v1/policy";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
  policyTypeToJSON,
} from "@/types/proto/v1/org_policy_service";
import { policyNamePrefix } from "@/store/modules/v1/common";

const getSlowQueryPolicyName = (parentPath: string): string => {
  return `${parentPath}/${policyNamePrefix}${policyTypeToJSON(
    PolicyType.SLOW_QUERY
  )}`;
};

export const useSlowQueryPolicyStore = defineStore("slow-query-policy", () => {
  const policyMapByName = ref(new Map<string, Policy>());

  const getPolicyList = () => {
    return [...policyMapByName.value.values()];
  };

  const getPolicy = (name: string) => {
    return policyMapByName.value.get(name);
  };

  const setPolicyList = (policyList: Policy[]) => {
    for (const policy of policyList) {
      policyMapByName.value.set(policy.name, policy);
    }
  };

  const fetchPolicyList = async () => {
    const policyStore = usePolicyV1Store();
    const policyList = await policyStore.fetchPolicies({
      resourceType: PolicyResourceType.INSTANCE,
      policyType: PolicyType.SLOW_QUERY,
      showDeleted: true,
    });

    setPolicyList(policyList);
    return policyList;
  };

  const upsertPolicy = async ({
    parentPath,
    active,
  }: {
    parentPath: string;
    active: boolean;
  }) => {
    const policy = await usePolicyV1Store().upsertPolicy({
      parentPath,
      policy: {
        type: PolicyType.SLOW_QUERY,
        slowQueryPolicy: {
          active,
        },
      },
      updateMask: ["payload"],
    });

    if (policy) {
      policyMapByName.value.set(policy.name, policy);
    }

    return policy;
  };

  return {
    getPolicyList,
    fetchPolicyList,
    upsertPolicy,
  };
});

export const useSlowQueryPolicyList = () => {
  const store = useSlowQueryPolicyStore();
  const currentUser = useCurrentUser();
  const ready = ref(false);

  watchEffect(() => {
    if (currentUser.value.id === UNKNOWN_ID) return;
    ready.value = false;
    store.fetchPolicyList().finally(() => {
      ready.value = true;
    });
  });

  const list = computed(() => {
    return store.getPolicyList();
  });

  return { list, ready };
};
