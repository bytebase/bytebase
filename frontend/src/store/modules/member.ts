import { defineStore, storeToRefs } from "pinia";
import axios from "axios";
import { watchEffect } from "vue";
import {
  Member,
  MemberId,
  MemberCreate,
  MemberPatch,
  MemberState,
  ResourceObject,
  Principal,
  PrincipalId,
  unknown,
  empty,
  EMPTY_ID,
  RoleType,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";

function convert(
  member: ResourceObject,
  includedList: ResourceObject[]
): Member {
  const principal = getPrincipalFromIncludedList(
    member.relationships!.principal.data,
    includedList
  );
  principal.role = member.attributes.role as RoleType;

  return {
    ...(member.attributes as Omit<Member, "id" | "principal">),
    id: parseInt(member.id),
    principal: principal,
  };
}

export const useMemberStore = defineStore("member", {
  state: (): MemberState => ({
    memberList: [],
  }),
  actions: {
    memberByPrincipalId(id: PrincipalId): Member {
      if (id == EMPTY_ID) {
        return empty("MEMBER") as Member;
      }

      return (
        this.memberList.find((item) => item.principal.id == id) ||
        (unknown("MEMBER") as Member)
      );
    },
    memberByEmail(email: string): Member {
      return (
        this.memberList.find((item) => item.principal.email == email) ||
        (unknown("MEMBER") as Member)
      );
    },
    setMemberList(memberList: Member[]) {
      this.memberList = memberList;
    },
    updatePrincipal(memberId: MemberId, principal: Principal) {
      const index = this.memberList.findIndex((m) => m.id === memberId);
      if (index >= 0) {
        this.memberList = [
          ...this.memberList.slice(0, index),
          {
            ...this.memberList[index],
            principal: principal,
          },
          ...this.memberList.slice(index + 1),
        ];
      }
    },
    appendMember(newMember: Member) {
      this.memberList.push(newMember);
    },
    replaceMemberInList(updatedMember: Member) {
      const i = this.memberList.findIndex(
        (item: Member) => item.id == updatedMember.id
      );
      if (i !== -1) {
        this.memberList[i] = updatedMember;
      }
    },
    async fetchMemberList() {
      const data = (await axios.get(`/api/member`)).data;
      const memberList: Member[] = data.data.map((member: ResourceObject) => {
        return convert(member, data.included);
      });

      // sort the member list
      memberList.sort((a: Member, b: Member) => {
        return a.id - b.id;
      });

      this.setMemberList(memberList);
      return memberList;
    },
    // Returns existing member if the principalId has already been created.
    async createdMember(newMember: MemberCreate) {
      const data = (
        await axios.post(`/api/member`, {
          data: {
            type: "MemberCreate",
            attributes: newMember,
          },
        })
      ).data;
      const createdMember = convert(data.data, data.included);

      this.appendMember(createdMember);

      return createdMember;
    },
    async patchMember({
      id,
      memberPatch,
    }: {
      id: MemberId;
      memberPatch: MemberPatch;
    }) {
      const data = (
        await axios.patch(`/api/member/${id}`, {
          data: {
            type: "memberPatch",
            attributes: memberPatch,
          },
        })
      ).data;
      const updatedMember = convert(data.data, data.included);

      this.replaceMemberInList(updatedMember);

      return updatedMember;
    },
    async deleteMemberById(id: MemberId) {
      await axios.delete(`/api/member/${id}`);

      const newList = this.memberList.filter((item: Member) => {
        return item.id != id;
      });

      this.setMemberList(newList);
    },
  },
});

export const useMemberList = () => {
  const store = useMemberStore();
  watchEffect(() => store.fetchMemberList());

  return storeToRefs(store).memberList;
};
