import dingtalkIcon from "@/assets/im/dingtalk.png";
import discordIcon from "@/assets/im/discord.svg";
import feishuIcon from "@/assets/im/feishu.webp";
import slackIcon from "@/assets/im/slack.png";
import teamsIcon from "@/assets/im/teams.svg";
import wecomIcon from "@/assets/im/wecom.png";
import { WebhookType } from "@/types/proto-es/v1/common_pb";

const iconMap: Record<number, string> = {
  [WebhookType.SLACK]: slackIcon,
  [WebhookType.DISCORD]: discordIcon,
  [WebhookType.TEAMS]: teamsIcon,
  [WebhookType.DINGTALK]: dingtalkIcon,
  [WebhookType.FEISHU]: feishuIcon,
  [WebhookType.LARK]: feishuIcon,
  [WebhookType.WECOM]: wecomIcon,
};

export function WebhookTypeIcon({
  type,
  className,
}: {
  type: WebhookType;
  className?: string;
}) {
  const src = iconMap[type];
  if (!src) return null;
  return <img src={src} alt="" className={className} />;
}
