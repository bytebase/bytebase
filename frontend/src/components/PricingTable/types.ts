import { Plan } from "@/types";

export interface LocalPlan extends Plan {
  label: string;
  image: string;
  buttonText: string;
  highlight: boolean;
  isAvailable: boolean;
  pricing: string;
  priceSuffix: string;
  canTrial: boolean;
}
