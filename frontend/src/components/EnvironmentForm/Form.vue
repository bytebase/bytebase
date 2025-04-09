<template>
  <div>
    <div class="flex flex-col gap-y-6">
      <div v-if="features.includes('BASE')" class="flex flex-col gap-y-2">
        <div for="name" class="flex item-center space-x-2">
          <div class="w-4 h-4 relative">
            <component :is="renderColorPicker()" />
          </div>
          <span for="name" class="font-medium">
            {{ $t("common.environment-name") }}
            <span class="text-red-600">*</span>
          </span>
        </div>
        <NInput
          v-model:value="state.environment.title"
          :disabled="!allowEdit"
          required
        />

        <ResourceIdField
          ref="resourceIdField"
          resource-type="environment"
          :readonly="!create"
          :value="extractEnvironmentResourceName(state.environment.name)"
          :resource-title="state.environment.title"
          :validate="validateResourceId"
        />
      </div>

      <div v-if="features.includes('TIER')" class="flex flex-col gap-y-2">
        <label class="font-medium flex items-center">
          {{ $t("policy.environment-tier.name") }}
          <FeatureBadge feature="bb.feature.environment-tier-policy" />
        </label>
        <p class="text-sm text-gray-600">
          <i18n-t tag="span" keypath="policy.environment-tier.description">
            <template #newline><br /></template>
          </i18n-t>
          <a
            class="inline-flex items-center text-blue-600 ml-1 hover:underline"
            href="https://www.bytebase.com/docs/administration/environment-policy/overview/?source=console#environment-tier"
            target="_blank"
            >{{ $t("common.learn-more")
            }}<heroicons-outline:external-link class="w-4 h-4"
          /></a>
        </p>
        <NCheckbox
          :checked="state.environmentTier === EnvironmentTier.PROTECTED"
          :disabled="!allowEdit"
          @update:checked="
            (on: boolean) => {
              state.environmentTier = on
                ? EnvironmentTier.PROTECTED
                : EnvironmentTier.UNPROTECTED;
            }
          "
        >
          {{ $t("policy.environment-tier.mark-env-as-production") }}
        </NCheckbox>
      </div>

      <div
        v-if="features.includes('ROLLOUT_POLICY')"
        class="flex flex-col gap-y-2"
      >
        <div class="flex items-baseline space-x-2">
          <label class="font-medium">
            {{ $t("policy.rollout.name") }}
          </label>
          <span
            v-show="!create && valueChanged('rolloutPolicy')"
            class="textlabeltip"
          >
            {{ $t("policy.rollout.tip") }}
          </span>
        </div>
        <div class="textinfolabel">
          {{ $t("policy.rollout.info") }}
          <a
            class="inline-flex items-center text-blue-600 ml-1 hover:underline"
            href="https://www.bytebase.com/docs/administration/environment-policy/rollout-policy"
            target="_blank"
          >
            {{ $t("common.learn-more") }}
            <heroicons-outline:external-link class="w-4 h-4" />
          </a>
        </div>
        <RolloutPolicyConfig
          v-model:policy="state.rolloutPolicy"
          :disabled="!allowEdit"
        />
      </div>

      <SQLReviewForResource
        v-if="features.includes('SQL_REVIEW') && !create"
        ref="sqlReviewForResourceRef"
        :resource="environment.name"
        :allow-edit="allowEdit"
      />

      <AccessControlConfigure
        v-if="features.includes('ACCESS_CONTROL') && !create"
        ref="accessControlConfigureRef"
        :resource="environment.name"
        :allow-edit="allowEdit"
      />
    </div>

    <div
      v-if="!create && !hideArchiveRestore"
      class="mt-6 border-t border-block-border flex justify-between items-center pt-4 pb-2"
    >
      <template v-if="state.environment.state === State.ACTIVE">
        <BBButtonConfirm
          v-if="allowArchive"
          :type="'ARCHIVE'"
          :button-text="$t('environment.archive')"
          :ok-text="$t('common.archive')"
          :confirm-title="
            $t('environment.archive') + ` '${state.environment.title}'?`
          "
          :confirm-description="$t('environment.archive-info')"
          :require-confirm="true"
          @confirm="archiveEnvironment"
        />
      </template>
      <template v-else-if="state.environment.state === State.DELETED">
        <BBButtonConfirm
          v-if="allowRestore"
          :type="'RESTORE'"
          :button-text="$t('environment.restore')"
          :ok-text="$t('common.restore')"
          :confirm-title="
            $t('environment.restore') + ` '${state.environment.title}'?`
          "
          :confirm-description="''"
          :require-confirm="true"
          @confirm="restoreEnvironment"
        />
      </template>
      <div v-else></div>
    </div>
  </div>
</template>

<script lang="tsx" setup>
import { NCheckbox, NInput, NColorPicker } from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirm } from "@/bbkit";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import {
  useEnvironmentV1List,
  useEnvironmentV1Store,
  hasFeature,
  pushNotification,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import type { ResourceId, ValidatedMessage } from "@/types";
import { State } from "@/types/proto/v1/common";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import { extractEnvironmentResourceName } from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import { FeatureBadge } from "../FeatureGuard";
import SQLReviewForResource from "../SQLReview/components/SQLReviewForResource.vue";
import { ResourceIdField } from "../v2";
import AccessControlConfigure from "./AccessControlConfigure.vue";
import RolloutPolicyConfig from "./RolloutPolicyConfig.vue";
import { useEnvironmentFormContext } from "./context";

withDefaults(
  defineProps<{
    hideArchiveRestore?: boolean;
    features?: Array<
      "BASE" | "TIER" | "ROLLOUT_POLICY" | "SQL_REVIEW" | "ACCESS_CONTROL"
    >;
  }>(),
  {
    hideArchiveRestore: false,
    features: () => [
      "BASE",
      "TIER",
      "ROLLOUT_POLICY",
      "SQL_REVIEW",
      "ACCESS_CONTROL",
    ],
  }
);

const { t } = useI18n();
const {
  create,
  environment,
  state,
  allowEdit,
  valueChanged,
  missingFeature,
  hasPermission,
  events,
  resourceIdField,
} = useEnvironmentFormContext();
const environmentList = useEnvironmentV1List();

const accessControlConfigureRef =
  ref<InstanceType<typeof AccessControlConfigure>>();
const sqlReviewForResourceRef =
  ref<InstanceType<typeof SQLReviewForResource>>();

watch(
  () => [
    accessControlConfigureRef.value?.isDirty ?? false,
    sqlReviewForResourceRef.value?.isDirty ?? false,
  ],
  ([d1, d2]) => {
    if (d1 || d2) {
      state.value.policyChanged = true;
    } else if (!d1 && !d2) {
      state.value.policyChanged = false;
    }
  }
);

const hasEnvironmentPolicyFeature = computed(() =>
  hasFeature("bb.feature.environment-tier-policy")
);

const allowArchive = computed(() => {
  return (
    hasPermission("bb.environments.delete") && environmentList.value.length > 1
  );
});

// TODO(p0ny): hard-delete environment.
const allowRestore = computed(() => {
  return false;
});

const renderColorPicker = () => {
  return (
    <NColorPicker
      class="!w-full !h-full"
      modes={["hex"]}
      showAlpha={false}
      value={state.value.environment.color || "#4f46e5"}
      renderLabel={() => (
        <div
          class="w-5 h-5 rounded cursor-pointer relative"
          style={{
            backgroundColor: state.value.environment.color || "#4f46e5",
          }}
        ></div>
      )}
      onComplete={(color: string) => {
        if (color.toUpperCase() === "#FFFFFF") {
          pushNotification({
            module: "bytebase",
            style: "WARN",
            title: t("common.warning"),
            description: "Invalid color",
          });
          state.value.environment.color = "#4f46e5";
          return;
        }
      }}
      onUpdateValue={(color: string) => {
        if (!hasEnvironmentPolicyFeature.value) {
          missingFeature.value = "bb.feature.environment-tier-policy";
          return;
        }
        state.value.environment.color = color;
      }}
    />
  );
};

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const env = await useEnvironmentV1Store().getOrFetchEnvironmentByName(
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

const archiveEnvironment = () => {
  events.emit("archive", state.value.environment);
};

const restoreEnvironment = () => {
  events.emit("restore", state.value.environment);
};

useEmitteryEventListener(events, "update-access-control", async () => {
  if (accessControlConfigureRef.value?.isDirty) {
    await accessControlConfigureRef.value.update();
  }
});
useEmitteryEventListener(events, "revert-access-control", () => {
  accessControlConfigureRef.value?.revert();
});
useEmitteryEventListener(events, "update-sql-review", async () => {
  if (sqlReviewForResourceRef.value?.isDirty) {
    await sqlReviewForResourceRef.value.update();
  }
});
useEmitteryEventListener(events, "revert-sql-review", () => {
  sqlReviewForResourceRef.value?.revert();
});
</script>
