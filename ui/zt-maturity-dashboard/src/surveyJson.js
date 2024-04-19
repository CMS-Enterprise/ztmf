



export const json = {
  title: "Zero Trust Maturiy Questionnare",
  logoPosition: "right",
  pages: [
    {
      elements: [
        {
          type: "panel",
          elements: [
            {
              type: "html",
              name: "income_intro",
              html:
                "<article class='intro'>" +
                "<h4 class='intro__heading intro__heading--income title'>Introduction</h4>" +
                "<div class='intro__body wysiwyg'>" +
                "<p>There are 7 sections, one for information about your system, one each for the five pillars," +
                "and then one for the cross-cutting functions as a whole</p>" +
                "<p>Please choose the multiple choice answer that is closest to how your system works. We understand that the answers are not going to exactly match your setups</p>" +
                "<p>All multiple choice questions are required. The explanations are not required, but are highly encouraged because they help give us context. One or two sentences is plenty.</p>" +
                "</div>" +
                "</article>",
            },
          ],
          name: "panel1",
        },
      ],
      name: "Introduction Page",
    },
    {
      name: "Health history",
      elements: [
        {
          type: "panel",
          name: "health-history",
          elements: [
            {
              type: "dropdown",
              name: "car-brand",
              title: "Which is the brand of your car?",
              choices: ['Audi', 'Toyota'],
              // defaultValue: "Audi",
              // showOtherItem: true,
              // showNoneItem: true,
            },
            {
              type: "boolean",
              name: "diabetes",
              startWithNewLine: false,
              title: "Do you have diabetes?",
            },
            {
              type: "boolean",
              name: "high-blood-pressure",
              startWithNewLine: false,
              title: "High blood pressure?",
            },
            {
              type: "boolean",
              name: "high-cholesterol",
              startWithNewLine: false,
              title: "High cholesterol?",
            },
            {
              type: "comment",
              name: "other-health-conditions",
              title: "Do you have other health conditions?",
              maxLength: 300,
            },
          ],
        },
      ],
      title: "Health history",
    },
    {
      name: "Social history",
      elements: [
        {
          type: "panel",
          name: "social-history",
          elements: [
            {
              type: "panel",
              name: "smoking",
              elements: [
                {
                  type: "radiogroup",
                  name: "cigarettes",
                  title: "Do you smoke cigarettes?",
                  choices: [
                    {
                      value: "never",
                      text: "Never",
                    },
                    {
                      value: "yes",
                      text: "Yes",
                    },
                    {
                      value: "quit",
                      text: "Quit",
                    },
                  ],
                },
                {
                  type: "text",
                  name: "packs-a-day",
                  visibleIf: "{cigarettes} = 'yes'",
                  title: "How many packs a day?",
                  inputType: "number",
                  min: 0,
                },
                {
                  type: "panel",
                  name: "smoking-history",
                  elements: [
                    {
                      type: "text",
                      name: "date-quit",
                      title: "Date quit",
                      titleLocation: "left",
                      inputType: "date",
                      maxValueExpression: "today()",
                    },
                    {
                      type: "text",
                      name: "years-smoked",
                      title: "Years smoked",
                      titleLocation: "left",
                      inputType: "number",
                      min: 0,
                    },
                  ],
                  visibleIf: "{cigarettes} = 'quit'",
                },
                {
                  type: "boolean",
                  name: "vape",
                  title: "Do you vape (e-cigarettes)?",
                },
              ],
            },
            {
              type: "panel",
              name: "alcohol-use-history",
              elements: [
                {
                  type: "boolean",
                  name: "alcohol",
                  title: "Do you drink alcohol?",
                },
                {
                  type: "text",
                  name: "drinks-per-week",
                  visibleIf: "{alcohol} = true",
                  title: "How many drinks per week?",
                },
              ],
              startWithNewLine: false,
            },
            {
              type: "panel",
              name: "drug-use-history",
              elements: [
                {
                  type: "checkbox",
                  name: "recreational-drugs",
                  title: "Do you use recreational drugs?",
                  choices: [
                    {
                      value: "rarely",
                      text: "Rarely",
                    },
                    {
                      value: "marijuana",
                      text: "Marijuana",
                    },
                    {
                      value: "cocaine",
                      text: "Cocaine",
                    },
                    {
                      value: "opioids",
                      text: "Opioids",
                    },
                  ],
                  showOtherItem: true,
                  showNoneItem: true,
                  otherPlaceholder: "Please specify... ",
                  noneText: "Never",
                  otherText: "Other",
                  colCount: 3,
                },
                {
                  type: "text",
                  name: "drug-use-times-per-month",
                  visibleIf:
                    "{recreational-drugs} anyof ['rarely', 'marijuana', 'cocaine', 'opioids', 'other']",
                  title: "How many times per month",
                  description:
                    "If you take different types of drugs, please specify the frequency of use for each in a 'drug - # times/month' format.",
                },
              ],
            },
            {
              type: "panel",
              name: "personal-info",
              elements: [
                {
                  type: "dropdown",
                  name: "education",
                  title: "What is your highest level of education completed?",
                  choices: [
                    {
                      value: "high-school",
                      text: "High School",
                    },
                    {
                      value: "trade-school",
                      text: "Trade School",
                    },
                    {
                      value: "college",
                      text: "College",
                    },
                    {
                      value: "post-graduate",
                      text: "Post-graduate degree(s)",
                    },
                  ],
                },
                {
                  type: "dropdown",
                  name: "marital-status",
                  title: "What is your marital status?",
                  choices: [
                    {
                      value: "married",
                      text: "Married",
                    },
                    {
                      value: "partnership",
                      text: "Partnership",
                    },
                    {
                      value: "divorced",
                      text: "Divorced",
                    },
                    {
                      value: "separated",
                      text: "Separated",
                    },
                    {
                      value: "single",
                      text: "Single",
                    },
                    {
                      value: "widow",
                      text: "Widow(er)",
                    },
                  ],
                },
                {
                  type: "panel",
                  name: "sexual-life",
                  elements: [
                    {
                      type: "boolean",
                      name: "sexually-active",
                      title: "Are you sexually active?",
                    },
                    {
                      type: "text",
                      name: "sexual-partners-number",
                      title: "How many sexual partners do you have?",
                      inputType: "number",
                      min: 0,
                    },
                    {
                      type: "radiogroup",
                      name: "sexual-partners-gender",
                      titleLocation: "hidden",
                      choices: [
                        {
                          value: "men",
                          text: "Men",
                        },
                        {
                          value: "women",
                          text: "Women",
                        },
                        {
                          value: "both",
                          text: "Both",
                        },
                      ],
                      colCount: 3,
                    },
                    {
                      type: "radiogroup",
                      name: "contraception",
                      title: "Do you use contraception?",
                      showCommentArea: true,
                      commentText: "If yes, what method?",
                      choices: [
                        {
                          value: "yes",
                          text: "Yes",
                        },
                        {
                          value: "no",
                          text: "No",
                        },
                      ],
                    },
                  ],
                },
              ],
            },
            {
              type: "panel",
              name: "employment-exercises-children",
              elements: [
                {
                  type: "radiogroup",
                  name: "employment",
                  title: "Are you employed?",
                  showCommentArea: true,
                  commentText: "Type of work",
                  choices: [
                    {
                      value: "yes",
                      text: "Yes",
                    },
                    {
                      value: "no",
                      text: "No",
                    },
                    {
                      value: "retired",
                      text: "Retired",
                    },
                  ],
                  colCount: 3,
                },
                {
                  type: "panel",
                  name: "physical-activity",
                  elements: [
                    {
                      type: "radiogroup",
                      name: "do-exercise",
                      title: "Do you exercise?",
                      choices: [
                        {
                          value: "yes",
                          text: "Yes",
                        },
                        {
                          value: "no",
                          text: "No",
                        },
                      ],
                      colCount: 2,
                    },
                    {
                      type: "multipletext",
                      name: "activities",
                      visibleIf: "{do-exercise} = 'yes'",
                      titleLocation: "hidden",
                      items: [
                        {
                          name: "activity-type",
                          title: "Type of activity",
                        },
                        {
                          name: "activity-frequency",
                          title: "How often",
                        },
                        {
                          name: "activity-duration",
                          title: "How long per activity",
                        },
                      ],
                    },
                  ],
                },
                {
                  type: "panel",
                  name: "children",
                  elements: [
                    {
                      type: "boolean",
                      name: "have-children",
                      title: "Do you have children?",
                    },
                    {
                      type: "multipletext",
                      name: "children-ages",
                      visibleIf: "{have-children} = true",
                      titleLocation: "hidden",
                      items: [
                        {
                          name: "children-number",
                          title: "# of children",
                        },
                        {
                          name: "ages",
                          title: "Their ages",
                        },
                      ],
                    },
                  ],
                },
              ],
              startWithNewLine: false,
            },
          ],
        },
      ],
      title: "Social history",
    },
    {
      name: "Surgical history / recent hospitalizations",
      elements: [
        {
          type: "comment",
          name: "surgery-description",
          title: "Date and type of surgery / procedure",
        },
      ],
      title: "Surgical history / recent hospitalizations",
    },
    {
      name: "Family history",
      elements: [
        {
          type: "matrixdynamic",
          name: "family-history",
          titleLocation: "hidden",
          columns: [
            {
              name: "relation",
              title: "Relation",
            },
            {
              name: "health-conditions",
              title: "Health conditions",
            },
            {
              name: "cancer-history",
              title: "Family history of cancer",
            },
          ],
          cellType: "text",
        },
      ],
      title: "Family history",
    },
    {
      name: "Preventive care",
      elements: [
        {
          type: "panel",
          name: "preventive-care",
          elements: [
            {
              type: "panel",
              name: "recent-shots-panel",
              elements: [
                {
                  type: "matrixdropdown",
                  name: "recent-shots",
                  title: "Recent shots from a doctor or pharmacist",
                  columns: [
                    {
                      name: "date",
                      title: "Date",
                    },
                    {
                      name: "place",
                      title: "Place",
                    },
                  ],
                  cellType: "text",
                  rows: [
                    {
                      value: "flu",
                      text: "Flu",
                    },
                    {
                      value: "shingles",
                      text: "Shingles",
                    },
                    {
                      value: "pneumonia",
                      text: "Pneumonia",
                    },
                    {
                      value: "tetanus",
                      text: "Tetanus",
                    },
                    {
                      value: "other",
                      text: "Other",
                    },
                  ],
                },
              ],
            },
            {
              type: "panel",
              name: "recent-tests-panel",
              elements: [
                {
                  type: "matrixdropdown",
                  name: "recent-tests",
                  title: "Recent tests or procedures",
                  columns: [
                    {
                      name: "date",
                      title: "Date",
                    },
                    {
                      name: "place",
                      title: "Place",
                    },
                  ],
                  cellType: "text",
                  rows: [
                    {
                      value: "colonoscopy",
                      text: "Colonoscopy",
                    },
                    {
                      value: "cologuard",
                      text: "Cologuard",
                    },
                    {
                      value: "mammogram",
                      text: "Mammogram",
                    },
                    {
                      value: "pap",
                      text: "PAP",
                    },
                    {
                      value: "other",
                      text: "Other",
                    },
                  ],
                },
              ],
              startWithNewLine: false,
            },
            {
              type: "panel",
              name: "specialists-panel",
              elements: [
                {
                  type: "matrixdynamic",
                  name: "specialists",
                  title: "Specialists",
                  columns: [
                    {
                      name: "provider",
                      title: "Provider's first and last name",
                    },
                    {
                      name: "speciality",
                      title: "Speciality",
                    },
                    {
                      name: "city",
                      title: "Town/City",
                    },
                  ],
                  cellType: "text",
                  rowCount: 1,
                },
              ],
            },
            {
              type: "panel",
              name: "medications-and-allergies",
              elements: [
                {
                  type: "multipletext",
                  name: "medications",
                  title: "Medications",
                  items: [
                    {
                      name: "medication-name",
                      title: "Name",
                    },
                    {
                      name: "medication-dose",
                      title: "Dose",
                    },
                    {
                      name: "medication-times-per-day",
                      title: "Times per day",
                    },
                  ],
                },
                {
                  type: "multipletext",
                  name: "allergies",
                  startWithNewLine: false,
                  title: "Allergies",
                  items: [
                    {
                      name: "allergy-type",
                      title: "Type",
                    },
                    {
                      name: "allergy-reaction",
                      title: "Reaction",
                    },
                  ],
                },
              ],
            },
          ],
        },
      ],
      title: "Preventive care",
    },
  ],
  showQuestionNumbers: "off",
  showTOC: true,
  completeText: "Submit",
  widthMode: "static",
  width: "1200px",
};
