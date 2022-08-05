<template>
  <Drawer
    :show="active"
    :on-close="onClose"
    :title="state.frontmatter.title"
    @close-drawer="active = false"
  >
    <template #body>
      <!-- eslint-disable-next-line vue/no-v-html -->
      <div v-if="state.html" v-html="state.html"></div>
    </template>
  </Drawer>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, watch, computed } from "vue";
import Markdoc, { Node, Tag } from "@markdoc/markdoc";
import DOMPurify from "dompurify";
import yaml from "js-yaml";
import { storeToRefs } from "pinia";
import { useLanguage } from "@/composables/useLanguage";
import { useUIStateStore, useHelpStore } from "@/store";
import Drawer from "@/components/Drawer.vue";
import { markdocConfig, DOMPurifyConfig } from "./config";

interface State {
  frontmatter: Record<string, string>;
  html: string;
}

export default defineComponent({
  components: { Drawer },
  setup() {
    const active = ref(false);
    const { locale } = useLanguage();
    const uiStateStore = useUIStateStore();
    const helpStore = useHelpStore();
    const helpStoreState = storeToRefs(helpStore);
    const helpId = computed(() => helpStoreState.currHelpId.value);
    const isGuide = computed(() => helpStoreState.openByDefault.value);

    const state = reactive<State>({
      frontmatter: {},
      html: "",
    });

    watch(helpId, async (id) => {
      if (id) {
        const res = await fetch(
          `/help/${locale.value === "zh-CN" ? "zh" : "en"}/${id}.md`
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

    return {
      active,
      state,
      onClose,
    };
  },
});
</script>
