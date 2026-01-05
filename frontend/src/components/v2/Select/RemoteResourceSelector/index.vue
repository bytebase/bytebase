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
    :value="value"
    :disabled="disabled"
    :options="options"
    :fallback-option="fallbackOption"
    :render-label="customLabel"
    :render-tag="customTag"
    :placeholder="placeholder ?? $t('common.search-for-more')"
    :size="size"
    :consistent-menu-width="consistentMenuWidth"
    :reset-menu-on-options-change="false"
    @search="handleSearch"
    @click="onSelectorOpen"
    @update:value="$emit('update:value', $event)"
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

<script lang="tsx" setup generic="T">
import { useDebounceFn } from "@vueuse/core";
import type { SelectOption } from "naive-ui";
import { NButton, NSelect } from "naive-ui";
import type { SelectBaseOption } from "naive-ui/lib/select/src/interface";
import { computed, nextTick, type Ref, ref, type VNodeChild, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import { getDefaultPagination } from "@/utils";
import type { ResourceSelectOption, SelectSize } from "./types";

const props = withDefaults(
  defineProps<{
    placeholder?: string;
    multiple?: boolean;
    disabled?: boolean;
    value?: string[] | string | undefined;
    consistentMenuWidth?: boolean;
    size?: SelectSize;
    additionalOptions?: ResourceSelectOption<T>[];
    renderLabel?: (
      option: ResourceSelectOption<T>,
      selected: boolean,
      searchText: string
    ) => VNodeChild;
    renderTag?: (props: {
      option: ResourceSelectOption<T>;
      handleClose: () => void;
    }) => VNodeChild;
    filter?: (resource: T) => boolean;
    search?: (params: {
      search: string;
      pageToken: string;
      pageSize: number;
    }) => Promise<{
      nextPageToken: string | undefined;
      options: ResourceSelectOption<T>[];
    }>;
    fallbackOption?: false | ((value: string) => ResourceSelectOption<T>);
    filterable?: boolean;
  }>(),
  {
    fallbackOption: false,
    consistentMenuWidth: true,
    filterable: true,
    additionalOptions: () => [],
  }
);

const emit = defineEmits<{
  (event: "update:value", value: string[] | string | undefined): void;
  (event: "open"): void;
}>();

const isLoadingNextPage = ref(false);
const pageToken = ref<string | undefined>("");
const searchText = ref("");
const show = ref(false);
const rawOptions = ref([]) as Ref<ResourceSelectOption<T>[]>;

const appendDataToRawList = (data: ResourceSelectOption<T>[]) => {
  for (const item of data) {
    if (!rawOptions.value.find((raw) => raw.value === item.value)) {
      rawOptions.value.unshift(item);
    }
  }
};

watch(
  [
    () => props.additionalOptions,
    () => isLoadingNextPage.value,
    () => searchText.value,
  ],
  ([additionalOptions, isLoadingNextPage, searchText]) => {
    if (!isLoadingNextPage && !searchText) {
      appendDataToRawList(additionalOptions);
    }
  },
  { deep: true }
);

const onNextPage = async (openOptions: boolean) => {
  if (!props.search) {
    return;
  }
  isLoadingNextPage.value = true;
  const { nextPageToken, options } = await props.search({
    pageToken: pageToken.value ?? "",
    pageSize: getDefaultPagination(),
    search: searchText.value,
  });

  if (!pageToken.value) {
    rawOptions.value = options;
  } else {
    appendDataToRawList(options);
  }

  pageToken.value = nextPageToken;
  isLoadingNextPage.value = false;

  if (openOptions) {
    nextTick(() => (show.value = true));
  }
};

const onSelectorOpen = async () => {
  if (props.disabled) {
    return;
  }
  await handleSearch("");
  emit("open");
};

const handleSearch = useDebounceFn(
  async (search: string, openOptions: boolean = true) => {
    searchText.value = search.trim().toLowerCase();
    pageToken.value = "";
    await onNextPage(openOptions);
  },
  DEBOUNCE_SEARCH_DELAY
);

const customLabel = computed(() => {
  const renderLabel = props.renderLabel;
  if (renderLabel) {
    return (option: SelectOption, selected: boolean) =>
      renderLabel(
        option as ResourceSelectOption<T>,
        selected,
        searchText.value
      );
  }
  return undefined;
});

const customTag = computed(() => {
  const renderTag = props.renderTag;
  if (renderTag) {
    return (params: { option: SelectBaseOption; handleClose: () => void }) =>
      renderTag({
        option: params.option as ResourceSelectOption<T>,
        handleClose: params.handleClose,
      });
  }
  return undefined;
});

const options = computed((): ResourceSelectOption<T>[] => {
  const filter = props.filter;
  if (!filter) {
    return rawOptions.value;
  }
  return rawOptions.value.filter((raw) => raw.resource && filter(raw.resource));
});

defineExpose({
  reset: async () => {
    await handleSearch("", false);
  },
  options,
});
</script>
