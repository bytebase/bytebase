import type dayjs from "dayjs";
import type { isEmpty } from "lodash-es";
import type {
  databaseSlug,
  dataSourceSlug,
  environmentName,
  environmentSlug,
  humanizeTs,
  humanizeDuration,
  humanizeDurationV1,
  humanizeDate,
  instanceName,
  instanceSlug,
  projectName,
  projectSlug,
  sizeToFit,
  urlfy,
} from "./utils";

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
    environmentName: typeof environmentName;
    environmentSlug: typeof environmentSlug;
    projectName: typeof projectName;
    projectSlug: typeof projectSlug;
    instanceName: typeof instanceName;
    instanceSlug: typeof instanceSlug;
    databaseSlug: typeof databaseSlug;
    dataSourceSlug: typeof dataSourceSlug;
  }
}
