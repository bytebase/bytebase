import { Ref } from "vue";

export type SubsetOf<T, S extends T> = S;
export type MaybeRef<T> = Ref<T> | T;

export type ValidatedMessage = {
  type: "warning" | "error";
  message: string;
};
