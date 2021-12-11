import type {
  databaseSlug,
  dataSourceSlug,
  environmentName,
  environmentSlug,
  humanizeTs,
  instanceName,
  instanceSlug,
  projectName,
  projectSlug,
  sizeToFit,
  urlfy,
} from "./utils";
import type { Moment } from "moment";
import type { isEmpty } from "lodash";

declare module "@vue/runtime-core" {
  export interface ComponentCustomProperties {
    window: Window & typeof globalThis;
    console: Console;
    moment: Moment;
    humanizeTs: typeof humanizeTs;
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
