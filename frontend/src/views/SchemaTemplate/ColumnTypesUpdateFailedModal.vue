<template>
  <BBModal
    :title="
      $t('schema-template.column-type-restriction.messages.unable-to-update')
    "
    class="!w-[32rem] !max-w-full"
    @close="emits('close')"
  >
    <div class="w-full">
      <p class="mt-1 mb-2">
        {{
          $t(
            "schema-template.column-type-restriction.messages.following-column-types-are-used"
          )
        }}
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
    <div class="mt-7 flex justify-end space-x-2">
      <button type="button" class="btn-normal" @click.prevent="$emit('close')">
        {{ $t("common.close") }}
      </button>
      <button
        type="button"
        class="btn-primary"
        @click="$emit('save-all', props.fieldTemplates)"
      >
        {{ $t("schema-template.column-type-restriction.save-all") }}
      </button>
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
  (
    event: "save-all",
    fieldTemplates: SchemaTemplateSetting_FieldTemplate[]
  ): void;
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
