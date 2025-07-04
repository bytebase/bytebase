<template>
  <div class="w-full">
    <div v-if="!isEditing" class="flex items-center justify-center gap-1">
      <template v-if="modelValue">
        <!-- Display uploaded image -->
        <img :src="modelValue" class="w-6 h-6 object-contain" alt="" />
        <MiniActionButton v-if="!readonly" @click="startEditing">
          <PencilIcon class="w-4 h-4" />
        </MiniActionButton>
      </template>
      <MiniActionButton v-else-if="!readonly" @click="startEditing">
        <PencilIcon class="w-4 h-4" />
      </MiniActionButton>
      <span v-else class="text-gray-400">-</span>
    </div>

    <!-- Edit mode -->
    <NPopover
      v-else
      :show="true"
      trigger="manual"
      placement="bottom"
      @clickoutside="cancelEdit"
    >
      <template #trigger>
        <div class="w-full" />
      </template>
      <div class="p-2 space-y-2">
        <SingleFileSelector
          class="w-48 h-32"
          :max-file-size-in-mi-b="2"
          :support-file-extensions="supportImageExtensions"
          @on-select="onFileSelect"
        >
          <template #default>
            <div
              class="w-full h-full flex flex-col items-center justify-center border-2 border-dashed border-gray-300 rounded-md hover:border-gray-400 transition-colors"
            >
              <heroicons-outline:upload class="w-8 h-8 text-gray-400" />
              <span class="text-xs text-gray-500 mt-1">
                {{ $t("settings.general.workspace.drag-logo") }}
              </span>
              <span class="text-xs text-gray-400">
                {{ supportedFormatsText }}
              </span>
            </div>
          </template>
        </SingleFileSelector>
        <div v-if="tempValue" class="mt-2">
          <img :src="tempValue" class="w-8 h-8 object-contain mx-auto" alt="" />
        </div>
        <div class="flex justify-end gap-2">
          <NButton size="tiny" @click="clearIcon">
            {{ $t("common.clear") }}
          </NButton>
          <NButton size="tiny" @click="cancelEdit">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton size="tiny" type="primary" @click="confirmEdit">
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </div>
    </NPopover>
  </div>
</template>

<script setup lang="ts">
import { PencilIcon } from "lucide-vue-next";
import { NPopover, NButton } from "naive-ui";
import { ref, computed } from "vue";
import SingleFileSelector from "@/components/SingleFileSelector.vue";
import { MiniActionButton } from "@/components/v2";

const props = defineProps<{
  modelValue?: string;
  readonly?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

const isEditing = ref(false);
const tempValue = ref("");

const supportImageExtensions = [".jpg", ".jpeg", ".png", ".webp", ".svg"];

const supportedFormatsText = computed(() => {
  return supportImageExtensions.join(", ") + " (max 2MB)";
});

const startEditing = () => {
  if (props.readonly) return;
  isEditing.value = true;
  tempValue.value = props.modelValue || "";
};

const confirmEdit = () => {
  emit("update:modelValue", tempValue.value);
  isEditing.value = false;
};

const cancelEdit = () => {
  tempValue.value = "";
  isEditing.value = false;
};

const clearIcon = () => {
  tempValue.value = "";
};

const onFileSelect = async (file: File) => {
  const fileInBase64 = await convertFileToBase64(file);
  tempValue.value = fileInBase64;
};

const convertFileToBase64 = (file: File) =>
  new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.readAsDataURL(file);
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = (error) => reject(error);
  });
</script>
