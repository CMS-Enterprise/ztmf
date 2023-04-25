This is the Google AppScript automation for generating the resulting scores and reports for ADO responses to the CMS Cloud ZTMM data call.

Dependencies:
	- Source Google Sheets results for data call
	- Google Drive space
	- Google Sheets report template

The source Google sheets results for the data call is an output of the Typeform survey created for this data call. Tabs within this document include a listing of all questions and their respective full text answers, a list of all questions with the score for each answer, and a list of each pillar and functions mapped to the question number.

Associating this script with the sheets file within a google workspace will add a menu bar option named 'Score & Report'. Under this menu will be the options for each step within the process of creating reports.

First, the 'Tabulate' function. This function takes the output from the TypeForm data call, compares each answer to the scoresheet, and outputs the corresponding score for each question into another sheet. In this output sheet the first 5 columns are copied directly as they are the submitter's association and contact information.

Next the 'vizTabulate' function. This function serves to slice out the scores for foundational categories (Visibility & Analytics, and Governance) into their individual pillar associated scores. Using the same process as the 'Tabulate' function, pillar associated question scores are appended to each row of the results sheet.

The 'Pillar_Score' function should be run next. This summarizes each response into scores by pillar into another new sheet. Scores are calculated using the data from the scores output and averaged based on the number of answered questions per pillar. This ensures that skipped or unknown questions are not factored into the scoring.

Once all this supporting information is calculated and stored the 'Generate_Slides' function is run. This function creates a copy of the report slides template. Then all of the relevant information for each entry is generated and populated. Submission information is copied from the score results sheet into the slides. A radar chart of the associated pillar scores is generated within sheets and then copied to the slides presentation. Then a score for each function within a pillar is calculated using the question mapping reference sheet and averaged based on how many questions were answered. This is copied into the slides report and then the raw scores are replaced with their respective level in the Zero Trust Maturity Model (T,A,O). Finally this renames the completed slides file to the name of the submitter's program.

The 'Generate_Slides' function will create reports for every entry row in the results.
