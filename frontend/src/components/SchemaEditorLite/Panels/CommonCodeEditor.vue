<template>
  <div class="h-full flex flex-col gap-1 overflow-y-hidden pt-1">
    <div class="flex justify-between items-center text-sm">
      <div class="flex justify-start items-center gap-2">
        <template
          v-if="!readonly && (status === 'normal' || status === 'updated')"
        >
          <NButton
            v-if="!state.unlocked"
            size="small"
            @click="state.unlocked = true"
          >
            <template #icon>
              <SquarePenIcon class="w-3.5 h-3.5" />
            </template>
            {{ $t("common.edit") }}
          </NButton>
          <template v-else>
            <NButton size="small" @click="cancelEdit">
              <template #icon>
                <XIcon class="w-4 h-4" />
              </template>
              {{ $t("common.cancel") }}
            </NButton>
            <NButton
              size="small"
              type="primary"
              :disabled="props.code === state.code"
              @click="confirmEdit"
            >
              <template #icon>
                <CheckIcon class="w-4 h-4" />
              </template>
              {{ $t("common.confirm") }}
            </NButton>
          </template>
        </template>
      </div>
      <slot name="header-suffix" />
    </div>
    <MonacoEditor
      :content="state.code"
      :readonly="!editable"
      :auto-complete-context="{
        instance: db.instance,
        database: db.name,
        scene: 'all',
      }"
      class="border w-full rounded flex-1 relative"
      @update:content="handleUpdateCode"
    />
  </div>
</template>

<script setup lang="ts">
import { XIcon, CheckIcon, SquarePenIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import type { ComposedDatabase } from "@/types";
import type { EditStatus } from "..";

type LocalState = {
  code: string;
  unlocked: boolean;
};

const props = defineProps<{
  db: ComposedDatabase;
  code: string;
  readonly: boolean;
  status: EditStatus;
}>();

const emit = defineEmits<{
  (e: "update:code", code: string): void;
}>();

const state = reactive<LocalState>({
  code: props.code,
  unlocked: false,
});

const editable = computed(() => {
  if (props.readonly) {
    return false;
  }
  if (props.status === "dropped") {
    return false;
  }
  if (props.status === "created") {
    return true;
  }
  return state.unlocked;
});

const cancelEdit = () => {
  state.unlocked = false;
  state.code = props.code;
};

const confirmEdit = () => {
  state.unlocked = false;
  emit("update:code", state.code);
};

const handleUpdateCode = (code: string) => {
  state.code = code;
  if (props.status === "created") {
    emit("update:code", code);
  }
};

watch(
  () => props.code,
  (code) => {
    state.code = code;
  }
);
</script>
