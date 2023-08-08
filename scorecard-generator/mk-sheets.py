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


with open("in.csv") as csvfile:
    reader = csv.DictReader(csvfile)

    for in_row in reader:
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

        number_dot = re.compile(r"\d+\.")
        for out_row in sheet:
            if not out_row[2].value:
                continue

            match = number_dot.match(out_row[2].value)
            if match:
                # look for relevant in columns
                for col_name in in_row:
                    if col_name.startswith(match[0]) or (
                        col_name[0] == "*" and col_name[1:].startswith(match[0])
                    ):
                        value = in_row[col_name]
                        if value != "":
                            if value[0].isdigit() and value[1] == ".":
                                out_row[4].value = int(in_row[col_name][0])
                            else:
                                out_row[5].value = value

        wb.save(f"out/{name}.xlsx")

