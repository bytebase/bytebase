<template>
  <NSwitch
    v-model:value="state.encryptEnabled"
    :disabled="!editable"
    size="small"
  />
  <template v-if="editable && state.encryptEnabled">
    <span class="textinfolabel pl-4 pr-2"
      >{{ $t("common.password") }}
      <RequiredStar />
    </span>
    <NInput
      v-model:value="state.password"
      class="!w-auto"
      size="small"
      type="password"
      :input-props="{ autocomplete: 'new-password' }"
      :placeholder="$t('common.password')"
    />
  </template>
</template>

<script setup lang="ts">
import { NSwitch, NInput } from "naive-ui";
import { reactive, watch } from "vue";
import RequiredStar from "@/components/RequiredStar.vue";

interface LocalState {
  encryptEnabled: boolean;
  password: string;
}

const props = defineProps<{
  password?: string;
  editable?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:password", value: string): void;
}>();

const state = reactive<LocalState>({
  encryptEnabled: Boolean(props.password),
  password: props.password || "",
});

watch(
  () => state,
  () => {
    if (!state.encryptEnabled) {
      emit("update:password", "");
    } else {
      emit("update:password", state.password);
    }
  },
  { deep: true }
);
</script>
