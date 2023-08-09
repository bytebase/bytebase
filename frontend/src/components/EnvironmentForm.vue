<template>
  <component :is="drawer ? DrawerContent : 'div'" v-bind="bindings">
    <div
      class="divide-y divide-block-border"
      :class="drawer ? 'w-[36rem]' : 'w-full px-4 pb-4'"
    >
      <div class="grid grid-cols-1 gap-x-4">
        <div class="col-span-1">
          <label for="name" class="textlabel">
            {{ $t("common.environment-name") }}
            <span class="text-red-600">*</span>
          </label>
          <BBTextField
            class="mt-2 w-full"
            :disabled="!allowEdit"
            :required="true"
            :value="state.environment.title"
            @input="handleEnvironmentNameChange"
          />
        </div>

        <div class="mt-2">
          <ResourceIdField
            ref="resourceIdField"
            resource-type="environment"
            :readonly="!create"
            :value="extractEnvironmentResourceName(state.environment.name)"
            :resource-title="state.environment.title"
            :validate="validateResourceId"
          />
        </div>

        <div class="col-span-1 mt-6">
          <label class="textlabel flex items-center">
            {{ $t("policy.environment-tier.name") }}
            <FeatureBadge feature="bb.feature.environment-tier-policy" />
          </label>
          <p class="mt-2 text-sm text-gray-600">
            <i18n-t tag="span" keypath="policy.environment-tier.description">
              <template #newline><br /></template>
            </i18n-t>
            <a
              class="inline-flex items-center text-blue-600 ml-1 hover:underline"
              href="https://www.bytebase.com/docs/administration/environment-policy/tier"
              target="_blank"
              >{{ $t("common.learn-more")
              }}<heroicons-outline:external-link class="w-4 h-4"
            /></a>
          </p>
          <div class="mt-4 flex flex-col space-y-4">
            <div class="flex space-x-4">
              <BBCheckbox
                :value="state.environmentTier === EnvironmentTier.PROTECTED"
                :disabled="!allowEdit"
                @toggle="(on: boolean) => {
                state.environmentTier = on ? EnvironmentTier.PROTECTED : EnvironmentTier.UNPROTECTED
              }"
              />
              <div>
                <div class="textlabel">
                  {{ $t("policy.environment-tier.mark-env-as-production") }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div class="col-span-1 mt-6">
          <label class="textlabel">
            {{ $t("policy.rollout.name") }}
          </label>
          <span v-show="valueChanged('approvalPolicy')" class="textlabeltip">{{
            $t("policy.rollout.tip")
          }}</span>
          <div class="mt-1 textinfolabel">
            {{ $t("policy.rollout.info") }}
          </div>
          <div class="mt-4 flex flex-col space-y-4">
            <div class="flex space-x-4">
              <input
                v-model="state.approvalPolicy.deploymentApprovalPolicy!.defaultStrategy"
                name="manual-approval-never"
                tabindex="-1"
                type="radio"
                class="text-accent disabled:text-accent-disabled focus:ring-accent"
                :value="ApprovalStrategy.AUTOMATIC"
                :disabled="!allowEdit"
              />
              <div class="-mt-0.5">
                <div class="textlabel">
                  {{ $t("policy.rollout.auto") }}
                </div>
                <div class="mt-1 textinfolabel">
                  {{ $t("policy.rollout.auto-info") }}
                </div>
              </div>
            </div>

            <div class="flex space-x-4">
              <input
                v-model="state.approvalPolicy.deploymentApprovalPolicy!.defaultStrategy"
                name="manual-approval-always"
                tabindex="-1"
                type="radio"
                class="text-accent disabled:text-accent-disabled focus:ring-accent"
                :value="ApprovalStrategy.MANUAL"
                :disabled="!allowEdit"
              />
              <div class="-mt-0.5">
                <div class="textlabel flex">
                  {{ $t("policy.rollout.manual") }}
                  <FeatureBadge feature="bb.feature.approval-policy" />
                </div>
                <div class="mt-1 textinfolabel">
                  {{ $t("policy.rollout.manual-info") }}
                </div>
              </div>
            </div>

            <AssigneeGroupEditor
              class="ml-8"
              :policy="state.approvalPolicy"
              :allow-edit="allowEdit"
              @update="(assigneeGroupList) => {
                state.approvalPolicy.deploymentApprovalPolicy!.deploymentApprovalStrategies = assigneeGroupList
              }"
            />
          </div>
        </div>
        <div class="col-span-1 mt-6">
          <label class="textlabel"> {{ $t("policy.backup.name") }} </label>
          <span v-show="valueChanged('backupPolicy')" class="textlabeltip">{{
            $t("policy.backup.tip")
          }}</span>
          <div class="mt-4 flex flex-col space-y-4">
            <div class="flex space-x-4">
              <input
                v-model="state.backupPolicy.backupPlanPolicy!.schedule"
                tabindex="-1"
                type="radio"
                class="text-accent disabled:text-accent-disabled focus:ring-accent"
                :value="BackupPlanSchedule.UNSET"
                :disabled="!allowEdit"
              />
              <div class="-mt-0.5">
                <div class="textlabel">
                  {{ $t("policy.backup.not-enforced") }}
                </div>
                <div class="mt-1 textinfolabel">
                  {{ $t("policy.backup.not-enforced-info") }}
                </div>
              </div>
            </div>
            <div class="flex space-x-4">
              <input
                v-model="state.backupPolicy.backupPlanPolicy!.schedule"
                tabindex="-1"
                type="radio"
                class="text-accent disabled:text-accent-disabled focus:ring-accent"
                :value="BackupPlanSchedule.DAILY"
                :disabled="!allowEdit"
              />
              <div class="-mt-0.5">
                <div class="textlabel flex">
                  {{ $t("policy.backup.daily") }}
                  <FeatureBadge feature="bb.feature.backup-policy" />
                </div>
                <div class="mt-1 textinfolabel">
                  {{ $t("policy.backup.daily-info") }}
                </div>
              </div>
            </div>
            <div class="flex space-x-4">
              <input
                v-model="state.backupPolicy.backupPlanPolicy!.schedule"
                tabindex="-1"
                type="radio"
                class="text-accent disabled:text-accent-disabled focus:ring-accent"
                :value="BackupPlanSchedule.WEEKLY"
                :disabled="!allowEdit"
              />
              <div class="-mt-0.5">
                <div class="textlabel flex">
                  {{ $t("policy.backup.weekly") }}
                  <FeatureBadge feature="bb.feature.backup-policy" />
                </div>
                <div class="mt-1 textinfolabel">
                  {{ $t("policy.backup.weekly-info") }}
                </div>
              </div>
            </div>
          </div>
        </div>
        <div v-if="!create" class="col-span-1 mt-6">
          <label class="textlabel">
            {{ $t("sql-review.title") }}
          </label>
          <div class="mt-3">
            <div v-if="sqlReviewPolicy" class="inline-flex items-center">
              <BBSwitch
                v-if="allowEditSQLReviewPolicy"
                class="mr-2"
                :text="true"
                :value="sqlReviewPolicy.enforce"
                @toggle="toggleSQLReviewPolicy"
              />
              <button
                type="button"
                class="text-sm font-medium text-accent hover:underline-2"
                @click.prevent="onSQLReviewPolicyClick"
              >
                {{ sqlReviewPolicy.name }}
              </button>
            </div>
            <button
              v-else-if="hasPermission"
              type="button"
              class="btn-normal py-2 px-4 gap-x-1 items-center"
              @click.prevent="onSQLReviewPolicyClick"
            >
              {{ $t("sql-review.configure-policy") }}
            </button>
            <span v-else class="textinfolabel">
              {{ $t("sql-review.no-policy-set") }}
            </span>
          </div>
        </div>
      </div>

      <div v-if="!drawer" class="mt-6 flex justify-between items-center pt-5">
        <template
          v-if="(state.environment as Environment).state === State.ACTIVE"
        >
          <BBButtonConfirm
            v-if="allowArchive"
            :style="'ARCHIVE'"
            :button-text="$t('environment.archive')"
            :ok-text="$t('common.archive')"
            :confirm-title="
            $t('environment.archive') + ` '${(state.environment as Environment).title}'?`
          "
            :confirm-description="$t('environment.archive-info')"
            :require-confirm="true"
            @confirm="archiveEnvironment"
          />
        </template>
        <template
          v-else-if="(state.environment as Environment).state === State.DELETED"
        >
          <BBButtonConfirm
            v-if="allowRestore"
            :style="'RESTORE'"
            :button-text="$t('environment.restore')"
            :ok-text="$t('common.restore')"
            :confirm-title="
            $t('environment.restore') + ` '${(state.environment as Environment).title}'?`
          "
            :confirm-description="''"
            :require-confirm="true"
            @confirm="restoreEnvironment"
          />
        </template>
        <div v-else></div>
        <div v-if="allowEdit">
          <button
            type="button"
            class="btn-normal py-2 px-4"
            :disabled="!valueChanged()"
            @click.prevent="revertEnvironment"
          >
            {{ $t("common.revert") }}
          </button>
          <button
            type="submit"
            class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
            :disabled="!valueChanged()"
            @click.prevent="updateEnvironment"
          >
            {{ $t("common.update") }}
          </button>
        </div>
      </div>
    </div>

    <template v-if="drawer" #footer>
      <div class="flex justify-end items-center gap-x-3">
        <NButton @click.prevent="$emit('cancel')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowCreate"
          @click.prevent="createEnvironment"
        >
          {{ $t("common.create") }}
        </NButton>
      </div>
      <!-- Update button group -->
    </template>
  </component>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual, isEmpty } from "lodash-es";
import { NButton } from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, reactive, PropType, watch, watchEffect, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSwitch } from "@/bbkit";
import { DrawerContent } from "@/components/v2";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import {
  pushNotification,
  useCurrentUserV1,
  useEnvironmentV1List,
  useSQLReviewStore,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { useEnvironmentV1Store } from "@/store/modules/v1/environment";
import type { ResourceId, SQLReviewPolicy, ValidatedMessage } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import {
  Policy,
  PolicyType,
  BackupPlanSchedule,
  ApprovalStrategy,
} from "@/types/proto/v1/org_policy_service";
import {
  extractEnvironmentResourceName,
  hasWorkspacePermissionV1,
  sqlReviewPolicySlug,
} from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import AssigneeGroupEditor from "./EnvironmentForm/AssigneeGroupEditor.vue";

interface LocalState {
  environment: Environment;
  approvalPolicy: Policy;
  backupPolicy: Policy;
  environmentTier: EnvironmentTier;
}

const ROUTE_NAME = "setting.workspace.sql-review";

const props = defineProps({
  create: {
    type: Boolean,
    default: false,
  },
  environment: {
    required: true,
    type: Object as PropType<Environment>,
  },
  approvalPolicy: {
    required: true,
    type: Object as PropType<Policy>,
  },
  backupPolicy: {
    required: true,
    type: Object as PropType<Policy>,
  },
  environmentTier: {
    required: true,
    type: Number as PropType<EnvironmentTier>,
  },
  drawer: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits([
  "create",
  "update",
  "cancel",
  "archive",
  "restore",
  "update-policy",
]);

const { t } = useI18n();
const state = reactive<LocalState>({
  environment: cloneDeep(props.environment),
  approvalPolicy: cloneDeep(props.approvalPolicy),
  backupPolicy: cloneDeep(props.backupPolicy),
  environmentTier: props.environmentTier,
});

const router = useRouter();
const environmentV1Store = useEnvironmentV1Store();
const sqlReviewStore = useSQLReviewStore();
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

const bindings = computed(() => {
  if (props.drawer) {
    return {
      title: t("environment.create"),
    };
  }
  return {};
});

const environmentId = computed(() => {
  if (props.create) {
    return;
  }
  return (props.environment as Environment).uid;
});

const prepareSQLReviewPolicy = () => {
  if (!environmentId.value) {
    return;
  }
  return sqlReviewStore.getOrFetchReviewPolicyByEnvironmentUID(
    environmentId.value
  );
};
watchEffect(prepareSQLReviewPolicy);

const sqlReviewPolicy = computed((): SQLReviewPolicy | undefined => {
  if (!environmentId.value) {
    return;
  }
  return sqlReviewStore.getReviewPolicyByEnvironmentUID(environmentId.value);
});

const handleEnvironmentNameChange = (event: InputEvent) => {
  state.environment.title = (event.target as HTMLInputElement).value;
};

const onSQLReviewPolicyClick = () => {
  if (sqlReviewPolicy.value) {
    router.push({
      name: `${ROUTE_NAME}.detail`,
      params: {
        sqlReviewPolicySlug: sqlReviewPolicySlug(sqlReviewPolicy.value),
      },
    });
  } else {
    router.push({
      name: `${ROUTE_NAME}.create`,
      query: {
        environmentId: environmentId.value,
      },
    });
  }
};

watch(
  () => props.environment,
  (cur) => {
    state.environment = cloneDeep(cur);
  }
);

watch(
  () => props.approvalPolicy,
  (cur: Policy) => {
    state.approvalPolicy = cloneDeep(cur);
  }
);

watch(
  () => props.backupPolicy,
  (cur: Policy) => {
    state.backupPolicy = cloneDeep(cur);
  }
);

watch(
  () => props.environmentTier,
  (cur: EnvironmentTier) => {
    state.environmentTier = cur;
  }
);

const currentUserV1 = useCurrentUserV1();

const environmentList = useEnvironmentV1List();

const hasPermission = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-environment",
    currentUserV1.value.userRole
  );
});

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const env = await environmentV1Store.getOrFetchEnvironmentByName(
      environmentNamePrefix + resourceId,
      true /* silent */
    );
    if (env) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t("resource.environment"),
          }),
        },
      ];
    }
  } catch (error) {
    if (getErrorCode(error) !== Status.NOT_FOUND) {
      throw error;
    }
  }
  return [];
};

const allowArchive = computed(() => {
  return allowEdit.value && environmentList.value.length > 1;
});

const allowRestore = computed(() => {
  return hasPermission.value;
});

const allowEdit = computed(() => {
  return (
    props.create ||
    ((state.environment as Environment).state === State.ACTIVE &&
      hasPermission.value)
  );
});

const allowCreate = computed(() => {
  return (
    !isEmpty(state.environment?.title) &&
    resourceIdField.value?.resourceId &&
    resourceIdField.value?.isValidated
  );
});

const allowEditSQLReviewPolicy = computed(() => {
  return hasWorkspacePermissionV1(
    "bb.permission.workspace.manage-sql-review-policy",
    currentUserV1.value.userRole
  );
});

const valueChanged = (
  field?: "environment" | "approvalPolicy" | "backupPolicy" | "environmentTier"
): boolean => {
  switch (field) {
    case "environment":
      return !isEqual(props.environment, state.environment);
    case "approvalPolicy":
      return !isEqual(props.approvalPolicy, state.approvalPolicy);
    case "backupPolicy":
      return !isEqual(props.backupPolicy, state.backupPolicy);
    case "environmentTier":
      return !isEqual(props.environmentTier, state.environmentTier);

    default:
      return (
        !isEqual(props.environment, state.environment) ||
        !isEqual(props.approvalPolicy, state.approvalPolicy) ||
        !isEqual(props.backupPolicy, state.backupPolicy) ||
        !isEqual(props.environmentTier, state.environmentTier)
      );
  }
};

const revertEnvironment = () => {
  state.environment = cloneDeep(props.environment!);
  state.approvalPolicy = cloneDeep(props.approvalPolicy!);
  state.backupPolicy = cloneDeep(props.backupPolicy!);
  state.environmentTier = cloneDeep(props.environmentTier!);
};

const createEnvironment = () => {
  emit(
    "create",
    {
      name: resourceIdField.value?.resourceId,
      title: state.environment.title,
    },
    state.approvalPolicy,
    state.backupPolicy,
    state.environmentTier
  );
};

const updateEnvironment = () => {
  const env = props.environment;
  if (
    state.environment.title !== env.title ||
    state.environmentTier !== env.tier
  ) {
    const patchedEnvironment = {
      title: state.environment.title,
      tier: state.environmentTier,
    };
    emit("update", patchedEnvironment);
  }

  if (!isEqual(props.approvalPolicy, state.approvalPolicy)) {
    emit(
      "update-policy",
      state.environment,
      PolicyType.DEPLOYMENT_APPROVAL,
      state.approvalPolicy
    );
  }

  if (!isEqual(props.backupPolicy, state.backupPolicy)) {
    emit(
      "update-policy",
      state.environment,
      PolicyType.BACKUP_PLAN,
      state.backupPolicy
    );
  }
};

const archiveEnvironment = () => {
  emit("archive", state.environment);
};

const restoreEnvironment = () => {
  emit("restore", state.environment);
};

const toggleSQLReviewPolicy = async (on: boolean) => {
  const policy = sqlReviewPolicy.value;
  if (!policy) return;
  const originalOn = policy.enforce;
  if (on === originalOn) return;
  await useSQLReviewStore().updateReviewPolicy({
    id: policy.id,
    enforce: on,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("sql-review.policy-updated"),
  });
};
</script>
