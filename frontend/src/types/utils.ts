import type { Ref } from "vue";

export type MaybeRef<T> = Ref<T> | T;

export type ValidatedMessage = {
  type: "warning" | "error";
  message: string;
};

export type PickLiteral<A, B extends A> = B;
