<template>
  <div class="space-y-5">
    <div>
      <label class="textlabel mt-4">
        {{ $t("schame-review.create.basic-info.display-name") }}
        <span style="color: red">*</span>
      </label>
      <p class="mt-1 textinfolabel">
        {{ $t("schame-review.create.basic-info.display-name-label") }}
      </p>
      <BBTextField
        class="mt-2 w-full"
        placeholder="Database review name"
        :value="name"
        @input="$emit('name-change', $event.target.value)"
      />
    </div>
    <div>
      <label class="textlabel mt-4">
        {{ $t("schame-review.create.basic-info.environments") }}
        <span style="color: red">*</span>
      </label>
      <p class="mt-1 textinfolabel mb-5">
        {{ $t("schame-review.create.basic-info.environments-label") }}
      </p>
      <BBAttention
        v-if="availableEnvironmentNameList.length === 0"
        :style="'WARN'"
        :description="
          $t('schame-review.create.basic-info.no-available-environment')
        "
        class="mb-5"
      />
      <div class="flex">
        <LabelSelect
          v-model:value="selectedEnvNameList"
          :options="availableEnvironmentNameList"
          :multiple="true"
          :placeholder="
            $t('schame-review.create.basic-info.environments-select')
          "
          class="flex items-center relative values py-1 border border-gray-300 rounded cursor-pointer"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { useSchemaSystemStore } from "@/store";
import { Environment } from "../../types";

const props = defineProps({
  name: {
    required: true,
    type: String,
  },
  selectedEnvNameList: {
    required: true,
    type: Array as PropType<string[]>,
  },
  environmentList: {
    required: true,
    type: Array as PropType<Environment[]>,
  },
});

const emit = defineEmits(["name-change"]);

const store = useSchemaSystemStore();

const availableEnvironmentNameList = computed(() => {
  const filteredList = store.availableEnvironments(props.environmentList);

  return [
    ...new Set([
      ...props.selectedEnvNameList,
      ...filteredList.map((e) => e.name),
    ]),
  ];
});
</script>
