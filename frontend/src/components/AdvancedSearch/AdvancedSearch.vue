<template>
  <div ref="containerRef" class="w-full relative">
    <NInput
      ref="inputRef"
      v-model:value="inputText"
      :placeholder="placeholder ?? $t('issue.advanced-search.self')"
      style="--n-padding-left: 8px; --n-padding-right: 4px"
      @click="handleInputClick"
      @keyup="handleKeyUp"
      @keydown="handleKeyDown"
    >
      <template #prefix>
        <div
          class="flex flex-row items-center justify-start gap-x-2"
          :style="{
            'max-width': `calc(${containerWidth}px - 14rem)`,
          }"
        >
          <div class="flex items-center space-x-2">
            <FilterIcon class="w-4 h-4 text-control-placeholder" />
            <span class="textinfolabel">
              {{ $t("issue.advanced-search.filter") }}
            </span>
          </div>
          <div
            ref="tagsContainerRef"
            class="flex-1 flex flex-row items-center flex-nowrap gap-1 overflow-auto hide-scrollbar"
          >
            <ScopeTags
              :params="params"
              :scope-options="scopeOptions"
              :focused-tag-id="focusedTagId"
              @select-scope="selectScopeFromTag"
              @remove-scope="removeScope"
            />
          </div>
        </div>
      </template>
      <template #suffix>
        <NButton
          v-show="clearable"
          quaternary
          circle
          size="tiny"
          @click.stop.prevent="handleClear"
        >
          <template #icon>
            <XIcon class="w-3 h-3" />
          </template>
        </NButton>
      </template>
    </NInput>

    <Transition name="fade-slide-up" :appear="true">
      <div
        v-show="showMenu"
        v-zindexable="{ enabled: true }"
        class="absolute top-[36px] w-full bg-gray-100 shadow-xl origin-top-left rounded-[3px] overflow-clip"
      >
        <ScopeMenu
          :show="state.menuView === 'scope'"
          :options="visibleScopeOptions"
          :menu-index="menuIndex"
          @select-scope="selectScope"
          @hover-item="menuIndex = $event"
        />
        <ValueMenu
          :show="state.menuView === 'value'"
          :scope-option="currentScopeOption"
          :value-options="visibleValueOptions"
          :menu-index="menuIndex"
          :fetch-state="currentFetchState"
          :show-empty-placeholder="
            (currentScopeOption?.options ?? []).length > 0
          "
          @select-value="selectValue"
          @hover-item="menuIndex = $event"
          @fetch-next-page="() => handleSearch(currentValueForScope)"
        />
      </div>
    </Transition>
  </div>
</template>

<script lang="ts" setup>
import { useDebounceFn, onClickOutside } from "@vueuse/core";
import { useElementSize } from "@vueuse/core";
import { cloneDeep, last } from "lodash-es";
import { FilterIcon } from "lucide-vue-next";
import { XIcon } from "lucide-vue-next";
import { NButton, NInput, type InputInst } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { zindexable as vZindexable } from "vdirs";
import { reactive, watch, onMounted, ref, computed, nextTick } from "vue";
import { useRoute, useRouter } from "vue-router";
import { DEBOUNCE_SEARCH_DELAY } from "@/types";
import type { SearchParams, SearchScopeId } from "@/utils";
import {
  emptySearchParams,
  getValueFromSearchParams,
  minmax,
  upsertScope,
  buildSearchTextBySearchParams,
  buildSearchParamsBySearchText,
  mergeSearchParams,
} from "@/utils";
import ScopeMenu from "./ScopeMenu.vue";
import ScopeTags from "./ScopeTags.vue";
import ValueMenu from "./ValueMenu.vue";
import type { ScopeOption } from "./types";

const props = withDefaults(
  defineProps<{
    params: SearchParams;
    scopeOptions?: ScopeOption[];
    placeholder?: string | undefined;
    autofocus?: boolean;
    overrideRouteQuery?: boolean;
  }>(),
  {
    scopeOptions: () => [],
    autofocus: false,
    placeholder: undefined,
    overrideRouteQuery: true,
  }
);

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
  (event: "select-unsupported-scope", id: SearchScopeId): void;
}>();

interface LocalState {
  searchText: string;
  currentScope?: SearchScopeId;
  menuView?: "value" | "scope";
  scopeOptions: ScopeOption[];
  fetchDataStateMap: Map<
    SearchScopeId,
    {
      loading: boolean;
      nextPageToken?: string;
    }
  >;
}

const route = useRoute();
const router = useRouter();

const defaultSearchParams = () => {
  const params = emptySearchParams();
  for (const scope of props.params.scopes) {
    if (scope.readonly) {
      params.scopes.push({ ...scope });
    }
  }
  return params;
};

const buildSearchTextByParams = (params: SearchParams | undefined): string => {
  const prefix = (params?.scopes ?? [])
    .map((scope) => `${scope.id}:${scope.value}`)
    .join(" ");
  const query = params?.query ?? "";
  if (!prefix && !query) {
    return "";
  }
  return `${prefix} ${query}`;
};

const state = reactive<LocalState>({
  searchText: buildSearchTextByParams(props.params),
  scopeOptions: [],
  fetchDataStateMap: new Map(),
});

watch(
  () => props.scopeOptions,
  () => {
    state.scopeOptions = cloneDeep(props.scopeOptions);
  },
  { deep: true, immediate: true }
);

const containerRef = ref<HTMLElement>();
const tagsContainerRef = ref<HTMLElement>();
const inputText = ref(props.params.query);
const inputRef = ref<InputInst>();
const menuIndex = ref(-1);
const { width: containerWidth } = useElementSize(containerRef);
const focusedTagId = ref<SearchScopeId>();

const editableScopes = computed(() => {
  return props.params.scopes.filter((s) => !s.readonly);
});

watch(
  () => props.params,
  (params) => {
    state.searchText = buildSearchTextByParams(params);
  }
);

const valueOptions = computed(() => {
  if (state.menuView === "value" && currentScopeOption.value) {
    return currentScopeOption.value.options ?? [];
  }
  return [];
});

const currentScopeOption = computed(() => {
  if (state.currentScope) {
    return state.scopeOptions.find((opt) => opt.id === state.currentScope);
  }
  return undefined;
});

const currentValueForScope = computed(() => {
  if (!state.currentScope) return "";
  const scopePrefix = `${state.currentScope}:`;
  return inputText.value.trim().toLowerCase().substring(scopePrefix.length);
});

const currentFetchState = computed(() => {
  return state.currentScope
    ? (state.fetchDataStateMap.get(state.currentScope) ?? {
        loading: false,
      })
    : { loading: false };
});

const handleSearch = useDebounceFn(async (search: string) => {
  if (!currentScopeOption.value?.search) {
    return;
  }

  const fetchState = { ...currentFetchState.value };
  if (fetchState.loading) {
    return;
  }
  fetchState.loading = true;

  try {
    const { options, nextPageToken } = await currentScopeOption.value.search({
      keyword: search,
      nextPageToken: fetchState.nextPageToken,
    });
    if (!currentScopeOption.value.options) {
      currentScopeOption.value.options = [];
    }
    if (!fetchState.nextPageToken) {
      currentScopeOption.value.options = [...options];
    } else {
      currentScopeOption.value.options.push(...options);
    }
    fetchState.nextPageToken = nextPageToken;
  } finally {
    fetchState.loading = false;
    state.fetchDataStateMap.set(currentScopeOption.value.id, fetchState);
  }
}, DEBOUNCE_SEARCH_DELAY);

watch(
  [() => currentScopeOption.value, () => currentValueForScope.value],
  async ([scopeOption, valueForScope]) => {
    if (!scopeOption || !scopeOption.search) {
      return;
    }

    state.fetchDataStateMap.set(scopeOption.id, {
      loading: false,
      nextPageToken: "",
    });
    await handleSearch(valueForScope);
  },
  { immediate: true }
);

// availableScopeOptions will hide chosen search scope.
// For example, if uses already select the instance, we should NOT show the instance scope in the dropdown.
const availableScopeOptions = computed((): ScopeOption[] => {
  const existedScopes = new Set<SearchScopeId>(
    props.params.scopes.map((scope) => scope.id)
  );

  return state.scopeOptions.filter((scope) => {
    if (existedScopes.has(scope.id) && !scope.allowMultiple) {
      return false;
    }
    return true;
  });
});

const visibleScopeOptions = computed(() => {
  if (currentScopeOption.value) {
    return [currentScopeOption.value];
  }

  const keyword = inputText.value.trim().replace(/:.*$/, "").toLowerCase();
  if (!keyword) return availableScopeOptions.value;

  return availableScopeOptions.value.filter(
    (option) =>
      option.id.toLowerCase().includes(keyword) ||
      option.title.toLowerCase().includes(keyword)
  );
});

const visibleValueOptions = computed(() => {
  if (!state.currentScope) return [];

  const keyword = currentValueForScope.value
    .trim()
    .replace(/:.*$/, "")
    .toLowerCase();
  if (!keyword || currentScopeOption.value?.search) {
    return valueOptions.value;
  }

  const filtered = valueOptions.value.filter(
    (option) =>
      option.value.toLowerCase().includes(keyword) ||
      option.keywords.some((key) => key.includes(keyword))
  );

  const currentValue = getValueFromSearchParams(
    props.params,
    state.currentScope
  );
  const option = valueOptions.value.find((opt) => opt.value === currentValue);
  if (currentValue && option) {
    // If we have current value, put it to the first if it doesn't match the keyword
    const index = filtered.findIndex((opt) => opt.value === currentValue);
    if (index < 0) {
      filtered.unshift(option);
    }
  }
  return filtered;
});

const visibleOptions = computed(() => {
  return state.menuView === "scope"
    ? visibleScopeOptions.value
    : state.menuView === "value"
      ? visibleValueOptions.value
      : ([] as unknown[]);
});

const showMenu = computed(() => {
  if (state.menuView === "scope") {
    return visibleScopeOptions.value.length > 0;
  }
  if (state.menuView === "value") {
    return true;
  }
  return false;
});

const clearable = computed(() => {
  return (
    props.params.query.trim().length > 0 || editableScopes.value.length > 0
  );
});

const hideMenu = () => {
  nextTick(() => {
    state.menuView = undefined;
    focusedTagId.value = undefined;
  });
};

onClickOutside(containerRef, hideMenu);

const moveMenuIndex = (delta: -1 | 1) => {
  const options = visibleOptions.value;
  if (options.length === 0) return;

  const target = minmax(menuIndex.value + delta, 0, options.length - 1);
  menuIndex.value = target;
};

const removeScope = (id: SearchScopeId) => {
  const updated = upsertScope({
    params: props.params,
    scopes: {
      id,
      value: "",
    },
  });
  emit("update:params", updated);
};

const selectScope = (
  id: SearchScopeId | undefined,
  value: string | undefined = undefined
) => {
  state.currentScope = id;
  if (id) {
    state.menuView = "value";
    // Fill-in the scope prefix if needed
    if (!inputText.value.startsWith(`${id}:`)) {
      inputText.value = `${id}:${value ?? ""}`;
    }
    scrollScopeTagIntoViewIfNeeded(id);
  } else {
    state.menuView = "scope";
  }
};

const extractValue = () => {
  const id = state.currentScope;
  if (!id) {
    return;
  }
  const text = inputText.value;
  if (!text.startsWith(`${id}:`)) {
    return;
  }
  return text.slice(`${id}:`.length);
};

const selectValue = (value: string) => {
  const id = state.currentScope;
  if (!id || !currentScopeOption.value) {
    state.menuView = undefined;
    return;
  }
  const { allowMultiple } = currentScopeOption.value;
  const updated = upsertScope({
    params: props.params,
    scopes: {
      id,
      value,
    },
    allowMultiple,
  });
  updated.query = "";
  inputText.value = "";
  selectScope(undefined);
  emit("update:params", updated);

  scrollScopeTagIntoViewIfNeeded(id);
  hideMenu();
};

const selectScopeFromTag = (id: SearchScopeId) => {
  if (state.scopeOptions.find((opt) => opt.id === id)) {
    // For AdvancedSearch supported scopes
    selectScope(id);
    return;
  }

  // Unsupported scope for AdvancedSearch
  // emit an event and wish the parent UI can handle this
  emit("select-unsupported-scope", id);
  hideMenu();
};

const maybeSelectMatchedScope = () => {
  if (!state.menuView || state.menuView === "scope") {
    const matchedScope = visibleScopeOptions.value.find((opt) =>
      inputText.value.startsWith(`${opt.id}:`)
    );
    if (matchedScope) {
      // select the scope if the inputText matches its prefix
      selectScope(matchedScope.id);
      return true;
    }
    if (!state.menuView) {
      // Show scope menu if none of the menus are shown
      state.menuView = "scope";
      return true;
    }
  }
  return false;
};

const maybeDeselectMismatchedScope = () => {
  if (state.menuView === "value" && state.currentScope) {
    if (!inputText.value.startsWith(`${state.currentScope}:`)) {
      // de-select current scope since the inputText doesn't match its prefix.
      state.menuView = "scope";
      selectScope(undefined);
      return true;
    }
  }
  return false;
};

const maybeEmitIncompleteValue = () => {
  if (!inputText.value.startsWith(`${state.currentScope}:`)) {
    const updated = cloneDeep(props.params);
    updated.query = inputText.value;
    updateParams(updated);
  }
};

const updateParams = useDebounceFn((params: SearchParams) => {
  emit("update:params", params);
}, DEBOUNCE_SEARCH_DELAY);

const handleInputClick = () => {
  maybeSelectMatchedScope();
  maybeDeselectMismatchedScope();
};

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.isComposing) return;
  if (e.defaultPrevented) return;
  const { key } = e;
  if (key === "Backspace" && inputText.value === "") {
    // Pressing "backspace" when the input box is empty
    if (focusedTagId.value) {
      e.stopPropagation();
      e.preventDefault();
      // Delete the focusedTag if it exists
      const id = focusedTagId.value;
      focusedTagId.value = undefined;
      removeScope(id);
      return;
    } else {
      e.stopPropagation();
      e.preventDefault();
      // Otherwise mark the last editable scope as focused.
      const id = last(editableScopes.value)?.id;
      if (id) {
        focusedTagId.value = id;
        scrollScopeTagIntoViewIfNeeded(id);
      }
      return;
    }
  }
  focusedTagId.value = undefined;

  if (key === "ArrowUp") {
    moveMenuIndex(-1);
    e.preventDefault();
    return;
  }
  if (key === "ArrowDown") {
    moveMenuIndex(1);
    e.preventDefault();
    return;
  }
};

const handleKeyUp = (e: KeyboardEvent) => {
  if (e.isComposing) return;
  if (e.defaultPrevented) return;
  const { key } = e;
  if (key === "Escape") {
    maybeEmitIncompleteValue();
    state.menuView = undefined;
    return;
  }
  if (key === "Backspace" && inputText.value === "") {
    // backspace key might be processed by KeyDown
    if (focusedTagId.value) {
      return;
    }
  }
  if (maybeSelectMatchedScope()) {
    maybeEmitIncompleteValue();
    return;
  }
  if (maybeDeselectMismatchedScope()) {
    maybeEmitIncompleteValue();
    return;
  }
  if (key === "Enter") {
    // Press enter to select scope (dive into the next step)
    // or select value
    const index = menuIndex.value;
    if (index < 0) {
      hideMenu();
      return;
    }
    if (state.menuView === "scope") {
      const option = visibleScopeOptions.value[index];
      if (option) {
        selectScope(option.id);
        maybeEmitIncompleteValue();
        return;
      }
    }
    if (state.menuView === "value") {
      if (visibleValueOptions.value.length === 0) {
        const val = extractValue();
        if (val) {
          selectValue(val);
          return;
        }
      } else if (visibleValueOptions.value[index]) {
        selectValue(visibleValueOptions.value[index].value);
        return;
      }
    }
  }

  maybeEmitIncompleteValue();
};

const handleClear = () => {
  const params = defaultSearchParams();
  emit("update:params", params);
  hideMenu();
};

const scrollScopeTagIntoViewIfNeeded = (id: SearchScopeId) => {
  nextTick(() => {
    const tagsContainerEl = tagsContainerRef.value;
    if (!tagsContainerEl) return;
    const tagEl = tagsContainerEl.querySelector(
      `[data-search-scope-id="${id}"]`
    );
    if (tagEl) {
      scrollIntoView(tagEl, {
        scrollMode: "if-needed",
      });
    }
  });
};

onMounted(() => {
  if (props.autofocus) {
    inputRef.value?.inputElRef?.focus();
  }
  const { qs } = route.query;
  if (typeof qs === "string" && qs.length > 0) {
    const params = buildSearchParamsBySearchText(qs);
    const existedScopes = props.params.scopes.reduce((map, scope) => {
      map.set(scope.id, scope.readonly ?? false);
      return map;
    }, new Map<SearchScopeId, boolean>());
    params.scopes = params.scopes.map((scope) => ({
      ...scope,
      readonly: existedScopes.get(scope.id),
    }));
    emit("update:params", mergeSearchParams(cloneDeep(props.params), params));
  }
});

watch(
  () => state.menuView,
  () => {
    focusedTagId.value = undefined;
    menuIndex.value = -1;
    if (state.menuView === "value" && state.currentScope) {
      const value = getValueFromSearchParams(props.params, state.currentScope);
      if (value) {
        const index = valueOptions.value.findIndex(
          (option) => option.value === value
        );
        if (index >= 0) {
          menuIndex.value = index;
        }
      }
    }
  }
);

watch(visibleScopeOptions, (newOptions, oldOptions) => {
  if (state.menuView !== "scope") return;
  const highlightedScope = oldOptions[menuIndex.value]?.id;
  if (highlightedScope) {
    const index = newOptions.findIndex((opt) => opt.id === highlightedScope);
    if (index >= 0) {
      menuIndex.value = index;
      return;
    }
  }
});

watch(visibleValueOptions, (newOptions, oldOptions) => {
  if (state.menuView !== "value") return;
  const highlightedValue = oldOptions[menuIndex.value]?.value;
  if (highlightedValue) {
    const index = newOptions.findIndex((opt) => opt.value === highlightedValue);
    if (index >= 0) {
      menuIndex.value = index;
      return;
    }
  }
  menuIndex.value = minmax(menuIndex.value, 0, newOptions.length - 1);
});

watch(
  () => props.params,
  (params) => {
    if (!inputText.value) {
      inputText.value = params.query;
    }
    if (props.overrideRouteQuery) {
      router.replace({
        query: {
          ...route.query,
          qs: buildSearchTextBySearchParams(params),
        },
      });
    }
  },
  { deep: true }
);
</script>
