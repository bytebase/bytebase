<template>
  <div class="hidden md:flex justify-center items-center border m-4">
    <img class="m-4 h-72" src="../assets/illustration/guide.webp" alt="" />
    <div class="m-4">
      <h2 class="text-xl text-main" v-html="content"></h2>
      <div class="mt-4">
        <button class="btn-normal" @click.prevent="dismiss">
          {{ $t("common.dismiss") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { useStore } from "vuex";
import { useI18n } from "vue-i18n";
import { formatStringTemplate } from "../utils/util"

export default {
  name: "IntroBanner",
  components: {},
  setup() {
    const { t } = useI18n();
    const store = useStore();
    const dismiss = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "general.overview",
        newState: true,
      });
    };
    return {
      dismiss,
      content: formatStringTemplate(
        t("intro.content"),
        `<span class="text-accent">${t("intro.quickstart")}</span>`,
        `<a class="normal-link" href="https://docs.bytebase.com" target="__blank">
          ${t("intro.doc")}
        </a>`,
        `<a
          class="normal-link"
          href="https://github.com/bytebase/bytebase/issues"
          target="__blank"
          >${t("intro.issue")}</a
        >`
      )
    };
  },
};
</script>
