<template>
  <n-drawer
    v-model:show="active"
    :width="380"
    :placement="placement"
    :on-after-leave="onClose"
  >
    <n-drawer-content
      :title="state.frontmatter.title"
      closable
      :native-scrollbar="false"
    >
      <template #default>
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div v-if="state.html" v-html="state.html"></div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts">
import { defineComponent, ref, reactive, watch, computed } from "vue";
import { NDrawer, NDrawerContent, DrawerPlacement } from "naive-ui";
import Markdoc, { Node, Tag } from "@markdoc/markdoc";
import DOMPurify from "dompurify";
import yaml from "js-yaml";
import { storeToRefs } from "pinia";
import { useLanguage } from "@/composables/useLanguage";
import { useUIStateStore, useHelpStore } from "@/store";

interface State {
  frontmatter: Record<string, string>;
  html: string;
}

export default defineComponent({
  components: { NDrawer, NDrawerContent },
  setup() {
    const active = ref(false);
    const placement = ref<DrawerPlacement>("right");
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
        const content = Markdoc.transform(ast) as Tag;
        content.attributes.class = "prose"; // style help content
        const html: string = Markdoc.renderers.html(content);

        state.frontmatter = ast.attributes.frontmatter
          ? (yaml.load(ast.attributes.frontmatter) as Record<string, string>)
          : {};
        state.html = DOMPurify.sanitize(html);
        activate("right");
      }
    });

    const onClose = () => {
      if (isGuide.value) {
        if (!uiStateStore.getIntroStateByKey(`guide.${helpId.value}`)) {
          uiStateStore.saveIntroStateByKey({
            key: `guide.${helpId.value}`,
            newState: true,
          });
        }
      }
      helpStore.exitHelp();
    };

    const activate = (place: DrawerPlacement) => {
      active.value = true;
      placement.value = place;
    };
    return {
      active,
      placement,
      activate,
      state,
      onClose,
    };
  },
});
</script>
