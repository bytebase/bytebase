import { useLocalStorage } from "@vueuse/core";
import { NSwitch } from "naive-ui";
import { defineComponent, h } from "vue";
import { useActuatorV1Store } from "@/store";
import { isDev } from "@/utils";

export const SupportedLSPTypes = ["OLD", "NEW"] as const;
export type LSPType = typeof SupportedLSPTypes[number];

export const StoredLSPType = useLocalStorage<LSPType>(
  "bb.sql-editor.dev-lsp-type",
  "OLD",
  {
    serializer: {
      read(raw: LSPType) {
        if (!SupportedLSPTypes.includes(raw)) return "OLD";
        return raw;
      },
      write(value) {
        return value;
      },
    },
  }
);

export const shouldUseNewLSP = () => {
  if (!isDev()) {
    // In release mode, look up the actuator service for CLI flag
    return useActuatorV1Store().serverInfo?.lsp;
  }

  // Use the value from UI switch in dev mode
  return StoredLSPType.value === "NEW";
};

export const LSPTypeSwitch = defineComponent({
  name: "LSPTypeSwitch",
  render() {
    const label = h("span", {}, "[DEV] LSP Type");
    const switcher = h(
      NSwitch,
      {
        text: true,
        value: StoredLSPType.value === "NEW",
        onUpdateValue(on: boolean) {
          StoredLSPType.value = on ? "NEW" : "OLD";
          location.reload();
        },
      },
      {
        checked: () => h("span", { class: "font-medium scale-x-75" }, "NEW"),
        unchecked: () => h("span", { class: "font-medium scale-x-75" }, "OLD"),
      }
    );
    return h(
      "div",
      {
        class: "flex flex-row items-center gap-x-2 justify-between",
        onClick: (e: MouseEvent) => {
          e.stopPropagation();
          e.preventDefault();
        },
      },
      [label, switcher]
    );
  },
});
