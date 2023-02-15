<template>
  <div class="space-y-9">
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
      <label class="textlabel">
        {{ $t("sql-review.create.basic-info.environments") }}
        <span style="color: red">*</span>
      </label>
      <p class="mt-1 textinfolabel mb-5">
        {{ $t("sql-review.create.basic-info.environments-label") }}
      </p>
      <BBAttention
        v-if="availableEnvironmentList.length === 0"
        :style="'WARN'"
        :title="$t('common.environment')"
        :description="
          $t('sql-review.create.basic-info.no-available-environment-desc')
        "
        class="mb-5"
      />
      <div v-if="!isEdit" class="flex flex-wrap gap-x-3">
        <div
          v-for="env in environmentList"
          :key="env.id"
          class="flex items-center"
        >
          <input
            :id="`${env.id}`"
            type="radio"
            :value="env.id"
            :disabled="env.disabled"
            :checked="env.id === selectedEnvironment?.id"
            class="text-accent disabled:text-accent-disabled focus:ring-accent"
            :class="env.disabled ? 'cursor-not-allowed' : 'cursor-pointer'"
            @change="$emit('env-change', env)"
          />
          <label
            :for="`${env.id}`"
            class="ml-2 items-center text-sm"
            :class="
              env.disabled
                ? 'cursor-not-allowed text-gray-400'
                : 'cursor-pointer'
            "
          >
            {{ env.displayName }}
          </label>
        </div>
      </div>
      <BBBadge
        v-else-if="isEdit && selectedEnvironment"
        :text="environmentName(selectedEnvironment)"
        :can-remove="false"
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
        <SQLReviewTemplates
          v-if="state.openTemplate"
          :required="false"
          :template-list="templateList"
          :selected-template-index="selectedTemplateIndex"
          class="mx-5 mt-5"
          @select="(index) => $emit('select-template', index)"
        />
      </div>
      <SQLReviewTemplates
        v-else
        :required="true"
        :template-list="templateList"
        :selected-template-index="selectedTemplateIndex"
        :title="$t('sql-review.create.basic-info.choose-template')"
        @select="(index) => $emit('select-template', index)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive, computed } from "vue";
import { useEnvironmentList } from "@/store";
import { Environment, SQLReviewPolicyTemplate } from "@/types";
import { environmentName } from "@/utils";
import { BBTextField } from "@/bbkit";
import { SQLReviewTemplates } from "./components";

interface LocalEnvironment extends Environment {
  disabled: boolean;
  displayName: string;
}

interface LocalState {
  openTemplate: boolean;
}

const props = defineProps({
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
  templateList: {
    required: true,
    type: Object as PropType<SQLReviewPolicyTemplate[]>,
  },
  selectedTemplateIndex: {
    required: true,
    type: Number,
  },
  isEdit: {
    required: true,
    type: Boolean,
  },
});

const emit = defineEmits(["name-change", "env-change", "select-template"]);

const state = reactive<LocalState>({
  openTemplate: false,
});

const environmentList = computed((): LocalEnvironment[] => {
  const environmentList = useEnvironmentList(["NORMAL"]);
  const availableIdSet = new Set(
    props.availableEnvironmentList.map((env) => env.id)
  );

  return environmentList.value.map((env) => ({
    ...env,
    disabled: !availableIdSet.has(env.id),
    displayName: environmentName(env),
  }));
});

const onNameChange = (event: Event) => {
  emit("name-change", (event.target as HTMLInputElement).value);
};
</script>
