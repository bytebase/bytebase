import { merge } from "lodash-es";

const fetchJSONData = async (path: string) => {
  const res = await fetch("/static/guide" + path);
  const data = await res.json();
  return data;
};

export const fetchGuideDataWithName = async (guideName: string) => {
  const recorderData = await fetchJSONData(`/recorder/${guideName}.json`);
  const guideData = await fetchJSONData(`/${guideName}.json`);

  return merge(recorderData, guideData);
};
