import { defineStore } from "pinia";
import { ref } from "vue";

export type PageMode = "BUNDLED" | "STANDALONE";

export const useVendersStore = defineStore("venders", () => {
  const mode = ref<PageMode>("BUNDLED");

  return {
    mode,
  };
});
