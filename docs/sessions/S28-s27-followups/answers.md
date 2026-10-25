# S28 answers

Asked by Claude in chat, 25 Oct 2026 (after the S27 run).

1. **When the staff-side join happens:** when staff open **Fill in for the patient** (`behalf.view`), before the form is pre-filled (recommended).
2. **What the staff-side join may do:** it joins **only this record**, and **only to an account that already exists**. It never creates an account, never joins the person's other records (`joinMatching`), and copies no contacts either way. One `account.join` row in Trust A's log names the staff member (recommended).
3. **What "first open only" covers:** the **whole pre-fill runs once per form**, on its first open (this trust's answers and other trusts'). The form records that it was pre-filled, and later opens never read earlier HQs again (recommended).
