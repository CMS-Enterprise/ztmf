# Generate Excel sheets for Zero Trust data call

1. Standard Python setup: `python3 -m venv venv && source venv/bin/activate && pip install -r requirements.txt`
2. Download the Typeform results from Google Sheets and put them in this directory as `in.csv`
3. `python3 ./mk-sheets.py`

This should create a number of Excel files in out/.
If you are on the VPN, you made need to turn it off to make pip work
