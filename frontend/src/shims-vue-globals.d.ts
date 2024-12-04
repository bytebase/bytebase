import type dayjs from "dayjs";
import type { isEmpty } from "lodash-es";
import type {
  humanizeTs,
  humanizeDuration,
  humanizeDurationV1,
  humanizeDate,
  sizeToFit,
  urlfy,
} from "./utils";

export {};

declare module "vue" {
  export interface ComponentCustomProperties {
    window: Window & typeof globalThis;
    console: Console;
    dayjs: typeof dayjs;
    humanizeTs: typeof humanizeTs;
    humanizeDuration: typeof humanizeDuration;
    humanizeDurationV1: typeof humanizeDurationV1;
    humanizeDate: typeof humanizeDate;
    isDev: boolean;
    isRelease: boolean;
    sizeToFit: typeof sizeToFit;
    urlfy: typeof urlfy;
    isEmpty: typeof isEmpty;
  }
}
