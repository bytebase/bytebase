<template>
  <!-- TODO: When all engines migrate to simplified input, replace NInput with InlineInput component -->
  <NInput
    v-if="useSimpleInput"
    ref="inputRef"
    :value="simpleInputValue"
    :placeholder="simpleInputPlaceholder"
    :disabled="disabled"
    :style="simpleInputStyle"
    @focus="focused = true"
    @blur="focused = false"
    @update:value="handleSimpleInput"
  />

  <NDropdown
    v-else
    trigger="click"
    placement="bottom-start"
    :value="dropdownValue"
    :options="options"
    :consistent-menu-width="false"
    :disabled="disabled"
    class="bb-schema-editor--column-default-value-select"
    style="max-height: 20rem; overflow-y: auto; overflow-x: hidden"
    @select="handleSelect"
  >
    <NInput
      ref="inputRef"
      :value="inputValue"
      :placeholder="placeholder"
      :disabled="inputDisabled"
      :style="inputStyle"
      @focus="focused = true"
      @blur="focused = false"
      @update:value="handleInput"
    >
      <template #suffix>
        <!-- use the same icon and style with NSelect -->
        <NElement
          v-if="!disabled"
          tag="button"
          class="absolute top-1/2 right-[3px] -translate-y-1/2"
          :class="[
            disabled
              ? 'text-[var(--placeholder-color-disabled)] cursor-not-allowed'
              : 'text-[var(--placeholder-color)] hover:text-[var(--primary-color-hover)] cursor-pointer',
          ]"
        >
          <svg
            viewBox="0 0 16 16"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            class="w-4 h-4"
          >
            <path
              d="M3.14645 5.64645C3.34171 5.45118 3.65829 5.45118 3.85355 5.64645L8 9.79289L12.1464 5.64645C12.3417 5.45118 12.6583 5.45118 12.8536 5.64645C13.0488 5.84171 13.0488 6.15829 12.8536 6.35355L8.35355 10.8536C8.15829 11.0488 7.84171 11.0488 7.64645 10.8536L3.14645 6.35355C2.95118 6.15829 2.95118 5.84171 3.14645 5.64645Z"
              fill="currentColor"
            ></path>
          </svg>
        </NElement>
      </template>
    </NInput>
  </NDropdown>
</template>

<script lang="ts" setup>
import {
  type InputInst,
  type SelectOption,
  NDropdown,
  NElement,
  NInput,
} from "naive-ui";
import type { CSSProperties } from "vue";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { DefaultValueOption } from "@/components/SchemaEditorLite/utils";
import {
  DEFAULT_EXPRESSION_OPTION,
  DEFAULT_NULL_OPTION,
  DEFAULT_STRING_OPTION,
  NO_DEFAULT_OPTION,
  getColumnDefaultDisplayString,
  getColumnDefaultValuePlaceholder,
} from "@/components/SchemaEditorLite/utils";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";

type DefaultValueSelectOption = SelectOption & {
  value: string;
  defaults: DefaultValueOption;
};

const props = defineProps<{
  column: ColumnMetadata;
  engine: Engine;
  disabled?: boolean;
}>();
const emit = defineEmits<{
  (event: "input", value: string): void;
  (event: "select", option: DefaultValueOption): void;
}>();

const { t } = useI18n();
const focused = ref(false);
const inputRef = ref<InputInst>();

const dropdownValue = computed(() => {
  const { hasDefault, default: defaultString } = props.column;
  if (!hasDefault) return "no-default";
  if (defaultString === "NULL") return "null";
  if (typeof defaultString === "string") return "string";
  return null;
});

const inputValue = computed(() => {
  return getColumnDefaultDisplayString(props.column) ?? null;
});

const placeholder = computed(() => {
  return getColumnDefaultValuePlaceholder(props.column);
});

// Computed property for engines that use simple input (all migrated engines)
const useSimpleInput = computed(() => {
  return (
    props.engine === Engine.POSTGRES ||
    props.engine === Engine.MYSQL ||
    props.engine === Engine.MSSQL ||
    props.engine === Engine.ORACLE ||
    props.engine === Engine.TIDB ||
    props.engine === Engine.MARIADB ||
    props.engine === Engine.OCEANBASE ||
    props.engine === Engine.SNOWFLAKE ||
    props.engine === Engine.CLICKHOUSE ||
    props.engine === Engine.COCKROACHDB ||
    props.engine === Engine.SPANNER ||
    props.engine === Engine.BIGQUERY ||
    props.engine === Engine.REDSHIFT ||
    props.engine === Engine.STARROCKS ||
    props.engine === Engine.DORIS
  );
});

const simpleInputValue = computed(() => {
  // For all migrated engines, we use defaultString field which contains the properly formatted expression
  if (
    props.engine === Engine.POSTGRES ||
    props.engine === Engine.MYSQL ||
    props.engine === Engine.MSSQL ||
    props.engine === Engine.ORACLE ||
    props.engine === Engine.TIDB ||
    props.engine === Engine.MARIADB ||
    props.engine === Engine.OCEANBASE ||
    props.engine === Engine.SNOWFLAKE ||
    props.engine === Engine.CLICKHOUSE ||
    props.engine === Engine.COCKROACHDB ||
    props.engine === Engine.SPANNER ||
    props.engine === Engine.BIGQUERY ||
    props.engine === Engine.REDSHIFT ||
    props.engine === Engine.STARROCKS ||
    props.engine === Engine.DORIS
  ) {
    return props.column.default || "";
  }
  return "";
});

const simpleInputPlaceholder = computed(() => {
  return t("schema-editor.default.placeholder");
});

// TODO: This styling mimics InlineInput component. When all engines use simplified input,
// replace NInput + simpleInputStyle with InlineInput component directly
const simpleInputStyle = computed(() => {
  const style: CSSProperties = {
    cursor: "default",
    "--n-color": "transparent",
    "--n-color-disabled": "transparent",
    "--n-padding-left": "6px",
    "--n-padding-right": "4px",
    "--n-text-color-disabled": "rgb(var(--color-main))",
  };
  const border = focused.value
    ? "1px solid rgb(var(--color-control-border))"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

const options = computed((): DefaultValueSelectOption[] => {
  return [
    {
      key: NO_DEFAULT_OPTION.key,
      value: NO_DEFAULT_OPTION.key,
      label: t("schema-editor.default.no-default"),
      defaults: NO_DEFAULT_OPTION,
    },
    {
      key: DEFAULT_NULL_OPTION.key,
      value: DEFAULT_NULL_OPTION.key,
      label: t("schema-editor.default.null"),
      defaults: DEFAULT_NULL_OPTION,
      disabled: !props.column.nullable,
    },
    {
      key: DEFAULT_STRING_OPTION.key,
      value: DEFAULT_STRING_OPTION.key,
      label: t("schema-editor.default.value"),
      defaults: DEFAULT_STRING_OPTION,
    },
    {
      key: DEFAULT_EXPRESSION_OPTION.key,
      value: DEFAULT_EXPRESSION_OPTION.key,
      label: t("schema-editor.default.expression"),
      defaults: DEFAULT_EXPRESSION_OPTION,
    },
  ];
});

const inputDisabled = computed(() => {
  return props.disabled;
});

const handleSelect = (value: string) => {
  const option = options.value.find((opt) => opt.value === value);
  if (!option) {
    return;
  }

  const { defaults } = option;
  emit("select", defaults);

  if (typeof defaults.value.default === "string") {
    requestAnimationFrame(() => {
      inputRef.value?.focus();
    });
  }
};

const handleInput = (value: string) => {
  if (dropdownValue.value === "no-default" || dropdownValue.value === "null") {
    handleSelect("string");
  }
  emit("input", value);
};

const handleSimpleInput = (value: string) => {
  // Emit the value for the input handler
  emit("input", value);

  // Also emit a select event to maintain consistency with the existing API
  const defaultOption = {
    key: value.trim() ? "string" : "no-default",
    value: {
      hasDefault: !!value.trim(),
      default: value.trim(), // Both PostgreSQL and MySQL use default for now
    },
  };
  emit("select", defaultOption);
};

const inputStyle = computed(() => {
  const style: CSSProperties = {
    "--n-padding-left": "6px",
    "--n-padding-right": "16px",
    "--n-color": "transparent",
    "--n-color-disabled": "transparent",
    "--n-text-color-disabled": "rgb(var(--color-main))",
    cursor: "default",
  };
  const border = focused.value
    ? "1px solid rgb(var(--color-control-border))"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

watch(
  () => props.column.nullable,
  (nullable) => {
    if (!nullable && props.column.default === "NULL") {
      handleSelect(NO_DEFAULT_OPTION.key);
    }
  }
);
</script>

<style lang="postcss" scoped>
.bb-schema-editor--column-default-value-select :deep(.n-base-selection) {
  --n-padding-single: 0 16px 0 6px !important;
  --n-color: transparent !important;
  --n-color-disabled: transparent !important;
  --n-border: none !important;
  --n-text-color-disabled: rgb(var(--color-main)) !important;
}
.bb-schema-editor--column-default-value-select
  :deep(.n-base-selection .n-base-suffix) {
  right: 4px;
}
</style>
