import { skill as databaseChangeSkill } from "./databaseChange";
import { skill as grantPermissionSkill } from "./grantPermission";
import { skill as querySkill } from "./query";

interface Skill {
  name: string;
  description: string;
  content: string;
}

const skills: Skill[] = [querySkill, databaseChangeSkill, grantPermissionSkill];

const skillMap = new Map(skills.map((s) => [s.name, s]));

export interface GetSkillArgs {
  name?: string;
}

export function getSkill(args: GetSkillArgs): string {
  if (!args.name) {
    const list = skills
      .map((s) => `- **${s.name}**: ${s.description}`)
      .join("\n");
    return `Available skills:\n${list}\n\nCall get_skill(name="skill-name") to load a specific skill.`;
  }

  const skill = skillMap.get(args.name);
  if (!skill) {
    const names = skills.map((s) => s.name).join(", ");
    return `Skill "${args.name}" not found. Available skills: ${names}`;
  }

  return skill.content;
}
