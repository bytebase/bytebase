<template>
  <BBModal
    :title="'Unable to update restriction'"
    class="!w-[32rem] !max-w-full"
    @close="emits('close')"
  >
    <template #header>
      <div>{{ $t("custom-approval.security-rule.template.view") }}</div>
    </template>
    <div class="w-full">
      <p class="mt-1 mb-2">
        {{ "Following column types are still used by field tempalte:" }}
      </p>
      <div
        v-for="unmatchedField in unmatchedFieldMap.keys()"
        :key="unmatchedField"
        class="w-full flex flex-row justify-start items-center mt-2"
      >
        <div class="mx-2 shrink-0">•</div>
        <span class="mr-1 shrink-0">{{ unmatchedField }}:</span>
        <div
          class="flex flex-row justify-start items-center flex-wrap break-all"
        >
          <span>{{ unmatchedFieldMap.get(unmatchedField)?.join(", ") }}</span>
        </div>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { SchemaTemplateSetting_FieldTemplate } from "@/types/proto/v1/setting_service";

const props = defineProps<{
  fieldTemplates: SchemaTemplateSetting_FieldTemplate[];
}>();

const emits = defineEmits<{
  (event: "close"): void;
}>();

const unmatchedFieldMap = computed(() => {
  const fieldMap = new Map<string, string[]>();
  for (const fieldTemplate of props.fieldTemplates) {
    const field = fieldMap.get(fieldTemplate.column?.type || "");
    if (!field) {
      fieldMap.set(fieldTemplate.column?.type || "", [
        fieldTemplate.column?.name || "",
      ]);
    } else {
      field.push(fieldTemplate.column?.name || "");
    }
  }
  return fieldMap;
});
</script>
