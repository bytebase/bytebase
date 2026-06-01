import { useAppStore } from "@/react/stores/app";
import { CommonText } from "./CommonText";

type Props = {
  readonly database: string;
  readonly schema?: string;
  readonly view: string;
};

/** Replaces `HoverPanel/ViewInfo.vue`. Just the view's comment, wrapped. */
export function ViewInfo({ database, schema, view }: Props) {
  const viewMetadata = useAppStore((s) =>
    s.getViewMetadata({ database, schema, view })
  );
  return <CommonText content={viewMetadata.comment} />;
}
