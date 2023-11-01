<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent :title="$t('settings.sensitive-data.semantic-types.self')">
      <div class="divide-block-border space-y-8 w-[40rem] h-full">
        <SemanticTypesTable
          :readonly="true"
          :row-clickable="true"
          :semantic-item-list="semanticItemList"
          @on-select="onApply($event)"
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
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { SemanticTypeSetting_SemanticType } from "@/types/proto/v1/setting_service";
import SemanticTypesTable, { SemanticItem } from "./SemanticTypesTable.vue";

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
