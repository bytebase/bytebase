<template>
  <div class="space-y-6">
    <div v-if="selectedEnvironment">
      <label class="textlabel">
        {{ $t("sql-review.create.basic-info.environments") }}
        <span style="color: red">*</span>
      </label>
      <p class="mt-1 textinfolabel">
        {{ $t("sql-review.create.basic-info.environments-label") }}
      </p>
      <BBAttention
        v-if="availableEnvironmentList.length === 0"
        class="mt-2"
        type="warning"
        :title="$t('common.environment')"
        :description="
          $t('sql-review.create.basic-info.no-available-environment-desc')
        "
      />
      <BBTextField
        class="mt-2 w-full"
        :value="environmentV1Name(selectedEnvironment)"
        :disabled="true"
      />
    </div>
    <div>
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
        :value="name"
        @update:value="emit('name-change', $event)"
      />
      <ResourceIdField
        ref="resourceIdField"
        class="mt-1"
        editing-class="mt-6"
        resource-type="review-config"
        :value="resourceId"
        :readonly="false"
        :resource-title="name"
        :suffix="true"
        :validate="validateResourceId"
        @update:value="emit('resource-id-change', $event)"
      />
    </div>
    <div>
      <SQLReviewTemplateSelector
        :required="true"
        :selected-template="selectedTemplate"
        @select-template="$emit('select-template', $event)"
        @templates-change="onTemplatesChange($event)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Status } from "nice-grpc-common";
import { type PropType, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBTextField } from "@/bbkit";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { reviewConfigServiceClient } from "@/grpcweb";
import { reviewConfigNamePrefix } from "@/store/modules/v1/common";
import type { SQLReviewPolicyTemplate } from "@/types";
import type { ResourceId, ValidatedMessage } from "@/types";
import type { Environment } from "@/types/proto/v1/environment_service";
import { environmentV1Name } from "@/utils";
import { getErrorCode } from "@/utils/grpcweb";
import { SQLReviewTemplateSelector } from "./components";

const props = defineProps({
  name: {
    required: true,
    type: String,
  },
  resourceId: {
    required: true,
    type: String,
  },
  isCreate: {
    required: true,
    type: Boolean,
  },
  selectedEnvironment: {
    required: true,
    type: Object as PropType<Environment>,
  },
  availableEnvironmentList: {
    required: true,
    type: Array as PropType<Environment[]>,
  },
  selectedTemplate: {
    type: Object as PropType<SQLReviewPolicyTemplate>,
    default: undefined,
  },
  isEdit: {
    required: true,
    type: Boolean,
  },
});

const emit = defineEmits<{
  (event: "name-change", name: string): void;
  (event: "resource-id-change", resourceId: string): void;
  (event: "select-template", template: SQLReviewPolicyTemplate): void;
}>();

const { t } = useI18n();

const onTemplatesChange = (templates: {
  policy: SQLReviewPolicyTemplate[];
  builtin: SQLReviewPolicyTemplate[];
}) => {
  if (!props.selectedTemplate) {
    emit("select-template", templates.policy[0] ?? templates.builtin[0]);
  }
};

const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

const validateResourceId = async (
  resourceId: ResourceId
): Promise<ValidatedMessage[]> => {
  if (!resourceId) {
    return [];
  }

  try {
    const existed = await reviewConfigServiceClient.getReviewConfig(
      {
        name: `${reviewConfigNamePrefix}${resourceId}`,
      },
      { silent: true }
    );
    if (existed) {
      return [
        {
          type: "error",
          message: t("resource-id.validation.duplicated", {
            resource: t("resource.review-config"),
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
</script>
