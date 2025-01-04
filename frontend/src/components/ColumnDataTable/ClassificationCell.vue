<template>
  <div class="flex items-center gap-x-1">
    <ClassificationLevelBadge
      :classification="classification"
      :classification-config="classificationConfig"
    />
    <template v-if="!readonly && !disabled">
      <MiniActionButton
        v-if="classification"
        @click.prevent="removeClassification"
      >
        <XIcon class="w-3 h-3" />
      </MiniActionButton>
      <MiniActionButton @click.prevent="openDrawer">
        <PencilIcon class="w-3 h-3" />
      </MiniActionButton>
    </template>
  </div>

  <SelectClassificationDrawer
    :show="showClassificationDrawer"
    :classification-config="classificationConfig"
    @dismiss="showClassificationDrawer = false"
    @apply="$emit('apply', $event)"
  />
</template>

<script lang="ts" setup>
import { PencilIcon, XIcon } from "lucide-vue-next";
import { ref } from "vue";
import ClassificationLevelBadge from "@/components/SchemaTemplate/ClassificationLevelBadge.vue";
import { MiniActionButton } from "@/components/v2";
import type { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto/v1/setting_service";
import SelectClassificationDrawer from "../SchemaTemplate/SelectClassificationDrawer.vue";

defineProps<{
  classification?: string | undefined;
  readonly?: boolean;
  disabled?: boolean;
  classificationConfig: DataClassificationConfig;
}>();

const emit = defineEmits<{
  (event: "apply", id: string): void;
}>();

const showClassificationDrawer = ref(false);

const openDrawer = (e: MouseEvent) => {
  e.stopPropagation();
  showClassificationDrawer.value = true;
};

const removeClassification = (e: MouseEvent) => {
  e.stopPropagation();
  emit("apply", "");
};
</script>
