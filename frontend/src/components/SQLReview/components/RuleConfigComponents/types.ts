import { Engine } from "@/types/proto/v1/common";

export type PayloadValueType = boolean | string | number | string[];

export type PayloadForEngine = Map<Engine, PayloadValueType[]>;
