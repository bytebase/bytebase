<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="$t('settings.sensitive-data.semantic-types.add-from-template')"
      class="max-w-[100vw]"
    >
      <div class="w-2xl flex flex-col gap-y-4">
        <p class="textinfolabel">
          {{
            $t("settings.sensitive-data.semantic-types.template.description")
          }}
        </p>
        <SemanticTypesTable
          :readonly="true"
          :row-clickable="true"
          :semantic-item-list="semanticTemplateList"
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

<script setup lang="ts">
import { NButton } from "naive-ui";
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { getSemanticTemplateList } from "@/types";
import type { SemanticTypeSetting_SemanticType } from "@/types/proto-es/v1/setting_service_pb";
import type { SemanticItem } from "./SemanticTypesTable.vue";
import SemanticTypesTable from "./SemanticTypesTable.vue";

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
