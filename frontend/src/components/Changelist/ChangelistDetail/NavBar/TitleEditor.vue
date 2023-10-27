<template>
  <span v-if="!allowEdit" class="text-xl font-bold ml-2 py-1 leading-[34px]">
    {{ state.title }}
  </span>
  <NInput
    v-else
    :value="state.title"
    :style="style"
    :loading="state.isUpdating"
    :disabled="!allowEdit || state.isUpdating"
    autosize
    required
    @focus="state.isEditing = true"
    @blur="onBlur"
    @keyup.enter="onEnter"
    @update:value="onUpdateValue"
  />
</template>

<script setup lang="ts">
import { NInput } from "naive-ui";
import { CSSProperties, computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useChangelistStore } from "@/store";
import { Changelist } from "@/types/proto/v1/changelist_service";
import { useChangelistDetailContext } from "../context";

const { t } = useI18n();
const { changelist, allowEdit } = useChangelistDetailContext();

const state = reactive({
  isEditing: false,
  isUpdating: false,
  title: changelist.value.description,
});

const style = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    minWidth: "10rem",
    "--n-color-disabled": "transparent",
    "--n-font-size": "20px",
  };
  const border = state.isEditing
    ? "1px solid rgb(var(--color-control-border))"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

const onBlur = async () => {
  const cleanup = () => {
    state.isEditing = false;
    state.isUpdating = false;
  };

  if (state.title === changelist.value.description) {
    cleanup();
    return;
  }
  try {
    state.isUpdating = true;
    const patch = Changelist.fromPartial({
      ...changelist.value,
      description: state.title,
    });
    useChangelistStore().patchChangelist(patch, ["description"]);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    cleanup();
  }
};

const onEnter = (e: Event) => {
  const input = e.target as HTMLInputElement;
  input.blur();
};

const onUpdateValue = (title: string) => {
  state.title = title;
};

watch(
  () => changelist.value.description,
  (title) => {
    state.title = title;
  }
);
</script>
