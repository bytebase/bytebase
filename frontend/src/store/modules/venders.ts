import { defineStore } from "pinia";
import { ref } from "vue";

export type PageMode =
  // General mode. Console is full-featured and SQL Editor is bundled in the layout.
  | "BUNDLED"
  // Vender customized mode. Hide certain parts (e.g., headers, sidebars) and
  // some features are disabled or hidden.
  | "STANDALONE";

export const useVendersStore = defineStore("venders", () => {
  const mode = ref<PageMode>("BUNDLED");

  return {
    mode,
  };
});
