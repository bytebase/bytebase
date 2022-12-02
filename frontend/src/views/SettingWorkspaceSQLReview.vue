<template>
  <div class="mx-auto">
    <div class="textinfolabel">
      {{ $t("sql-review.description") }}
      <a
        href="https://www.bytebase.com/docs/sql-review/review-rules/overview"
        target="_blank"
        class="normal-link inline-flex flex-row items-center"
      >
        {{ $t("common.learn-more") }}
        <heroicons-outline:external-link class="w-4 h-4" />
      </a>
    </div>
    <FeatureAttention
      v-if="!hasSQLReviewPolicyFeature"
      custom-class="mt-5"
      feature="bb.feature.sql-review"
      :description="$t('subscription.features.bb-feature-sql-review.desc')"
    />
    <div v-if="store.reviewPolicyList.length > 0" class="space-y-6 my-5">
      <div v-if="hasPermission" class="flex items-center justify-end">
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click.prevent="goToCreationView"
        >
          {{ $t("sql-review.create-policy") }}
        </button>
      </div>
      <template v-for="review in store.reviewPolicyList" :key="review.id">
        <SQLReviewCard
          :review-policy="review"
          @click="onClick"
          @duplicate="onDuplicate"
        />
      </template>
    </div>
    <template v-else>
      <SQLReviewEmptyView @create="goToCreationView" />
    </template>
  </div>
  <BBModal
    v-if="state.showDuplicateModal && state.duplicatePolicy"
    :title="$t('sql-review.duplicate-policy')"
    @close="state.showDuplicateModal = false"
  >
    <div class="min-w-0 md:min-w-400">
      <div class="mt-2">
        <label class="textlabel">
          {{ $t("sql-review.create.basic-info.display-name") }}
          <span style="color: red">*</span>
        </label>
        <p class="mt-1 textinfolabel">
          {{ $t("sql-review.create.basic-info.display-name-label") }}
        </p>
        <BBTextField
          class="mt-2 w-full"
          :placeholder="
            $t('sql-review.create.basic-info.display-name-placeholder')
          "
          :value="state.duplicatePolicy.name"
          @input="onNameChange"
        />
      </div>
      <div class="mt-5">
        <label class="textlabel">
          {{ $t("sql-review.create.basic-info.environments") }}
          <span style="color: red">*</span>
        </label>
        <p class="mt-1 textinfolabel mb-5">
          {{ $t("sql-review.create.basic-info.environments-label") }}
        </p>
        <BBAttention
          v-if="availableEnvironmentList.length === 0"
          :style="'WARN'"
          :title="$t('common.environment')"
          :description="
            $t('sql-review.create.basic-info.no-available-environment-desc')
          "
          class="mb-5"
        />
        <div class="flex flex-wrap gap-x-3 px-1">
          <div
            v-for="env in availableEnvironmentList"
            :key="env.id"
            class="flex items-center"
          >
            <input
              :id="`${env.id}`"
              type="radio"
              :value="env.id"
              :checked="env.id === state.duplicatePolicy.environment?.id"
              class="text-accent disabled:text-accent-disabled cursor-pointer focus:ring-accent"
              @change="onEnvChange(env)"
            />
            <label
              :for="`${env.id}`"
              class="ml-2 items-center cursor-pointer text-sm"
            >
              {{ environmentName(env) }}
            </label>
          </div>
        </div>
      </div>
      <div class="mt-7 flex justify-end space-x-3">
        <button
          type="button"
          class="btn-normal"
          @click.prevent="state.showDuplicateModal = false"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          type="button"
          class="btn-primary"
          :disabled="duplicateButtonDisabled"
          @click.prevent="duplicatePolicy"
        >
          {{ $t("common.duplicate") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useSQLReviewStore,
  useCurrentUser,
  featureToRef,
  useEnvironmentList,
} from "@/store";
import { unknown, UNKNOWN_ID, SQLReviewPolicy, Environment } from "@/types";
import {
  environmentName,
  hasWorkspacePermission,
  sqlReviewPolicySlug,
} from "@/utils";

interface LocalState {
  showDuplicateModal: boolean;
  duplicatePolicy?: SQLReviewPolicy;
}

const state = reactive<LocalState>({
  showDuplicateModal: false,
});

const router = useRouter();
const store = useSQLReviewStore();
const ROUTE_NAME = "setting.workspace.sql-review";
const currentUser = useCurrentUser();
const { t } = useI18n();

watchEffect(() => {
  store.fetchReviewPolicyList();
});

const hasPermission = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-sql-review-policy",
    currentUser.value.role
  );
});

const hasSQLReviewPolicyFeature = featureToRef("bb.feature.sql-review");

const goToCreationView = () => {
  if (hasPermission.value) {
    router.push({
      name: `${ROUTE_NAME}.create`,
    });
  } else {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-review.no-permission"),
    });
  }
};

const onClick = (review: SQLReviewPolicy) => {
  router.push({
    name: `${ROUTE_NAME}.detail`,
    params: {
      sqlReviewPolicySlug: sqlReviewPolicySlug(review),
    },
  });
};

const onDuplicate = (policy: SQLReviewPolicy) => {
  state.showDuplicateModal = true;
  state.duplicatePolicy = {
    ...policy,
    environment: unknown("ENVIRONMENT") as Environment,
  };
};

const onNameChange = (event: Event) => {
  if (!state.duplicatePolicy) {
    return;
  }
  state.duplicatePolicy.name = (event.target as HTMLInputElement).value;
};

const onEnvChange = (env: Environment) => {
  if (!state.duplicatePolicy) {
    return;
  }
  state.duplicatePolicy.environment = env;
};

const availableEnvironmentList = computed((): Environment[] => {
  const environmentList = useEnvironmentList(["NORMAL"]);

  const filteredList = store.availableEnvironments(
    environmentList.value,
    undefined // undefined means we don't know the policy id, this shoud be a create action.
  );

  return filteredList;
});

const duplicateButtonDisabled = computed((): boolean => {
  if (!state.duplicatePolicy) {
    return true;
  }

  if (!state.duplicatePolicy.name) {
    return true;
  }

  if (state.duplicatePolicy.environment.id === UNKNOWN_ID) {
    return true;
  }

  return false;
});

const duplicatePolicy = () => {
  if (!state.duplicatePolicy || duplicateButtonDisabled.value) {
    return;
  }

  store
    .addReviewPolicy({
      name: state.duplicatePolicy.name,
      ruleList: state.duplicatePolicy.ruleList,
      environmentId: state.duplicatePolicy.environment.id,
    })
    .then(() => {
      state.duplicatePolicy = undefined;
      state.showDuplicateModal = false;
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-review.policy-duplicated"),
      });
    });
};
</script>
