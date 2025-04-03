<template>
  <div class="flex flex-col gap-y-2">
    <p class="textlabel">
      {{ $t("sql-review.create.basic-info.choose-template") }}
      <span v-if="required" style="color: red">*</span>
    </p>

    <div
      class="flex flex-col sm:flex-row sm:flex-wrap justify-start items-stretch gap-x-10 gap-y-4"
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
        <div class="text-left space-y-2">
          <span class="text-base font-medium">
            {{ template.review.name }}
          </span>
          <div class="space-y-2">
            <BBBadge
              v-for="resource in template.review.resources"
              :key="resource"
              :can-remove="false"
            >
              <Resource :resource="resource" :show-prefix="true" />
            </BBBadge>
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

    <NDivider />

    <div
      class="flex flex-col sm:flex-row sm:flex-wrap justify-start items-stretch gap-x-10 gap-y-4"
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
        <div class="flex justify-center items-center space-x-1">
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
import { NDivider } from "naive-ui";
import { computed } from "vue";
import { BBBadge } from "@/bbkit";
import Resource from "@/components/v2/ResourceOccupiedModal/Resource.vue";
import { useSQLReviewPolicyList } from "@/store";
import type { SQLReviewPolicyTemplateV2 } from "@/types";
import { TEMPLATE_LIST_V2 as builtInTemplateList } from "@/types";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
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
  return reviewPolicyList.value.map((r) => rulesToTemplate(r, false));
});

const isSelectedTemplate = (template: SQLReviewPolicyTemplateV2) => {
  return template.id === props.selectedTemplateId;
};

const enabledRuleCount = (template: SQLReviewPolicyTemplateV2) => {
  return template.ruleList.filter(
    (rule) => rule.level !== SQLReviewRuleLevel.DISABLED
  ).length;
};
</script>
