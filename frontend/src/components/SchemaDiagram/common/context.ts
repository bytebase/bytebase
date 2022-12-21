import { inject, InjectionKey, provide } from "vue";
import { SchemaDiagramContext } from "../types";

export const KEY = Symbol(
  "bb.schema-diagram"
) as InjectionKey<SchemaDiagramContext>;

export const useSchemaDiagramContext = () => {
  return inject(KEY)!;
};

export const provideSchemaDiagramContext = (context: SchemaDiagramContext) => {
  provide(KEY, context);
};
