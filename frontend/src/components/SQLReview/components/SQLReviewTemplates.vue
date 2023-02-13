<template>
  <div>
    <p v-if="title" class="textlabel">
      {{ title }}
      <span v-if="required" style="color: red">*</span>
    </p>
    <div
      class="flex flex-col sm:flex-row justify-start items-center gap-x-10 gap-y-10 mt-4"
    >
      <div
        v-for="(template, index) in templateList"
        :key="index"
        class="relative border border-gray-300 hover:bg-gray-100 rounded-lg p-6 transition-all flex flex-col justify-center items-center w-full sm:max-w-xs"
        :class="
          index == selectedTemplateIndex
            ? 'bg-gray-100'
            : 'bg-transparent cursor-pointer'
        "
        @click="$emit('select', index)"
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
          </div>
        </div>
        <heroicons-solid:check-circle
          v-if="index == selectedTemplateIndex"
          class="w-7 h-7 text-gray-500 absolute top-3 left-3"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType } from "vue";
import { SQLReviewPolicyTemplate } from "@/types/sqlReview";

defineProps({
  templateList: {
    required: true,
    type: Object as PropType<SQLReviewPolicyTemplate[]>,
  },
  selectedTemplateIndex: {
    required: false,
    default: -1,
    type: Number,
  },
  title: {
    required: false,
    default: "",
    type: String,
  },
  required: {
    required: true,
    type: Boolean,
  },
});

defineEmits(["select"]);

const getTemplateImage = (id: string) =>
  new URL(`../../../assets/${id}.webp`, import.meta.url).href;
</script>
