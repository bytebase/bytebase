<template>
  <div class="hidden md:flex justify-center items-center border m-4">
    <img class="m-4 h-72" src="../assets/illustration/guide.webp" alt="" />
    <div class="m-4">
      <h2 class="text-xl text-main">
        <i18n-t keypath="intro.content">
          <template #quickstart>
            <span class="text-accent">{{ $t("common.quickstart") }}</span>
          </template>
          <template #doc>
            <a
              class="normal-link"
              href="https://bytebase.com/docs"
              target="__blank"
            >
              {{ $t("intro.doc") }}
            </a>
          </template>
          <template #issue>
            <a
              class="normal-link"
              href="https://github.com/bytebase/bytebase/issues"
              target="__blank"
            >
              {{ $t("intro.issue") }}
            </a>
          </template>
        </i18n-t>
      </h2>
      <div class="mt-4">
        <button class="btn-normal" @click.prevent="dismiss">
          {{ $t("common.dismiss") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { useUIStateStore } from "@/store";
import { defineComponent } from "vue";

export default defineComponent({
  name: "IntroBanner",
  components: {},
  setup() {
    const dismiss = () => {
      useUIStateStore().saveIntroStateByKey({
        key: "general.overview",
        newState: true,
      });
    };
    return {
      dismiss,
    };
  },
});
</script>
