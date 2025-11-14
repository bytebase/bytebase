import type { ButtonProps } from "naive-ui";
import { computed, type MaybeRef, unref } from "vue";

export const useButton = (options: {
  active: MaybeRef<boolean>;
  disabled: MaybeRef<boolean>;
}) => {
  const props = computed(() => {
    const props: ButtonProps = {
      tag: "div",
      disabled: unref(options.disabled),
      size: "small",
    };
    if (unref(options.active)) {
      props.secondary = true;
      props.type = "primary";
    } else {
      props.quaternary = true;
      props.type = "default";
    }
    props.disabled = unref(options.disabled);

    return props;
  });
  const style = computed(() => {
    const parts: string[] = [];

    parts.push("--n-height: 32px");
    parts.push("--n-padding: 4px 8px");
    parts.push("--n-icon-size: 16px");
    return parts.join("; ");
  });
  return { props, style };
};
