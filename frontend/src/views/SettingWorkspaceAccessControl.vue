<template>
  <div class="w-full mt-4 space-y-4">
    <FeatureAttention
      v-if="!hasDataAccessControlFeature"
      feature="bb.feature.access-control"
    />
    <div class="flex justify-between">
      <i18n-t
        tag="div"
        keypath="settings.access-control.description"
        class="textinfolabel"
      >
        <template #link>
          <LearnMoreLink
            url="https://www.bytebase.com/docs/security/data-access-control"
          />
        </template>
      </i18n-t>
    </div>

    <div class="relative">
      <BBTable
        :column-list="COLUMN_LIST"
        :data-source="state.environmentPolicyList"
        :show-header="true"
        :left-bordered="true"
        :right-bordered="true"
        :row-clickable="false"
      >
        <template #body="{ rowData: policy }: { rowData: EnvironmentPolicy }">
          <BBTableCell>
            <EnvironmentV1Name
              class="pl-2"
              :environment="policy.environment"
              :link="false"
            />
          </BBTableCell>
          <BBTableCell>
            <NCheckbox
              v-model:checked="policy.allowQueryData"
              :disabled="!allowAdmin"
            >
              {{ $t("settings.access-control.skip-approval") }}
            </NCheckbox>
          </BBTableCell>
          <BBTableCell>
            <NCheckbox
              v-model:checked="policy.allowExportData"
              :disabled="!allowAdmin"
            >
              {{ $t("settings.access-control.skip-approval") }}
            </NCheckbox>
          </BBTableCell>
          <BBTableCell>
            <NCheckbox
              v-model:checked="policy.disallowCopyingDataFromSQLEditor"
              :disabled="!allowAdmin"
            >
              {{ $t("settings.access-control.disallowed") }}
            </NCheckbox>
          </BBTableCell>
        </template>
      </BBTable>

      <div
        v-if="state.isLoading"
        class="absolute w-full h-full inset-0 z-1 bg-white/50 flex flex-col items-center justify-center"
      >
        <BBSpin />
      </div>

      <div
        v-if="!hasDataAccessControlFeature"
        class="absolute w-full h-full inset-0 z-10 bg-gray-300/80 flex flex-col items-center justify-center"
      >
        <UpgradeSubscriptionButton />
      </div>
    </div>
  </div>

  <FeatureModal
    feature="bb.feature.access-control"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { NCheckbox } from "naive-ui";
import { computed, reactive, watch, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { BBTableColumn } from "@/bbkit/types";
import UpgradeSubscriptionButton from "@/components/UpgradeSubscriptionButton.vue";
import { EnvironmentV1Name } from "@/components/v2";
import { resolveCELExpr } from "@/plugins/cel";
import {
  featureToRef,
  pushNotification,
  useCurrentUserV1,
  useEnvironmentV1List,
} from "@/store";
import { usePolicyV1Store } from "@/store/modules/v1/policy";
import { Expr } from "@/types/proto/google/type/expr";
import { Environment } from "@/types/proto/v1/environment_service";
import { IamPolicy } from "@/types/proto/v1/iam_policy";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { hasWorkspacePermissionV1 } from "@/utils";

interface EnvironmentPolicy {
  environment: Environment;
  allowQueryData: boolean;
  allowExportData: boolean;
  disallowCopyingDataFromSQLEditor: boolean;
}

interface LocalState {
  showFeatureModal: boolean;
  isLoading: boolean;
  environmentPolicyList: EnvironmentPolicy[];
}

const { t } = useI18n();
const environmentList = useEnvironmentV1List();
const policyStore = usePolicyV1Store();
const currentUserV1 = useCurrentUserV1();
const hasDataAccessControlFeature = featureToRef("bb.feature.access-control");
const state = reactive<LocalState>({
  showFeatureModal: false,
  isLoading: true,
  environmentPolicyList: environmentList.value.map((environment) => {
    const defaultValue = false;
    return {
      environment,
      allowQueryData: defaultValue,
      allowExportData: defaultValue,
      disallowCopyingDataFromSQLEditor: defaultValue,
    };
  }),
});

const allowAdmin = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-access-control",
    currentUserV1.value.userRole
  );
});

const COLUMN_LIST = computed((): BBTableColumn[] => [
  {
    title: t("common.environment"),
  },
  {
    title: t("settings.access-control.query-data"),
  },
  {
    title: t("settings.access-control.export-data"),
  },
  {
    title: t("settings.access-control.copy-data-from-sql-editor"),
  },
]);

onMounted(async () => {
  if (!hasDataAccessControlFeature.value) {
    state.isLoading = false;
    return;
  }

  const processWorkspaceIAMPolicy = async () => {
    const policy = await policyStore.getOrFetchPolicyByName(
      "policies/WORKSPACE_IAM"
    );
    if (!policy || !policy.workspaceIamPolicy) {
      return;
    }

    for (const binding of policy.workspaceIamPolicy.bindings) {
      if (!binding.members.includes("allUsers")) {
        continue;
      }

      if (binding.parsedExpr?.expr) {
        const simpleExpr = resolveCELExpr(binding.parsedExpr.expr);
        const args = simpleExpr.args;
        if (
          simpleExpr.operator !== "@in" ||
          args[0] !== "resource.environment_name"
        ) {
          continue;
        }

        const environmentNameList = args[1] as string[];
        for (const environmentPolicy of state.environmentPolicyList) {
          if (
            environmentNameList.includes(environmentPolicy.environment.name)
          ) {
            if (binding.role === "roles/QUERIER") {
              environmentPolicy.allowQueryData = true;
            } else if (binding.role === "roles/EXPORTER") {
              environmentPolicy.allowExportData = true;
            }
          }
        }
      }
    }
  };
  const processEnvironmentDisableCopyDataPolicy = async () => {
    const policyList = await policyStore.fetchPolicies({
      resourceType: PolicyResourceType.ENVIRONMENT,
      policyType: PolicyType.DISABLE_COPY_DATA,
    });
    policyList.forEach((policy) => {
      if (policy.disableCopyDataPolicy?.active) {
        const environmentPolicy = state.environmentPolicyList.find(
          (ep) => ep.environment.uid === policy.resourceUid
        );
        if (environmentPolicy) {
          environmentPolicy.disallowCopyingDataFromSQLEditor = true;
        }
      }
    });
  };

  try {
    await Promise.allSettled([
      processWorkspaceIAMPolicy(),
      processEnvironmentDisableCopyDataPolicy(),
    ]);
    state.isLoading = true;
  } finally {
    state.isLoading = false;
  }
});

const buildWorkspaceIAMPolicy = (envPolicyList: EnvironmentPolicy[]) => {
  const workspaceIamPolicy: IamPolicy = IamPolicy.fromPartial({});

  const allowQueryEnvNameList: string[] = envPolicyList
    .filter((item) => item.allowQueryData)
    .map((item) => item.environment.name);
  const allowExportEnvNameList: string[] = envPolicyList
    .filter((item) => item.allowExportData)
    .map((item) => item.environment.name);
  if (allowQueryEnvNameList.length > 0) {
    workspaceIamPolicy.bindings.push({
      role: "roles/QUERIER",
      members: ["allUsers"],
      condition: Expr.fromPartial({
        expression: `resource.environment_name in ["${allowQueryEnvNameList.join(
          '", "'
        )}"]`,
      }),
    });
  }
  if (allowExportEnvNameList.length > 0) {
    workspaceIamPolicy.bindings.push({
      role: "roles/EXPORTER",
      members: ["allUsers"],
      condition: Expr.fromPartial({
        expression: `resource.environment_name in ["${allowExportEnvNameList.join(
          '", "'
        )}"]`,
      }),
    });
  }
  return workspaceIamPolicy;
};

const upsertPolicy = useDebounceFn(async () => {
  if (!hasDataAccessControlFeature.value) {
    state.showFeatureModal = true;
    return;
  }

  if (!allowAdmin.value) {
    return;
  }

  const upsertWorkspaceIAMPolicy = async () => {
    await policyStore.createPolicy("", {
      type: PolicyType.WORKSPACE_IAM,
      resourceType: PolicyResourceType.WORKSPACE,
      resourceUid: "1",
      workspaceIamPolicy: buildWorkspaceIAMPolicy(state.environmentPolicyList),
    });
  };

  const upsertEnvironmentDisableCopyDataPolicy = async () => {
    for (let i = 0; i < state.environmentPolicyList.length; i++) {
      const environmentPolicy = state.environmentPolicyList[i];
      await policyStore.createPolicy(environmentPolicy.environment.name, {
        type: PolicyType.DISABLE_COPY_DATA,
        resourceType: PolicyResourceType.ENVIRONMENT,
        disableCopyDataPolicy: {
          active: environmentPolicy.disallowCopyingDataFromSQLEditor,
        },
      });
    }
  };

  await Promise.allSettled([
    upsertWorkspaceIAMPolicy(),
    upsertEnvironmentDisableCopyDataPolicy(),
  ]);

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
}, 200);

watch(
  () => state.environmentPolicyList,
  async () => {
    if (state.isLoading) {
      return;
    }

    await upsertPolicy();
  },
  {
    deep: true,
  }
);
</script>
