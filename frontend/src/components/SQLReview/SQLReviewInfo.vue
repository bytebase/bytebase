<template>
  <div class="space-y-6">
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
        :readonly="!isCreate"
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
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBTextField } from "@/bbkit";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useSQLReviewStore } from "@/store";
import { reviewConfigNamePrefix } from "@/store/modules/v1/common";
import type { SQLReviewPolicyTemplateV2 } from "@/types";
import type { ResourceId, ValidatedMessage } from "@/types";
import { getErrorCode } from "@/utils/grpcweb";
import { SQLReviewTemplateSelector } from "./components";

const props = defineProps<{
  name: string;
  resourceId: string;
  isCreate: boolean;
  selectedTemplate?: SQLReviewPolicyTemplateV2;
  isEdit: boolean;
}>();

const emit = defineEmits<{
  (event: "name-change", name: string): void;
  (event: "resource-id-change", resourceId: string): void;
  (event: "select-template", template: SQLReviewPolicyTemplateV2): void;
}>();

const sqlReviewStore = useSQLReviewStore();
const { t } = useI18n();

const onTemplatesChange = (templates: {
  policy: SQLReviewPolicyTemplateV2[];
  builtin: SQLReviewPolicyTemplateV2[];
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
    const existed = await sqlReviewStore.fetchReviewPolicyByName({
      name: `${reviewConfigNamePrefix}${resourceId}`,
      silent: true,
    });
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
