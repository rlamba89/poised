# United States: health data residency — does the law regulate where PHI is STORED, or also where it is PROCESSED / ACCESSED?

Scope: hospital / healthcare-provider patient data (PHI), with a UK SaaS vendor acting as a HIPAA business associate. Two designs are tested: (1) **hybrid**: Postgres and S3 in a US region (e.g. us-east-1), with the application code (Lambdas) in AWS London processing data in memory and UK staff giving support; (2) **all-London**: everything hosted in eu-west-2. Research date: 8 Oct 2026.

Legend used below: **[LAW]** = binding statute or regulation; **[GUIDANCE]** = regulator guidance; **[CONTRACT]** = contract or procurement terms (binding only on the parties who sign them); **[COMMENTARY]** = law-firm or other secondary analysis.

---

## Q1. HIPAA (Privacy and Security Rules) and HHS OCR guidance: is there any data-location requirement, and what must a BAA contain?

### Takeaway
HIPAA has **no data-location requirement of any kind**: not for storage, processing or access. OCR's own cloud FAQ says in terms that a covered entity may use a business associate that stores ePHI outside the US. OCR's only condition is that the BAA is in place, and that foreign-location risks are covered in the Security Rule risk analysis. The BAA must contain HIPAA's standard terms, none of which concern geography. Both designs, hybrid and all-London, are therefore HIPAA-permissible.

### Cited Findings
- **[GUIDANCE]** OCR cloud computing FAQ, Q9, quoted verbatim: *"Do the HIPAA Rules allow a covered entity or business associate to use a CSP that stores ePHI on servers outside of the United States? Yes, provided the covered entity (or business associate) enters into a business associate agreement (BAA) with the CSP and otherwise complies with the applicable requirements of the HIPAA Rules. However, while the HIPAA Rules do not include requirements specific to protection of electronic protected health information (ePHI) **processed or stored** by a CSP or any other business associate outside of the United States, OCR notes that the risks to such ePHI may vary greatly depending on its geographic location. In particular, outsourcing storage or other services for ePHI overseas may increase the risks and vulnerabilities to the information or present special considerations with respect to enforceability of privacy and security protections over the data. Covered entities (and business associates, including the CSP) should take these risks into account when conducting the risk analysis and risk management required by the Security Rule. See 45 CFR §§ 164.308(a)(1)(ii)(A) and (a)(1)(ii)(B). For example, if ePHI is maintained in a country where there are documented increased attempts at hacking or other malware attacks, such risks should be considered, and entities must implement reasonable and appropriate technical safeguards to address such threats."* The page says "Content last reviewed December 23, 2022". — [HHS OCR, Guidance on HIPAA & Cloud Computing (archived Nov 2025)](https://web.archive.org/web/2025/https://www.hhs.gov/hipaa/for-professionals/special-topics/health-information-technology/cloud-computing/index.html); standalone FAQ: [HHS FAQ 2083](https://www.hhs.gov/hipaa/for-professionals/faq/2083/do-the-hipaa-rules-allow-a-covered-entity-or-business-associate-to-use-a-csp-that-stores-ephi-on-servers-outside-of-the-united-states/index.html). hhs.gov returned 403 to automated fetches, so the text was read from the Wayback Machine copy.
  - **Note:** OCR explicitly names both **"processed or stored"** and still finds no location rule. HIPAA itself therefore does not distinguish processing from storage; it regulates neither by location.
- **[GUIDANCE]** OCR cloud FAQ, Q1: a covered entity may use a cloud service "to store or process ePHI", *"provided the covered entity or business associate enters into a HIPAA-compliant business associate contract or agreement (BAA) with the CSP … The BAA also contractually requires the business associate to appropriately safeguard the ePHI, including implementing the requirements of the Security Rule."* — [HHS OCR cloud guidance (archived)](https://web.archive.org/web/2025/https://www.hhs.gov/hipaa/for-professionals/special-topics/health-information-technology/cloud-computing/index.html)
- **[GUIDANCE]** The same guidance says a CSP is a business associate *"even if the CSP processes or stores only encrypted ePHI and lacks an encryption key for the data"*, and is *"both contractually liable for meeting the terms of the BAA and directly liable for compliance with the applicable requirements of the HIPAA Rules."* So AWS (US region) and Poised (UK) are both business associates or subcontractor business associates. — [HHS OCR cloud guidance (archived)](https://web.archive.org/web/2025/https://www.hhs.gov/hipaa/for-professionals/special-topics/health-information-technology/cloud-computing/index.html)
- **[LAW]** 45 CFR 164.314(a)(2)(i) says the BAA *"must provide that the business associate will— (A) Comply with the applicable requirements of this subpart; (B) … ensure that any subcontractors that create, receive, maintain, or transmit electronic protected health information on behalf of the business associate agree to comply with the applicable requirements of this subpart by entering into a contract …; and (C) Report to the covered entity any security incident of which it becomes aware, including breaches of unsecured protected health information as required by § 164.410."* None of these terms concern geography. (The Privacy Rule BAA terms are at 45 CFR 164.504(e); that text was not re-fetched.) — [eCFR 45 CFR 164.314](https://www.ecfr.gov/current/title-45/section-164.314)
- **[COMMENTARY]** McDermott Will & Schulte (1 Jul 2025): HIPAA *"does not prohibit PHI from being accessed or stored outside the US"*. The practical problem they identify is enforcement: *"If a foreign vendor violates HIPAA or experiences a data breach, there is limited recourse unless there are strong, binding, international arbitration provisions, or the foreign vendor maintains a substantial US-based presence."* — [McDermott, "US healthcare offshoring: Navigating patient data privacy"](https://www.mcdermottlaw.com/?p=322505)
- **[COMMENTARY]** Shumaker (12 Dec 2025): *"HIPAA does not prohibit PHI from being accessed or stored outside the United States."* Covered entities remain exposed for breach response if offshore vendors mishandle data. — [Shumaker, "Offshoring Patient Data"](https://www.shumaker.com/insight/offshoring-patient-data-what-health-care-providers-need-to-know-now/)
- **[LAW, pending]** The HIPAA Security Rule NPRM was published on 6 Jan 2025. Secondary sources report it was **still not final as of mid-2026**, with final action pushed to at least July 2027 on the federal regulatory agenda. — [Medcurity](https://medcurity.com/hipaa-security-rule-2026/); [myabt](https://www.myabt.com/blog/hipaa-security-rule-2026-final-changes). These are vendor blogs, not primary sources.

### Inferences
- Under HIPAA alone, **both** the hybrid and the all-London designs are lawful, provided there is a BAA and a documented risk analysis that addresses UK location, cross-border enforceability, and UK staff access.
- A UK vendor is directly liable under HIPAA as a business associate. US hospitals worry that OCR cannot practically enforce against a foreign entity, and this is the main reason they add contractual location clauses (see Q6).
- In the Security Rule risk analysis, the UK should rate as low-risk compared with OCR's example of a country with "documented increased attempts at hacking". The analysis should still be written down.

### Gaps
- I could not fetch the live hhs.gov pages (403). The text above is from the Wayback capture of Nov 2025. I found no later OCR guidance that changes Q9.
- I did not verify whether the January 2025 Security Rule NPRM text contains any location provision. No source I found suggests it does.

---

## Q2. Texas SB 1188 (2025): exact wording, storage vs access, who it applies to, guidance and enforcement, penalties

### Takeaway
Texas Health & Safety Code §183.002(a) requires EHRs "under the control of" a covered entity to be **"physically maintained in the United States or a territory"**. On its words this is a **storage** rule. A separate clause, §183.002(b), limits access by **role**, not by location. Law-firm consensus is that offshore *access* is permitted as long as nothing is stored, copied or cached offshore. However, one firm (Hall Render) calls the statute ambiguous on offshore view-only access. The rule took effect for storage on 1 Jan 2026. I found no AG guidance, agency rules or enforcement actions as of Oct 2026.

### Cited Findings
- **[LAW]** Enrolled text of **§183.002, "Requirements for Electronic Health Record Storage"**, quoted verbatim:
  - *"(a) A covered entity shall ensure that electronic health records under the control of the entity that contain patient information are **physically maintained in the United States or a territory of the United States**. This subsection applies to: (1) electronic health records that are stored by a third-party or subcontracted computing facility or an entity that provides cloud computing services; and (2) electronic health records that are stored using a technology through which patient information may be electronically retrieved, accessed, or transmitted.*
  - *(b) A covered entity shall ensure that the electronic health record information of this state's residents, other than open data, is accessible only to individuals who require the information to perform duties within the scope of the individual's employment related to treatment, payment, or health care operations.*
  - *(c) Each covered entity shall implement reasonable and appropriate administrative, physical, and technical safeguards to protect the confidentiality, integrity, and availability of electronic health record information."*

  — [Texas SB 1188, enrolled text (capitol.texas.gov)](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm)
- **[LAW] What it covers:**
  - The operative verb in (a) is "physically maintained", which is storage.
  - The words "retrieved, accessed, or transmitted" in (a)(2) describe *which stored records* are caught (records stored in technology that allows retrieval or access). They do **not** impose a rule on where access happens from.
  - (b) is a need-to-know (treatment/payment/operations) access limit with **no geographic element**.

  — [SB 1188 enrolled](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm)
- **[LAW] Who it covers:**
  - §183.001(2): *"'Covered entity' has the meaning assigned by Section 181.001. The term includes a health care practitioner."* It then excludes home and community support agencies, nursing facilities, continuing care facilities, assisted living, ICFs, day activity centres, and TxHmL/HCS waiver providers. **Hospitals are not excluded.** — [SB 1188 enrolled](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm)
  - Tex. Health & Safety Code §181.001(b)(2) (Texas Medical Records Privacy Act) defines "covered entity" as any person who: *"(A) for commercial, financial, or professional gain … engages, in whole or in part, and with real or constructive knowledge, in the practice of assembling, collecting, analyzing, using, evaluating, storing, or transmitting protected health information … (B) comes into possession of protected health information; (C) obtains or stores protected health information under this chapter; or (D) is an employee, agent, or contractor of a person described by Paragraph (A), (B), or (C) …"*. In §181.001(b)(2)(A) the term expressly includes "a business associate … information or computer management entity". — [Tex. Health & Safety Code §181.001 (texas.public.law)](https://texas.public.law/statutes/tex._health_and_safety_code_section_181.001). The statute portal statutes.capitol.texas.gov is JS-only and could not be fetched; texas.public.law mirrors the text.
- **[LAW] Effective dates:** SECTION 2(b): *"Section 183.002 … applies to the storage of an electronic health record on or after January 1, 2026, regardless of the date on which the electronic health record was prepared."* The Act as a whole took effect on 1 Sep 2025 (SECTION 4). The Senate concurred on 28 May 2025. — [SB 1188 enrolled](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm). The Governor signed it on 20 Jun 2025. — [McDonald Hopkins](https://www.mcdonaldhopkins.com/insights/news/s-b-1188-texas-ehr-law-data-localization-ai-and-access-requirements)
- **[LAW] Penalties and enforcement:**
  - §183.011(b): the AG may seek civil penalties of up to *"$5,000 for each violation that is committed negligently and that occurs in a single year, regardless of how long the violation continues during that year; $25,000 for each violation that is committed knowingly or intentionally …; or $250,000 for each violation in which the covered entity knowingly or intentionally used protected health information for financial gain."* §183.011(a) also allows injunctive relief.
  - §183.010: a regulatory agency may discipline a covered entity that violates the chapter *"three or more times"*, including licence suspension or revocation.
  - §183.009: HHSC or the appropriate regulator *"shall conduct an investigation of any credible allegation"*.
  - §183.012: the agencies are to make an MOU and "as necessary, adopt rules".
  - No cure period appears in the text.

  — [SB 1188 enrolled](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm)
- **[COMMENTARY] Storage vs access:**
  - McDonald Hopkins (25 Aug 2025): *"while offshore storage is banned, offshore access is still permitted, provided that appropriate safeguards are in place"*. Entities must ensure *"offshore access does not result in storage, copying, or caching of EHR data on non-U.S. servers or devices."* — [McDonald Hopkins](https://www.mcdonaldhopkins.com/insights/news/s-b-1188-texas-ehr-law-data-localization-ai-and-access-requirements)
  - Buchalter (1 Oct 2025): *"Offshore access is somewhat more permissible, so long as data is not stored, cached, or copied offshore in violation of the law"*, and *"Vendors must ensure that any caching, backup, replication, or disaster recovery processes comply with the requirement."* — [Buchalter](https://www.buchalter.com/insights/from-phi-to-ai-what-texas-sb-1188-means-for-healthcare-entities-and-vendors/)
  - Hall Render (19 Aug 2025) dissents in part: *"The law is ambiguous regarding whether the prohibition relates only to storage or would also apply to offshore view-only access to such data."* — [Hall Render](https://hallrender.com/2025/08/19/new-texas-law-on-ehr-data-localization-and-ai/)
- **[COMMENTARY] Vendors:**
  - Buchalter: *"The law also explicitly reaches third-party vendors, cloud service providers, and subcontractors that manage or store EHRs on behalf of covered entities."* — [Buchalter](https://www.buchalter.com/insights/from-phi-to-ai-what-texas-sb-1188-means-for-healthcare-entities-and-vendors/)
  - McDonald Hopkins says the law reaches hospitals and clinics, insurers, schools and "third-party vendors who handle EHRs for covered entities". — [McDonald Hopkins](https://www.mcdonaldhopkins.com/insights/news/s-b-1188-texas-ehr-law-data-localization-ai-and-access-requirements)
- **[COMMENTARY vs LAW] Texas residents only?**
  - Buchalter describes the storage rule as applying to "EHRs that contain patient information for Texas residents". — [Buchalter](https://www.buchalter.com/insights/from-phi-to-ai-what-texas-sb-1188-means-for-healthcare-entities-and-vendors/)
  - The enrolled text of §183.002(a), however, has **no residency qualifier**: it covers EHRs "under the control of the entity that contain patient information". Only §183.002(b) refers to "this state's residents". — [SB 1188 enrolled](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm)
  - **Conflict noted.** The statute's text is broader than Buchalter's summary.
- **[LAW] Guidance and enforcement since 1 Jan 2026:** Targeted searches (AG Paxton, HHSC, TMB rules, FAQs, investigations) found **no** published AG guidance, no §183.012 rules and no enforcement actions. Law firms said they were "monitoring" for agency guidance. — [McDonald Hopkins](https://www.mcdonaldhopkins.com/insights/news/s-b-1188-texas-ehr-law-data-localization-ai-and-access-requirements); search of natlawreview, Hall Render, Katten, Goodwin and others ([NatLawReview](https://natlawreview.com/article/new-texas-law-storage-health-care-data)).
- **[LAW] "Electronic health record" is not defined** in Chapter 183. The §183.001 definitions list covers only biological sex, covered entity, female, health care practitioner, male, and sexual development disorder. — [SB 1188 enrolled](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm)

### Inferences
- **All-London fails SB 1188** for any Texas hospital customer. EHR data "under the control of" the hospital, stored by a cloud provider, would be physically maintained in the UK.
- **Hybrid probably passes on the text.** The records are "physically maintained" in us-east-1, and nothing in §183.002 forbids access or processing from abroad.
  - The word "maintained" is in line with HIPAA usage, where "create, receive, **maintain**, or transmit" separates holding data from transmitting it. That supports a storage reading.
  - The law-firm "no caching or copying offshore" gloss carries **real residual risk** for the hybrid, because London app servers necessarily hold PHI transiently in memory. Under a strict reading of "caching", in-memory caches (for example ElastiCache or a CDN in London) could be treated as offshore maintenance. Transient in-process memory is the weakest form of that argument; persistent copies are the strongest.
- For the hybrid to be defensible, **no PHI should persist outside the US**:
  - application logs and CloudWatch in eu-west-2;
  - error trackers;
  - queues (SQS) and Lambda /tmp;
  - backups, snapshots and DR replicas;
  - SES/SMS message bodies queued in London;
  - CDN/edge caching of API responses at non-US edge locations (Amplify/CloudFront). Set no-store headers.
  - These points are my architectural inference from the statute and the law-firm commentary, not from any official guidance.
- **Poised itself may be a "covered entity" under §183.001(2).** It does not depend only on the hospital, because the §181.001 definition sweeps in anyone "storing or transmitting" PHI and expressly includes business associates. Practical AG enforcement against a UK entity is uncertain, but Texas hospitals will push the obligation onto Poised by contract regardless.
- §183.002(b) (role-based access) means UK support staff should only touch Texas EHR data when their support work counts as performing "health care operations" for the covered entity. That needs least-privilege, logged, purpose-limited support access.

### Gaps
- No official Texas guidance or enforcement exists yet, so the storage-only reading has not been tested in court or by any regulator.
- Whether a pre-operative questionnaire and assessment platform holds "electronic health records" (the term is undefined) is open. The conservative assumption is yes.
- Whether transient in-memory processing outside the US counts as being "maintained" or "cached" offshore has not been addressed by any authority I found.

---

## Q3. Other states with health-data location or offshore restrictions (enacted or pending, as of Oct 2026)

### Takeaway
**Florida** is the only other enacted *statutory* location rule for provider health records that I found. Fla. Stat. §408.051(3), in force since 2023, requires offsite patient information to be "physically maintained in the continental United States or its territories or **Canada**". It is again a storage rule, enforced through a licensure affidavit. I found no other state that copied Texas or Florida as of Oct 2026. Other "offshore" restrictions exist only through Medicaid contracts and state executive orders (see Q4).

### Cited Findings
- **[LAW] Fla. Stat. §408.051(3)**, quoted verbatim: *"SECURITY AND STORAGE OF PERSONAL MEDICAL INFORMATION.—In addition to the requirements in 45 C.F.R. part 160 and subparts A and C of part 164, a health care provider that utilizes certified electronic health record technology must ensure that all patient information stored in an offsite physical or virtual environment, including through a third-party or subcontracted computing facility or an entity providing cloud computing services, is **physically maintained in the continental United States or its territories or Canada**. This subsection applies to all qualified electronic health records that are stored using any technology that can allow information to be electronically retrieved, accessed, or transmitted."*
  - History: s. 9, ch. 2023-33 (SB 264, 2023), with later amendments in ch. 2024-15, 2025-6 and 2025-204.
  - "Qualified electronic health record" is defined in §408.051(2) as *"an electronic record of health-related information concerning an individual which includes patient demographic and clinical health information, such as medical history and problem lists, and which has the capacity to provide clinical decision support, to support physician order entry, to capture and query information relevant to health care quality, and to exchange electronic health information with, and integrate such information from, other sources."*

  — [Fla. Stat. §408.051 (2025), flsenate.gov](https://www.flsenate.gov/Laws/Statutes/2025/408.051)
- **[LAW] Fla. Stat. §408.810(14)** (enforcement through licensure): *"The licensee must sign an affidavit at the time of his or her initial application for a license and on any renewal applications thereafter that attests under penalty of perjury that he or she is in compliance with s. 408.051(3). The licensee must remain in compliance with s. 408.051(3) or the licensee shall be subject to disciplinary action by the agency."*
  - §408.810(15) separately bars controlling-interest holders from interests in entities with business relationships with a "foreign country of concern": China, Russia, Iran, North Korea, Cuba, the Maduro regime and Syria. **The UK is not one.**

  — [Fla. Stat. §408.810 (2025)](https://www.flsenate.gov/Laws/Statutes/2025/408.810)
- **[COMMENTARY]** Jackson Lewis (16 Mar 2026), "A Reminder About Florida's Ban on Offshore Health Data Storage":
  - The law applies across *"hospitals, clinics, ambulatory surgical centers, home health agencies, hospices, nursing homes, labs, pharmacies, and many individual licensed practitioners"*, and in practice reaches *"vendors and subcontractors that support healthcare operations."*
  - The fetch summariser described the article as covering "storage AND access/retrieval", based on the "retrieved, accessed, or transmitted" clause. As in Texas, that clause scopes *which stored records* are caught; the operative duty is "physically maintained". **Treat the "access" characterisation with caution.**

  — [Jackson Lewis Workplace Privacy Report](https://www.workplaceprivacyreport.com/2026/03/articles/health-information-technology/a-reminder-about-floridas-ban-on-offshore-health-data-storage-what-providers-and-vendors-should-know/)
- **[COMMENTARY]** Shumaker describes Florida's law as prohibiting providers *"from storing qualified electronic health records outside the United States, its territories, or Canada"*, which is a storage-specific description. — [Shumaker](https://www.shumaker.com/insight/offshoring-patient-data-what-health-care-providers-need-to-know-now/)
- **[LAW, other Texas bill]** Texas HB 3233 (89R) would bar pharmacy benefit managers from storing patient data in a country designated a state sponsor of terrorism. This is irrelevant to the UK and to hospitals. Bill analysis is at the "engrossed" stage, and I did not confirm final enactment. — [HB 3233 bill analysis](https://capitol.texas.gov/tlodocs/89R/analysis/html/HB03233E.htm)
- **[LAW, pending/other]**
  - DC Bill 26-0525 (Personal Health Data Security Amendment Act of 2025) covers geofencing, consent and deletion. It has **no** location requirement. — [ACLU-DC testimony, 2026](https://www.acludc.org/?p=7838)
  - New 2026 state consumer privacy laws (for example Louisiana SB 386 and Oklahoma SB 546, both effective 1 Jan 2027) were reported as general privacy laws, not localisation laws. — search results summarising [PrivacyWorld (Aug 2026)](https://www.privacyworld.blog/2026/08/adding-to-the-count-the-latest-in-state-consumer-privacy-laws/). Not independently verified.
- **[COMMENTARY]** An RXNT vendor piece (2026) covers "Texas and Florida" as *the* 2026 EHR data-localisation laws, which is consistent with there being only two. — [RXNT](https://www.rxnt.com/what-healthcare-providers-in-texas-and-florida-need-to-know-about-2026-ehr-data-localization-laws/)

### Inferences
- **All-London fails Florida** for any Florida-licensed hospital using certified EHR technology (effectively all of them). **Hybrid passes on the text**, since storage is in the US.
  - Canada is also allowed in Florida (ca-central-1 would satisfy Florida, though not Texas).
  - Florida's affidavit "under penalty of perjury" makes Florida hospitals very conservative, so expect them to require contractual US-storage warranties from Poised.
- Florida's literal scope is "patient information" held by a provider *that uses* certified EHR technology, together with the clause "applies to all qualified electronic health records". Whether a pre-op tool's data falls in scope is ambiguous. The conservative assumption is yes.
- On what I found, the US statutory map for provider health-data localisation is **two states (TX and FL), both storage-only**.

### Gaps
- My searches for 2026 copycat bills in other states (for example Louisiana, Oklahoma, Tennessee, Arizona) found none, but I did not check every legislature's 2026 session database. Some pending bill could exist that I did not find.
- I found no AHCA (Florida) enforcement actions or guidance on §408.051(3).

---

## Q4. Public payers and government contracts: Medicare/CMS, Medicaid managed-care contracts, VA, DoD/TRICARE. Do they bind a private hospital's software vendor?

### Takeaway
This is where US rules **do** reach *access and processing*, not just storage. They do so mostly through **contract**, not statute:
- **Texas Medicaid MCO contracts** (and plan flow-downs to network providers) ban offshore work, offshore storage, transfer outside the US, *and remote access from outside the US*.
- **Wisconsin** Medicaid bars offshore work involving access to patient information.
- **Medicare Advantage** permits offshore subcontractors but requires attestations triggered by offshore *access or processing* as well as storage.
- **DoD's DFARS** cloud clause is storage-only and binds only DoD contractors.

These terms bind a private hospital's vendor only through flow-down: the hospital signs them with a plan or agency, then passes them on to its vendors.

### Cited Findings
- **[CONTRACT] Texas Medicaid (HHSC Uniform Managed Care Contract), as flowed down to providers.** Superior HealthPlan (a Texas Medicaid/CHIP MCO) notice to its providers, dated 15 Jul 2025, quoted verbatim:
  - *"As a reminder, per your contract with Superior HealthPlan and our contract with the Texas Health and Human Services Commission (HHSC), all work performed under your agreement must be conducted exclusively within the United States. In addition, all information obtained through performance of the agreement must be stored and maintained within the United States. Confidential information must not be transferred outside the United States under any circumstances. **No individual is permitted to access information, systems, or deliverables remotely from a location outside the United States.** This includes, but is not limited to, any access to HHSC information or systems.*
  - *This requirement applies to all services, including but not limited to: Information Technology (IT) services; **Data processing**, transmission, storage, archiving and data center operations; Disaster recovery sites and services; Customer support; Medical, dental, laboratory, and clinical services. Any violation of these provisions will be considered a breach of your contract and may result in Superior utilizing remedies including contract termination. Providers will also be liable to Superior for any monetary damages incurred."*

  — [Superior HealthPlan newsroom, 15 Jul 2025](https://superiorhealthplan.com/newsroom/important-compliance-with-us-only-performance-and-data-handling-requirements-07152025.html)
- **[CONTRACT, COMMENTARY]** Texas UMCC (as described by law firms):
  - Shumaker: it *"prohibits performance of work or maintenance of information outside the United States and bars remote offshore access to Texas Health and Human Services systems or data."* — [Shumaker](https://www.shumaker.com/insight/offshoring-patient-data-what-health-care-providers-need-to-know-now/)
  - McDermott: the UMCC requires MCOs to *"provide all services within the US"* and requires information to be *"stored and maintained within the United States."* — [McDermott](https://www.mcdermottlaw.com/?p=322505)
  - A search-result summary of the UMCC Terms & Conditions (§§4.10–4.11) quoted it as barring Confidential Information from being *"moved outside of the United States by any means (physical or electronic) at any time, for any period of time, for any reason"*, and barring *"remote access to HHSC information, systems, or Deliverables from a location outside of the United States"*, with an HHSC written-approval exception. — [HHS Texas STAR Health Uniform Terms (link from search)](https://www.hhs.texas.gov/sites/default/files/documents/doing-business-with-hhs/contracting/exhibit-a-star-health-uniform-terms-conditions.pdf). Direct download failed, so this exact wording is not verified by me.
- **[CONTRACT] Other Medicaid programmes:**
  - McDermott: *"Wisconsin: Contractors 'are prohibited from performing work outside the US that involves access to or disclosure of patient health and related information.'"* It also lists Arizona (AHCCCS), Ohio, Missouri and New Jersey as having *"regulatory or executive order restrictions."* — [McDermott](https://www.mcdermottlaw.com/?p=322505)
  - Shumaker: Ohio executive orders *"prohibit executive agencies from contracting for services to be performed outside the United States"* and extend to subcontractors. — [Shumaker](https://www.shumaker.com/insight/offshoring-patient-data-what-health-care-providers-need-to-know-now/)
  - A search-result summary (source page not individually fetched) stated that Missouri's managed-care contract prohibits health plans from "storing and/or transmitting any data related to the Managed Care Program outside of the United States", and that Wisconsin, Alaska, Ohio and Arizona prohibit offshore subcontractors. **Treat as unverified.**
- **[LAW] Federal Medicaid statute and regulation** (these concern location of entities and payments, not data):
  - 42 CFR 438.602(i): *"Entities located outside of the U.S. The State must ensure that the MCO, PIHP, PAHP, PCCM, or PCCM entity with which the State contracts under this part is not located outside of the United States and that no claims paid by an MCO, PIHP, or PAHP to a network provider, out-of-network provider, subcontractor or financial institution located outside of the U.S. are considered in the development of actuarially sound capitation rates."* — [eCFR 42 CFR 438.602](https://www.ecfr.gov/current/title-42/section-438.602)
  - ACA §6505 bars Medicaid payments for items or services furnished by entities outside the US. Per Shumaker, *"CMS clarified this targets benefits/services, not administrative support."* — [Shumaker](https://www.shumaker.com/insight/offshoring-patient-data-what-health-care-providers-need-to-know-now/)
- **[GUIDANCE/CONTRACT] Medicare Advantage / Part D:**
  - Offshore subcontracting is **permitted**, but CMS requires plans to *"obtain detailed attestations regarding any offshore subcontractors that 'receive, process, transfer, handle, store, or access' beneficiary PHI."* — [Shumaker](https://www.shumaker.com/insight/offshoring-patient-data-what-health-care-providers-need-to-know-now/)
  - McDermott ties this to 42 CFR §422.503, under which MAOs obtain *"detailed information regarding the offshore vendors' safeguards protecting patient information"* via *"a signed attestation certificate"*. It adds that CMS *"maintains the authority to audit compliance and may penalise MAOs."* — [McDermott](https://www.mcdermottlaw.com/?p=322505)
  - Plans push these attestations to providers. Examples: [Health Net Oregon "Offshore Subcontracting Attestation – Participating Provider"](https://www.healthnetoregon.com/content/dam/centene/healthnet/pdfs/provider/or/OR-Offshore-Subcontracting-Attestation-Participating-Provider.pdf); [Wellcare CA provider offshore attestation form 2025](https://www.wellcare.com/-/media/pdfs/california/provider/2025/ca_care_prov_offshore_subcontracting_attestation_form_2025_r.ashx); [Johns Hopkins Health Plans offshore attestation](https://www.hopkinsmedicine.org/-/media/johns-hopkins-health-plans/documents/amd/offshore_subcontracting_attestation.pdf). I could not extract these PDFs' text.
- **[LAW, contract clause] DoD: DFARS 252.239-7010 (Cloud Computing Services, JAN 2023):**
  - (b)(3): *"The Contractor shall maintain within the United States or outlying areas all Government data that is not physically located on DoD premises, unless the Contractor receives written notification from the Contracting Officer to use another location."* This is **storage**.
  - (c): *"The Contractor shall not access, use, or disclose Government data unless specifically authorized by the terms of"* the contract. This is an authorisation limit, **not a location limit**.
  - The clause applies only "when using cloud computing to provide information technology services in the performance of the contract" with DoD.

  — [eCFR 48 CFR 252.239-7010](https://www.ecfr.gov/current/title-48/section-252.239-7010)
- **[LAW] VA:** I searched the text of VAAR Part 852 (the solicitation provisions and contract clauses) for "outside the United States", "offshore" and "within the United States" and found **no offshore-hosting or offshore-access clause**. The hits were only about mailed invoices from foreign vendors and IRB location. — [VAAR Part 852 (acquisition.gov)](https://www.acquisition.gov/vaar/part-852-solicitation-provisions-and-contract-clauses). VA Handbook 6500.6 ("Contract Security") governs VA contractors' handling of VA sensitive information, but I did not obtain its text.
- **[GUIDANCE] TRICARE:** Searches of the TRICARE manuals found encryption rules for portable devices holding PII/PHI, but **no explicit offshore ban**. — [TRICARE Systems Manual C1S1.1](https://manuals.health.mil/pages/DisplayManualHtmlFile/2022-12-05/AsOf/TST5/C1S1_1.html); [GAO-06-676 on offshore outsourcing by Medicare/Medicaid/TRICARE contractors](https://www.gao.gov/assets/gao-06-676.pdf) (background, 2006).

### Inferences
- **Are these binding on a hospital's vendor?** Not directly by law. They bind the plan or agency contractor, and then whoever signs the flow-down.
  - The Superior notice shows Texas Medicaid MCOs **do** flow the terms down to network providers (hospitals) and expressly cover IT, "data processing", storage, DR and customer support.
  - A Texas hospital in a Medicaid MCO network would therefore have to impose US-only *work, storage and access* on Poised for any information "obtained through performance of" that MCO agreement. Pre-op assessments of that hospital's Medicaid members very plausibly qualify.
  - Under this flow-down **the hybrid design fails**: processing happens in London and UK staff have remote access. All-London fails too.
- The MA attestation regime **permits** offshore vendors, including the hybrid, but expects disclosure and attestation for offshore access or processing. A hospital in MA networks may ask Poised to complete such an attestation.
- DFARS, VA and TRICARE terms bind only entities with direct federal contracts. They cover government data, not a private hospital's own clinical records about TRICARE or VA beneficiaries. A typical private hospital vendor is not bound, unless the customer is a federal facility or the hospital is performing a DoD or VA contract with data owned by the government. That case is an inference; I found no source addressing it directly.

### Gaps
- I did not obtain primary CMS memo text (HPMS offshore subcontracting guidance) or verify that 42 CFR 422.503 itself contains the attestation requirement. McDermott cites it, but the requirement may live in CMS sub-regulatory guidance.
- I did not verify verbatim primary UMCC §§4.10–4.11 text (the download failed), nor the Wisconsin, Arizona, Ohio, Missouri, New Jersey or Alaska primary contract text.
- I did not find the VA Handbook 6500.6 text or any DHA/TRICARE offshore policy.

---

## Q5. DOJ "bulk sensitive data" rule (28 CFR Part 202) and PADFA: do they restrict access from the UK?

### Takeaway
Both regimes treat **access** (not just storage or transfer) as the regulated act. However, both apply **only to countries of concern / foreign adversaries**: China (including Hong Kong and Macau), Cuba, Iran, North Korea, Russia, Venezuela. **The UK is not covered**, and a UK vendor with UK-resident staff is not a "covered person" unless it is owned by, or employs people resident in, a country of concern. Neither rule restricts the hybrid or all-London designs as such.

### Cited Findings
- **[LAW]** 28 CFR 202.601(a): the countries of concern are *"(1) China; (2) Cuba; (3) Iran; (4) North Korea; (5) Russia; and (6) Venezuela."* — [eCFR 28 CFR 202.601](https://www.ecfr.gov/current/title-28/section-202.601)
- **[LAW]** 28 CFR 202.211(a) defines a "covered person" as an entity 50%+ owned by countries of concern or covered persons, or organised or principally based in one; an individual employee or contractor of a country of concern or covered entity; an individual *"primarily a resident in the territorial jurisdiction of a country of concern"*; or a person the AG designates.
  - Example 3 states that citizens of a country of concern *"primarily resident in a third country, such as Russian citizens primarily resident in a European Union country … would not be"* covered persons.

  — [eCFR 28 CFR 202.211](https://www.ecfr.gov/current/title-28/section-202.211)
- **[LAW]** 28 CFR 202.201: *"The term access means logical or physical access, including the ability to obtain, read, copy, decrypt, edit, divert, release, affect, alter the state of, or otherwise view or receive, in any form, including through information systems, information technology systems, cloud-computing platforms, networks, security systems, equipment, or software. For purposes of determining whether a transaction is a covered data transaction, access is determined without regard for the application or effect of any security requirements."* — [eCFR 28 CFR 202.201](https://www.ecfr.gov/current/title-28/section-202.201)
- **[LAW]** 28 CFR 202.258 defines a "vendor agreement" as one in which a person provides goods or services, *"including cloud-computing services"*. Example 2 is close to Poised's situation: *"A medical facility in the United States contracts with a company headquartered in a country of concern to provide IT-related services. … The medical facility has bulk personal health data on its U.S. patients. The IT services provided under the contract involve access to the medical facility's systems containing the bulk personal health data. This vendor agreement is a restricted transaction."* — [eCFR 28 CFR 202.258](https://www.ecfr.gov/current/title-28/section-202.258)
- **[LAW/GUIDANCE]** The rule took effect on 8 Apr 2025. The bulk threshold for personal health data is 10,000 US persons. — [UNC Research DOJ bulk data rule page](https://research.unc.edu/compliance/library/doj-data/); [Cornell Research Services](https://researchservices.cornell.edu/policies/department-justice-doj-bulk-data-rule)
- **[LAW] PADFA (Protecting Americans' Data from Foreign Adversaries Act of 2024)**, signed 24 Apr 2024 and effective 23 Jun 2024, enforced by the FTC. It prohibits **"data brokers"** from making personally identifiable sensitive data of US individuals available to *foreign adversary countries* (North Korea, China, Russia, Iran) or entities they control. — [Akin Gump alert](https://www.akingump.com/en/insights/alerts/protecting-americans-data-from-foreign-adversaries-act-of-2024); [Baker McKenzie](https://connectontech.bakermckenzie.com/u-s-prohibits-data-brokers-from-making-u-s-sensitive-data-available-to-foreign-adversaries)

### Inferences
- The DOJ rule is the clearest example in US law of **access being regulated the same as storage or transfer**, regardless of encryption. That scope is limited to the six countries of concern.
- For Poised (UK-owned, UK staff, AWS), the DOJ rule is a **due-diligence item, not a blocker**. Hospitals with more than 10,000 patients' health data will ask whether any owner, staff member, contractor or sub-processor is a covered person, for example offshore developers in China or Russia, or investors from a country of concern. Keep that answer "no".
- PADFA is unlikely to apply. Poised is a service provider, not a data broker selling data to foreign adversaries.

### Gaps
- I did not fetch PADFA's statutory definition of "data broker" (including any service-provider exclusion) from the US Code; the analysis above relies on secondary summaries.
- I found no DOJ NSD guidance specific to non-concern-country (for example UK) vendors. None is expected, because they are outside scope.

---

## Q6. Practice: do US hospitals' procurement, BAAs and security questionnaires require US-only hosting and US-only support access even where the law does not?

### Takeaway
Yes, often, but not uniformly. Many US health-system BAAs ban offshore **storage** without consent. Some go further and ban offshore **access or processing**; some explicitly **allow** remote access to US-stored data with approved controls. Payers and Medicaid networks are the strictest. A UK vendor should expect US-storage clauses almost everywhere and US-only-access clauses at a meaningful share of customers.

### Cited Findings
- **[CONTRACT] UAB (University of Alabama at Birmingham) Health System standard BAA** (form F# 231r10, issued 11/01/20), §2.1.5. The business associate must ensure subcontractors *"do not store PHI beyond the borders of the United States of America. **Remote access to data stored within the borders of the United States of America is allowed** as long as appropriate HIPAA technical controls are utilized. These controls must be approved by the UAB Health System Information Services (HSIS) Information Security Team. Printing of PHI beyond the borders of the United States of America is not allowed."* — [UAB Business Associate Agreement (5.10.21)](https://www.ccts.cme.uab.edu/financialaffairs/images/documents/forms/UAB_Business_Associate_Agreement_5.10.21.pdf). Verified from the PDF text.
  - This is a real-world example of a US academic health system contractually blessing the **hybrid pattern**: US storage with offshore remote access.
- **[CONTRACT] University of Houston System general BAA (2025)**, §2.5: *"Nor may PHI be stored outside the United States without prior written consent from the Covered Entity."* This is storage-only. — [UH System General BAA](https://uhsystem.edu/offices/legal-affairs/contract-administration/contract-documents/miscellaneous-contracts-and-forms/_2025/general-business-associate-agreement.pdf). Verified from the PDF text.
- **[CONTRACT]** Other BAA clauses seen in circulation (from a search-result summary of LawInsider and health-system BAAs; not individually verified per document):
  - *"Business Associate shall not … store, transmit or make available any PHI to any entity or individual outside the United States without prior written consent of Covered Entity"*;
  - *"No person physically located outside of the United States shall have access to PHI unless Customer provides prior written consent"*;
  - *"Neither Business Associate nor any subcontractor … shall store, process or otherwise handle PHI outside of the United States."*

  — [LawInsider "Offshore Work" clauses](https://www.lawinsider.com/clause/offshore-work/_1); [Covenant Health BAA](https://www.covenanthealth.com/wp-content/uploads/2023/04/Business-Associate-Agreement-vers.-9.7.2021-002.pdf) (could not download)
- **[COMMENTARY]** McDermott: *"Many healthcare providers' and health plans' contracts include restrictions on their business associates"*, prohibiting PHI from leaving US territory or requiring enhanced security. — [McDermott](https://www.mcdermottlaw.com/?p=322505)
  - A search summary of the same and related material described payer and provider-network terms that *"prohibit PHI from leaving US territory or accessing PHI outside the US"*. — [McDermott](https://www.mcdermottlaw.com/?p=322505); [AccountableHQ](https://www.accountablehq.com/post/hipaa-and-data-localization-do-you-need-to-keep-phi-in-the-u-s)
- **[CONTRACT]** Payer flow-downs (Superior HealthPlan; MA offshore attestations) reach hospitals and, through them, their vendors (see Q4). — [Superior HealthPlan](https://superiorhealthplan.com/newsroom/important-compliance-with-us-only-performance-and-data-handling-requirements-07152025.html)
- **[MARKET]** Some US health-data vendors market "all healthcare data operations in the United States" as a selling point (for example 1upHealth, "Why 1upHealth keeps all healthcare data operations in the United States"). — [1upHealth blog](https://1up.health/blog/why-1uphealth-keeps-all-healthcare-data-operations-in-the-united-states/). The page body could not be read, so only the title is cited.

### Inferences
- **Storage-in-US is close to table stakes** for US hospital SaaS: it is the law in TX and FL and a common BAA term elsewhere. All-London will struggle commercially even in states where it is legal.
- **Offshore access and processing is negotiable but frequently restricted.** Expect three customer types:
  1. storage-only clauses, where the hybrid is fine;
  2. "remote access allowed with our security team's approval" (UAB-style), where the hybrid is fine if Poised documents controls;
  3. "no access or processing outside the US without consent", or a Medicaid/payer flow-down, where the hybrid needs written consent or fails.
- Security questionnaires (SIG, HITRUST-based, custom) routinely ask where data is stored *and* where support personnel are located. A clear, honest answer that data at rest is in a US region, compute is in the UK, and support is in the UK with logged, just-in-time access will be read against the customer's own BAA template.

### Gaps
- I found no survey data quantifying how many US hospitals require US-only *access* (as opposed to storage). The evidence is anecdotal: individual BAAs and law-firm statements.
- I did not obtain standard questionnaire text (for example SIG or the HITRUST questions on data location and offshore personnel) to quote.

---

## Q7. Is "remote access by foreign staff" or "processing in another country" treated differently from storage anywhere in US law? (And: is the user's belief right that residency rules are only about storage?)

### Takeaway
**Partly right.** The two US *statutes* that localise provider health records (Texas §183.002(a) and Florida §408.051(3)) regulate only **where records are physically maintained**, and neither expressly restricts offshore access or processing. HIPAA regulates neither. But US rules **do** reach access and processing in three places:
- the DOJ bulk-data rule (access = covered, but only for countries of concern);
- Medicaid managed-care contracts and their flow-downs (Texas: no remote access from outside the US and no offshore data processing; Wisconsin: no offshore work involving access);
- Medicare Advantage attestations (offshore access or processing must be disclosed and attested, not banned).

So "never about processing" is wrong as a general statement about US obligations. It is accurate only for the state statutes.

### Cited Findings
- **Storage-only (statute):**
  - Texas §183.002(a), "physically maintained in the United States" — [SB 1188](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm)
  - Florida §408.051(3), "physically maintained in the continental United States or its territories or Canada" — [Fla. Stat. 408.051](https://www.flsenate.gov/Laws/Statutes/2025/408.051)
  - DFARS 252.239-7010(b)(3), "maintain within the United States … all Government data" — [eCFR](https://www.ecfr.gov/current/title-48/section-252.239-7010)
- **Neither storage nor access (federal privacy law):** HIPAA. OCR says the rules "do not include requirements specific to … ePHI processed or stored … outside of the United States". — [HHS OCR cloud FAQ Q9](https://web.archive.org/web/2025/https://www.hhs.gov/hipaa/for-professionals/special-topics/health-information-technology/cloud-computing/index.html)
- **Access-inclusive (regulation, country-limited):** 28 CFR 202.201 "access means logical or physical access … to … view or receive", applying only to the six countries of concern. — [eCFR 202.201](https://www.ecfr.gov/current/title-28/section-202.201); [202.601](https://www.ecfr.gov/current/title-28/section-202.601)
- **Access- and processing-inclusive (contract):**
  - Texas Medicaid MCO flow-down: "No individual is permitted to access information, systems, or deliverables remotely from a location outside the United States", covering "Data processing …". — [Superior HealthPlan](https://superiorhealthplan.com/newsroom/important-compliance-with-us-only-performance-and-data-handling-requirements-07152025.html)
  - Wisconsin: "performing work outside the US that involves access to … patient health … information" — [McDermott](https://www.mcdermottlaw.com/?p=322505)
- **Access- and processing-triggered disclosure (CMS MA):** attestation for offshore subcontractors that "receive, process, transfer, handle, store, or access" PHI. — [Shumaker](https://www.shumaker.com/insight/offshoring-patient-data-what-health-care-providers-need-to-know-now/)
- **Interpretive split on Texas:** McDonald Hopkins and Buchalter say offshore access is allowed as long as there is no offshore storage, copying or caching. Hall Render says it is "ambiguous" whether offshore view-only access is caught. — [McDonald Hopkins](https://www.mcdonaldhopkins.com/insights/news/s-b-1188-texas-ehr-law-data-localization-ai-and-access-requirements); [Buchalter](https://www.buchalter.com/insights/from-phi-to-ai-what-texas-sb-1188-means-for-healthcare-entities-and-vendors/); [Hall Render](https://hallrender.com/2025/08/19/new-texas-law-on-ehr-data-localization-and-ai/)

### Inferences
- Even the storage-only statutes have a **"no copies offshore" corollary** in practitioner guidance (no caching, backup, replication or DR offshore). The line that matters for the hybrid is whether anything **persists** in London, not merely whether code runs there.

### Gaps
- No US court or regulator has ruled on whether transient in-memory processing abroad counts as offshore "maintenance" or "storage".

---

## Q8. Verdict: would (1) the hybrid design and (2) all-London hosting comply, for a typical US hospital customer?

### Takeaway
- **(2) All-London:**
  - lawful under HIPAA;
  - **unlawful for Texas and Florida hospitals** (state storage statutes);
  - contractually blocked at many other US hospitals (BAA storage clauses).
  - It is not a viable US default.
- **(1) Hybrid (DB and S3 in a US region; London compute; UK support):**
  - lawful under HIPAA;
  - **likely compliant with the Texas and Florida statutes on their text**, provided no PHI persists outside the US;
  - legally untested, with one law firm flagging ambiguity on offshore access in Texas;
  - **blocked by contract** wherever a customer's BAA, or a Medicaid/payer flow-down, bans offshore access or processing. Texas Medicaid network hospitals are the clearest case.
  - For those customers, only full US hosting (compute and data in the US) plus US-only support access would satisfy the terms. The alternative is written consent from the customer.

### Cited Findings
- HIPAA allows offshore storage and processing with a BAA and risk analysis. — [HHS OCR cloud FAQ Q9](https://web.archive.org/web/2025/https://www.hhs.gov/hipaa/for-professionals/special-topics/health-information-technology/cloud-computing/index.html)
- Texas requires US physical maintenance from 1 Jan 2026, with penalties up to $5k/$25k/$250k per violation and licensing discipline. — [SB 1188](https://capitol.texas.gov/tlodocs/89R/billtext/html/SB01188F.htm)
- Florida requires continental-US, territories or Canada storage, with a licensure affidavit under penalty of perjury. — [Fla. Stat. 408.051](https://www.flsenate.gov/Laws/Statutes/2025/408.051); [408.810](https://www.flsenate.gov/Laws/Statutes/2025/408.810)
- Offshore access is "still permitted" under SB 1188 if not "storage, copying, or caching of EHR data on non-U.S. servers or devices". — [McDonald Hopkins](https://www.mcdonaldhopkins.com/insights/news/s-b-1188-texas-ehr-law-data-localization-ai-and-access-requirements)
  - Contrast Hall Render: "ambiguous" on offshore view-only access. — [Hall Render](https://hallrender.com/2025/08/19/new-texas-law-on-ehr-data-localization-and-ai/)
- The Texas Medicaid MCO flow-down bans remote offshore access and offshore data processing. — [Superior HealthPlan](https://superiorhealthplan.com/newsroom/important-compliance-with-us-only-performance-and-data-handling-requirements-07152025.html)
- An example health-system BAA allows remote access to US-stored PHI with approved controls. — [UAB BAA](https://www.ccts.cme.uab.edu/financialaffairs/images/documents/forms/UAB_Business_Associate_Agreement_5.10.21.pdf)
- The DOJ rule does not cover the UK. — [28 CFR 202.601](https://www.ecfr.gov/current/title-28/section-202.601)

### Inferences

Compliance matrix (my synthesis):

| Regime | Binding? | Covers | All-London | Hybrid (US DB/S3, UK compute + support) |
|---|---|---|---|---|
| HIPAA Privacy/Security Rules | Law | No location rule (storage or processing) | OK with BAA + risk analysis | OK with BAA + risk analysis |
| Texas H&S Code §183.002(a) (SB 1188) | Law (TX covered entities; arguably Poised too) | Storage ("physically maintained") | **Fails** | Likely OK on text. Risk: caching/copies in London, untested; Hall Render ambiguity on offshore access |
| Texas §183.002(b) | Law | Role-based access (not location) | Need-to-know support access | Need-to-know support access |
| Florida §408.051(3) + §408.810(14) | Law (FL-licensed providers using CEHRT) | Storage (US, territories or Canada) | **Fails** | Likely OK on text |
| DOJ 28 CFR 202 / PADFA | Law | Access + storage + transfer, **countries of concern only** | Not triggered (UK) | Not triggered (UK). Keep staff and sub-processors out of countries of concern |
| Texas Medicaid UMCC / MCO provider flow-downs | Contract | Work, storage, transfer, **remote access, data processing** | **Fails** | **Fails** (UK processing and support) unless HHSC/MCO approval |
| Wisconsin (and reportedly AZ, OH, AK, MO, NJ) Medicaid | Contract / EO | Offshore work or access (varies) | Likely fails | Likely fails where access or work is restricted |
| Medicare Advantage offshore attestation | CMS contract guidance | Disclosure if offshore receive/process/store/**access** | Allowed with attestation | Allowed with attestation |
| DFARS 252.239-7010 / VA / TRICARE | Federal contract only | DoD: storage of Government data | N/A to a private hospital's own records | N/A to a private hospital's own records |
| Hospital BAAs / questionnaires | Contract | Varies: storage-only → access-allowed-with-approval → no offshore access/processing | Usually fails | Passes storage-only and UAB-type clauses; needs consent where access/processing is banned |

- **Engineering conditions for the hybrid to hold up** (inference):
  - no PHI in London logs, metrics, traces or error tools;
  - no PHI in London queues, caches, Lambda /tmp, SES/SMS payload stores, or backups and snapshots;
  - Cache-Control: no-store on PHI API responses so non-US edge locations don't cache them;
  - DR/replicas only in US regions;
  - support access that is just-in-time, logged and least-privilege, with no local downloads or printing in the UK (UAB's clause bans printing offshore).
- **Commercial reality:** the hybrid will satisfy the laws and many customers, but a material share of US hospitals (Medicaid-heavy Texas systems, payer-integrated systems, and strict BAAs) will require US compute and US-only support. Poised should plan for a full US deployment (compute and data in the US) as the eventual US offering, or at least as an option.
- The user's premise ("residency rules are only about storage, never about processing") holds for **US statutes** (TX, FL), but **not** for US *contractual* regimes or the DOJ rule. In US hospital sales, contracts are often the binding constraint.

### Gaps
- No official interpretation of SB 1188 or Fla. §408.051(3) on in-memory offshore processing exists.
- I did not determine whether a pre-op assessment platform's data is an "electronic health record" (TX, undefined) or within "qualified electronic health records" (FL, defined by capabilities a pre-op tool may lack). The conservative assumption is that both apply.
- I could not quantify how common US-only-access clauses are among US hospitals.
- This is legal research, not legal advice. A US healthcare regulatory lawyer should confirm the Texas and Florida analyses before Poised relies on the hybrid design.
