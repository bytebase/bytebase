<template>
  <div class="space-y-6">
    <div v-if="attachedResources.length > 0">
      <label class="textlabel">
        {{ $t("sql-review.attach-resource.self") }}
        <span style="color: red">*</span>
      </label>
      <p class="mt-1 textinfolabel">
        {{ $t("sql-review.attach-resource.label") }}
      </p>
      <NRadioGroup
        v-model:value="attachResourceType"
        :disabled="!allowChangeAttachedResource"
        class="space-x-2 mt-2"
      >
        <NRadio value="environment">{{ $t("common.environment") }}</NRadio>
        <NRadio value="project">{{ $t("common.project") }}</NRadio>
      </NRadioGroup>
      <BBAttention type="info" class="my-2">
        {{ $t(`sql-review.attach-resource.label-${attachResourceType}`) }}
      </BBAttention>
      <EnvironmentSelect
        v-if="attachResourceType === 'environment'"
        class="mt-2"
        required
        name="environment"
        :environment="attachedResources[0]"
        :use-resource-id="true"
        :disabled="!allowChangeAttachedResource"
        :filter="(env: Environment, _: number) => filterResource(env.name)"
        @update:environment="
          (val: string | undefined) => {
            if (!val) {
              $emit('attached-resources-change', []);
            } else {
              $emit('attached-resources-change', [val]);
            }
          }
        "
      />
      <ProjectSelect
        v-if="attachResourceType === 'project'"
        class="mt-2"
        style="width: 100%"
        required
        :project="attachedResources[0]"
        :use-resource-id="true"
        :disabled="!allowChangeAttachedResource"
        :filter="(proj: Project, _: number) => filterResource(proj.name)"
        @update:project="
          (val: string | undefined) => {
            if (!val) {
              $emit('attached-resources-change', []);
            } else {
              $emit('attached-resources-change', [val]);
            }
          }
        "
      />
      <DatabaseSelect
        v-if="attachResourceType === 'database'"
        class="mt-2"
        style="width: 100%"
        required
        :multiple="true"
        :databases="attachedResources"
        :use-resource-id="true"
        :disabled="!allowChangeAttachedResource"
        :filter="(db: Database, _: number) => filterResource(db.name)"
        @update:databases="$emit('attached-resources-change', $event)"
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
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBTextField } from "@/bbkit";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useSQLReviewStore } from "@/store";
import {
  reviewConfigNamePrefix,
  projectNamePrefix,
  isDatabaseName,
} from "@/store/modules/v1/common";
import type { SQLReviewPolicyTemplateV2 } from "@/types";
import type { ResourceId, ValidatedMessage } from "@/types";
import type { Database } from "@/types/proto/v1/database_service";
import type { Environment } from "@/types/proto/v1/environment_service";
import type { Project } from "@/types/proto/v1/project_service";
import { getErrorCode } from "@/utils/grpcweb";
import { SQLReviewTemplateSelector } from "./components";
import { type ResourceType } from "./components/useReviewConfigAttachedResource";

const props = defineProps<{
  name: string;
  resourceId: string;
  attachedResources: string[];
  isCreate: boolean;
  selectedTemplate?: SQLReviewPolicyTemplateV2;
  isEdit: boolean;
  allowChangeAttachedResource: boolean;
}>();

const emit = defineEmits<{
  (event: "name-change", name: string): void;
  (event: "resource-id-change", resourceId: string): void;
  (event: "attached-resources-change", resourceId: string[]): void;
  (event: "select-template", template: SQLReviewPolicyTemplateV2): void;
}>();

const sqlReviewStore = useSQLReviewStore();

const attachResourceType = ref<ResourceType>(
  (function (): ResourceType {
    if (props.attachedResources.length === 0) {
      return "environment";
    }
    if (props.attachedResources.every((resource) => isDatabaseName(resource))) {
      return "database";
    }
    if (
      props.attachedResources.every((resource) =>
        resource.startsWith(projectNamePrefix)
      )
    ) {
      return "project";
    }
    return "environment";
  })()
);
const { t } = useI18n();

watch(
  () => attachResourceType.value,
  () => emit("attached-resources-change", [])
);

const filterResource = (name: string): boolean => {
  if (!props.allowChangeAttachedResource) {
    return true;
  }
  return !sqlReviewStore.getReviewPolicyByResouce(name);
};

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
