import axios from "axios";
import {
  Member,
  MemberId,
  MemberNew,
  MemberPatch,
  MemberState,
  ResourceObject,
  PrincipalId,
  unknown,
} from "../../types";

function convert(member: ResourceObject): Member {
  return {
    ...(member.attributes as Omit<Member, "id">),
    id: member.id,
  };
}

const state: () => MemberState = () => ({
  memberList: [],
});

const getters = {
  memberList: (state: MemberState) => (): Member[] => {
    return state.memberList;
  },
  memberByPrincipalId: (state: MemberState) => (id: PrincipalId): Member => {
    return (
      state.memberList.find((item) => item.principalId == id) ||
      (unknown("ROLE_MAPPING") as Member)
    );
  },
};

const actions = {
  async fetchMemberList({ commit }: any) {
    const memberList = (await axios.get(`/api/Member`)).data.data.map(
      (member: ResourceObject) => {
        return convert(member);
      }
    );

    commit("setMemberList", memberList);
    return memberList;
  },

  // Returns existing member if the principalId has already been created.
  async createdMember({ commit }: any, newMember: MemberNew) {
    const createdMember = convert(
      (
        await axios.post(`/api/Member`, {
          data: {
            type: "member",
            attributes: newMember,
          },
        })
      ).data.data
    );

    commit("appendMember", createdMember);

    return createdMember;
  },

  async patchMember({ commit }: any, member: MemberPatch) {
    const { id, ...attrs } = member;
    const updatedMember = convert(
      (
        await axios.patch(`/api/Member/${member.id}`, {
          data: {
            type: "member",
            attributes: attrs,
          },
        })
      ).data.data
    );

    commit("replaceMemberInList", updatedMember);

    return updatedMember;
  },

  async deleteMemberById(
    { state, commit }: { state: MemberState; commit: any },
    id: MemberId
  ) {
    await axios.delete(`/api/Member/${id}`);

    const newList = state.memberList.filter((item: Member) => {
      return item.id != id;
    });

    commit("setMemberList", newList);
  },
};

const mutations = {
  setMemberList(state: MemberState, memberList: Member[]) {
    state.memberList = memberList;
  },

  appendMember(state: MemberState, newMember: Member) {
    state.memberList.push(newMember);
  },

  replaceMemberInList(state: MemberState, updatedMember: Member) {
    const i = state.memberList.findIndex(
      (item: Member) => item.id == updatedMember.id
    );
    if (i != -1) {
      state.memberList[i] = updatedMember;
    }
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
