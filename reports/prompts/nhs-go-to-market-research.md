# Prompt: NHS go-to-market research for Poised

Paste everything below the line into a new Claude Code session opened in the `poised` repo.

---

You're the research lead for a new go-to-market team. Your job is to produce a research report, and a plan built on it, for selling **Poised** to NHS trusts and hospitals in England (and later the rest of the UK). The founder, Rahul, wants Poised to **beat every competitor on price and features**. He needs to know:
- what he has to have in place before any trust can buy;
- how NHS organisations actually buy software like this;
- who to approach, how to find them, and what to say.

Treat this as setting up a real sales function from nothing: research first, then the plan.

## Context: read these first

- `CLAUDE.md`: what Poised is, the architecture and the hosting.
- `docs/saas-requirements.md`: the product requirements. §0 is the decision log; OPS-01 to OPS-07 cover hosting, the DSPT, clinical safety and pen tests.
- `docs/requirements.md`: the feature list.
- `docs/plans/README.md`: the roadmap and what's built today.
- `reports/Health data residency storage vs processing.md`: earlier research, as an example of the expected quality.

**In short:** Poised is a perioperative SaaS covering pre-operative assessment and post-operative follow-up (PROMs). It has:
- a clinical questionnaire authoring tool;
- patient questionnaires on the phone (magic link plus date of birth);
- clinician validation of each question set;
- a POA (pre-operative assessment) summary;
- scores, an ASA suggestion and auto-triage;
- reminders, PROMs, an audit log and multi-trust tenancy.

It runs serverless on AWS London, with UK data kept in the UK, and it's built so new countries can be added later. Today it's a working local prototype: no customers, no certifications, no real patient data. Rahul knows the incumbent product well. Treat it as a competitor to research, never as a template.

## Before you start: ask Rahul these questions

Ask them together. Give a recommended answer with each, then carry on with your best assumption for any question he skips.
1. **Company:** is there a UK limited company yet? Who are the founders, and how many people are on the team?
2. **Budget and runway** for certification, insurance and sales tools over the first 12 months.
3. **Links to any incumbent.** Is Rahul employed or contracted by a competitor, or was he recently? Is there a non-compete, an IP assignment or a confidentiality clause? Did any questionnaire content come from a competitor's product? This decides whether he can sell at all, so treat it as a blocker.
4. **First target:** NHS England only, or the devolved nations too? Acute trusts only, or independent-sector providers (Spire, Circle, Nuffield) as well? These often buy faster.

## What to research

Use primary sources wherever they exist: gov.uk, NHS England, MHRA, ICO, NHS Digital, the procurement portals, and the competitors' own sites and G-Cloud listings. Give every claim:
- a link;
- the date you checked it;
- a confidence rating: **Confirmed** (primary source), **Likely** (secondary sources agree) or **Unverified**.

NHS structures and rules changed a lot in 2025–26, including the plan to abolish NHS England and fold it into DHSC, ICB cuts, the 10 Year Health Plan and the Procurement Act 2023. So check the **current** state of each rule and never rely on memory. If the anthropic-skills:deep-research skill is available, use it, with subagents running the sections below in parallel.

### 1. Market and competitors
- **How pre-op assessment works in NHS trusts today:** paper forms, phone calls, EPR modules, dedicated software. Cover the national drivers too: elective recovery targets, GIRFT, CPOC and RCoA guidance, surgical hubs, the NHS App's role in the 10 Year Health Plan, and the Federated Data Platform.
- **Market size:** the number of acute trusts and independent-sector providers, surgical volumes, and how much is already digital.
- **Competitors:**
  - dedicated pre-op and perioperative products (for example the incumbent Rahul knows, Ultramed, Synopsis, Cievert; check these and find the rest);
  - patient-engagement platforms with pre-op modules (for example DrDoctor, Isla, Patients Know Best);
  - EPR-native modules (Epic, Oracle Health, System C, Nervecentre and others).

  For each one, record: the product, its customers (named trusts where public), its certifications (DTAC, DCB0129, MHRA registration), its integrations, its pricing model and published prices, and its weak spots (reviews, case studies, job ads, board papers).
- **A feature matrix** comparing Poised (built and planned) with the top 5–8 competitors, plus a list of features that would win deals and that no one does well yet.

### 2. Pricing: how to beat everyone
- **What trusts pay today.** Sources:
  - competitors' G-Cloud pricing documents (Digital Marketplace);
  - contract awards on Contracts Finder and Find a Tender;
  - existing FOI responses on WhatDoTheyKnow;
  - trust board papers and annual reports.
- **Common pricing models:** per patient, per episode, per bed, a flat annual licence, or a set-up fee plus a subscription. How do trusts budget for each (capital or revenue), and which is easiest to approve?
- **A recommended pricing model and price points** that undercut the market, with unit economics. Our cost base is serverless and close to £0 when idle; estimate the AWS cost per trust, plus support, certification and insurance overheads.
- **Under the Procurement Act 2023, trusts pick the "most advantageous tender", not the cheapest.** Say how far a low price actually helps, and what else wins: clinical safety evidence, integrations, implementation support and references.
- **Pilot options:** free or discounted pilots, gain-share, outcome-based pricing (cancelled operations avoided, nurse hours saved), and how other NHS start-ups structured their first deals.

### 3. What must be in place before a trust can buy
For each item, give what it is, whether it's mandatory or expected, who checks it, the cost range (GBP), how long it takes, and the order it has to happen in. Cover at least the following, and add anything that's missing:
- **Clinical safety:**
  - DCB0129 (our side): a Clinical Safety Officer, a clinical risk management file and a hazard log;
  - DCB0160 (the trust's side): what we must give trusts to support it.
- **Medical device status (MHRA):** does auto-triage, scoring or an ASA suggestion make Poised software as a medical device under UK MDR 2002? Which class? What would registration, UKCA/CE marking and post-market surveillance need? Which features could be designed to stay outside the definition, and what would that cost the product? Check the MHRA's registration database (PARD) to see how competitors are registered.
- **NHS digital assurance:** DTAC (or its current successor) and the DSPT.
- **Security:** Cyber Essentials Plus, ISO 27001 (needed or just nice to have?) and pen testing (CREST/CHECK).
- **Information governance:** UK GDPR, the DPIA template, the data processing agreement (the trust is the controller and we're the processor), ICO registration, and data residency. Also cover changes from the Data (Use and Access) Act 2025.
- **Accessibility:** WCAG 2.2 AA, the accessibility statement and the NHS service standard.
- **Interoperability that buyers expect:** NHS login, PDS/Spine (NHS number lookup), FHIR/HL7 interfaces to EPRs, NHS App integration, and the IM1/GP Connect equivalents if they apply.
- **Commercial and legal:**
  - company set-up;
  - professional indemnity, cyber and product liability insurance, at the levels trusts ask for;
  - standard NHS terms (for example the NHS Standard Contract and the G-Cloud call-off);
  - supplier onboarding: due diligence and financial checks for a young company.
- **Licensing of clinical content:**
  - scored instruments such as STOP-Bang;
  - PROMs such as EQ-5D and the Oxford Hip and Knee Scores, which are licensed;
  - any content that came from a competitor.
- **Founder risk:** employment-contract restrictions and IP ownership, using what Rahul answers in question 3.

The output of this section is a **compliance roadmap**: phases, dependencies, costs, and the minimum set needed to (a) run an unpaid pilot with made-up data, (b) run a pilot with real patients, and (c) sign a paid contract.

### 4. How the NHS buys
- **Routes to market:**
  - G-Cloud (which lot, and when the next round opens);
  - other frameworks (for example HealthTrust Europe, NHS SBS, Crown Commercial Service and NHS Supply Chain);
  - direct awards under the Procurement Act thresholds;
  - pilots funded by innovation money (SBRI Healthcare, Health Innovation Networks, the NHS Innovation Accelerator, elective recovery or frontline digitisation funding).

  For each route, give the effort involved, the time to the first sale, and which one to do first.
- **The buying journey inside a trust:** who starts it, who signs it off, and how long each stage takes. Stages include the clinical champion, digital or IT, IG, clinical safety, procurement, finance, the business case or capital committee, and the ICB.
- **What a trust business case for a product like this has to contain,** and the evidence that makes it easy: time saved, day-of-surgery cancellations avoided, theatre utilisation and patient experience.

### 5. Who to approach and how to find them
- **Buyer and champion roles, ranked by influence:**
  - clinical: anaesthesia clinical directors, POA lead nurses and matrons, theatre managers, surgical division managers;
  - digital: the CCIO, CNIO, CDIO and CIO;
  - management: elective recovery and transformation leads, procurement leads;
  - ICB digital leads.

  Who champions, who signs and who can block?
- **Where to find the organisations:**
  - the NHS ODS data for every trust and site;
  - CQC data;
  - HES or NHS England statistics on elective volumes, for choosing which trusts to target.
- **Where to find the people:**
  - LinkedIn Sales Navigator, Apollo, Cognism and Lusha (compare how well each covers NHS contacts and whether they comply with UK GDPR);
  - HSJ Intelligence or similar NHS org-chart services;
  - trust websites and board papers;
  - NHS Jobs adverts (these often name the pre-op system a trust uses);
  - conference speaker lists.
- **Communities and events:**
  - the Preoperative Association (PAN), CPOC, the RCoA and the Association of Anaesthetists;
  - Digital Health Rewired, HETT and NHS ConfedExpo;
  - the Health Innovation Networks' regional events.

  Include dates and costs for the next 12 months.
- **Outreach rules:**
  - PECR and UK GDPR for B2B email to NHS staff (`nhs.net`) and for LinkedIn messages;
  - the legitimate-interests assessment needed;
  - opt-out handling;
  - NHS anti-bribery and hospitality rules that limit what sales staff can offer.

### 6. The sales function
- **Ideal customer profile and first targets:** score every acute trust in England on surgical volume, current system (FOI, job ads and board papers), digital maturity, ICB funding, and contract end dates. Recommend 20 trusts to start with, and say why.
- **Positioning:** the main message and proof points for each audience (clinicians, digital leads, finance). Include objection handling, especially:
  - "you're a start-up";
  - "we already have it in Epic";
  - "is it a medical device?";
  - "what about data security?"
- **Outreach sequences:** email and LinkedIn templates for each role, a demo script, a one-page leave-behind outline, and a pilot proposal template.
- **Sales stack:**
  - a CRM (HubSpot or Pipedrive; recommend one);
  - a contact-data tool;
  - email sequencing;
  - a tool for tracking FOI requests and tenders.

  Give the cost of each and a recommended minimal stack for one or two people.
- **Team and process:** who to hire, and when (a clinical advisor or CSO first?); pipeline stages and conversion assumptions; and a 30/60/90-day and 12-month plan with measurable targets.

## Deliverables

Save these in `reports/` and don't commit them:
1. **`reports/NHS go-to-market research.md`**, which contains:
   - a one-page executive summary first: the top 10 actions, in order, each with its cost and timeline;
   - sections 1–6;
   - a list of open questions for Rahul.

   Write it in plain British English, with short sections and tables where they help.
2. **`reports/nhs-target-trusts.csv`:** organisation-level data only. Columns: trust, ODS code, ICB, region, surgical volume, current POA system (with its source), known contract end, priority score and notes. **Don't put named individuals' contact details in the repo.** Instead, describe how to pull them into the CRM.
3. **`reports/nhs-outreach-templates.md`:** the sequences, demo script and pilot proposal outline.

## Rules
- **Research only.** Don't contact anyone, submit an FOI, sign up for paid tools, or send anything. Draft FOI wording if it would help, and Rahul will send it.
- **Separate fact from advice.** Where you recommend something, say so, and give options with the recommended one first.
- **Get legal and regulatory points right, and flag them for a professional.** Mark each one "check with a regulatory consultant or solicitor" where that applies, especially MHRA classification and anything to do with the founder's employment.
- Post progress updates as each section finishes.
