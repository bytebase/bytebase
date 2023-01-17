import { inject, InjectionKey, provide, unref, watchEffect } from "vue";
import type { MaybeRef } from "@/types";
import type { Geometry, SchemaDiagramContext } from "../types";

export const KEY = Symbol(
  "bb.schema-diagram"
) as InjectionKey<SchemaDiagramContext>;

export const useSchemaDiagramContext = () => {
  return inject(KEY)!;
};

export const provideSchemaDiagramContext = (context: SchemaDiagramContext) => {
  provide(KEY, context);
};

export const useGeometry = (geometry: MaybeRef<Geometry>) => {
  const context = useSchemaDiagramContext();
  watchEffect((onCleanup) => {
    const g = unref(geometry);
    context.geometries.value.add(g);
    onCleanup(() => {
      context.geometries.value.delete(g);
    });
  });
};
