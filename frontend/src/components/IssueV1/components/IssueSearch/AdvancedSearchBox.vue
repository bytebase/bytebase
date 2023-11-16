<template>
  <div
    ref="containerRef"
    class="bb-advanced-issue-search-box relative"
    v-bind="$attrs"
  >
    <NInput
      ref="inputRef"
      v-model:value="inputText"
      class="bb-advanced-issue-search-box__input"
      style="--n-padding-left: 8px"
      @click="handleInputClick"
      @blur="menuView = undefined"
      @keyup="handleKeyPress"
    >
      <template #prefix>
        <div
          class="flex flex-row items-center justify-start gap-x-2"
          :style="{
            'max-width': `calc(${containerWidth}px - 14rem)`,
          }"
        >
          <SearchIcon class="w-4 h-4 text-control-placeholder" />
          <div
            ref="tagsContainerRef"
            class="flex-1 flex flex-row items-center flex-nowrap gap-1 overflow-auto hide-scrollbar"
          >
            <ScopeTags
              :params="params"
              @update:params="$emit('update:params', $event)"
              @select-scope="(id) => selectScope(id)"
            />
          </div>
        </div>
      </template>
    </NInput>

    <Transition name="fade-slide-up" :appear="true">
      <div
        v-show="showMenu"
        v-zindexable="{ enabled: true }"
        class="absolute top-[36px] w-full py-1 divide-y divide-block-border bg-white shadow-md origin-top-left rounded-[3px] overflow-clip"
      >
        <ScopeMenu
          v-show="menuView === 'scope'"
          :input-text="inputText"
          :params="params"
          :options="visibleScopeOptions"
          :menu-index="menuIndex"
          @select-scope="selectScope"
        />
        <ValueMenu
          v-show="menuView === 'value'"
          :input-text="inputText"
          :params="params"
          :scope-option="currentScopeOption"
          :value-options="visibleValueOptions"
          :menu-index="menuIndex"
          @select-value="selectValue"
        />
      </div>
    </Transition>
  </div>
</template>

<script lang="ts" setup>
import { useElementSize } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { SearchIcon } from "lucide-vue-next";
import { InputInst, NInput } from "naive-ui";
import scrollIntoView from "scroll-into-view-if-needed";
import { zindexable as vZindexable } from "vdirs";
import {
  reactive,
  watch,
  onMounted,
  ref,
  toRef,
  computed,
  nextTick,
} from "vue";
import {
  SearchParams,
  SearchScopeId,
  getValueFromSearchParams,
  minmax,
  upsertScope,
} from "@/utils";
import ScopeMenu from "./ScopeMenu.vue";
import ScopeTags from "./ScopeTags.vue";
import ValueMenu from "./ValueMenu.vue";
import { useSearchScopeOptions } from "./useSearchScopeOptions";

const props = withDefaults(
  defineProps<{
    params: SearchParams;
    customClass?: string;
    autofocus?: boolean;
  }>(),
  {
    customClass: "",
    autofocus: false,
  }
);

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

interface LocalState {
  searchText: string;
  showSearchScopes: boolean;
  currentScope?: SearchScopeId;
}

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
  showSearchScopes: props.autofocus,
});
const containerRef = ref<HTMLElement>();
const tagsContainerRef = ref<HTMLElement>();
const inputText = ref(props.params.query);
const inputRef = ref<InputInst>();
const menuIndex = ref(0);
const { width: containerWidth } = useElementSize(containerRef);

watch(
  () => state.showSearchScopes,
  (show) => {
    if (show) state.currentScope = undefined;
  }
);

watch(
  () => props.params,
  (params) => {
    state.searchText = buildSearchTextByParams(params);
  }
);

const {
  menuView,
  availableScopeOptions,
  currentScope,
  currentScopeOption,
  valueOptions,
} = useSearchScopeOptions(toRef(props, "params"));

const visibleScopeOptions = computed(() => {
  if (currentScopeOption.value) {
    return [currentScopeOption.value];
  }

  const keyword = inputText.value.trim().replace(/:.*$/, "").toLowerCase();
  if (!keyword) return availableScopeOptions.value;
  return availableScopeOptions.value.filter((opt) => {
    return (
      opt.id.toLowerCase().includes(keyword) ||
      opt.description.toLowerCase().includes(keyword) ||
      opt.title.toLowerCase().includes(keyword)
    );
  });
});

const visibleValueOptions = computed(() => {
  if (!currentScope.value) return [];
  const scopePrefix = `${currentScope.value}:`;
  const keyword = inputText.value
    .trim()
    .toLowerCase()
    .substring(scopePrefix.length);
  if (!keyword) return valueOptions.value;
  const currentValue = getValueFromSearchParams(
    props.params,
    currentScope.value
  );
  return valueOptions.value.filter((opt) => {
    if (currentValue && opt.value === currentValue) return true;
    return opt.value.toLowerCase().includes(keyword);
  });
});

const visibleOptions = computed(() => {
  return menuView.value === "scope"
    ? visibleScopeOptions.value
    : menuView.value === "value"
    ? visibleValueOptions.value
    : ([] as unknown[]);
});

const showMenu = computed(() => {
  if (menuView.value === "scope") {
    return visibleScopeOptions.value.length > 0;
  }
  if (menuView.value === "value") {
    return true;
  }
  return false;
});

const moveMenuIndex = (delta: -1 | 1) => {
  const options = visibleOptions.value;
  if (options.length === 0) return;

  const target = minmax(menuIndex.value + delta, 0, options.length - 1);
  menuIndex.value = target;
};

const selectScope = (id: SearchScopeId | undefined) => {
  currentScope.value = id;
  if (id) {
    menuView.value = "value";
    // Fill-in the scope prefix if needed
    if (!inputText.value.startsWith(`${id}:`)) {
      inputText.value = `${id}:`;
    }
    scrollScopeTagIntoViewIfNeeded(id);
  } else {
    menuView.value = "scope";
  }
};
const selectValue = (value: string) => {
  const id = currentScope.value;
  if (!id) {
    menuView.value = undefined;
    return;
  }
  const updated = upsertScope(props.params, {
    id,
    value,
  });
  updated.query = "";
  inputText.value = "";
  selectScope(undefined);
  emit("update:params", updated);

  scrollScopeTagIntoViewIfNeeded(id);
};

const maybeSelectMatchedScope = () => {
  if (!menuView.value || menuView.value === "scope") {
    const matchedScope = visibleScopeOptions.value.find((opt) =>
      inputText.value.startsWith(`${opt.id}:`)
    );
    if (matchedScope) {
      // select the scope if the inputText matches its prefix
      selectScope(matchedScope.id);
      return true;
    }
    if (!menuView.value) {
      // Show scope menu if none of the menus are shown
      menuView.value = "scope";
      return true;
    }
  }
  return false;
};
const maybeDeselectMismatchedScope = () => {
  if (menuView.value === "value" && currentScope.value) {
    if (!inputText.value.startsWith(`${currentScope.value}:`)) {
      // de-select current scope since the inputText doesn't match its prefix.
      menuView.value = "scope";
      selectScope(undefined);
      return true;
    }
  }
  return false;
};

const maybeEmitIncompleteValue = () => {
  const updated = cloneDeep(props.params);
  updated.query = inputText.value;
  emit("update:params", updated);
};

const handleInputClick = () => {
  maybeSelectMatchedScope();
  maybeDeselectMismatchedScope();
  maybeEmitIncompleteValue();
};

const handleKeyPress = (e: KeyboardEvent) => {
  if (!inputRef.value || !inputRef.value.inputElRef) {
    return;
  }

  if (e.isComposing) return;
  if (e.defaultPrevented) return;
  const { key } = e;
  if (key === "Escape") {
    maybeEmitIncompleteValue();
    menuView.value = undefined;
    return;
  }

  if (maybeSelectMatchedScope()) {
    maybeEmitIncompleteValue();
    return;
  }
  if (maybeDeselectMismatchedScope()) {
    maybeEmitIncompleteValue();
    return;
  }

  if (key === "ArrowUp") {
    moveMenuIndex(-1);
    return;
  }
  if (key === "ArrowDown") {
    moveMenuIndex(1);
    return;
  }
  if (key === "Enter") {
    // Press enter to select scope (dive into the next step)
    // or select value
    const index = menuIndex.value;
    if (menuView.value === "scope") {
      const option = visibleScopeOptions.value[index];
      if (option) {
        selectScope(option.id);
        maybeEmitIncompleteValue();
        return;
      }
    }
    if (menuView.value === "value") {
      const option = visibleValueOptions.value[index];
      if (option) {
        selectValue(option.value);
        return;
      }
    }
  }

  maybeEmitIncompleteValue();
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
});
watch(menuView, () => {
  menuIndex.value = 0;
  if (menuView.value === "value" && currentScope.value) {
    const value = getValueFromSearchParams(props.params, currentScope.value);
    if (value) {
      const index = valueOptions.value.findIndex(
        (option) => option.value === value
      );
      if (index >= 0) menuIndex.value = index;
    }
  }
});
watch(visibleScopeOptions, (newOptions, oldOptions) => {
  if (menuView.value !== "scope") return;
  const highlightedScope = oldOptions[menuIndex.value]?.id;
  if (highlightedScope) {
    const index = newOptions.findIndex((opt) => opt.id === highlightedScope);
    if (index >= 0) {
      menuIndex.value = index;
      return;
    }
  }
  menuIndex.value = minmax(menuIndex.value, 0, newOptions.length - 1);
});
watch(visibleValueOptions, (newOptions, oldOptions) => {
  if (menuView.value !== "value") return;
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
    inputText.value = params.query;
  }
);
</script>

<style lang="postcss" scoped>
.bb-advanced-issue-search-box
  .bb-advanced-issue-search-box__input
  :deep(.n-input__input) {
  @apply flex flex-row items-center;
}
</style>
