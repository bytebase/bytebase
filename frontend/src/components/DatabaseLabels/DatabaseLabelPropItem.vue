<template>
  <span class="textlabel relative inline-flex items-center gap-x-1">
    <template v-if="!state.editing">
      {{ value || $t("label.empty-label-value") }}

      <heroicons-outline:pencil
        v-if="editable"
        class="w-5 h-5 p-0.5 rounded cursor-pointer hover:bg-control-bg-hover"
        @click="startEditing"
      />
    </template>
    <template v-else-if="state.editing">
      <input
        ref="input"
        v-model="state.value"
        type="text"
        autocomplete="off"
        class="textfield"
        :class="{ error: !!state.error && state.changed }"
        :placeholder="$t('setting.label.value-placeholder')"
        @blur="cancel"
        @keyup.esc="cancel"
        @keyup.enter="save"
      />
      <div class="icon-btn cancel" @click="cancel">
        <heroicons-solid:x class="w-4 h-4" />
      </div>
      <NPopover trigger="hover" :disabled="!state.error">
        <template #trigger>
          <div
            class="icon-btn save"
            :class="{ disabled: !!state.error }"
            @mousedown.prevent.stop="save"
          >
            <heroicons-solid:check class="w-4 h-4" />
          </div>
        </template>

        <div class="text-red-600 whitespace-nowrap">
          {{ state.error }}
        </div>
      </NPopover>
    </template>
  </span>
</template>

<script lang="ts" setup>
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Database, LabelKeyType, LabelValueType } from "../../types";
import { LABEL_VALUE_EMPTY, MAX_LABEL_VALUE_LENGTH } from "../../utils";

type LocalState = {
  editing: boolean;
  value: LabelValueType | undefined;
  error: string | undefined;
  changed: boolean;
};

const props = defineProps<{
  labelKey: LabelKeyType;
  required: boolean;
  value: LabelValueType | undefined;
  database: Database;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (e: "update:value", value: LabelValueType): void;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  editing: false,
  value: props.value,
  error: undefined,
  changed: false,
});

const input = ref<HTMLInputElement>();

watch(
  () => props.value,
  (value) => (state.value = value)
);

const editable = computed((): boolean => {
  if (!props.allowEdit) return false;

  return !props.required;
});

const startEditing = () => {
  state.value = props.value;
  state.error = undefined;
  state.changed = false;
  state.editing = true;

  nextTick(() => {
    input.value?.focus();
  });
};
const validate = (): boolean => {
  const value = state.value?.trim();

  if (!value) {
    // Empty value is allowed
    state.error = undefined;
    return true;
  }

  if (value.length > MAX_LABEL_VALUE_LENGTH) {
    // Max length exceeded
    state.error = t("label.error.max-length-exceeded", {
      len: MAX_LABEL_VALUE_LENGTH,
    });
  } else {
    state.error = undefined;
  }

  return !state.error;
};

const cancel = () => {
  state.editing = false;
};

const save = () => {
  if (!validate()) return;

  const value = state.value?.trim() ?? LABEL_VALUE_EMPTY;
  emit("update:value", value);
  cancel();
};

watch([() => state.editing, () => state.value], ([editing, value]) => {
  if (!editing) return;
  if (value !== props.value) {
    state.changed = true;
    validate();
  }
});
</script>

<style scoped lang="postcss">
.icon-btn {
  @apply w-[20px] h-[20px] inline-flex items-center justify-center
    rounded bg-white border border-control-border
    hover:bg-control-bg-hover
    cursor-pointer;
}
.icon-btn.disabled {
  @apply cursor-not-allowed bg-control-bg;
}
.textfield {
  @apply rounded px-2 py-0 text-sm w-32 h-[20px];
}
.textfield.error {
  @apply border-error focus:ring-error focus:border-error;
}
.cancel {
  @apply text-control;
}
.save {
  @apply text-success;
}
</style>
