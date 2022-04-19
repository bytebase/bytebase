import { defineStore } from "pinia";
import axios from "axios";
import {
  PrincipalId,
  Principal,
  PrincipalCreate,
  PrincipalPatch,
  PrincipalState,
  ResourceObject,
  unknown,
  empty,
  EMPTY_ID,
  PrincipalType,
  ResourceIdentifier,
  RoleType,
} from "@/types";
import { randomString } from "@/utils";
import { useAuthStore } from "./auth";

function convert(principal: ResourceObject): Principal {
  return {
    id: parseInt(principal.id),
    creatorId: principal.attributes.creatorId as PrincipalId,
    createdTs: principal.attributes.createdTs as number,
    updaterId: principal.attributes.updaterId as PrincipalId,
    updatedTs: principal.attributes.updatedTs as number,
    type: principal.attributes.type as PrincipalType,
    name: principal.attributes.name as string,
    email: principal.attributes.email as string,
    role: principal.attributes.role as RoleType,
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

export const usePrincipalStore = defineStore("principal", {
  state: (): PrincipalState => ({
    principalList: [],
  }),
  actions: {
    convert(principal: ResourceObject): Principal {
      return convert(principal);
    },
    principalById(principalId: PrincipalId): Principal {
      if (principalId == EMPTY_ID) {
        return empty("PRINCIPAL") as Principal;
      }

      return (
        this.principalList.find((item) => item.id == principalId) ||
        (unknown("PRINCIPAL") as Principal)
      );
    },
    setPrincipalList(principalList: Principal[]) {
      this.principalList = principalList;
    },
    appendPrincipal(newPrincipal: Principal) {
      this.principalList.push(newPrincipal);
    },
    upsertPrincipalInList(updatedPrincipal: Principal) {
      const i = this.principalList.findIndex(
        (item: Principal) => item.id == updatedPrincipal.id
      );
      if (i == -1) {
        this.principalList.push(updatedPrincipal);
      } else {
        this.principalList[i] = updatedPrincipal;
      }
    },
    async fetchPrincipalList() {
      const userList: ResourceObject[] = (await axios.get(`/api/principal`))
        .data.data;
      const principalList = userList.map((user) => {
        return convert(user);
      });

      this.setPrincipalList(principalList);

      return principalList;
    },
    async fetchPrincipalById(principalId: PrincipalId) {
      const principal = convert(
        (await axios.get(`/api/principal/${principalId}`)).data.data
      );

      this.upsertPrincipalInList(principal);

      return principal;
    },
    // Returns existing user if already created.
    async createPrincipal(newPrincipal: PrincipalCreate) {
      const createdPrincipal = convert(
        (
          await axios.post(`/api/principal`, {
            data: {
              type: "PrincipalCreate",
              attributes: {
                name: newPrincipal.name,
                email: newPrincipal.email,
                password: randomString(),
              },
            },
          })
        ).data.data
      );

      this.appendPrincipal(createdPrincipal);

      return createdPrincipal;
    },
    async patchPrincipal({
      principalId,
      principalPatch,
    }: {
      principalId: PrincipalId;
      principalPatch: PrincipalPatch;
    }) {
      const updatedPrincipal = convert(
        (
          await axios.patch(`/api/principal/${principalId}`, {
            data: {
              type: "principalPatch",
              attributes: principalPatch,
            },
          })
        ).data.data
      );

      this.upsertPrincipalInList(updatedPrincipal);

      useAuthStore().refreshUserIfNeeded(updatedPrincipal.id);

      return updatedPrincipal;
    },
  },
});
