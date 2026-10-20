# Ireland (Republic of Ireland): health data residency, storage vs processing/access

Scope: hospital and healthcare-provider patient data. Poised is a UK SaaS vendor acting as processor. The design being tested is (1) a hybrid: Postgres and S3 in AWS eu-west-1 (Dublin), app compute in eu-west-2 (London), UK staff doing support; and (2) everything in London. Researched 8 Oct 2026.

How to read the labels:
- **[BINDING LAW]**: statute, regulation, or Commission decision.
- **[REGULATOR GUIDANCE]**: EDPB, DPC, or government advice notes. Persuasive or policy, not law.
- **[PROCUREMENT / PRACTICE]**: HSE tenders, privacy notices, and contracts.

For each rule, the scope tag says whether it covers **STORAGE**, **ACCESS**, **PROCESSING** or **TRANSFER**.

---

## Q1. GDPR and the Irish Data Protection Act 2018 (including the s.36(2) Health Research Regulations and s.52): is there any localisation requirement?

### Takeaway
No. No binding Irish or EU data protection law requires hospital patient data to be stored in Ireland or in the EU/EEA.
- GDPR regulates **transfers** to third countries (Chapter V), and a transfer is triggered by making data *available*, including by access. It is not about where the disk sits.
- The Irish DPA 2018 adds health-specific safeguards: who may process health data, confidentiality, access limits, logging and encryption. None of them concerns location.
- The Health Research Regulations (SI 314/2018) contain no location rule either, and they cover research only, not care delivery.

### Cited Findings
- **[BINDING LAW, TRANSFER]** GDPR Art. 44: "Any transfer of personal data which are undergoing processing or are intended for processing after transfer to a third country or to an international organisation shall take place only if, subject to the other provisions of this Regulation, the conditions laid down in this Chapter are complied with by the controller and processor…" — [GDPR, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
  - Chapter V is a *conditions-on-transfer* regime, not a storage-location rule.
- **[BINDING LAW, TRANSFER]** GDPR Art. 45(1): "A transfer of personal data to a third country … may take place where the Commission has decided that the third country … ensures an adequate level of protection. Such a transfer shall not require any specific authorisation." — [GDPR, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
- **[BINDING LAW, TRANSFER/PROCESSING]** GDPR Art. 28(3)(a): the processor must process "only on documented instructions from the controller, including with regard to transfers of personal data to a third country…". The hospital's data processing agreement must therefore authorise the UK processing and UK access. — [GDPR, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
- **[BINDING LAW, within-EU flows]** GDPR Art. 1(3): "The free movement of personal data within the Union shall be neither restricted nor prohibited for reasons connected with the protection of natural persons with regard to the processing of personal data." So an "Ireland-only" (as opposed to EEA) rule could not be based on data protection grounds. — [GDPR, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
- **[BINDING LAW, PROCESSING: who, not where]** DPA 2018 s.52(1): processing of special category data is lawful where necessary "(c) for medical diagnosis, (d) for the provision of medical care, treatment or social care, (e) for the management of health or social care systems and services…". s.52(2) adds: "where it is undertaken by or under the responsibility of— (a) a health practitioner, or (b) a person who in the circumstances owes a duty of confidentiality to the data subject that is equivalent to that which would exist if that person were a health practitioner." There is no location element. — [Data Protection Act 2018, s.52](https://www.irishstatutebook.ie/eli/2018/act/7/section/52/enacted/en/html)
- **[BINDING LAW, ACCESS controls, not location]** DPA 2018 s.36(1) lists "suitable and specific measures". They include:
  - "(b) limitations on access to the personal data undergoing processing within a workplace…";
  - "(e)(i) logging mechanisms to permit verification of whether and by whom the personal data have been consulted…";
  - "(iii) where the processing involves data relating to the health of a data subject, a requirement that the processing is undertaken by a person referred to in section 52(2)";
  - "(iv) pseudonymisation" and "(v) encryption".

  s.36(2) lets regulations add further measures. — [Data Protection Act 2018, s.36](https://www.irishstatutebook.ie/eli/2018/act/7/section/36/enacted/en/html)
- **[BINDING LAW, research only]** The Data Protection Act 2018 (Section 36(2)) (Health Research) Regulations 2018 (SI 314/2018), reg. 3(1), apply to "A controller who is processing or further processing personal data for the purposes of health research". They require governance measures: ethics approval, "specification of any data processors involved", training, DPIA and so on. A full-text search of the SI found **no** reference to transfers, storage location, "outside the State" or third countries. — [SI 314/2018](https://www.irishstatutebook.ie/eli/2018/si/314/made/en/print)
- **[BINDING LAW, non-personal data only]** For context: the EU Free Flow of Non-Personal Data Regulation (EU) 2018/1807 prohibits national data localisation requirements for *non-personal* data, except on public security grounds. It does not cover patient data. — [Regulation (EU) 2018/1807, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2018/1807/oj)
- **[SECONDARY, law-firm/handbook summary]** "Ireland has no data localization or data residency laws that require the retention and storage of data in the local jurisdiction or prohibit the transfer of data out of the jurisdiction." This wording came from a search-engine summary of the Legal 500 Ireland guide and the Baker McKenzie handbook. The Baker McKenzie page returned 404 on fetch, so the exact wording is unverified. — [Legal 500 Ireland: Data Protection & Cybersecurity](https://www.legal500.com/guides/chapter/ireland-data-protection-cybersecurity); [Baker McKenzie Global Data & Cyber Handbook, Ireland](https://resourcehub.bakermckenzie.com/en/resources/global-data-and-cyber-handbook/emea/ireland/topics/data-localization-and-regulation-of-non-personal-data)

### Inferences
- Irish binding law answers (c): **neither storage nor processing must be in Ireland or the EEA**, as long as GDPR transfer conditions are met for any third-country access or processing.
- The health-specific Irish rules (s.36, s.52) govern *who* may process health data and *how* (confidentiality, access limits, logging, encryption). Poised's UK support staff must be bound by confidentiality equivalent to a health practitioner's (s.52(2)(b)). This is a contract and HR control, not a location one.
- The research regulations are irrelevant to routine pre-op care delivery. They would matter only if a hospital used Poised data for health research.

### Gaps
- I did not search for every other Irish statute that might touch health records (for example, the Patient Safety (Licensing) Act, Mental Health Acts, or Medical Council guidance) for location terms. Nothing surfaced, but this is not proof of absence.
- I did not check whether any further s.36(2) regulations (other than SI 314/2018) were made after 2018.

---

## Q2. DPC and EDPB guidance: is remote access by UK staff, or processing in the UK, a "transfer"?

### Takeaway
Yes. EU guidance, which the DPC and HSE follow, treats any access from a third country as a transfer. That includes a UK processor's software reading an EU-hosted database, or UK staff viewing data on a screen for support.
- EDPB Guidelines 05/2021 Example 11 is almost exactly the hybrid design: a third-country processor remotely accessing data stored in the EU "results in transfers".
- Putting the database in Dublin therefore does **not** remove the Chapter V transfer. It only changes where the data rests.

### Cited Findings
- **[REGULATOR GUIDANCE, TRANSFER definition]** The EDPB's three cumulative criteria for a transfer: "1) A controller or a processor ('exporter') is subject to the GDPR for the given processing. 2) The exporter discloses by transmission or otherwise makes personal data, subject to this processing, available to another controller, joint controller or processor ('importer'). 3) The importer is in a third country, irrespective of whether or not this importer is subject to the GDPR for the given processing in accordance with Article 3…" (para. 9). — [EDPB Guidelines 05/2021 on the interplay between Art. 3 and Chapter V, v2.0, adopted 14 Feb 2023](https://www.edpb.europa.eu/system/files/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **[REGULATOR GUIDANCE, ACCESS = TRANSFER]** Section 2.2: "Some examples of how personal data could be 'made available' are by creating an account, granting access rights to an existing account, 'confirming'/'accepting' an effective request for remote access… It should be kept in mind that remote access from a third country (even if it takes place only by means of displaying personal data on a screen, for example in support situations, troubleshooting or for administration purposes) and/or storage in a cloud situated outside the EEA offered by a service provider, is also considered to be a transfer, provided that the three criteria outlined in paragraph 9 above are met." — [EDPB Guidelines 05/2021 v2.0](https://www.edpb.europa.eu/system/files/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **[REGULATOR GUIDANCE, the hybrid scenario]** Example 11, "Remote access to data in the EU by a third country processor acting on behalf of EU controllers": "A company in a third country (Company Z), with no establishment in the EU, offers services as a processor to companies in the EU. Company Z, acting as processor on behalf of the EU controllers, is remotely accessing, e.g. for support purposes, the data which is stored in the EU. Since Company Z is located in a third country, such remote access results in transfers of data from the EU controllers to their processor (Company Z) in a third country under Chapter V." — [EDPB Guidelines 05/2021 v2.0](https://www.edpb.europa.eu/system/files/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **[REGULATOR GUIDANCE, PROCESSING in third country]** Example 5, "Controller in the EU sends data to a processor in a third country": "…data are provided from a controller, which as regards the processing in question, is subject to the GDPR, to a processor in a third country. Hence, the provision of data will be considered as a transfer…" — [EDPB Guidelines 05/2021 v2.0](https://www.edpb.europa.eu/system/files/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **[REGULATOR GUIDANCE, internal processing]** "Conversely, Chapter V does not apply to 'internal processing', i.e. where data is not disclosed by transmission or otherwise made available to another controller or processor, including where such processing takes place outside the EU." The hospital's data going to Poised (a separate processor) is not internal processing. — [EDPB Guidelines 05/2021 v2.0](https://www.edpb.europa.eu/system/files/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **[REGULATOR GUIDANCE, EU processor under foreign law]** Example 12 contrasts the case of an EU processor that is subject to third-country law but where "there is no one outside the EU… who has access to the data". That is *not* a transfer, but the controller must assess the processor's reliability under Art. 28(1). — [EDPB Guidelines 05/2021 v2.0](https://www.edpb.europa.eu/system/files/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **[REGULATOR GUIDANCE, DPC]** The DPC's transfers page explains the Chapter V mechanisms: adequacy (Art. 45), appropriate safeguards (Art. 46: SCCs, BCRs and others) and derogations (Art. 49). It does **not** say that data must be stored in Ireland or the EEA. The fetched page did not explicitly mention remote access or the UK. — [DPC: Transfers of personal data to third countries or international organisations](https://www.dataprotection.ie/en/organisations/international-transfers/transfers-personal-data-third-countries-or-international-organisations)
- **[REGULATOR GUIDANCE, DPC cloud]** The DPC cloud guidance says "Important for both control and security is the location of the data". The landing page sets no Ireland or EEA requirement. The full note dates from November 2019 and I did not read it. — [DPC: Guidance for organisations engaging cloud service providers](https://www.dataprotection.ie/en/dpc-guidance/guidance-organisations-engaging-cloud-service-providers)
- **[HSE INTERNAL GUIDANCE, ACCESS = TRANSFER]** HSE's Schrems II FAQ (last updated 6 Dec 2021), Q1: "Chapter 5 of the GDPR sets out the permitted transfer mechanisms to transfer personal data, to or permit access to the data from, third countries such as adequacy decisions, appropriate safeguards (including standard contractual clauses)…". The HSE itself treats *permitting access from* a third country as within Chapter V. — [HSE: FAQs in relation to Schrems II](https://healthservice.hse.ie/documents/1098/faqs-in-relation-to-schrems-ii.pdf)

### Inferences
- The user's premise ("residency is only about storage, never processing") is **wrong for GDPR**. The transfer test is about *who can access the data from where*, so a UK processor's compute and support are what trigger Chapter V.
- **Both designs involve the same Chapter V transfer** (Irish hospital → Poised in the UK). The hybrid does not make the transfer go away.
- With UK adequacy in force (see Q3), that transfer needs no SCCs and no transfer impact assessment (TIA). So, as a matter of law, both designs are equally lawful.
- In the hybrid, every page view processed in London is a transfer. The data lives in Dublin, but it is made available to the UK processor on every request.

### Gaps
- I did not find a DPC decision or DPC-authored text (as distinct from EDPB guidance) that specifically discusses remote access from the UK. The DPC is bound to apply EDPB guidelines consistently, but I found no Ireland-specific statement.
- I did not read the full 2019 DPC cloud guidance PDF.

---

## Q3. UK–EU adequacy status as of October 2026, and any Ireland–UK or Common Travel Area arrangement for health data

### Takeaway
The European Commission **renewed** the UK GDPR adequacy decision on **19 December 2025**, valid until **27 December 2031**. Transfers from Ireland to the UK therefore need no additional safeguards, though GDPR's other rules (Art. 28, Art. 32, DPIA) still apply. I found no Ireland–UK or Common Travel Area arrangement specific to health data protection.

### Cited Findings
- **[BINDING LAW, TRANSFER]** "On 19 December 2025, the European Commission reaffirmed that the United Kingdom (UK) ensures an adequate level of data protection so that personal data can continue to be transferred to the UK as third country." The Commission adopted two decisions, one under the GDPR and one under the Law Enforcement Directive. The renewal took account of an EDPB opinion adopted on 16 October 2025, and the decision "will expire on 27 December 2031 unless it is extended." — [eucrim: Commission renewed adequacy decisions for data transfers to the UK](https://eucrim.eu/news/commission-renewed-adequacy-decisions-for-data-transfers-to-the-uk/); Commission press release IP/25/3059 is linked from that page ([press release](https://ec.europa.eu/commission/presscorner/detail/en/ip_25_3059), not separately fetched); confirmed by [Hunton: European Commission renews UK data adequacy decisions](https://www.hunton.com/privacy-and-cybersecurity-law-blog/european-commission-renews-uk-data-adequacy-decisions)
- **History (superseded):**
  - The 2021 decisions would have expired on 27 June 2025.
  - An interim extension was endorsed in June 2025 while the Commission reviewed the UK Data (Use and Access) Act 2025. — [eucrim](https://eucrim.eu/news/commission-renewed-adequacy-decisions-for-data-transfers-to-the-uk/)
  - HSE's own FAQ still states the old position: "adequacy decision remains valid … until 27 June 2025". That page is **out of date** (last updated Dec 2021). — [HSE Schrems II FAQ](https://healthservice.hse.ie/documents/1098/faqs-in-relation-to-schrems-ii.pdf)
- **[HSE INTERNAL GUIDANCE]** HSE FAQ Q5 and Q6: "Where adequacy decisions are in place, the data exporter is not required to implement any other appropriate safeguards or put any supplementary measures in place to validate the transfer." And: "no further steps need to be taken in respect of personal data transfers to the UK where this adequacy decision applies… From a general data protection perspective, relationships with any UK-based service providers or partners should consider whether any other formal arrangements are required such as a data processing agreement (as required by Article 28 of the GDPR)…" — [HSE Schrems II FAQ](https://healthservice.hse.ie/documents/1098/faqs-in-relation-to-schrems-ii.pdf)
- **Ireland-specific point:** eucrim notes that the adequacy decision "is also binding on EU countries with opt-out/opt-in reservations in justice and home affairs (i.e., Ireland and Denmark)". This concerns the **Law Enforcement Directive** decision. The GDPR decision applies across the EEA anyway. — [eucrim](https://eucrim.eu/news/commission-renewed-adequacy-decisions-for-data-transfers-to-the-uk/)

### Inferences
- In October 2026, **Ireland → UK processing or access by Poised is a lawful transfer under Art. 45**, with no SCCs or TIA needed.
- The residual legal risk is that adequacy is suspended, revoked or not renewed after 2031. If that happens, *both* designs would need SCCs (Module 2, controller → processor) and a TIA, because the UK compute and support access are the transfer.
- Keeping the database in Dublin would not avoid that. It would only make it easier to move compute into the EU.

### Gaps
- I found **no** Common Travel Area instrument, or Ireland–UK bilateral data protection arrangement, specific to health data. The Common Travel Area deals with movement and residence rights, and I found no primary source linking it to data transfers.
- I did not fetch the Official Journal reference number for the December 2025 decision.
- I did not check whether any formal CJEU challenge to the renewed decision is pending as of October 2026.

---

## Q4. The Health Information Act 2026, the EU European Health Data Space Regulation and HIQA: any rule on where health information is stored or processed?

### Takeaway
- **Health Information Act 2026:** the Health Information Bill was enacted as the Health Information Act 2026 (No. 10 of 2026). A full-text search found **no storage or processing location requirement**. Its only third-country provision (s.18) is about the HSE sharing EHR information with foreign health bodies.
- **EHDS Art. 86, the one rule here that is about storage only:** the EU European Health Data Space Regulation (EU) 2025/327 contains the only explicit residency hook. Art. 86 *permits* (but does not require) Member States to mandate that **storage** of priority-category health data processed by healthcare providers be located within the EU. It concerns **storage only**, and applies from 26 March 2027.
- **Ireland has not used it so far:** I found no evidence that Ireland has exercised the Art. 86 option. The 2026 Act does not.

### Cited Findings
- **[BINDING LAW]** Health Information Act 2026, long title: "An Act to give further effect to Regulation (EU) 2025/327 … on the European Health Data Space…; to provide for a duty for health services providers to share, in certain circumstances, a patient's personal health data with other health services providers…; to provide for the creation and assignment by the Health Service Executive of an Electronic Health Record in respect of every patient…". Its Parts cover:
  - Duty to Share (ss.7–9);
  - Electronic Health Records (ss.10–21);
  - Provision of Health Information to the Executive (ss.22–24);
  - amendment of the Patient Safety Act 2023 (s.25). — [Health Information Act 2026 (No. 10 of 2026)](https://www.irishstatutebook.ie/eli/2026/act/10/enacted/en/html); [PDF](https://www.irishstatutebook.ie/eli/pdf/2026/en.act.2026.0010.pdf)
- **[BINDING LAW, no location rule]** I searched the full text of the 2026 Act PDF for "outside the State", "Economic Area", "stored", "storage", "located", "processor" and "within the State". The only hits for "third country" are in s.18, and there are none for storage location or processors. — [Health Information Act 2026 PDF](https://www.irishstatutebook.ie/eli/pdf/2026/en.act.2026.0010.pdf)
- **[BINDING LAW, format and timing, not location]** s.7(2): forwarded data "shall be forwarded … (a) in a digital format, utilising the standards and interoperability requirements specified in guidelines, and (b) within the period specified in guidelines." — [s.7](https://www.irishstatutebook.ie/eli/2026/act/10/section/7/enacted/en/html)
- **[BINDING LAW, ACCESS by third-country health bodies]** s.18(1)–(3): the HSE "may … enter into a reciprocal arrangement or other agreement with a third country health services provider for the purpose of sharing relevant information included in an Electronic Health Record…". Such arrangements must "permit secure access by a third country health services provider", and the HSE "shall not enter into an arrangement … except after consultation with the Data Protection Commission". s.18(5) defines "third country" as "a country or territory other than the State or another Member State". This covers HSE-to-foreign-health-body EHR sharing, not vendor hosting. — [s.18](https://www.irishstatutebook.ie/eli/2026/act/10/section/18/enacted/en/html)
- **Commencement (unverified):** search-engine summaries state that the Act was signed on 30 April 2026, and that certain EHR provisions were commenced by Ministerial Order on 22 July 2026, taking effect on 17 August 2026. I did not open the commencement order. — [Health Information Act 2026 index (irishstatutebook)](https://www.irishstatutebook.ie/eli/2026/en/act/pub/0010/index.html)
- **[BINDING LAW (EU), STORAGE only, Member State option]** EHDS Regulation (EU) 2025/327, Art. 86, "Storage of personal electronic health data for primary use": "…Member States shall ensure that a particularly high level of protection and security is in place when processing personal electronic health data for primary use, by means of appropriate technical and organisational measures. In this respect, this Regulation shall not preclude a requirement under national law, taking into account the national context, that, in cases where personal electronic health data are processed by healthcare providers for the provision of healthcare or by the national contact points for digital health connected to MyHealth@EU, the storage of personal electronic health data referred to in Article 14 of this Regulation for the purpose of primary use be located within the Union, in compliance with Union law and international commitments." — [EHDS Art. 86 (streamlex.eu mirror of the OJ text)](https://streamlex.eu/articles/ehds-en-art-86/); official text at [EUR-Lex, Regulation (EU) 2025/327](https://eur-lex.europa.eu/eli/reg/2025/327/oj). EUR-Lex blocked automated fetch, so the wording was taken from the mirror.
- **[BINDING LAW (EU)]** EHDS Art. 105: "This Regulation shall apply from 26 March 2027", with staggered later dates (2029, 2031) for specific provisions. Art. 86 has no special date of its own. — [EHDS Art. 105 (streamlex.eu)](https://streamlex.eu/articles/ehds-en-art-105/)
- **HIQA:** I found no HIQA standard or guidance that sets a storage or processing location for health information (see Gaps).

### Inferences
- EHDS Art. 86 is the one place where the user's "storage, not processing" intuition is **literally right**: the permitted national rule covers *where storage is located*, nothing more.
- If Ireland ever legislated under Art. 86, the **hybrid** design (database and S3 in Dublin) would meet it. **All-London** would not, at least for "Article 14" priority-category data (patient summaries, e-prescriptions and similar). Whether pre-op questionnaire answers fall into those categories is unclear.
- As of October 2026, Ireland's implementing Act has not imposed such a requirement.
- Any such national requirement would still sit alongside GDPR Chapter V. Processing or access from the UK would remain a transfer, which is lawful under adequacy.

### Gaps
- I did not verify whether regulations or guidelines under the 2026 Act (s.3 regulations, s.21 guidelines) impose hosting conditions. None were found.
- I did not check whether Poised would count as an "EHR system" under EHDS Chapter III (certification and CE-style obligations), which is a separate compliance question.
- I did not research HIQA's information-management standards in depth. I found no location rule, but I have no affirmative source to cite.
- Secondary-use EHDS provisions (secure processing environments) were out of scope.

---

## Q5. HSE policies and procurement, and the Irish government cloud policy: any EEA or Ireland-only hosting, or limits on offshore support?

### Takeaway
- **Government policy:** the Cloud Computing Advice Note, updated in 2025, sets **no** central data-classification or residency rule. Each body sets its own, and it says data sensitivity alone should not prevent use of public cloud.
- **HSE practice:**
  - The HSE treats non-EEA access or processing as a Chapter V transfer, with TIAs and SCCs where needed.
  - It describes its own systems as EEA-hosted.
  - At least one recent HSE SaaS tender made it **mandatory that the solution be "hosted, backed-up and supported from within the EEA"**. That clause covers storage, compute **and support access**.
- **Where the real constraint is:** for HSE work, procurement terms (not law) are the binding constraint. They can explicitly reach processing and support location.

### Cited Findings
- **[GOVERNMENT POLICY, applies to public bodies]** Cloud Computing Advice Note, 2025 Update, from the Department of Public Expenditure, Infrastructure, Public Service Reform and Digitalisation. Page metadata gives publication on 21 May 2025 and modification on 19 June 2025. It updates the October 2019 Note; "much of the advice from then still applies". — [gov.ie: Cloud Computing Advice Note 2025](https://www.gov.ie/en/department-of-public-expenditure-infrastructure-public-service-reform-and-digitalisation/publications/cloud-computing-advice-note-2025/); [PDF](https://assets.gov.ie/static/documents/Cloud_Computing_Advice_Note_2025_-_PDF.pdf); superseded original: [Cloud Computing Advice Note October 2019](https://www.gov.ie/en/publication/078d54-cloud-computing-advice-note-october-2019); [Arthur Cox summary of the 2019 note](https://www.arthurcox.com/knowledge/department-of-public-expenditure-and-reform-publishes-advice-on-cloud-computing/)
- **[GOVERNMENT POLICY, covers processing, storage and transit; no residency rule]** Advice Note 2025, §8.2 Data:
  - "organisations must be aware of how their data is protected while being processed (in use), while stored (at rest) and while moving (in transit)."
  - "In deciding what information can be put in the cloud and where (public, private, hybrid or not at all), organisations need to be able to identify the sensitivity of their data… Sensitivity of data alone should not prevent use of public cloud."
  - "There are no central data classification requirements in place except for information defined as 'top secret'. Individual public service organisations are able to define their own requirements for their own data and some will have existing specific requirements in place."
  - "organisations must ensure that these [data protection] obligations are not used as a reason to avoid using cloud services without clearly documented justification."

  It then points to the DPC's transfer guidance: adequacy (Art. 45), SCCs and other safeguards (Art. 46), and derogations (Art. 49). — [Cloud Computing Advice Note 2025 PDF](https://assets.gov.ie/static/documents/Cloud_Computing_Advice_Note_2025_-_PDF.pdf)
- **[PROCUREMENT, STORAGE + PROCESSING + SUPPORT ACCESS]** HSE 21795, Electronic and Digital Signature Solution (SaaS), published 14/11/2024: "It is a mandatory requirement that the solution will be hosted, backed-up and supported from within the European Economic Area (EEA)." — [EU Publications Office tender notice, HSE 21795](https://op.europa.eu/en/web/public-procurement/procurement-details/-/procurement/23523b78-bfdc-48d2-8db5-51454b958bc5)
- **[HSE PRACTICE, TRANSFER]** HSE staff guidance: "If you transfer personal data to a destination outside the EEA (a 'third country'), you must follow the transfer mechanisms set out in GDPR". It provides an HSE TIA form, and routes queries to Technology and Transformation and the Deputy DPOs. — [HSE: International transfer of personal data](https://healthservice.hse.ie/staff/procedures-guidelines/data-protection/international-transfer-of-personal-data/)
- **[HSE PRACTICE, TRANSFER]** HSE privacy statement (website): "We do not transfer any other personal data about you outside the European Economic Area (EEA). If in the future, we need to transfer your personal data outside the EEA, we will comply with Chapter 5 of the General Data Protection Regulation (GDPR)." — [HSE privacy statement](https://www.hse.ie/privacy-statement/)
- **[HSE PRACTICE, from search-result snippets only; not fetched]**
  - An HSE IFMS CRM privacy notice says the system "is hosted within the European Economic Area (EEA) and no data will be transferred to countries outside the EEA". — [HSE IFMS CRM privacy notice](https://hse.ie/eng/about/who/finance/financereformprogramme/technologyfinancereform/ifms-stakeholder-management-crm-privacy-notice1.pdf)
  - The HSE Live notice says no data is transferred outside the EEA. — [HSE Live data protection notice](https://about.hse.ie/publications/hse-live-phone-service-data-protection-information-notice/)
- **[HSE PRACTICE, unverified; PDF would not download]** A search summary of HSE parliamentary question response PQ 46795/25 (September 2025) states that HSE data is held in the EEA or in a country covered by an EU adequacy decision, with EU model agreements where data is held elsewhere. Treat this as unverified. — [HSE PQ 46795/25](https://www.hse.ie/eng/about/personalpq/pq/2025-pq-responses/september-2025/pq-46795-25-padraig-rice.pdf)
- **[HSE CONTRACT TEMPLATES, from snippet]** The HSE's published data processing terms for non-statutory providers include "Standard Contractual Clauses that deal with the transfer of Personal Data outside the European Economic Area". This means the templates expect, and provide for, non-EEA transfers rather than banning them. — [HSE data processing terms (provider to HSE)](https://www.hse.ie/eng/services/publications/non-statutory-sector/hse-non-stat-sector-data-processing-terms-provider-to-hse.pdf)

### Inferences
- **For an HSE customer, the binding constraint is the tender and contract.**
  - HSE 21795 shows that HSE drafters write residency clauses covering **hosting, backups and support**, not just storage.
  - A clause like that would be failed by **both** the hybrid (compute and support in London) and all-London.
  - The hybrid would pass only if the tender said something narrower, such as "data stored in the EEA".
- **HSE's data protection function is not hostile to UK processing.** Its own guidance says UK transfers need "no further steps" beyond an Art. 28 agreement while adequacy applies. Some HSE procurements may accept UK or adequacy-country processing. It is project-by-project.
- **Backups:** if the tender says "backed-up … within the EEA", S3 or RDS cross-region backups or replication to London would breach it, even in the hybrid design.

### Gaps
- I did not find a published, current HSE-wide "cloud policy" or information-classification policy with a residency rule. HSE policies such as the Personal Data Protection Policy exist, but I could not retrieve the PDFs.
- I could not find a standard HSE ICT framework clause on offshore support. The evidence is one tender (HSE 21795), so frequency across HSE tenders is unknown.
- I did not check NIS2 transposition in Ireland (the National Cyber Security Bill) for any location rules. The Advice Note mentions NIS2 only in connection with security responsibilities.

---

## Q6. Private hospitals (e.g. Bon Secours, Beacon, Blackrock): any sector rules or practice?

### Takeaway
I found no private-hospital-specific residency rule. Private hospitals are bound by the same GDPR and DPA 2018 regime (no localisation) and by the Health Information Act 2026's duty-to-share rules (no localisation). Visible practice is "adequacy or safeguards", not "EEA-only".

### Cited Findings
- **[PRACTICE, from search-result summary only]** Bon Secours Ireland's candidate privacy policy (a staff/recruitment notice, not a patient notice) reportedly says it transfers personal data only where the Commission has found the third country adequate, or where other safeguards are in place. The full page did not load on fetch. — [Bon Secours Ireland candidate privacy policy](https://www.bonsecours.ie/careers/candidate-privacy-policy)
- **[BINDING LAW]** The Health Information Act 2026 duty to share applies to "health services providers" generally, which includes private providers, but contains no location rule (see Q4). — [Health Information Act 2026](https://www.irishstatutebook.ie/eli/2026/act/10/enacted/en/html)

### Inferences
- For a private Irish hospital, both designs are lawful under adequacy. Whether a residency clause applies depends on the hospital's own procurement or information-security policy. Expect requests for an Art. 28 data processing agreement, a DPIA input, sub-processor lists and possibly "EEA hosting".

### Gaps
- I did not retrieve patient-facing privacy notices or procurement terms for Beacon, Blackrock, Mater Private or Bon Secours. I have no evidence either way on whether they demand EEA-only processing or support.
- I did not check whether HIQA licensing of private hospitals (under the Patient Safety (Licensing) Act) creates any information-governance requirement about hosting.

---

## Q7. Verdict: does (1) the hybrid design and (2) all-in-London hosting comply, for an Irish HSE or private hospital customer?

### Takeaway
- **Binding law:** both designs comply with Irish and EU law in October 2026. No storage or processing localisation applies. The UK processing and support access is a Chapter V transfer, and it is covered by the UK adequacy decision valid to 27 December 2031.
- **What the hybrid buys:** nothing under GDPR. It helps only against **storage-only** requirements: an "EEA-stored" contract clause, or a possible future national EHDS Art. 86 rule.
- **What it does not satisfy:** HSE-style clauses requiring hosting, backups **and support** from within the EEA. For those, neither design complies. Only EU-resident compute and EEA-based support would.
- **The user's belief:** "residency rules are only about storage, never processing" is **partly wrong**.
  - It is right for EHDS Art. 86 and for some contract clauses.
  - It is wrong for GDPR's transfer test (access counts).
  - It is wrong for at least one HSE procurement clause ("supported from within the EEA").

### Cited Findings
- **Remote access by a third-country processor to EU-stored data is a transfer.** (EDPB Example 11, quoted in Q2.) — [EDPB Guidelines 05/2021 v2.0](https://www.edpb.europa.eu/system/files/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **The UK adequacy renewal makes Ireland → UK transfers lawful without specific authorisation.** (Renewed 19 Dec 2025, expires 27 Dec 2031.) — [eucrim](https://eucrim.eu/news/commission-renewed-adequacy-decisions-for-data-transfers-to-the-uk/); [GDPR Art. 45(1)](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
- **EHDS Art. 86 permits a national rule on storage location only.** — [EHDS Art. 86](https://streamlex.eu/articles/ehds-en-art-86/)
- **The Health Information Act 2026 has no location rule.** — [Health Information Act 2026](https://www.irishstatutebook.ie/eli/pdf/2026/en.act.2026.0010.pdf)
- **HSE 21795 makes EEA hosting, backup and support mandatory.** "hosted, backed-up and supported from within the European Economic Area (EEA)" — [HSE 21795 notice](https://op.europa.eu/en/web/public-procurement/procurement-details/-/procurement/23523b78-bfdc-48d2-8db5-51454b958bc5)
- **Government cloud policy sets no central residency rule.** "There are no central data classification requirements in place except for information defined as 'top secret'." — [Cloud Computing Advice Note 2025](https://assets.gov.ie/static/documents/Cloud_Computing_Advice_Note_2025_-_PDF.pdf)

### Inferences

**Hybrid (DB and S3 in Dublin; compute and support in London):**

| Customer | Assessment |
|---|---|
| Law (any customer) | Compliant. Chapter V transfer to the UK on every request, covered by adequacy. Needs an Art. 28 data processing agreement that authorises UK processing and support (Art. 28(3)(a)), s.52(2)-style confidentiality for staff, Art. 32 security and a DPIA. |
| HSE | Fails any "hosted/supported from within the EEA" clause like HSE 21795. Passes a narrower "data stored/at rest in the EEA" clause, *if* backups and replicas also stay in the EEA. |
| Private hospital | Likely acceptable. Depends on contract. |
| Future EHDS Art. 86 rule in Ireland (none exists today) | Would likely pass, since the rule concerns storage. |

**All-in-London:**

| Customer | Assessment |
|---|---|
| Law (any customer) | Compliant on the same basis (adequacy). Same Art. 28, s.52(2), Art. 32 and DPIA duties. |
| HSE | Fails any EEA-hosting or EEA-storage clause. |
| Private hospital | Likely acceptable unless the hospital's policy demands EEA hosting. |
| Future EHDS Art. 86 rule in Ireland | Would fail for priority-category data. |

**Engineering consequences:**
- If HSE work is a target, the hybrid gives limited extra value over all-London. Clauses like HSE 21795 require compute and support inside the EEA too, which means a full EU deployment (compute in eu-west-1) plus an EEA-based support arrangement or an EU entity.
- A hybrid that also needs low latency puts every query on a London→Dublin network hop, and still fails "hosted/supported" clauses.
- **Adequacy risk:** if UK adequacy lapses or is struck down, both designs immediately need SCCs plus a TIA. The hybrid does not mitigate this, because the UK access is the transfer. Only EU compute plus EU-only support would.

### Gaps
- I have only one confirmed HSE tender example of the "hosted, backed-up and supported" clause. I did not establish how common it is across HSE or hospital-group tenders, or whether HSE accepts adequacy-country processing on clinical SaaS (the PQ 46795/25 summary suggests it might, but it is unverified).
- I did not get an Irish law firm opinion that applies EDPB Example 11 to UK-based SaaS vendors serving Irish hospitals after the 2025 adequacy renewal. The analysis above rests on EDPB primary guidance.
