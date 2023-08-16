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
        class="relative border border-gray-300 hover:bg-gray-100 rounded-lg p-6 transition-all w-full h-full sm:max-w-xs"
        :class="
          isSelectedTemplate(template)
            ? 'bg-gray-100'
            : 'bg-transparent cursor-pointer'
        "
        @click="$emit('select-template', template)"
      >
        <div class="ml-6 text-left space-y-2">
          <span class="text-base mt-4 font-medium">
            {{ template.review.name }}
          </span>
          <p class="text-sm">
            <span class="mr-2">{{ $t("common.environment") }}:</span>
            <EnvironmentV1Name
              :environment="template.review.environment"
              :link="false"
            />
          </p>
          <p class="text-sm">
            <span class="mr-2">{{ $t("sql-review.enabled-rules") }}:</span>
            <span>{{ enabledRuleCount(template) }}</span>
          </p>
        </div>
        <heroicons-solid:check-circle
          v-if="isSelectedTemplate(template)"
          class="w-7 h-7 text-gray-500 absolute top-3 left-3"
        />
      </div>
    </div>

    <hr />

    <div
      class="flex flex-col sm:flex-row sm:flex-wrap justify-start items-stretch gap-x-10 gap-y-4"
    >
      <div
        v-for="template in builtInTemplateList"
        :key="template.id"
        class="relative border border-gray-300 hover:bg-gray-100 rounded-lg p-6 transition-all flex flex-col justify-center items-center w-full sm:max-w-xs"
        :class="
          isSelectedTemplate(template)
            ? 'bg-gray-100'
            : 'bg-transparent cursor-pointer'
        "
        @click="$emit('select-template', template)"
      >
        <div class="flex justify-center items-center space-x-1">
          <img class="w-24" :src="getTemplateImage(template.id)" alt="" />
          <div class="text-left">
            <span class="text-base mt-4 font-medium">
              {{
                $t(`sql-review.template.${template.id.split(".").join("-")}`)
              }}
            </span>
            <p class="mt-2 text-xs">
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
          class="w-7 h-7 text-gray-500 absolute top-3 left-3"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, watch } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useSQLReviewPolicyList } from "@/store";
import { SQLReviewPolicyTemplate } from "@/types";
import { TEMPLATE_LIST as builtInTemplateList } from "@/types";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import { rulesToTemplate } from "./utils";

const props = withDefaults(
  defineProps<{
    title?: string;
    required?: boolean;
    selectedTemplate?: SQLReviewPolicyTemplate | undefined;
  }>(),
  {
    title: "",
    required: true,
    selectedTemplate: undefined,
  }
);

const emit = defineEmits<{
  (event: "select-template", template: SQLReviewPolicyTemplate): void;
  (
    event: "templates-change",
    templateList: {
      policy: SQLReviewPolicyTemplate[];
      builtin: SQLReviewPolicyTemplate[];
    }
  ): void;
}>();

const reviewPolicyList = useSQLReviewPolicyList();

const reviewPolicyTemplateList = computed(() => {
  return reviewPolicyList.value.map((policy) =>
    rulesToTemplate(policy, false /* withDisabled=false */)
  );
});

const isSelectedTemplate = (template: SQLReviewPolicyTemplate) => {
  return template.id === props.selectedTemplate?.id;
};

const enabledRuleCount = (template: SQLReviewPolicyTemplate) => {
  return template.ruleList.filter(
    (rule) => rule.level !== SQLReviewRuleLevel.DISABLED
  ).length;
};

const getTemplateImage = (id: string) => {
  return new URL(`../../../assets/${id}.webp`, import.meta.url).href;
};

watch(
  reviewPolicyTemplateList,
  () => {
    emit("templates-change", {
      policy: reviewPolicyTemplateList.value,
      builtin: builtInTemplateList,
    });
  },
  { immediate: true }
);
</script>
