import type dayjs from "dayjs";
import type { humanizeDate, humanizeTs } from "./utils";

export {};

declare module "vue" {
  export interface ComponentCustomProperties {
    window: Window & typeof globalThis;
    console: Console;
    dayjs: typeof dayjs;
    humanizeTs: typeof humanizeTs;
    humanizeDate: typeof humanizeDate;
    isDev: boolean;
    isRelease: boolean;
  }
}
