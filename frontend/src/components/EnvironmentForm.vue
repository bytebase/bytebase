<template>
  <div class="px-4 py-2 space-y-6 divide-y divide-block-border">
    <div class="grid grid-cols-1 gap-y-6 gap-x-4">
      <div class="col-span-1">
        <label for="name" class="textlabel">
          {{ $t("common.environments") }} <span class="text-red-600">*</span>
        </label>
        <BBTextField
          class="mt-2 w-full"
          :disabled="!allowEdit"
          :required="true"
          :value="state.environment.name"
          @input="state.environment.name = $event.target.value"
        />
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
              v-model="state.approvalPolicy.payload.value"
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
          <div class="flex space-x-4">
            <input
              v-model="state.approvalPolicy.payload.value"
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
              v-model="state.backupPolicy.payload.schedule"
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
              v-model="state.backupPolicy.payload.schedule"
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
              v-model="state.backupPolicy.payload.schedule"
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
          {{ $t("schema-review-policy.title") }}
        </label>
        <div class="mt-3">
          <button
            v-if="schemaReviewPolicy"
            type="button"
            class="text-sm font-medium text-accent hover:underline"
            @click.prevent="onSchemaReviewPolicyClick"
          >
            {{ schemaReviewPolicy.name }}
            <span v-if="schemaReviewPolicy.rowStatus == 'ARCHIVED'">
              ({{ $t("schema-review-policy.disabled") }})
            </span>
          </button>
          <button
            v-else-if="hasPermission"
            type="button"
            class="btn-normal py-2 px-4 gap-x-1 items-center"
            @click.prevent="onSchemaReviewPolicyClick"
          >
            {{ $t("schema-review-policy.configure-policy") }}
          </button>
          <span v-else class="textinfolabel">
            {{ $t("schema-review-policy.no-policy-set") }}
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
      <template v-if="state.environment.rowStatus == 'NORMAL'">
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
      <template v-else-if="state.environment.rowStatus == 'ARCHIVED'">
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

<script lang="ts">
import {
  computed,
  reactive,
  PropType,
  watch,
  watchEffect,
  defineComponent,
} from "vue";
import { cloneDeep, isEqual, isEmpty } from "lodash-es";
import { useRouter } from "vue-router";
import {
  Environment,
  EnvironmentCreate,
  EnvironmentPatch,
  Policy,
  DatabaseSchemaReviewPolicy,
} from "../types";
import { isDev, isDBAOrOwner, schemaReviewPolicySlug } from "../utils";
import {
  useCurrentUser,
  useEnvironmentList,
  useSchemaSystemStore,
} from "@/store";

interface LocalState {
  environment: Environment | EnvironmentCreate;
  approvalPolicy: Policy;
  backupPolicy: Policy;
}

const ROUTE_NAME = "setting.workspace.schema-review-policy";

export default defineComponent({
  name: "EnvironmentForm",
  props: {
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
  },
  emits: ["create", "update", "cancel", "archive", "restore", "update-policy"],
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      environment: cloneDeep(props.environment),
      approvalPolicy: cloneDeep(props.approvalPolicy),
      backupPolicy: cloneDeep(props.backupPolicy),
    });

    const router = useRouter();
    const schemaSystemStore = useSchemaSystemStore();

    const environmentId = computed(() => {
      if (props.create) {
        return;
      }
      return (props.environment as Environment).id;
    });

    const prepareSchemaReviewPolicy = () => {
      if (!environmentId.value) {
        return;
      }
      return schemaSystemStore.fetchReviewPolicyByEnvironmentId(
        environmentId.value
      );
    };
    watchEffect(prepareSchemaReviewPolicy);

    const schemaReviewPolicy = computed(
      (): DatabaseSchemaReviewPolicy | undefined => {
        if (!environmentId.value) {
          return;
        }
        return schemaSystemStore.getReviewPolicyByEnvironmentId(
          environmentId.value
        );
      }
    );

    const onSchemaReviewPolicyClick = () => {
      if (schemaReviewPolicy.value) {
        router.push({
          name: `${ROUTE_NAME}.detail`,
          params: {
            schemaReviewPolicySlug: schemaReviewPolicySlug(
              schemaReviewPolicy.value
            ),
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
      field?: "environment" | "approvalPolicy" | "backupPolicy"
    ): boolean => {
      switch (field) {
        case "environment":
          return !isEqual(props.environment, state.environment);
        case "approvalPolicy":
          return !isEqual(props.approvalPolicy, state.approvalPolicy);
        case "backupPolicy":
          return !isEqual(props.backupPolicy, state.backupPolicy);

        default:
          return (
            !isEqual(props.environment, state.environment) ||
            !isEqual(props.approvalPolicy, state.approvalPolicy) ||
            !isEqual(props.backupPolicy, state.backupPolicy)
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
        state.backupPolicy
      );
    };

    const updateEnvironment = () => {
      if (state.environment.name != props.environment!.name) {
        const patchedEnvironment: EnvironmentPatch = {};

        patchedEnvironment.name = state.environment.name;
        emit("update", patchedEnvironment);
      }

      if (!isEqual(props.approvalPolicy, state.approvalPolicy)) {
        emit(
          "update-policy",
          (state.environment as Environment).id,
          "bb.policy.pipeline-approval",
          state.approvalPolicy
        );
      }

      if (!isEqual(props.backupPolicy, state.backupPolicy)) {
        emit(
          "update-policy",
          (state.environment as Environment).id,
          "bb.policy.backup-plan",
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

    return {
      isDev,
      state,
      allowArchive,
      allowRestore,
      allowEdit,
      valueChanged,
      allowCreate,
      hasPermission,
      revertEnvironment,
      createEnvironment,
      updateEnvironment,
      archiveEnvironment,
      restoreEnvironment,
      schemaReviewPolicy,
      onSchemaReviewPolicyClick,
    };
  },
});
</script>
