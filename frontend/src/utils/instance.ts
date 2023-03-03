import { computed, unref } from "vue";
import {
  Environment,
  Instance,
  Language,
  languageOfEngine,
  MaybeRef,
} from "../types";

export function instanceName(instance: Instance) {
  let name = instance.name;
  if (instance.rowStatus == "ARCHIVED") {
    name += " (Archived)";
  }
  return name;
}

// Sort the list to put prod items first.
export function sortInstanceList(
  list: Instance[],
  environmentList: Environment[]
): Instance[] {
  return list.sort((a: Instance, b: Instance) => {
    let aEnvIndex = -1;
    let bEnvIndex = -1;

    for (let i = 0; i < environmentList.length; i++) {
      if (environmentList[i].id == a.environment.id) {
        aEnvIndex = i;
      }
      if (environmentList[i].id == b.environment.id) {
        bEnvIndex = i;
      }

      if (aEnvIndex != -1 && bEnvIndex != -1) {
        break;
      }
    }
    return bEnvIndex - aEnvIndex;
  });
}

export const useInstanceEditorLanguage = (
  instance: MaybeRef<Instance | undefined>
) => {
  return computed((): Language => {
    return languageOfEngine(unref(instance)?.engine);
  });
};

export const isValidSpannerHost = (host: string) => {
  const RE =
    /^projects\/(?<PROJECT_ID>(?:[a-z]|[-.:]|[0-9])+)\/instances\/(?<INSTANCE_ID>(?:[a-z]|[-]|[0-9])+)$/;
  return RE.test(host);
};
