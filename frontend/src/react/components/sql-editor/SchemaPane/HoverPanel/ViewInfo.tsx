import { useVueState } from "@/react/hooks/useVueState";
import { useDBSchemaV1Store } from "@/store";
import { CommonText } from "./CommonText";

type Props = {
  readonly database: string;
  readonly schema?: string;
  readonly view: string;
};

/** Replaces `HoverPanel/ViewInfo.vue`. Just the view's comment, wrapped. */
export function ViewInfo({ database, schema, view }: Props) {
  const dbSchema = useDBSchemaV1Store();
  const viewMetadata = useVueState(() =>
    dbSchema.getViewMetadata({ database, schema, view })
  );
  return <CommonText content={viewMetadata.comment} />;
}
