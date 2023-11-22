<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="$t('settings.sensitive-data.semantic-types.add-from-template')"
    >
      <div class="w-[52rem]">
        <SemanticTypesTable
          :readonly="true"
          :row-clickable="true"
          :semantic-item-list="semanticTemplateList"
          @select="onApply($event)"
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

<script setup lang="ts">
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { getSemanticTemplateList } from "@/types";
import { SemanticTypeSetting_SemanticType } from "@/types/proto/v1/setting_service";
import SemanticTypesTable, { SemanticItem } from "./SemanticTypesTable.vue";

defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "apply", item: SemanticTypeSetting_SemanticType): void;
}>();

const semanticTemplateList = computed((): SemanticItem[] =>
  getSemanticTemplateList().map((item) => ({
    dirty: false,
    item,
    mode: "NORMAL",
  }))
);

const onApply = (id: string) => {
  const data = semanticTemplateList.value.find((data) => data.item.id === id);
  if (data) {
    emit("apply", data.item);
  }
  emit("dismiss");
};
</script>
