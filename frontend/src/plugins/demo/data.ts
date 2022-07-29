import { merge } from "lodash-es";
import { validateStepData } from "./guide";
import { GuideData, HintData } from "./types";

const hintCache = new Map<string, HintData[]>();

const fetchJSONData = async (path: string) => {
  const res = await fetch("/demo" + path);
  const data = await res.json();
  return data;
};

export const fetchGuideDataWithName = async (guideName: string) => {
  const recorderData = await fetchJSONData(`/${guideName}/recorder.json`);
  const guideRawData = await fetchJSONData(`/${guideName}/guide.json`);
  const guideData = merge(recorderData, guideRawData) as GuideData;
  guideData.steps = guideData.steps.filter((s) => validateStepData(s));
  return guideData;
};

export const fetchHintDataWithName = async (hintName: string) => {
  if (hintCache.has(hintName)) {
    return hintCache.get(hintName) as HintData[];
  }

  const hintData = (await fetchJSONData(
    `/${hintName}/hint.json`
  )) as HintData[];
  hintCache.set(hintName, hintData);

  return hintData;
};
