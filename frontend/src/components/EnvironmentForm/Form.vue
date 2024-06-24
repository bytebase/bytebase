<template>
  <div>
    <div class="flex flex-col gap-y-6">
      <div class="flex flex-col gap-y-2">
        <label for="name" class="textlabel">
          {{ $t("common.environment-name") }}
          <span class="text-red-600">*</span>
        </label>
        <NInput
          v-model:value="state.environment.title"
          :disabled="!allowEdit"
          size="large"
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

      <div class="flex flex-col gap-y-2">
        <label class="textlabel flex items-center">
          {{ $t("policy.environment-tier.name") }}
          <FeatureBadge feature="bb.feature.environment-tier-policy" />
        </label>
        <p class="text-sm text-gray-600">
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
        <NCheckbox
          :checked="state.environmentTier === EnvironmentTier.PROTECTED"
          :disabled="!allowEdit"
          style="--n-label-padding: 0 0 0 1rem"
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

      <template v-if="!simple">
        <div class="flex flex-col gap-y-2">
          <label class="textlabel">
            {{ $t("policy.rollout.name") }}
          </label>
          <span
            v-show="!create && valueChanged('rolloutPolicy')"
            class="textlabeltip !ml-0"
            >{{ $t("policy.rollout.tip") }}</span
          >
          <div class="textinfolabel">
            {{ $t("policy.rollout.info") }}
            <a
              class="inline-flex items-center text-blue-600 ml-1 hover:underline"
              href="https://www.bytebase.com/docs/administration/environment-policy/rollout-policy"
              target="_blank"
              >{{ $t("common.learn-more")
              }}<heroicons-outline:external-link class="w-4 h-4"
            /></a>
          </div>
          <RolloutPolicyConfig
            v-model:policy="state.rolloutPolicy"
            :disabled="!allowEdit"
          />
        </div>

        <SQLReviewForResource
          v-if="!create"
          :resource="environment.name"
          :allow-edit="allowEdit"
        />

        <div v-if="!create" class="flex flex-col gap-y-2">
          <div class="textlabel flex items-center space-x-1">
            <label>
              {{ $t("environment.access-control.title") }}
            </label>
            <FeatureBadge feature="bb.feature.access-control" />
          </div>
          <div>
            <div class="inline-flex items-center gap-x-2">
              <Switch
                :value="disableCopyDataPolicy"
                :text="true"
                :disabled="!allowEditDisableCopyData"
                @update:value="upsertPolicy"
              />
              <span class="textlabel">{{
                $t(
                  "environment.access-control.disable-copy-data-from-sql-editor"
                )
              }}</span>
            </div>
          </div>
        </div>
      </template>
    </div>

    <div
      v-if="!create && !hideArchiveRestore"
      class="mt-6 flex justify-between items-center pt-5"
    >
      <template v-if="state.environment.state === State.ACTIVE">
        <BBButtonConfirm
          v-if="allowArchive"
          :style="'ARCHIVE'"
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
          :style="'RESTORE'"
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

<script lang="ts" setup>
import { NCheckbox } from "naive-ui";
import { Status } from "nice-grpc-common";
import { computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import {
  hasFeature,
  pushNotification,
  useEnvironmentV1List,
  useEnvironmentV1Store,
  usePolicyV1Store,
} from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import type { ResourceId, ValidatedMessage } from "@/types";
import { State } from "@/types/proto/v1/common";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import {
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { extractEnvironmentResourceName } from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import { useEnvironmentFormContext } from "./context";

defineProps<{
  simple?: boolean;
  hideArchiveRestore?: boolean;
}>();

const { t } = useI18n();
const {
  create,
  environment,
  state,
  allowEdit,
  valueChanged,
  hasPermission,
  events,
  resourceIdField,
} = useEnvironmentFormContext();
const policyStore = usePolicyV1Store();
const environmentList = useEnvironmentV1List();

const disableCopyDataPolicy = computed(() => {
  const policies = policyStore.policyList.filter(
    (policy) =>
      policy.resourceType === PolicyResourceType.ENVIRONMENT &&
      policy.type === PolicyType.DISABLE_COPY_DATA &&
      policy.resourceUid === environment.value.uid &&
      policy.disableCopyDataPolicy?.active
  );
  return policies.length > 0;
});

const allowEditDisableCopyData = computed(() => {
  return hasPermission("bb.policies.update");
});

const allowArchive = computed(() => {
  return (
    hasPermission("bb.environments.delete") && environmentList.value.length > 1
  );
});

const allowRestore = computed(() => {
  return hasPermission("bb.environments.undelete");
});

const prepareEnvironmentDisableCopyDataPolicy = async () => {
  await policyStore.fetchPolicies({
    resourceType: PolicyResourceType.ENVIRONMENT,
    policyType: PolicyType.DISABLE_COPY_DATA,
  });
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

const upsertPolicy = async (on: boolean) => {
  if (!hasFeature("bb.feature.access-control")) {
    state.value.missingRequiredFeature = "bb.feature.access-control";
    return;
  }

  await policyStore.createPolicy(environment.value.name, {
    type: PolicyType.DISABLE_COPY_DATA,
    resourceType: PolicyResourceType.ENVIRONMENT,
    disableCopyDataPolicy: {
      active: on,
    },
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const archiveEnvironment = () => {
  events.emit("archive", state.value.environment);
};

const restoreEnvironment = () => {
  events.emit("restore", state.value.environment);
};

onMounted(() => {
  prepareEnvironmentDisableCopyDataPolicy();
});
</script>
