<template>
  <div class="relative" :class="customClass">
    <NInput
      ref="inputRef"
      :value="state.searchText"
      :clearable="!!state.searchText"
      :placeholder="$t('issue.advanced-search.self')"
      style="width: 100%"
      @update:value="onUpdate($event)"
      @focus="state.showSearchScopes = true"
      @blur="onClear"
      @keyup="onKeydown"
    >
      <template #prefix>
        <heroicons-outline:search class="h-4 w-4 text-control-placeholder" />
      </template>
    </NInput>
    <div
      v-if="state.showSearchScopes"
      class="absolute z-50 top-full w-full divide-y divide-block-border bg-white shadow-md"
    >
      <div
        v-for="item in searchScopes"
        :key="item.id"
        class="flex gap-x-3 p-3 items-center cursor-pointer hover:bg-gray-100"
        @mousedown.prevent.stop="
          () => {
            state.showSearchScopes = false;
            state.currentScope = item.id;
          }
        "
      >
        <heroicons-outline:filter class="h-4 w-4 text-control" />
        <span class="text-accent">{{ item.title }}</span>
        <span class="text-control-light">{{ item.description }}</span>
      </div>
    </div>
    <div
      v-if="state.currentScope && searchOptions.length > 0"
      class="absolute z-50 top-full w-full divide-y divide-block-border bg-white shadow-md"
    >
      <div class="p-3 text-lg text-control-light">
        {{ searchKeyword }}
      </div>
      <div class="max-h-60 overflow-y-auto divide-y divide-block-border">
        <div
          v-for="option in searchOptions"
          :key="option.id"
          class="flex gap-x-3 p-3 items-baseline cursor-pointer hover:bg-gray-100"
          @mousedown.prevent.stop="
            onOptionSelect(state.currentScope, option.id)
          "
        >
          <component :is="option.label" class="text-control text-sm" />
          <span class="text-control-light text-xs">{{ option.id }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { debounce } from "lodash-es";
import { NInput } from "naive-ui";
import { reactive, computed, h, VNode, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import GitIcon from "@/components/GitIcon.vue";
import { useProjectV1ListByCurrentUser } from "@/store";
import { Workflow } from "@/types/proto/v1/project_service";
import { projectV1Name } from "@/utils";

type SearchScopeId = "project";

export interface SearchParams {
  query: string;
  scopes: {
    id: SearchScopeId;
    value: string;
  }[];
}

const props = withDefaults(
  defineProps<{
    customClass?: string;
    params?: SearchParams;
    autofocus?: boolean;
  }>(),
  {
    customClass: "",
    params: undefined,
    autofocus: false,
  }
);

const emit = defineEmits<{
  (event: "update", params: SearchParams): void;
}>();

const { t } = useI18n();

interface LocalState {
  searchText: string;
  showSearchScopes: boolean;
  currentScope?: SearchScopeId;
}

interface SearchOption {
  id: string;
  label: VNode;
}

interface SearchScope {
  id: SearchScopeId;
  title: string;
  description: string;
  options: SearchOption[];
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
  showSearchScopes: false,
});
const inputRef = ref<InstanceType<typeof NInput>>();

const { projectList } = useProjectV1ListByCurrentUser();

const searchScopes = computed((): SearchScope[] => {
  return [
    {
      id: "project",
      title: t("issue.advanced-search.scope.project.title"),
      description: t("issue.advanced-search.scope.project.description"),
      options: projectList.value.map((proj) => {
        const children: VNode[] = [
          h("span", { innerHTML: projectV1Name(proj) }),
        ];
        if (proj.workflow === Workflow.VCS) {
          children.push(h(GitIcon, { class: "w-5" }));
        }
        return {
          id: proj.name,
          label: h("div", { class: "flex gap-x-2" }, children),
        };
      }),
    },
  ];
});

const searchOptions = computed((): SearchOption[] => {
  const item = searchScopes.value.find(
    (item) => item.id === state.currentScope
  );
  return item?.options ?? [];
});

const searchKeyword = computed(() => {
  const scope = searchScopes.value.find(
    (item) => item.id === state.currentScope
  );
  return scope?.title ?? "";
});

const onOptionSelect = (keyword: SearchScopeId, id: string) => {
  const search = `${keyword}:${id} ${query.value}`;
  state.searchText = search;
  debouncedUpdate();
  onClear();
};

const onClear = () => {
  state.showSearchScopes = false;
  state.currentScope = undefined;
};

const debouncedUpdate = debounce(() => {
  emit("update", getSearchParamsByText(state.searchText));
}, 500);

const onUpdate = (value: string) => {
  state.searchText = value;
  debouncedUpdate();
};

const query = computed(() => {
  const sections = state.searchText.split(" ");
  let i = 0;
  while (i < sections.length) {
    const section = sections[i];
    const keyword = section.split(":")[0];
    const exist =
      searchScopes.value.findIndex((item) => item.id === keyword) >= 0;
    if (!exist) {
      break;
    }
    i++;
  }
  return sections.slice(i).join(" ");
});

const getSearchParamsByText = (text: string): SearchParams => {
  const plainQuery = query.value;
  const scopeText = text.split(` ${plainQuery}`)[0] || "";
  return {
    query: plainQuery,
    scopes: scopeText.split(" ").map((scope) => {
      return {
        id: scope.split(":")[0] as SearchScopeId,
        value: scope.split(":")[1],
      };
    }),
  };
};

onMounted(() => {
  if (props.autofocus) {
    inputRef.value?.inputElRef?.focus();
  }
});

const onKeydown = (e: KeyboardEvent) => {
  if (!inputRef.value || !inputRef.value.inputElRef) {
    return;
  }
  if (!state.searchText) {
    state.showSearchScopes = true;
    return;
  }

  const start = inputRef.value.inputElRef.selectionStart ?? -1;
  const end = inputRef.value.inputElRef.selectionEnd ?? -1;
  if (start !== end) {
    onClear();
    return;
  }

  const sections = state.searchText.split(" ");
  let i = 0;
  let len = 0;
  while (i < sections.length) {
    len += sections[i].length;
    if (i < sections.length - 1) {
      len += 1;
    }
    if (len >= start) {
      break;
    }
    i++;
  }
  if (i >= sections.length) {
    onClear();
    return;
  }

  const currentScope = sections[i].split(":")[0] as SearchScopeId;
  const existed =
    searchScopes.value.findIndex((item) => item.id === currentScope) >= 0;
  if (!existed) {
    onClear();
    return;
  }

  state.currentScope = currentScope;
};
</script>
