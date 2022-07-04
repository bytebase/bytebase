import { merge } from "lodash-es";
import { validateStepData } from "./guide";
import { GuideData, HintData } from "./types";

const fetchJSONData = async (path: string) => {
  const res = await fetch("/static/demo" + path);
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
  const hintData = (await fetchJSONData(
    `/${hintName}/hint.json`
  )) as HintData[];
  return hintData;
};
