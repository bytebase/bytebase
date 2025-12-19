<template>
  <NSelect
    v-bind="$attrs"
    v-model:show="show"
    :loading="isLoadingNextPage"
    :filterable="true"
    :clearable="true"
    :remote="true"
    :virtual-scroll="true"
    :multiple="multiple"
    :value="selected"
    :disabled="disabled"
    :options="options"
    :fallback-option="false"
    :render-label="renderLabel"
    :render-tag="renderTag"
    :placeholder="placeholder ?? $t('common.search-for-more')"
    :size="size"
    :consistent-menu-width="consistentMenuWidth"
    :reset-menu-on-options-change="false"
    @search="handleSearch"
    @click="onSelectorOpen"
    @update:value="handleValueUpdated"
  >
    <template #empty>
      <slot name="empty">
        <BBSpin v-if="isLoadingNextPage" />
      </slot>
    </template>
    <template #action>
      <NButton
        v-if="pageToken"
        class="w-full!"
        quaternary
        :loading="isLoadingNextPage"
        @click="() => onNextPage(true)">
        {{ $t("common.load-more") }}
      </NButton>
    </template>
  </NSelect>
</template>

<script lang="tsx" setup generic="T extends { name: string }">
import { useDebounceFn } from "@vueuse/core";
import type { SelectOption } from "naive-ui";
import { NButton, NCheckbox, NSelect, NTag } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, nextTick, type Ref, ref, type VNodeChild, watch } from "vue";
import { BBSpin } from "@/bbkit";
import EllipsisText from "@/components/EllipsisText.vue";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import { getDefaultPagination } from "@/utils";

type ResourceSelectOption = SelectOption & {
  resource: T;
  value: string;
  label: string;
};

const props = withDefaults(
  defineProps<{
    placeholder?: string;
    multiple?: boolean;
    disabled?: boolean;
    value?: string | undefined | null;
    values?: string[] | undefined | null;
    consistentMenuWidth?: boolean;
    size?: "tiny" | "small" | "medium" | "large";
    showResourceName?: boolean;
    resourceNameClass?: string;
    additionalData?: T[];
    customLabel?: (resource: T) => VNodeChild;
    filter?: (resource: T) => boolean;
    search?: (params: {
      search: string;
      pageToken: string;
      pageSize: number;
    }) => Promise<{ nextPageToken: string | undefined; data: T[] }>;
    getOption: (resource: T) => { value: string; label: string };
  }>(),
  {
    customLabel: undefined,
    size: "medium",
    fallbackOption: false,
    disabled: false,
    multiple: false,
    showResourceName: true,
    resourceNameClass: "",
    consistentMenuWidth: true,
    additionalData: () => [],
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    filter: (resource: T) => true,
  }
);

const emit = defineEmits<{
  (event: "update:value", value: string | undefined): void;
  (event: "update:values", value: string[]): void;
}>();

const isLoadingNextPage = ref(false);
const pageToken = ref<string | undefined>("");
const searchText = ref("");
const show = ref(false);
const rawList = ref([]) as Ref<T[]>;

const appendDataToRawList = (data: T[]) => {
  for (const item of data) {
    if (!rawList.value.find((raw) => raw.name === item.name)) {
      rawList.value.push(item);
    }
  }
};

watch(
  [
    () => props.additionalData,
    () => isLoadingNextPage.value,
    () => searchText.value,
  ],
  ([additionalData, isLoadingNextPage, searchText]) => {
    if (!isLoadingNextPage && !searchText) {
      appendDataToRawList(additionalData);
    }
  },
  { deep: true }
);

const onNextPage = async (openOptions: boolean) => {
  if (!props.search) {
    return;
  }
  isLoadingNextPage.value = true;
  const { nextPageToken, data } = await props.search({
    pageToken: pageToken.value ?? "",
    pageSize: getDefaultPagination(),
    search: searchText.value,
  });

  if (!pageToken.value) {
    rawList.value = data;
  } else {
    appendDataToRawList(data);
  }

  pageToken.value = nextPageToken;
  isLoadingNextPage.value = false;

  if (openOptions) {
    nextTick(() => (show.value = true));
  }
};

const selected = computed(() => {
  if (props.multiple) {
    return props.values || [];
  } else {
    return props.value;
  }
});

const onSelectorOpen = async () => {
  if (props.disabled) {
    return;
  }
  await handleSearch("");
};

const handleSearch = useDebounceFn(
  async (search: string, openOptions: boolean = true) => {
    searchText.value = search.trim().toLowerCase();
    pageToken.value = "";
    await onNextPage(openOptions);
  },
  DEBOUNCE_SEARCH_DELAY
);

const renderLabel = (option: SelectOption, selected: boolean) => {
  const { resource, label } = option as ResourceSelectOption;
  const node = (
    <div class="py-1">
      {props.customLabel ? props.customLabel(resource) : label}
      {props.showResourceName && (
        <div>
          <EllipsisText
            class={`opacity-60 textinfolabel ${props.resourceNameClass}`}
          >
            {resource.name}
          </EllipsisText>
        </div>
      )}
    </div>
  );
  if (props.multiple) {
    return (
      <div class="flex items-center gap-x-2 py-2">
        <NCheckbox checked={selected} size="small" />
        {node}
      </div>
    );
  }

  return node;
};

const renderTag = ({
  option,
  handleClose,
}: {
  option: SelectBaseOption;
  handleClose: () => void;
}) => {
  const { resource, label } = option as ResourceSelectOption;
  const node = props.customLabel ? props.customLabel(resource) : label;
  if (props.multiple) {
    return (
      <NTag size={props.size} closable={!props.disabled} onClose={handleClose}>
        {node}
      </NTag>
    );
  }
  return node;
};

const handleValueUpdated = (value: string | string[] | undefined | null) => {
  if (props.multiple) {
    if (!value) {
      // normalize value
      value = [];
    }
    emit("update:values", value as string[]);
  } else {
    if (value === null) {
      // normalize value
      value = "";
    }
    emit("update:value", value as string);
  }
};

const options = computed((): ResourceSelectOption[] => {
  return rawList.value.filter(props.filter).map((data) => {
    return {
      resource: data,
      ...props.getOption(data),
    };
  });
});
</script>
