<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent :title="$t('settings.sensitive-data.semantic-types.self')">
      <div class="flex flex-col gap-y-8 w-200 h-full">
        <SemanticTypesTable
          :readonly="true"
          :row-clickable="true"
          :size="'small'"
          :semantic-item-list="semanticItemList"
          @select="onApply($event)"
        />
      </div>
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-2">
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
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import type { SemanticTypeSetting_SemanticType } from "@/types/proto-es/v1/setting_service_pb";
import type { SemanticItem } from "./SemanticTypesTable.vue";
import SemanticTypesTable from "./SemanticTypesTable.vue";

const props = defineProps<{
  show: boolean;
  semanticTypeList: SemanticTypeSetting_SemanticType[];
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "apply", id: string): void;
}>();

const semanticItemList = computed((): SemanticItem[] => {
  return props.semanticTypeList.map((semanticType) => {
    return {
      dirty: false,
      item: semanticType,
      mode: "NORMAL",
    };
  });
});

const onApply = (semanticTypeId: string) => {
  emit("apply", semanticTypeId);
  emit("dismiss");
};
</script>
