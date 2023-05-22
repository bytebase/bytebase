import slug from "slug";

import { UNKNOWN_ID } from "@/types";
import { Instance } from "@/types/proto/v1/instance_service";
import { State } from "@/types/proto/v1/common";

export const UNKNOWN_INSTANCE_NAME = `instances/${UNKNOWN_ID}`;

export const instanceV1Slug = (instance: Instance) => {
  return [slug(instance.environment), slug(instance.title), instance.uid].join(
    "-"
  );
};

export function instanceV1Name(instance: Instance) {
  let name = instance.title;
  if (instance.state === State.DELETED) {
    name += " (Archived)";
  }
  return name;
}
