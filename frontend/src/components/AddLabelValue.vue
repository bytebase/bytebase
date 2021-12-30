<template>
  <div class="add-label-value">
    <template v-if="!state.isAdding">
      <NPopover trigger="hover" :disabled="!reserved">
        <template #trigger>
          <div
            class="icon-btn"
            :class="{ disabled: reserved }"
            @click="toggleAdding"
          >
            <heroicons-solid:plus class="w-4 h-4" />
          </div>
        </template>

        <span class="text-red-600">
          {{
            $t("label.error.cannot-edit-reserved-label", {
              key: label.key,
            })
          }}
        </span>
      </NPopover>
    </template>
    <template v-else>
      <input
        ref="input"
        v-model="state.text"
        type="text"
        autocomplete="off"
        class="textfield"
        :class="{ error: !!state.error && state.changed }"
        :placeholder="$t('setting.label.value-placeholder')"
        @blur="cancel"
        @keyup.esc="cancel"
        @keyup.enter="tryAdd"
      />
      <div class="icon-btn cancel" @click="cancel">
        <heroicons-solid:x class="w-4 h-4" />
      </div>
      <NPopover trigger="hover" :disabled="!state.error">
        <template #trigger>
          <div
            class="icon-btn save"
            :class="{ disabled: !!state.error }"
            @mousedown.prevent.stop="tryAdd"
          >
            <heroicons-solid:check class="w-4 h-4" />
          </div>
        </template>

        <div class="text-red-600 whitespace-nowrap">
          {{ state.error }}
        </div>
      </NPopover>
    </template>
  </div>
</template>

<script lang="ts">
import {
  defineComponent,
  PropType,
  ref,
  nextTick,
  watch,
  reactive,
  computed,
} from "vue";
import { useI18n } from "vue-i18n";
import { Label } from "../types";
import { NPopover } from "naive-ui";
import { isReservedLabel } from "../utils";

type LocalState = {
  isAdding: boolean;
  text: string;
  error: string | undefined;
  changed: boolean;
};

const MAX_VALUE_LENGTH = 63;

export default defineComponent({
  name: "AddLabelValue",
  components: { NPopover },
  props: {
    label: {
      type: Object as PropType<Label>,
      required: true,
    },
  },
  emits: ["add"],
  setup(props, { emit }) {
    const { t } = useI18n();
    const input = ref<HTMLInputElement>();
    const state = reactive<LocalState>({
      isAdding: false,
      text: "",
      error: undefined,
      changed: false,
    });

    const reserved = computed(() => isReservedLabel(props.label));

    const toggleAdding = () => {
      if (isReservedLabel(props.label)) return;
      state.isAdding = !state.isAdding;
    };

    const validate = () => {
      const v = state.text.trim();
      if (!v) {
        // can't be empty
        state.error = t("label.error.value-necessary");
      } else if (props.label.valueList.includes(v)) {
        // must be unique
        state.error = t("label.error.value-duplicated");
      } else if (v.length > MAX_VALUE_LENGTH) {
        // max length exceeded
        state.error = t("label.error.max-length-exceeded", {
          len: MAX_VALUE_LENGTH,
        });
      } else {
        // ok
        state.error = undefined;
      }
      return !state.error;
    };

    watch(
      () => state.text,
      () => {
        state.changed = true;
        validate();
      }
    );

    const cancel = () => {
      state.isAdding = false;
      state.text = "";
    };

    const tryAdd = () => {
      if (state.error) {
        return;
      }
      const v = state.text.trim();
      emit("add", v);
      cancel();
    };

    watch(
      () => state.isAdding,
      (isAdding) => {
        if (isAdding) {
          // reset input state
          validate();
          state.changed = false;
          // auto focus if possible
          nextTick(() => input.value?.focus());
        }
      }
    );

    return { state, reserved, toggleAdding, tryAdd, cancel, input };
  },
});
</script>

<style scoped lang="postcss">
.add-label-value {
  @apply inline-flex flex-nowrap items-center gap-1 h-6;
}
.add-label-value > * {
  @apply h-full;
}
.icon-btn {
  @apply px-1 py-1 inline-flex items-center
    rounded bg-white border border-control-border
    hover:bg-control-bg-hover
    cursor-pointer;
}
.icon-btn.disabled {
  @apply cursor-not-allowed bg-control-bg;
}
.textfield {
  @apply rounded px-2 py-0 text-sm w-32;
}
.textfield.error {
  @apply border-error focus:ring-error focus:border-error;
}
.cancel {
  @apply text-error;
}
.save {
  @apply text-success;
}
</style>
