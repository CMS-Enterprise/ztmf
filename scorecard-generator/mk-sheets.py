from openpyxl import load_workbook
import re
import random
import csv
import os

if not os.path.exists("out"):
    os.mkdir("out")

syslookup = {}
# import a list of the FISMA systems and some info for use as a lookup table
with open("hhs-list-withpeeps.csv") as csvfile:
    reader = csv.DictReader(csvfile)
    for l in reader:
        # Name,Acronym,UUID,FIPSRating,TLCPhase,BusinessOwner,BusinessEmail,ISSO,ISSOEmail
        l["BusinessEmail"] = l["BusinessEmail"].strip()
        l["ISSOEmail"] = l["ISSOEmail"].strip()
        syslookup[l["Acronym"]] = l

nextaction = {}
# {'\ufeffFunction': 'Authentication ', 'Question': '1. How does the system authenticate user identity? ', 'ID': '1.', 'Pillar': 'Identity', 'Subset': 'IDM/OKTA', 'NextAction': 'Upgrading Okta Classic to Okta Identity Engine; This will enable the use of device signals and improve the availability of FIDO2 and WebAuthn.', 'Cost': '5000000 (sharerd across multiple systems)', 'CompletionDate': 'Q3 FY2024', 'TypeFunds': 'Appropriated Funds', 'BudgetTimeframe': 'FY24 Q1', 'Comments': 'This is already scheduled and funded', '': ''}

with open("AWS-nextsteps.csv") as csvfile:
    reader = csv.DictReader(csvfile)
    for row in reader:
        del row['']
        del row['\ufeffFunction ']
        del row['Pillar']
        key = row.pop('ID')
        subset = row.pop('Subset')
        #print(row)
        if subset == '':
            subset = 'Default'
        if key in nextaction:
            nextaction[key].update({subset: row})
        elif key != '':
            nextaction[key] = {subset: row}

# print(nextaction)

CAPABILITY = 4
DESCRIBE = 5
PLANNED = 6
COST = 7
COMPLETION = 8
FUNDS = 9
TIMEFAME = 10
COMMENTS = 11

#with open("HHS Data Call AWS-Q4FY2023.csv") as csvfile:
with open("in.csv") as csvfile:
    reader = csv.DictReader(csvfile)

    for in_row in reader:
        # in_row is a dictionary where the keys are the questions and the values are the answers
        wb = load_workbook(filename="tpl.xlsx")
        sheet = wb["QUESTIONNAIRE"]

        name = in_row["What is the acronym for the FISMA system you are answering for?"]

        sheet["B2"] = "CMS"
        sheet["B3"] = name
        if name in syslookup:
            sheet["B5"] = syslookup[name].get("BusinessOwner", "none on record")
            sheet["B4"] = syslookup[name].get("ISSO", "none on record")
        else:
            print("No listing found for ", name)

        # find out if the likely IdP is IDM or EUA (there are others, but these are the most common)
        # Do we even need to do this?? Maybe there can be one explanation that will suit both
        idp = in_row["2. *Identity Stores:* What Identity Management Provider(s) do you use for authentication of people who work on the system (developers, admins, etc) and users of the system?"]
        if (idp.lower().find("idm") >= 0 or idp.lower().find("okta") >= 0):
            idp = "IDM/OKTA"
        elif (idp.lower().find("eua") >= 0 or idp.lower().find("ldap") >= 0):
            idp = "EUA"

        number_dot = re.compile(r"\d+\.")
        for out_row in sheet:
            if not out_row[2].value:
                continue

            match = number_dot.match(out_row[2].value)
            if match:
                # look for relevant in columns
                for col_name in in_row:
                    # There are 2 columns in in_row for each out_row question; the first starts with a number,
                    #  the second is just text. This checks that the column name starts with the matching number.
                    #  then it checks if the value is the score or the extra text
                    if col_name.startswith(match[0]) or (
                        col_name[0] == "*" and col_name[1:].startswith(match[0])
                    ):
                        value = in_row[col_name]
                        if value != "":
                            if value[0].isdigit() and value[1] == ".":
                                out_row[CAPABILITY].value = int(in_row[col_name][0])
                                if (out_row[4].value < 3):
                                    # add in possible next planned actions
                                    # check if there is a next planned action?  If none, state that and move on
                                    if match[0] in nextaction:
                                        print("Got it!")
                                        print(nextaction[match[0]])
                                        # next, check the subset -- Default if none
                                        subset = 'Default'
                                        if match[0] in ['1', '3', '4', '5', '6'] and idp in ['EUA', 'IDM/OKTA']:
                                            subset = idp
                                        out_row[PLANNED].value = nextaction[match[0]][subset]["NextAction"]
                                        out_row[COST].value = nextaction[match[0]][subset]["Cost"]
                                        out_row[COMPLETION].value = nextaction[match[0]][subset]['CompletionDate']
                                        out_row[FUNDS].value = nextaction[match[0]][subset]['TypeFunds']
                                        out_row[TIMEFAME].value = nextaction[match[0]][subset]['BudgetTimeframe']
                                        out_row[COMMENTS].value = nextaction[match[0]][subset]['Comments']
                                    else:
                                        level = 'Advanced'
                                        if (out_row[4].value == 4):
                                            level = 'Optimal'
                                        out_row[PLANNED].value = "No planned action at this time because the maturity level is " + level
                                        out_row[COST].value = nextaction[match[0]][subset]['n/a']
                            else:
                                out_row[DESCRIBE].value = value

        # TODO: check the name format, and update this
        wb.save(f"out/CMS-{name}-Q4FY2023.xlsx")

