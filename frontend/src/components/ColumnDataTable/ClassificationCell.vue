<template>
  <div class="flex items-center gap-x-1">
    <ClassificationLevelBadge
      :classification="classification"
      :classification-config="classificationConfig"
    />
    <template v-if="!readonly && !disabled">
      <NPopconfirm v-if="classification" @positive-click="removeClassification">
        <template #trigger>
          <MiniActionButton @click.stop="">
            <XIcon class="w-3 h-3" />
          </MiniActionButton>
        </template>
        <template #default>
          <div>
            {{ $t("settings.sensitive-data.remove-classification-tips") }}
          </div>
        </template>
      </NPopconfirm>
      <MiniActionButton v-if="classificationConfig" @click.prevent="openDrawer">
        <PencilIcon class="w-3 h-3" />
      </MiniActionButton>
    </template>
  </div>

  <SelectClassificationDrawer
    v-if="classificationConfig"
    :show="showClassificationDrawer"
    :classification-config="classificationConfig"
    @dismiss="showClassificationDrawer = false"
    @apply="$emit('apply', $event)"
  />
</template>

<script lang="ts" setup>
import { PencilIcon, XIcon } from "lucide-vue-next";
import { NPopconfirm } from "naive-ui";
import { ref } from "vue";
import ClassificationLevelBadge from "@/components/SensitiveData/components/ClassificationLevelBadge.vue";
import SelectClassificationDrawer from "@/components/SensitiveData/components/SelectClassificationDrawer.vue";
import { MiniActionButton } from "@/components/v2";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { DataClassificationSetting_DataClassificationConfig as DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";

defineProps<{
  classification?: string | undefined;
  readonly?: boolean;
  disabled?: boolean;
  classificationConfig?: DataClassificationConfig;
  engine: Engine;
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
