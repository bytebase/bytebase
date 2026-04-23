<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent :title="$t('schema-template.classification.select')">
      <div class="w-[25rem] h-full">
        <ReactPageMount
          page="ClassificationTree"
          :config="classificationConfig"
          :onApply="onApply"
        />
      </div>

      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-3">
            <NButton @click.prevent="$emit('dismiss')">
              {{ $t("common.cancel") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { Drawer, DrawerContent } from "@/components/v2";
import ReactPageMount from "@/react/ReactPageMount.vue";
import type { DataClassificationSetting_DataClassificationConfig } from "@/types/proto-es/v1/setting_service_pb";

defineProps<{
  show: boolean;
  classificationConfig: DataClassificationSetting_DataClassificationConfig;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "apply", classificationId: string): void;
}>();

const onApply = (classificationId: string) => {
  emit("apply", classificationId);
  emit("dismiss");
};
</script>
