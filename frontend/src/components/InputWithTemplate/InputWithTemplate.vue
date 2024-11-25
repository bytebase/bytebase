<template>
  <div class="border border-gray-300 rounded">
    <div class="flex flex-wrap gap-2 p-3 bg-gray-50 rounded">
      <div v-for="template in templateList" :key="template.id">
        <NTooltip :disabled="!template.description">
          <template #trigger>
            <div
              class="px-4 py-1 rounded text-sm font-sm font-normal border border-gray-300 bg-gray-100 cursor-pointer hover:bg-gray-200"
              @click="() => onTemplateAdd(template)"
            >
              {{ template.id }}
            </div>
          </template>
          <span v-if="template.description" class="whitespace-nowrap">
            {{ $t(template.description) }}
          </span>
        </NTooltip>
      </div>
    </div>
    <div class="p-2 border-t border-gray-300">
      <div ref="containerRef" class="flex flex-wrap items-center gap-1">
        <div
          v-for="(data, i) in state.templateInputs"
          :key="i"
          :ref="(el: any) => (itemRefs[i] = el)"
        >
          <BBBadge
            v-if="data.type == 'template'"
            :text="data.value"
            :can-remove="!disabled"
            @remove="() => onTemplateRemove(i)"
          />
          <AutoWidthInput
            v-else
            :value="data.value"
            :max-width="state.inputMaxWidth"
            :disabled="disabled"
            @keyup="(e) => onKeyup(i, e)"
            @keydown="onKeydown(i, itemRefs[i].querySelector('input'))"
            @mouseup="onKeydown(i, itemRefs[i].querySelector('input'))"
            @change="(val) => onTemplateChange(i, val)"
          />
        </div>
        <input
          ref="inputRef"
          v-model="state.inputData"
          class="flex-1 px-0 m-0 py-1 cleared-input outline-none"
          type="text"
          :disabled="disabled"
          @keydown.delete="onInputDataDeleteEnter"
          @keyup.delete="onInputDataDeleteLeave"
          @keydown="onKeydown(state.templateInputs.length, inputRef)"
          @mouseup="onKeydown(state.templateInputs.length, inputRef)"
          @keyup.left="(e) => onKeyup(state.templateInputs.length, e)"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NTooltip } from "naive-ui";
import type { PropType } from "vue";
import { reactive, watch, watchEffect, ref, onUnmounted, onMounted } from "vue";
import { BBBadge } from "@/bbkit";
import AutoWidthInput from "./AutoWidthInput.vue";
import type { Template, TemplateInput } from "./types";
import { InputType } from "./types";
import { getTemplateInputs, templateInputsToString, KEY_EVENT } from "./utils";

interface LocalState {
  inputData: string;
  inputMaxWidth: number;
  templateInputs: TemplateInput[];
  inputCursorPosition: Map<number, number>;
}

const props = defineProps({
  value: {
    default: "",
    type: String,
  },
  templateList: {
    require: true,
    default: () => [],
    type: Array as PropType<Template[]>,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(["change"]);

const templateInputs = getTemplateInputs(props.value, props.templateList);
let inputData = "";

if (
  templateInputs.length > 0 &&
  templateInputs[templateInputs.length - 1].type === InputType.String
) {
  inputData = templateInputs.pop()?.value ?? inputData;
}

const state = reactive<LocalState>({
  inputData,
  inputMaxWidth: 0,
  templateInputs,
  inputCursorPosition: new Map<number, number>(),
});

watch(
  () => state.templateInputs,
  (val) => {
    emit("change", `${templateInputsToString(val)}${state.inputData}`);
  },
  { deep: true }
);

watch(
  () => state.inputData,
  (val) => {
    emit("change", `${templateInputsToString(state.templateInputs)}${val}`);
  }
);

const itemRefs = ref<HTMLElement[]>([]);
const containerRef = ref<HTMLDivElement>();
const inputRef = ref<HTMLInputElement>();

watchEffect(() => {
  if (containerRef.value) {
    state.inputMaxWidth = containerRef.value.offsetWidth;
  }
});

const onWindowResize = () => {
  if (containerRef.value) {
    state.inputMaxWidth = containerRef.value.offsetWidth;
  }
};

const onInputDataDeleteEnter = () => {
  if (!state.inputData && state.templateInputs.length > 0) {
    const last = state.templateInputs.slice(-1)[0];
    if (last.type === InputType.Template) {
      state.templateInputs.pop();
    }
  }
};

const onInputDataDeleteLeave = () => {
  if (!state.inputData && state.templateInputs.length > 0) {
    const last = state.templateInputs.slice(-1)[0];
    if (last && last.type === InputType.String) {
      state.inputData = state.templateInputs.pop()?.value ?? state.inputData;
    }
  }
};

// Store the cursor position in key down event.
const onKeydown = (i: number, input: HTMLInputElement | null | undefined) => {
  const selectionEnd = input?.selectionEnd;
  if (!Number.isNaN(selectionEnd)) {
    state.inputCursorPosition.set(i, selectionEnd!);
  }
};

const onKeyup = (i: number, e: KeyboardEvent) => {
  switch (e.key) {
    case KEY_EVENT.BACKSPACE:
    case KEY_EVENT.DELETE:
      if (state.templateInputs[i].value === "") {
        if (i === 0 || state.inputCursorPosition.get(i) === 0) {
          // remove the empty data
          onTemplateRemove(i);
          if (
            i - 1 >= 0 &&
            state.templateInputs[i - 1].type === InputType.Template
          ) {
            // remove the previous data (should be the template type)
            onTemplateRemove(i - 1);
          }
          // try to focus the input from i-1 index
          focusPreInput(i - 1);
        }
      }
      break;
    case KEY_EVENT.LEFT: {
      const left = state.inputCursorPosition.get(i);
      // if the cursor is at the first position, we try to focus on the previous input.
      if (left === 0) {
        focusPreInput(i - 1);
        state.inputCursorPosition.delete(i);
      }
      break;
    }
    case KEY_EVENT.RIGHT: {
      const right = state.inputCursorPosition.get(i);
      // if the cursor is at the last position, we try to focus on the next input.
      if (right === state.templateInputs[i].value.length) {
        focusNextInput(i + 1);
        state.inputCursorPosition.delete(i);
      }
      break;
    }
  }
};

const focusNextInput = (i: number) => {
  let j = i;
  while (j <= state.templateInputs.length - 1) {
    const input = state.templateInputs[j];
    if (input && input.type === InputType.String) {
      // make sure the next input will be focused on the first position.
      itemRefs.value[j].querySelector("input")?.setSelectionRange(0, 0);
      itemRefs.value[j].querySelector("input")?.focus();
      break;
    }
    j++;
  }
  if (j === state.templateInputs.length) {
    inputRef.value?.setSelectionRange(0, 0);
    inputRef.value?.focus();
  }
};

const focusPreInput = (i: number) => {
  let j = i;
  while (j >= 0) {
    const input = state.templateInputs[j];
    if (input && input.type === InputType.String) {
      // make sure the next input will be focused on the last position.
      itemRefs.value[j]
        .querySelector("input")
        ?.setSelectionRange(input.value.length, input.value.length);
      itemRefs.value[j].querySelector("input")?.focus();
      break;
    }
    j--;
  }
};

const onTemplateChange = (i: number, data: string) => {
  const target = state.templateInputs[i];
  if (!target) {
    return;
  }

  state.templateInputs = [
    ...state.templateInputs.slice(0, i),
    {
      value: data,
      type: target.type,
    },
    ...state.templateInputs.slice(i + 1),
  ];
};

const onTemplateAdd = (template: Template) => {
  // clear position cache.
  state.inputCursorPosition = new Map<number, number>();

  if (state.inputData) {
    // If the last input contains user's input, we also need to add it
    state.templateInputs.push({
      value: state.inputData,
      type: InputType.String,
    });
  }

  state.templateInputs.push({
    value: template.id,
    type: InputType.Template,
  });

  state.inputData = "";
  inputRef.value?.focus();
};

const onTemplateRemove = (i: number) => {
  if (i < 0 || i >= state.templateInputs.length) {
    return;
  }

  // clear position cache.
  state.inputCursorPosition = new Map<number, number>();

  state.templateInputs = [
    ...state.templateInputs.slice(0, i),
    ...state.templateInputs.slice(i + 1),
  ];

  if (state.templateInputs.length === 0) {
    return;
  }

  const index = i - 1;
  if (index < 0 || index >= state.templateInputs.length) {
    return;
  }

  const template = state.templateInputs[index];
  if (template.type !== InputType.String) {
    return;
  }

  if (i === state.templateInputs.length) {
    // If the last value is string, we need to extract it into the last input.
    state.inputData = `${state.templateInputs.pop()?.value ?? ""}${
      state.inputData
    }`;
  } else if (state.templateInputs[i].type === InputType.String) {
    // Join the adjacent string value
    state.templateInputs = [
      ...state.templateInputs.slice(0, index),
      {
        value: `${template.value}${state.templateInputs[i].value}`,
        type: InputType.String,
      },
      ...state.templateInputs.slice(i + 1),
    ];
  }
};

onMounted(() => {
  window.addEventListener("resize", onWindowResize);
});

onUnmounted(() => {
  window.removeEventListener("resize", onWindowResize);
});
</script>

<style scoped>
.cleared-input,
.cleared-input:focus {
  @apply shadow-none ring-0 border-0 border-none;
}
</style>
