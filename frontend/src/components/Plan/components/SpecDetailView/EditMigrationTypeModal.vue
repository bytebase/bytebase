<template>
  <NModal
    v-model:show="showModal"
    :title="$t('plan.change-type')"
    preset="dialog"
    :positive-text="$t('common.save')"
    :negative-text="$t('common.cancel')"
    @positive-click="handleSave"
    @negative-click="handleClose"
  >
    <NRadioGroup v-model:value="selectedType" class="space-y-4">
      <NRadio :value="MigrationType.DDL" class="w-full">
        <div class="flex items-start space-x-2 w-full ml-2">
          <FileDiffIcon class="w-6 h-6 flex-shrink-0" :stroke-width="1.5" />
          <div class="flex-1">
            <div class="flex items-center space-x-2">
              <span class="text-base text-gray-900">
                <span>{{ $t("plan.schema-migration") }}</span>
              </span>
            </div>
            <p class="text-sm text-gray-600 mt-1">
              {{ $t("plan.schema-migration-description") }}
            </p>
          </div>
        </div>
      </NRadio>
      <NRadio :value="MigrationType.DML" class="w-full">
        <div class="flex items-start space-x-2 w-full ml-2">
          <EditIcon class="w-6 h-6 flex-shrink-0" :stroke-width="1.5" />
          <div class="flex-1">
            <div class="flex items-center space-x-2">
              <span class="text-base text-gray-900">
                <span>{{ $t("plan.data-change") }}</span>
              </span>
            </div>
            <p class="text-sm text-gray-600 mt-1">
              {{ $t("plan.data-change-description") }}
            </p>
          </div>
        </div>
      </NRadio>
    </NRadioGroup>
  </NModal>
</template>

<script setup lang="ts">
import { FileDiffIcon, EditIcon } from "lucide-vue-next";
import { NModal, NRadioGroup, NRadio } from "naive-ui";
import { ref, watch } from "vue";
import { MigrationType } from "@/types/proto-es/v1/common_pb";

const props = defineProps<{
  show: boolean;
  migrationType: MigrationType;
}>();

const emit = defineEmits<{
  (event: "update:show", value: boolean): void;
  (event: "save", type: MigrationType): void;
}>();

const showModal = ref(props.show);
const selectedType = ref(props.migrationType);

watch(
  () => props.show,
  (newVal) => {
    showModal.value = newVal;
    if (newVal) {
      selectedType.value = props.migrationType;
    }
  }
);

watch(showModal, (newVal) => {
  emit("update:show", newVal);
});

const handleSave = () => {
  emit("save", selectedType.value);
};

const handleClose = () => {
  emit("update:show", false);
};
</script>
