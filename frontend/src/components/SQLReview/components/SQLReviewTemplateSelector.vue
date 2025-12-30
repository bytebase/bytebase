<template>
  <div class="flex flex-col gap-y-2">
    <p class="textlabel">
      {{ $t("sql-review.create.basic-info.choose-template") }}
      <span v-if="required" style="color: red">*</span>
    </p>

    <div
      class="flex flex-col sm:flex-row sm:flex-wrap justify-start items-stretch gap-x-4 gap-y-4"
    >
      <div
        v-for="template in reviewPolicyTemplateList"
        :key="template.id"
        class="relative border border-gray-300 hover:bg-gray-100 rounded-lg px-6 py-4 transition-all w-full h-full sm:max-w-xs"
        :class="
          isSelectedTemplate(template)
            ? 'bg-gray-100'
            : 'bg-transparent cursor-pointer'
        "
        @click="$emit('select-template', template)"
      >
        <div class="text-left flex flex-col gap-y-2">
          <span class="text-base font-medium">
            {{ template.review.name }}
          </span>
          <div class="flex flex-wrap gap-2">
            <NTag
              v-for="resource in template.review.resources"
              :key="resource"
              type="primary"
            >
              <Resource :resource="resource" :show-prefix="true" />
            </NTag>
          </div>
          <p class="text-sm">
            <span class="mr-2">{{ $t("sql-review.enabled-rules") }}:</span>
            <span>{{ enabledRuleCount(template) }}</span>
          </p>
        </div>
        <heroicons-solid:check-circle
          v-if="isSelectedTemplate(template)"
          class="w-7 h-7 text-accent absolute top-3 right-3"
        />
      </div>
    </div>

    <NDivider v-if="reviewPolicyTemplateList.length > 0" />

    <div
      class="flex flex-col sm:flex-row sm:flex-wrap justify-start items-stretch gap-x-4 gap-y-4"
    >
      <div
        v-for="template in builtInTemplateList"
        :key="template.id"
        class="relative border border-gray-300 hover:bg-gray-100 rounded-lg px-6 py-4 transition-all flex flex-col justify-center items-center w-full sm:max-w-xs"
        :class="
          isSelectedTemplate(template)
            ? 'bg-gray-100'
            : 'bg-transparent cursor-pointer'
        "
        @click="$emit('select-template', template)"
      >
        <div class="flex justify-center items-center gap-x-1">
          <div class="text-left">
            <span class="text-base font-medium">
              {{
                $t(`sql-review.template.${template.id.split(".").join("-")}`)
              }}
            </span>
            <p class="mt-1 text-sm text-gray-500">
              {{
                $t(
                  `sql-review.template.${template.id.split(".").join("-")}-desc`
                )
              }}
            </p>

            <p class="mt-1 text-xs">
              <span class="mr-2">{{ $t("sql-review.enabled-rules") }}:</span>
              <span>{{ enabledRuleCount(template) }}</span>
            </p>
          </div>
        </div>
        <heroicons-solid:check-circle
          v-if="isSelectedTemplate(template)"
          class="w-7 h-7 text-accent absolute top-3 right-3"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NDivider, NTag } from "naive-ui";
import { computed } from "vue";
import Resource from "@/components/v2/ResourceOccupiedModal/Resource.vue";
import { useSQLReviewPolicyList } from "@/store";
import type { SQLReviewPolicyTemplateV2 } from "@/types";
import { TEMPLATE_LIST_V2 as builtInTemplateList } from "@/types";
import { rulesToTemplate } from "./utils";

const props = withDefaults(
  defineProps<{
    title?: string;
    required?: boolean;
    selectedTemplateId?: string | undefined;
  }>(),
  {
    title: "",
    required: true,
    selectedTemplateId: undefined,
  }
);

defineEmits<{
  (event: "select-template", template: SQLReviewPolicyTemplateV2): void;
}>();

const reviewPolicyList = useSQLReviewPolicyList();

const reviewPolicyTemplateList = computed(() => {
  return reviewPolicyList.value.map((r) => rulesToTemplate(r));
});

const isSelectedTemplate = (template: SQLReviewPolicyTemplateV2) => {
  return template.id === props.selectedTemplateId;
};

const enabledRuleCount = (template: SQLReviewPolicyTemplateV2) => {
  return template.ruleList.length;
};
</script>
