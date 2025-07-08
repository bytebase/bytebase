<template>
  <NInput
    class="w-full"
    :value="state.title"
    :style="style"
    :loading="state.isUpdating"
    :disabled="state.isUpdating"
    :readonly="release.state !== State.ACTIVE"
    autosize
    required
    @focus="state.isEditing = true"
    @blur="onBlur"
    @keyup.enter="onEnter"
    @update:value="onUpdateValue"
  />
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NInput } from "naive-ui";
import type { CSSProperties } from "vue";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useReleaseStore } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { ReleaseSchema } from "@/types/proto-es/v1/release_service_pb";
import { useReleaseDetailContext } from "../context";

const { t } = useI18n();
const { release } = useReleaseDetailContext();

const state = reactive({
  isEditing: false,
  isUpdating: false,
  title: release.value.title,
});

const style = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    minWidth: "12rem",
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

  if (state.title === release.value.title) {
    cleanup();
    return;
  }
  try {
    state.isUpdating = true;
    const patch = create(ReleaseSchema, {
      ...release.value,
      title: state.title,
    });
    useReleaseStore().updateRelase(patch, ["title"]);
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
  () => release.value.title,
  (title) => {
    state.title = title;
  }
);
</script>
