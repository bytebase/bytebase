import type dayjs from "dayjs";
import type { Composer } from "vue-i18n";
import type { humanizeTs, humanizeDurationV1, humanizeDate } from "./utils";

export {};

declare module "vue" {
  export interface ComponentCustomProperties {
    window: Window & typeof globalThis;
    console: Console;
    dayjs: typeof dayjs;
    humanizeTs: typeof humanizeTs;
    humanizeDurationV1: typeof humanizeDurationV1;
    humanizeDate: typeof humanizeDate;
    isDev: boolean;
    isRelease: boolean;
    $t: Composer["t"];
    $te: Composer["te"];
    $tm: Composer["tm"];
  }
}
