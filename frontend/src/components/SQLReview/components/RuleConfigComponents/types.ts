export type PayloadValueType = boolean | string | number | string[];

export interface PayloadForEngine {
  [engine: string]: PayloadValueType[];
}
