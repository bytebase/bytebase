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
        :style="'WARN'"
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
        @input="(e) => onNameChange(e)"
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
import { PropType } from "vue";
import { BBTextField } from "@/bbkit";
import { SQLReviewPolicyTemplate } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { environmentV1Name } from "@/utils";
import { SQLReviewTemplateSelector } from "./components";

const props = defineProps({
  name: {
    required: true,
    type: String,
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
  (event: "select-template", template: SQLReviewPolicyTemplate): void;
}>();

const onNameChange = (event: Event) => {
  emit("name-change", (event.target as HTMLInputElement).value);
};

const onTemplatesChange = (templates: {
  policy: SQLReviewPolicyTemplate[];
  builtin: SQLReviewPolicyTemplate[];
}) => {
  if (!props.selectedTemplate) {
    emit("select-template", templates.policy[0] ?? templates.builtin[0]);
  }
};
</script>
