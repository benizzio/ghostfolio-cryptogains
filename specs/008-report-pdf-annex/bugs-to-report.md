# Bugs found to report using bugfix

## [ ] 1 - Markdown: general "Ghostfolio Capital Gains And Losses Report" summary section information classifier labels are not bold

Actual general summary section was
```markdown
- Year: 2025
- Cost Basis Method: Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order
- Generated At: 2026-07-05 17:36:46 CEST
- Report Calculation Currency: EUR
```

Expected general summary section was
```markdown
- **Year:** 2025
- **Cost Basis Method:** Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order
- **Generated At:** 2026-07-05 17:36:46 CEST
- **Report Calculation Currency:** EUR
```

## [ ] 2 - PDF: Document is illegible because it is being printed in Markdown format inside the PDF

The document seems to have the correct data and structure, but it is basically a Markdown syntax document inside a PDF, making it illegible to humans because the Markdown code is printed without any interpretation.
When printing to PDF, we need to format the data in a way it is human legible when generated. Re-assess research and verify if the current option of PDF generation library can format the text correctly in a way that fits the A4 pages. 
If viable and interpreted by the library, use HTML to properly format. If not, verify other possibilities.