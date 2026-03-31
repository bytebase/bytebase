<template>
  <Drawer
    :show="active"
    class="w-96! max-w-full"
    :close-on-esc="true"
    @update:show="(show: boolean) => !show && onClose()"
  >
    <DrawerContent
      class="w-full"
      :title="state.frontmatter.title"
      :closable="true"
    >
      <!-- eslint-disable-next-line vue/no-v-html -->
      <div v-if="state.html" class="overflow-auto" v-html="state.html"></div>
      <template #footer>
        <div class="flex flex-col items-center gap-y-4 pb-10">
          <NButton
            v-if="isGuide"
            text
            size="small"
            @click="onDismissAll"
          >
            {{ $t("help-drawer.dont-show-again") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import type { Node, Tag } from "@markdoc/markdoc";
import { NButton } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { Drawer, DrawerContent } from "@/components/v2";
import { useLanguage } from "@/composables/useLanguage";
import { useHelpStore, useUIStateStore } from "@/store";
import type { RouteMapList } from "@/types";

const [
  { default: Markdoc },
  { markdocConfig, DOMPurifyConfig },
  { default: yaml },
  { default: DOMPurify },
] = await Promise.all([
  import("@markdoc/markdoc"),
  import("./config"),
  import("js-yaml"),
  import("dompurify"),
]);

interface State {
  frontmatter: Record<string, string>;
  html: string;
  helpTimer: number | undefined;
  RouteMapList: RouteMapList | null;
}

const active = ref(false);
const { locale } = useLanguage();
const uiStateStore = useUIStateStore();
const helpStore = useHelpStore();
const helpStoreState = storeToRefs(helpStore);
const route = useRoute();

const helpId = computed(() => helpStoreState.currHelpId.value);
const isGuide = computed(() => helpStoreState.openByDefault.value);

const state = reactive<State>({
  frontmatter: {},
  html: "",
  helpTimer: undefined,
  RouteMapList: null,
});

// watch route change for help
watch(
  () => route.name,
  async (routeName) => {
    const uiStateStore = useUIStateStore();
    const helpStore = useHelpStore();

    // Clear timer after every route change.
    if (state.helpTimer) {
      clearTimeout(state.helpTimer);
      state.helpTimer = undefined;
    }

    // Hide opened help drawer if route changed.
    helpStore.exitHelp();

    if (!state.RouteMapList) {
      const res = await fetch("/help/routeMapList.json");
      state.RouteMapList = await res.json();
    }

    const helpId = state.RouteMapList?.find(
      (pair) => pair.routeName === routeName
    )?.helpName;

    if (
      helpId &&
      !uiStateStore.allHelpDrawersDismissed &&
      !uiStateStore.getIntroStateByKey(`${helpId}`)
    ) {
      state.helpTimer = window.setTimeout(() => {
        helpStore.showHelp(helpId, true);
        uiStateStore.saveIntroStateByKey({
          key: `${helpId}`,
          newState: true,
        });
      }, 500);
    }
  }
);

watch(helpId, async (id) => {
  if (id) {
    const res = await fetch(
      `/help/${
        locale.value === "zh-CN" ? "zh" : locale.value === "ja-JP" ? "ja" : "en"
      }/${id}.md`
    );
    const markdown = await res.text();
    const ast: Node = Markdoc.parse(markdown);
    const content = Markdoc.transform(ast, markdocConfig) as Tag;

    content.attributes.class = "markdown-body"; // style help content
    const html: string = Markdoc.renderers.html(content);

    state.frontmatter = ast.attributes.frontmatter
      ? (yaml.load(ast.attributes.frontmatter) as Record<string, string>)
      : {};
    state.html = DOMPurify.sanitize(html, DOMPurifyConfig);
    activate();
  } else {
    state.frontmatter = {};
    state.html = "";
    deactivate();
  }
});

const onDismissAll = () => {
  uiStateStore.allHelpDrawersDismissed = true;
  helpStore.exitHelp();
};

const onClose = () => {
  if (isGuide.value) {
    if (!uiStateStore.getIntroStateByKey(`${helpId.value}`)) {
      uiStateStore.saveIntroStateByKey({
        key: `${helpId.value}`,
        newState: true,
      });
    }
  }
  helpStore.exitHelp();
};

const activate = () => (active.value = true);

const deactivate = () => (active.value = false);
</script>
