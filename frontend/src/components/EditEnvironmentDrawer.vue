<template>
  <Drawer :show="show" @close="$emit('dismiss')">
    <DrawerContent
      :title="t('database.edit-environment')"
      class="w-96 max-w-[100vw]"
    >
      <EnvironmentSelect
        v-model:value="environment"
        class="mt-1"
        required
        name="environment"
      />
      <template #footer>
        <div class="w-full flex justify-between items-center">
          <div class="w-full flex justify-end items-center gap-x-3">
            <NButton @click.prevent="$emit('dismiss')">
              {{ t("common.cancel") }}
            </NButton>
            <NButton :disabled="!allowSave" type="primary" @click="onSave">
              {{ t("common.save") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent, EnvironmentSelect } from "@/components/v2";

const { t } = useI18n();

defineProps<{
  show: boolean;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
  (event: "update", environment: string): void;
}>();

const environment = ref<string>();

const allowSave = computed(() => {
  return environment.value !== undefined;
});

const onSave = () => {
  if (environment.value) {
    emit("update", environment.value);
  }
  emit("dismiss");
};
</script>
