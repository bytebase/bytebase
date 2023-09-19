import { defineStore } from "pinia";
import { computed, ref, watchEffect } from "vue";
import { UNKNOWN_USER_NAME } from "@/types";
import {
  Policy,
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { useCurrentUserV1 } from "./auth";
import { usePolicyV1Store } from "./v1/policy";

export const useSlowQueryPolicyStore = defineStore("slow-query-policy", () => {
  const policyMapByName = ref(new Map<string, Policy>());

  const getPolicyList = () => {
    return [...policyMapByName.value.values()];
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
  const currentUserV1 = useCurrentUserV1();
  const ready = ref(false);

  watchEffect(() => {
    if (currentUserV1.value.name === UNKNOWN_USER_NAME) return;
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
