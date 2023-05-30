import type { StyleValue } from "vue";

export type VueClass = string | Record<string, unknown> | Array<VueClass>;

export type VueStyle = StyleValue;

export type ExtractPromiseType<T> = T extends Promise<infer U> ? U : unknown;
