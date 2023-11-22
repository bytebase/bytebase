import { InjectionKey, Ref, inject, provide, ref } from "vue";
import { useDelayedValue } from "@/composables/useDelayedValue";
import { ComposedDatabase } from "@/types";
import {
  ColumnMetadata,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto/v1/database_service";

type UpdateFn = ReturnType<typeof useDelayedValue<any>>["update"];

type Position = {
  x: number;
  y: number;
};

export type HoverState = {
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
  view?: ViewMetadata;
  column?: ColumnMetadata;
};

export type HoverStateContext = {
  state: Ref<HoverState | undefined>;
  position: Ref<Position>;
  update: UpdateFn;
};

export const KEY = Symbol(
  "bb.sql-editor.schema-panel.hover-state"
) as InjectionKey<HoverStateContext>;

export const useHoverStateContext = () => {
  return inject(KEY)!;
};

export const provideHoverStateContext = () => {
  const { value: state, update } = useDelayedValue<HoverState | undefined>(
    undefined,
    {
      delayBefore: 500,
      delayAfter: 500,
    }
  );
  const position = ref<Position>({
    x: 0,
    y: 0,
  });
  const context: HoverStateContext = {
    state,
    position,
    update,
  };

  provide(KEY, context);

  return context;
};
