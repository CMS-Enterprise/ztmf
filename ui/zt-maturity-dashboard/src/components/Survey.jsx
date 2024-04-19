import { Model } from "survey-core";
import { Survey } from "survey-react-ui";
import "survey-core/defaultV2.min.css";
import * as SurveyTheme from "survey-core/themes";
import { useQuery, gql } from "@apollo/client";
import React, { useState, useEffect } from "react";

// import "./index.css";
// import { json } from "./json";
import {json} from "../surveyJson";
function SurveyComponent() {
  const survey = new Model(json);
  survey.applyTheme(SurveyTheme.PlainLight);
  survey.onComplete.add((sender, options) => {
    console.log(JSON.stringify(sender.data, null, 3));
  });
  // survey.setVaraible("fismaSystems", fismaSystemSet);
  return <Survey model={survey} />;
}

export default SurveyComponent;
