<template>
  <div class="px-4 py-2 space-y-6 divide-y divide-block-border">
    <div class="grid grid-cols-1 gap-y-6 gap-x-4">
      <div class="col-span-1">
        <label for="name" class="textlabel">
          {{ $t("common.environment-name") }}
          <span class="text-red-600">*</span>
        </label>
        <BBTextField
          class="mt-2 w-full"
          :disabled="!allowEdit"
          :required="true"
          :value="state.environment.name"
          @input="
            state.environment.name = ($event.target as HTMLInputElement).value
          "
        />
      </div>

      <div class="col-span-1">
        <label class="textlabel flex items-center">
          {{ $t("policy.environment-tier.name") }}
          <FeatureBadge
            feature="bb.feature.environment-tier-policy"
            class="text-accent"
          />
        </label>
        <div class="mt-4 flex flex-col space-y-4">
          <div class="flex space-x-4">
            <BBCheckbox
              :value="(state.environmentTierPolicy.payload as EnvironmentTierPolicyPayload).environmentTier === 'PROTECTED'"
              @toggle="on => {
                (state.environmentTierPolicy.payload as EnvironmentTierPolicyPayload).environmentTier = on ? 'PROTECTED' : 'UNPROTECTED'
              }"
            />
            <div>
              <div class="textlabel">
                {{ $t("policy.environment-tier.mark-env-as-protected") }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="col-span-1">
        <label class="textlabel"> {{ $t("policy.approval.name") }} </label>
        <span v-show="valueChanged('approvalPolicy')" class="textlabeltip">{{
          $t("policy.approval.tip")
        }}</span>
        <div class="mt-1 textinfolabel">
          {{ $t("policy.approval.info") }}
        </div>
        <div class="mt-4 flex flex-col space-y-4">
          <div class="flex space-x-4">
            <input
              v-model="(state.approvalPolicy.payload as PipelineApprovalPolicyPayload).value"
              name="manual-approval-always"
              tabindex="-1"
              type="radio"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
              value="MANUAL_APPROVAL_ALWAYS"
              :disabled="!allowEdit"
            />
            <div class="-mt-0.5">
              <div class="textlabel">{{ $t("policy.approval.manual") }}</div>
              <div class="mt-1 textinfolabel">
                {{ $t("policy.approval.manual-info") }}
              </div>
            </div>
          </div>

          <AssigneeGroupEditor
            class="ml-8"
            :policy="state.approvalPolicy"
            :allow-edit="allowEdit"
            @update="(assigneeGroupList) => {
              (state.approvalPolicy.payload as PipelineApprovalPolicyPayload).assigneeGroupList = assigneeGroupList
            }"
          />

          <div class="flex space-x-4">
            <input
              v-model="(state.approvalPolicy.payload as PipelineApprovalPolicyPayload).value"
              name="manual-approval-never"
              tabindex="-1"
              type="radio"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
              value="MANUAL_APPROVAL_NEVER"
              :disabled="!allowEdit"
            />
            <div class="-mt-0.5">
              <div class="textlabel flex">
                {{ $t("policy.approval.auto") }}
                <FeatureBadge
                  feature="bb.feature.approval-policy"
                  class="text-accent"
                />
              </div>
              <div class="mt-1 textinfolabel">
                {{ $t("policy.approval.auto-info") }}
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-span-1">
        <label class="textlabel"> {{ $t("policy.backup.name") }} </label>
        <span v-show="valueChanged('backupPolicy')" class="textlabeltip">{{
          $t("policy.backup.tip")
        }}</span>
        <div class="mt-4 flex flex-col space-y-4">
          <div class="flex space-x-4">
            <input
              v-model="(state.backupPolicy.payload as BackupPlanPolicyPayload).schedule"
              tabindex="-1"
              type="radio"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
              value="UNSET"
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
              v-model="(state.backupPolicy.payload as BackupPlanPolicyPayload).schedule"
              tabindex="-1"
              type="radio"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
              value="DAILY"
              :disabled="!allowEdit"
            />
            <div class="-mt-0.5">
              <div class="textlabel flex">
                {{ $t("policy.backup.daily") }}
                <FeatureBadge
                  feature="bb.feature.backup-policy"
                  class="text-accent"
                />
              </div>
              <div class="mt-1 textinfolabel">
                {{ $t("policy.backup.daily-info") }}
              </div>
            </div>
          </div>
          <div class="flex space-x-4">
            <input
              v-model="(state.backupPolicy.payload as BackupPlanPolicyPayload).schedule"
              tabindex="-1"
              type="radio"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
              value="WEEKLY"
              :disabled="!allowEdit"
            />
            <div class="-mt-0.5">
              <div class="textlabel flex">
                {{ $t("policy.backup.weekly") }}
                <FeatureBadge
                  feature="bb.feature.backup-policy"
                  class="text-accent"
                />
              </div>
              <div class="mt-1 textinfolabel">
                {{ $t("policy.backup.weekly-info") }}
              </div>
            </div>
          </div>
        </div>
      </div>
      <div v-if="!create" class="col-span-1">
        <label class="textlabel">
          {{ $t("sql-review.title") }}
        </label>
        <div class="mt-3">
          <button
            v-if="sqlReviewPolicy"
            type="button"
            class="text-sm font-medium text-accent hover:underline"
            @click.prevent="onSQLReviewPolicyClick"
          >
            {{ sqlReviewPolicy.name }}
            <span v-if="sqlReviewPolicy.rowStatus == 'ARCHIVED'">
              ({{ $t("sql-review.disabled") }})
            </span>
          </button>
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
    <!-- Create button group -->
    <div v-if="create" class="flex justify-end pt-5">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="$emit('cancel')"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        type="submit"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="createEnvironment"
      >
        {{ $t("common.create") }}
      </button>
    </div>
    <!-- Update button group -->
    <div v-else class="flex justify-between items-center pt-5">
      <template v-if="(state.environment as Environment).rowStatus == 'NORMAL'">
        <BBButtonConfirm
          v-if="allowArchive"
          :style="'ARCHIVE'"
          :button-text="$t('environment.archive')"
          :ok-text="$t('common.archive')"
          :confirm-title="
            $t('environment.archive') + ` '${state.environment.name}'?`
          "
          :confirm-description="$t('environment.archive-info')"
          :require-confirm="true"
          @confirm="archiveEnvironment"
        />
      </template>
      <template
        v-else-if="(state.environment as Environment).rowStatus == 'ARCHIVED'"
      >
        <BBButtonConfirm
          v-if="allowRestore"
          :style="'RESTORE'"
          :button-text="$t('environment.restore')"
          :ok-text="$t('common.restore')"
          :confirm-title="
            $t('environment.restore') + ` '${state.environment.name}'?`
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
</template>

<script lang="ts" setup>
import { computed, reactive, PropType, watch, watchEffect } from "vue";
import { cloneDeep, isEqual, isEmpty } from "lodash-es";
import { useRouter } from "vue-router";
import type {
  BackupPlanPolicyPayload,
  Environment,
  EnvironmentCreate,
  EnvironmentPatch,
  EnvironmentTierPolicyPayload,
  PipelineApprovalPolicyPayload,
  Policy,
  SQLReviewPolicy,
} from "../types";
import { isDBAOrOwner, sqlReviewPolicySlug } from "../utils";
import { useCurrentUser, useEnvironmentList, useSQLReviewStore } from "@/store";
import AssigneeGroupEditor from "./EnvironmentForm/AssigneeGroupEditor.vue";

interface LocalState {
  environment: Environment | EnvironmentCreate;
  approvalPolicy: Policy;
  backupPolicy: Policy;
  environmentTierPolicy: Policy;
}

const ROUTE_NAME = "setting.workspace.sql-review";

const props = defineProps({
  create: {
    type: Boolean,
    default: false,
  },
  environment: {
    required: true,
    type: Object as PropType<Environment | EnvironmentCreate>,
  },
  approvalPolicy: {
    required: true,
    type: Object as PropType<Policy>,
  },
  backupPolicy: {
    required: true,
    type: Object as PropType<Policy>,
  },
  environmentTierPolicy: {
    required: true,
    type: Object as PropType<Policy>,
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
const state = reactive<LocalState>({
  environment: cloneDeep(props.environment),
  approvalPolicy: cloneDeep(props.approvalPolicy),
  backupPolicy: cloneDeep(props.backupPolicy),
  environmentTierPolicy: cloneDeep(props.environmentTierPolicy),
});

const router = useRouter();
const sqlReviewStore = useSQLReviewStore();

const environmentId = computed(() => {
  if (props.create) {
    return;
  }
  return (props.environment as Environment).id;
});

const prepareSQLReviewPolicy = () => {
  if (!environmentId.value) {
    return;
  }
  return sqlReviewStore.fetchReviewPolicyByEnvironmentId(environmentId.value);
};
watchEffect(prepareSQLReviewPolicy);

const sqlReviewPolicy = computed((): SQLReviewPolicy | undefined => {
  if (!environmentId.value) {
    return;
  }
  return sqlReviewStore.getReviewPolicyByEnvironmentId(environmentId.value);
});

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
  (cur: Environment | EnvironmentCreate) => {
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
  () => props.environmentTierPolicy,
  (cur: Policy) => {
    state.environmentTierPolicy = cloneDeep(cur);
  }
);

const currentUser = useCurrentUser();

const environmentList = useEnvironmentList();

const hasPermission = computed(() => {
  return isDBAOrOwner(currentUser.value.role);
});

const allowArchive = computed(() => {
  return allowEdit.value && environmentList.value.length > 1;
});

const allowRestore = computed(() => {
  return hasPermission.value;
});

const allowEdit = computed(() => {
  return (
    props.create ||
    ((state.environment as Environment).rowStatus == "NORMAL" &&
      hasPermission.value)
  );
});

const allowCreate = computed(() => {
  return !isEmpty(state.environment?.name);
});

const valueChanged = (
  field?:
    | "environment"
    | "approvalPolicy"
    | "backupPolicy"
    | "environmentTierPolicy"
): boolean => {
  switch (field) {
    case "environment":
      return !isEqual(props.environment, state.environment);
    case "approvalPolicy":
      return !isEqual(props.approvalPolicy, state.approvalPolicy);
    case "backupPolicy":
      return !isEqual(props.backupPolicy, state.backupPolicy);
    case "environmentTierPolicy":
      return !isEqual(props.environmentTierPolicy, state.environmentTierPolicy);

    default:
      return (
        !isEqual(props.environment, state.environment) ||
        !isEqual(props.approvalPolicy, state.approvalPolicy) ||
        !isEqual(props.backupPolicy, state.backupPolicy) ||
        !isEqual(props.environmentTierPolicy, state.environmentTierPolicy)
      );
  }
};

const revertEnvironment = () => {
  state.environment = cloneDeep(props.environment!);
};

const createEnvironment = () => {
  emit(
    "create",
    state.environment,
    state.approvalPolicy,
    state.backupPolicy,
    state.environmentTierPolicy
  );
};

const updateEnvironment = () => {
  if (state.environment.name != props.environment!.name) {
    const patchedEnvironment: EnvironmentPatch = {};

    patchedEnvironment.name = state.environment.name;
    emit("update", patchedEnvironment);
  }

  const environmentId = (state.environment as Environment).id;

  if (!isEqual(props.approvalPolicy, state.approvalPolicy)) {
    emit(
      "update-policy",
      environmentId,
      "bb.policy.pipeline-approval",
      state.approvalPolicy
    );
  }

  if (!isEqual(props.backupPolicy, state.backupPolicy)) {
    emit(
      "update-policy",
      environmentId,
      "bb.policy.backup-plan",
      state.backupPolicy
    );
  }

  if (!isEqual(props.environmentTierPolicy, state.environmentTierPolicy)) {
    emit(
      "update-policy",
      environmentId,
      "bb.policy.environment-tier",
      state.environmentTierPolicy
    );
  }
};

const archiveEnvironment = () => {
  emit("archive", state.environment);
};

const restoreEnvironment = () => {
  emit("restore", state.environment);
};
</script>
