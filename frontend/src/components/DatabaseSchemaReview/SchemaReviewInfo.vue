<template>
  <div class="space-y-9">
    <div>
      <label class="textlabel">
        {{ $t("schema-review.create.basic-info.display-name") }}
        <span style="color: red">*</span>
      </label>
      <p class="mt-1 textinfolabel">
        {{ $t("schema-review.create.basic-info.display-name-label") }}
      </p>
      <BBTextField
        class="mt-2 w-full"
        placeholder="Database review name"
        :value="name"
        @input="$emit('name-change', $event.target.value)"
      />
    </div>
    <div>
      <label class="textlabel">
        {{ $t("schema-review.create.basic-info.environments") }}
      </label>
      <p class="mt-1 textinfolabel mb-5">
        {{ $t("schema-review.create.basic-info.environments-label") }}
      </p>
      <BBAttention
        v-if="availableEnvironmentList.length === 0"
        :style="'WARN'"
        :description="
          $t('schema-review.create.basic-info.no-available-environment')
        "
        class="mb-5"
      />
      <div class="flex flex-wrap gap-x-3">
        <div
          v-for="env in environmentList"
          :key="env.id"
          :class="[
            'flex items-center',
            env.disabled ? 'cursor-not-allowed text-gray-400' : 'text-gray-600',
          ]"
        >
          <input
            type="checkbox"
            :id="`${env.id}`"
            :value="env.id"
            :disabled="env.disabled"
            :checked="isEnvSelected(env)"
            @input="$emit('toggle-env', env)"
            class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
          />
          <label :for="`${env.id}`" class="ml-2 items-center text-sm">
            {{ env.displayName }}
          </label>
        </div>
      </div>
    </div>
    <div>
      <div class="mt-5" v-if="isEdit">
        <div
          class="flex cursor-pointer items-center px-2 text-indigo-500"
          @click="state.openTemplate = !state.openTemplate"
        >
          <heroicons-solid:chevron-right
            class="w-5 h-5 transform transition-all"
            :class="state.openTemplate ? 'rotate-90' : ''"
          />
          <span class="ml-3"> Change template </span>
        </div>
        <SchemaReviewTemplates
          v-if="state.openTemplate"
          :template-list="templateList"
          :selected-template-index="selectedTemplateIndex"
          :title="$t('schema-review.create.configure-rule.change-template')"
          @select="(index) => $emit('select-template', index)"
          class="mx-10 mt-5"
        />
      </div>
      <SchemaReviewTemplates
        v-else
        :template-list="templateList"
        :selected-template-index="selectedTemplateIndex"
        :title="$t('schema-review.create.basic-info.choose-template')"
        @select="(index) => $emit('select-template', index)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive, computed } from "vue";
import { useSchemaSystemStore, useEnvironmentList } from "@/store";
import { Environment, SchemaReviewTemplate } from "../../types";
import { environmentName } from "../../utils";

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
  selectedEnvironmentList: {
    required: true,
    type: Array as PropType<Environment[]>,
  },
  availableEnvironmentList: {
    required: true,
    type: Array as PropType<Environment[]>,
  },
  templateList: {
    required: true,
    type: Object as PropType<SchemaReviewTemplate[]>,
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

const emit = defineEmits(["name-change", "toggle-env", "select-template"]);

const state = reactive<LocalState>({
  openTemplate: false,
});

const store = useSchemaSystemStore();

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

const selectedEnvIdSet = computed(() => {
  return new Set(props.selectedEnvironmentList.map((env) => env.id));
});
const isEnvSelected = (env: Environment): boolean => {
  return selectedEnvIdSet.value.has(env.id);
};
</script>
