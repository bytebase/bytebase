<template>
  <div>
    <p class="textlabel">
      {{ title }}
    </p>
    <div
      class="flex flex-col sm:flex-row justify-start items-center gap-x-10 gap-y-10 mt-4"
    >
      <div
        v-for="(template, index) in templateList"
        :key="template.name"
        class="cursor-pointer bg-transparent border border-gray-300 hover:bg-gray-100 rounded-lg p-6 transition-all flex flex-col justify-center items-center w-full sm:max-w-xs"
        @click="$emit('select', index)"
      >
        <img class="h-24" :src="template.image" alt="" />
        <span class="text-lg lg:text-xl mt-4">{{ template.name }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { SchemaReviewTemplate } from "../../types/schemaSystem";

const props = defineProps({
  templateList: {
    required: true,
    type: Object as PropType<SchemaReviewTemplate[]>,
  },
  title: {
    required: true,
    type: String,
  },
});

const emit = defineEmits(["select"]);
</script>
