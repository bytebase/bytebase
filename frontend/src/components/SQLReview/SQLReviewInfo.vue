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
        :value="environmentName(selectedEnvironment)"
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
      <div v-if="isEdit" class="mt-5">
        <div
          class="flex cursor-pointer items-center text-indigo-500"
          @click="state.openTemplate = !state.openTemplate"
        >
          <heroicons-solid:chevron-right
            class="w-5 h-5 transform transition-all"
            :class="state.openTemplate ? 'rotate-90' : ''"
          />
          <span class="ml-l text-sm font-medium">
            {{ $t("sql-review.create.configure-rule.change-template") }}
          </span>
        </div>

        <template v-if="state.openTemplate">
          <SQLReviewTemplateSelector
            :required="false"
            :selected-template="selectedTemplate"
            @select-template="$emit('select-template', $event)"
          />
        </template>
      </div>
      <template v-else>
        <SQLReviewTemplateSelector
          :required="true"
          :selected-template="selectedTemplate"
          @select-template="$emit('select-template', $event)"
        />
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive } from "vue";
import { Environment, SQLReviewPolicyTemplate } from "@/types";
import { environmentName } from "@/utils";
import { BBTextField } from "@/bbkit";
import { SQLReviewTemplateSelector } from "./components";

interface LocalState {
  openTemplate: boolean;
}

defineProps({
  name: {
    required: true,
    type: String,
  },
  selectedEnvironment: {
    required: false,
    default: undefined,
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
  (event: "env-change", env: Environment): void;
  (event: "select-template", template: SQLReviewPolicyTemplate): void;
}>();

const state = reactive<LocalState>({
  openTemplate: false,
});

const onNameChange = (event: Event) => {
  emit("name-change", (event.target as HTMLInputElement).value);
};
</script>
