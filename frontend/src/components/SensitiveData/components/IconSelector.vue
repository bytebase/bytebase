<template>
  <div class="w-full">
    <div v-if="!isEditing" class="flex items-center justify-center gap-1">
      <template v-if="value">
        <!-- Display uploaded image -->
        <img :src="value" class="w-6 h-6 object-contain" alt="" />
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
      <div class="p-2 flex flex-col gap-y-4">
        <div class="w-48 h-48 flex justify-center items-center border border-gray-300 border-dashed rounded-md relative">
          <div
            class="w-1/3 h-1/3 bg-no-repeat bg-contain bg-center rounded-md pointer-events-none"
            :style="`background-image: url(${tempValue});`"
          ></div>

          <SingleFileSelector
            class="flex flex-col gap-y-1 text-center justify-center items-center absolute top-0 bottom-0 left-0 right-0"
            :class="[tempValue ? 'opacity-0 hover:opacity-90' : '']"
            :max-file-size-in-mi-b="2"
            :support-file-extensions="supportImageExtensions"
            :show-no-data-placeholder="!tempValue"
            @on-select="onFileSelect"
          />
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
import { NButton, NPopover } from "naive-ui";
import { ref, watch } from "vue";
import SingleFileSelector from "@/components/SingleFileSelector.vue";
import { MiniActionButton } from "@/components/v2";

const props = defineProps<{
  value?: string;
  readonly?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:value", value: string): void;
}>();

const isEditing = ref(false);
const tempValue = ref("");

watch(
  () => isEditing.value,
  (editing) => {
    if (editing) {
      tempValue.value = props.value ?? "";
    }
  }
);

const supportImageExtensions = [".jpg", ".jpeg", ".png", ".webp", ".svg"];

const startEditing = () => {
  if (props.readonly) return;
  isEditing.value = true;
};

const confirmEdit = () => {
  emit("update:value", tempValue.value);
  isEditing.value = false;
};

const cancelEdit = () => {
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
