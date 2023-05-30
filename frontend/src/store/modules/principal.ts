import {
  Principal,
  ResourceObject,
  empty,
  PrincipalType,
  ResourceIdentifier,
  RoleType,
} from "@/types";

function convert(principal: ResourceObject): Principal {
  return {
    id: parseInt(principal.id),
    type: principal.attributes.type as PrincipalType,
    name: principal.attributes.name as string,
    email: principal.attributes.email as string,
    role: principal.attributes.role as RoleType,
    serviceKey: principal.attributes.serviceKey as string,
  };
}

export function getPrincipalFromIncludedList(
  data:
    | ResourceIdentifier<ResourceObject>
    | ResourceIdentifier<ResourceObject>[]
    | undefined,
  includedList: ResourceObject[]
): Principal {
  if (data == null) {
    return empty("PRINCIPAL") as Principal;
  }
  for (const item of includedList || []) {
    if (item.type != "principal") {
      continue;
    }
    if (item.id == (data as ResourceIdentifier).id) {
      return convert(item);
    }
  }
  return empty("PRINCIPAL") as Principal;
}
