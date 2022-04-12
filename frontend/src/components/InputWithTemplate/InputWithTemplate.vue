<template>
  <div class="border border-gray-300 rounded">
    <div class="flex flex-wrap gap-4 p-3 bg-gray-50 rounded">
      <div
        v-for="template in templates"
        :key="template.id"
        class="px-4 py-1 rounded text-sm font-sm font-normal border border-gray-300 bg-gray-100 cursor-pointer tooltip-wrapper"
        @click="() => onTemplateAdd(template)"
      >
        {{ template.id }}
        <span class="tooltip whitespace-nowrap">{{
          template.description
        }}</span>
      </div>
    </div>
    <div class="p-2 border-t border-gray-300">
      <div ref="containerRef" class="flex flex-wrap items-center gap-1">
        <div v-for="(data, i) in state.templateInputs" :key="i">
          <BBBadge
            v-if="data.type === 'template'"
            :text="data.value"
            @remove="() => onTemplateRemove(i)"
          />
          <AutoWidthInput
            v-else
            :value="data.value"
            :max-width="state.inputMaxWidth"
            class-name="px-0 m-0 py-1 cleared-input outline-none"
            @keyup="(e) => onKeyup(i, e)"
            @change="(val) => onTemplateChange(i, val)"
          />
        </div>
        <input
          ref="inputRef"
          v-model="state.inputData"
          class="flex-1 px-0 m-0 py-1 cleared-input outline-none"
          type="text"
          @keydown.delete="onInputDataDeleteEnter"
          @keyup.delete="onInputDataDeleteLeave"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, watch, watchEffect, ref, PropType } from "vue";
import { Template } from "./types";

interface TemplateInput {
  value: string;
  type: "string" | "template";
}

interface LocalState {
  inputData: string;
  inputMaxWidth: number;
  templateInputs: TemplateInput[];
}

// getTemplateInputs will convert the string value like "abc{{template}}"
// into TemplateInput array: [{value: "abc", type: "string"}, {value: "template", type: "template"}]
const getTemplateInputs = (
  value: string,
  templates: Template[]
): TemplateInput[] => {
  let start = 0;
  let end = 0;
  const res: TemplateInput[] = [];
  const templateSet = new Set<string>(templates.map((t) => t.id));

  while (end <= value.length - 1) {
    if (
      value.slice(end, end + 2) === "}}" &&
      value.slice(start, start + 2) === "{{"
    ) {
      // When the end pointer meet the "}}" and the start pointer is "{{"
      // we can extract the string slice as template or normal string.
      const str = value.slice(start + 2, end);
      if (templateSet.has(str)) {
        res.push({
          value: str,
          type: "template",
        });
      } else {
        res.push({
          value: `{{${str}}}`,
          type: "string",
        });
      }
      end += 2;
      start = end;
    } else if (value.slice(end, end + 2) === "{{") {
      // When the end pointer meet the "{{"
      // we should reset the position of the start pointer.
      res.push({
        value: value.slice(start, end),
        type: "string",
      });
      start = end;
      end += 2;
    } else {
      end += 1;
    }
  }

  if (start < end) {
    res.push({
      value: value.slice(start, end),
      type: "string",
    });
  }

  // Join the adjacent string value
  return res.reduce((result, data) => {
    if (data.type === "template") {
      return [...result, data];
    }

    let str = data.value;

    if (result.length > 0 && result[result.length - 1].type === "string") {
      const last = result.pop();
      str = `${last ? last.value : ""}${str}`;
    }

    return [
      ...result,
      {
        value: str,
        type: "string",
      },
    ];
  }, [] as TemplateInput[]);
};

// templateInputsToString will convert TemplateInput array into string
const templateInputsToString = (inputs: TemplateInput[]): string => {
  return inputs
    .map((input) =>
      input.type === "string" ? input.value : `{{${input.value}}}`
    )
    .join("");
};

const props = defineProps({
  value: {
    default: "",
    type: String,
  },
  templates: {
    require: true,
    default: () => [],
    type: Array as PropType<Template[]>,
  },
});

const emit = defineEmits(["change"]);

const templateInputs = getTemplateInputs(props.value, props.templates);
let inputData = "";

if (
  templateInputs.length > 0 &&
  templateInputs[templateInputs.length - 1].type === "string"
) {
  const last = templateInputs.pop();
  if (last) {
    inputData = last.value;
  }
}

const state = reactive<LocalState>({
  inputData,
  inputMaxWidth: 0,
  templateInputs,
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

const containerRef = ref<HTMLDivElement>();
const inputRef = ref<HTMLInputElement>();

watchEffect(() => {
  if (containerRef.value) {
    state.inputMaxWidth = containerRef.value.offsetWidth;
  }
});

const onWindowResize = () => {
  if (containerRef && containerRef.value) {
    state.inputMaxWidth = containerRef.value.offsetWidth;
  }
};

const onInputDataDeleteEnter = (e: KeyboardEvent) => {
  if (!state.inputData && state.templateInputs.length > 0) {
    const last = state.templateInputs.slice(-1)[0];
    if (last.type === "template") {
      state.templateInputs.pop();
    }
  }
};

const onInputDataDeleteLeave = (e: KeyboardEvent) => {
  if (!state.inputData && state.templateInputs.length > 0) {
    const last = state.templateInputs.slice(-1)[0];
    if (last && last.type === "string") {
      const target = state.templateInputs.pop();
      if (target) {
        state.inputData = target.value;
      }
    }
  }
};

const onKeyup = (i: number, e: KeyboardEvent) => {
  const data = state.templateInputs[i];
  if (!data) {
    return;
  }

  if (e.key !== "Delete" || data.value !== "") {
    return;
  }

  onTemplateRemove(i);
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
  if (state.inputData) {
    // If the last input contains user's input, we also need to add it
    state.templateInputs.push({
      value: state.inputData,
      type: "string",
    });
  }

  state.templateInputs.push({
    value: template.id,
    type: "template",
  });

  state.inputData = "";
  if (inputRef && inputRef.value) {
    inputRef.value.focus();
  }
};

const onTemplateRemove = (i: number) => {
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
  if (template.type !== "string") {
    return;
  }

  if (i === state.templateInputs.length) {
    // If the last value is string, we need to extract it into the last input.
    const last = state.templateInputs.pop();

    state.inputData = `${last ? last.value : ""}${state.inputData}`;
  } else if (state.templateInputs[i].type === "string") {
    // Join the adjacent string value
    state.templateInputs = [
      ...state.templateInputs.slice(0, index),
      {
        value: `${template.value}${state.templateInputs[i].value}`,
        type: "string",
      },
      ...state.templateInputs.slice(i + 1),
    ];
  }
};

window.addEventListener("resize", onWindowResize);
</script>

<style scoped>
.cleared-input,
.cleared-input:focus {
  @apply shadow-none ring-0 border-0 border-none;
}

.tooltip-wrapper {
  @apply relative;
}

.tooltip {
  @apply invisible absolute -mt-8 ml-2 px-2 py-1 rounded bg-black bg-opacity-75 text-white;
}

.tooltip-wrapper:hover .tooltip {
  @apply visible z-50;
}
</style>
