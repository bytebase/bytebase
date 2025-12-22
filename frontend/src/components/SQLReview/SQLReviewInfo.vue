<template>
  <div class="flex flex-col gap-y-6">
    <div v-if="attachedResources.length > 0">
      <label class="textlabel">
        {{ $t("sql-review.attach-resource.self") }}
        <span style="color: red">*</span>
      </label>
      <p class="mt-1 textinfolabel">
        {{ $t("sql-review.attach-resource.label") }}
      </p>
      <NRadioGroup
        v-model:value="resourceType"
        :disabled="!allowChangeAttachedResource"
        class="flex gap-x-2 mt-2"
      >
        <NRadio value="environment">{{ $t("common.environment") }}</NRadio>
        <NRadio value="project">{{ $t("common.project") }}</NRadio>
      </NRadioGroup>
      <BBAttention type="info" class="my-2">
        {{ $t(`sql-review.attach-resource.label-${resourceType}`) }}
      </BBAttention>
      <EnvironmentSelect
        v-if="resourceType === 'environment'"
        class="mt-2"
        required
        name="environment"
        :value="attachedResources[0]"
        :disabled="!allowChangeAttachedResource"
        :filter="
          (env: Environment) =>
            filterResource(formatEnvironmentName(env.id))
        "
        @update:value="
          (val) => {
            const name = val as string
            if (!name) {
              $emit('attached-resources-change', []);
            } else {
              $emit('attached-resources-change', [name]);
            }
          }
        "
      />
      <ProjectSelect
        v-if="resourceType === 'project'"
        class="mt-2"
        style="width: 100%"
        required
        :value="attachedResources[0]"
        :disabled="!allowChangeAttachedResource"
        :filter="(proj) => filterResource(proj.name)"
        @update:value="
          (val) => {
            const name = val as string
            if (!val) {
              $emit('attached-resources-change', []);
            } else {
              $emit('attached-resources-change', [name]);
            }
          }
        "
      />
      <DatabaseSelect
        v-if="resourceType === 'database'"
        class="mt-2"
        style="width: 100%"
        required
        :value="attachedResources[0]"
        :disabled="!allowChangeAttachedResource"
        :filter="(db: Database) => filterResource(db.name)"
        @update:value="(val) => {
          const name = val as string
          if (!name) {
            $emit('attached-resources-change', []);
          } else {
            $emit('attached-resources-change', [name]);
          }
        }"
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
        class="mt-1"
        editing-class="mt-6"
        resource-type="review-config"
        :value="resourceId"
        :readonly="!isCreate"
        :resource-title="name"
        :suffix="true"
        :fetch-resource="
          (id) =>
            sqlReviewStore.fetchReviewPolicyByName({
              name: `${reviewConfigNamePrefix}${id}`,
              silent: true,
            })
        "
        @update:value="emit('resource-id-change', $event)"
      />
    </div>
    <div>
      <SQLReviewTemplateSelector
        :required="true"
        :selected-template-id="selectedTemplateId"
        @select-template="$emit('select-template', $event)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NRadio, NRadioGroup } from "naive-ui";
import { computed, watch } from "vue";
import { BBAttention, BBTextField } from "@/bbkit";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useResourceByName } from "@/components/v2/ResourceOccupiedModal/useResourceByName";
import { useSQLReviewStore } from "@/store";
import { reviewConfigNamePrefix } from "@/store/modules/v1/common";
import type { SQLReviewPolicyTemplateV2 } from "@/types";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  type Environment,
  formatEnvironmentName,
} from "@/types/v1/environment";
import { DatabaseSelect, EnvironmentSelect, ProjectSelect } from "../v2";
import { SQLReviewTemplateSelector } from "./components";

const props = withDefaults(
  defineProps<{
    name: string;
    resourceId: string;
    attachedResources?: string[];
    isCreate: boolean;
    selectedTemplateId?: string;
    isEdit: boolean;
    allowChangeAttachedResource: boolean;
  }>(),
  {
    attachedResources: () => [],
  }
);

const emit = defineEmits<{
  (event: "name-change", name: string): void;
  (event: "resource-id-change", resourceId: string): void;
  (event: "attached-resources-change", resourceId: string[]): void;
  (event: "select-template", template: SQLReviewPolicyTemplateV2): void;
}>();

const sqlReviewStore = useSQLReviewStore();
const { resourceType } = useResourceByName({
  resource: computed(() => props.attachedResources[0] ?? ""),
});

watch(
  () => resourceType.value,
  () => emit("attached-resources-change", [])
);

const filterResource = (name: string): boolean => {
  if (!props.allowChangeAttachedResource) {
    return true;
  }
  return !sqlReviewStore.getReviewPolicyByResouce(name);
};
</script>
