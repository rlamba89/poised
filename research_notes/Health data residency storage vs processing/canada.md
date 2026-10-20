# Canada: health data residency — storage vs processing/access (as of Oct 2026)

Scope: hospital / healthcare-provider patient data; a UK software vendor (Poised) acting as a service provider. Design (1) "hybrid" = Postgres + S3 in an AWS Canadian region, app compute in AWS London reading/writing over the network, UK staff support access. Design (2) = everything in London.

Legend used below: **[LAW]** binding statute/regulation; **[REG-GUIDE]** regulator / government guidance; **[PROCUREMENT]** contract, policy or procurement practice.

## Q1. Federal: PIPEDA, OPC guidance, and the Treasury Board (Protected B) rules

### Takeaway
PIPEDA has no data-localisation rule. The OPC's 2009 guidelines (reaffirmed in 2019) treat a transfer to a foreign service provider for processing as a "use", not a "disclosure": no extra consent is needed, but the organisation stays accountable, must protect the data "comparably" by contract, and must tell individuals their data "may be processed in a foreign country". The federal Protected B residency rule binds only Government of Canada departments, not hospitals.

### Cited Findings
- [REG-GUIDE] OPC, *Guidelines for Processing Personal Data Across Borders* (27 Jan 2009): "PIPEDA does not prohibit organizations in Canada from transferring personal information to an organization in another jurisdiction for processing." — [OPC 2009 Guidelines](https://www.priv.gc.ca/en/privacy-topics/airports-and-borders/gl_dab_090127/)
- [REG-GUIDE] Same guidelines: "A transfer for processing is a 'use' of the information; it is not a disclosure." — [OPC 2009 Guidelines](https://www.priv.gc.ca/en/privacy-topics/airports-and-borders/gl_dab_090127/)
- [LAW as quoted by OPC] PIPEDA Principle 4.1.3: "An organization is responsible for personal information in its possession or custody, including information that has been transferred to a third party for processing. The organization shall use contractual or other means to provide a comparable level of protection while the information is being processed by a third party." The OPC says "comparable" means "generally equivalent". — [OPC 2009 Guidelines](https://www.priv.gc.ca/en/privacy-topics/airports-and-borders/gl_dab_090127/)
- [REG-GUIDE] Notice duty: organisations must "make it plain", in "clear and understandable" language and at the time of collection, that individuals' "information may be processed in a foreign country and that it may be accessible to law enforcement and national security authorities." — [OPC 2009 Guidelines](https://www.priv.gc.ca/en/privacy-topics/airports-and-borders/gl_dab_090127/)
- [REG-GUIDE] Consent: "Assuming the information is being used for the purpose it was originally collected, additional consent for the transfer is not required." — [OPC 2009 Guidelines](https://www.priv.gc.ca/en/privacy-topics/airports-and-borders/gl_dab_090127/)
- [REG-GUIDE, history] In April 2019 the OPC proposed reversing this position, so that transfers to service providers would be "disclosures" needing consent. After stakeholder opposition it closed the consultation, and its 2009 guidelines "will remain unchanged under the current law". — [Goodmans](https://goodmans.ca/insights/article/Privacy_Commissioner_Retains_Original_Policy_on_Cross_Border_Transfers_of_Personal_Information); [Osgoode / The Court, Oct 2019](https://www.yorku.ca/osgoode/thecourt/2019/10/16/opc-reaffirms-2009-consent-standard-for-cross-border-data-processing/); [Bennett Jones, Apr 2019](https://www.bennettjones.com/Insights/Blogs/2019/04/Changes-to-Crossborder-Data-Flow-Consent-Requirements)
- [PROCUREMENT/POLICY – federal government only] Requirement 4.4.3.14 of the Treasury Board *Directive on Service and Digital* replaced the older *Direction for Electronic Data Residency*. It applies to initiatives that need storage or transmission of Protected B, Protected C and classified **Government of Canada** electronic data, and the departmental CIO approves any storage outside Canada. — [Canada.ca Direction for Electronic Data Residency (search summary; the page itself returned 404 when fetched)](https://www.canada.ca/en/government/system/digital-government/digital-government-innovations/cloud-services/direction-electronic-data-residency.html)
- [REG-GUIDE – federal] The GC White Paper *Data Sovereignty and Public Cloud* (2018) states that "as long as a CSP that operates in Canada is subject to the laws of a foreign country, Canada will not have full sovereignty over its data". It names the US FISA as allowing compelled disclosure "regardless of the data's location". It also notes that data must be decrypted to be processed. — [GC White Paper](https://www.canada.ca/en/government/system/digital-government/digital-government-innovations/cloud-services/digital-sovereignty/gc-white-paper-data-sovereignty-public-cloud.html)

### Inferences
- Under PIPEDA, the UK app tier processing data in memory, and UK support staff accessing it, are both a "transfer for processing" (a use). Both are lawful with a contract giving comparable protection and with transparency/notice. Storage location makes no legal difference under PIPEDA.
- PIPEDA is probably not the main law for public hospitals. They fall under provincial public-sector and health-information statutes, covered below. The TBS Protected B rule does not bind provincial hospitals, but hospital procurement teams often borrow the "Protected B" vocabulary. The GC's own view, that residency does not equal sovereignty, also undercuts the idea that storing data in Canada while processing it abroad solves the foreign-access concern.

### Gaps
- I did not verify whether a successor to Bill C-27 (CPPA) had been enacted or tabled by Oct 2026. The McCarthy "Doing Business in Canada 2026" page I fetched said nothing on federal reform.
- I could not retrieve the current text of Directive on Service and Digital 4.4.3.14: the TBS site rejected the request and the Canada.ca page returned 404. Its exact wording ("computing facilities located within the geographic boundaries of Canada…") is unverified here.

## Q2. Nova Scotia: PIIDPA (and PHIA)

### Takeaway
Nova Scotia is the strictest regime and explicitly regulates **access**, not just storage. Public bodies (including the health authority) and their service providers must ensure personal information is "stored only in Canada **and accessed only in Canada**". The exceptions are individual consent in a prescribed manner, a disclosure the Act allows, or a decision by the head of the public body that foreign storage or access "is to meet the necessary requirements of the public body's operation", reported annually to the Minister. Service providers face offences of up to $500,000 for a corporation.

### Cited Findings
- [LAW] PIIDPA s.5(1): "A public body shall ensure that personal information in its custody or under its control and a service provider or associate of a service provider shall ensure that personal information in its custody or under its control is stored only in Canada and accessed only in Canada, unless (a) where the individual the information is about has identified the information and has consented, in the manner prescribed by the regulations, to it being stored in or accessed from, as the case may be, outside Canada; (b) where it is stored in or accessed from outside Canada for the purpose of disclosure allowed under this Act; or (c) the head of the public body has allowed storage or access outside Canada pursuant to subsection (2)." — [PIIDPA, S.N.S. 2006, c.3 (as enacted PDF)](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf); [Bill 19 page](https://nslegislature.ca/legc/bills/60th_1st/3rd_read/b019.htm)
- [LAW] s.5(2): "The head of a public body may allow storage or access outside Canada of personal information in its custody or under its control, subject to any restrictions or conditions the head considers advisable, if the head considers the storage or access is to meet the necessary requirements of the public body's operation." — [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- [LAW] s.5(3): a head who allows storage or access outside Canada must, "within ninety days after the end of that year, report to the Minister all such decisions made during that year, together with the reasons therefor." — [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- [LAW] s.5(4): "In providing storage, access or disclosure of personal information outside Canada, a service provider shall only collect and use such personal information that is necessary to fulfill its obligation as a service provider, and shall at all times make reasonable security arrangements…" — [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- [LAW] Definitions (s.2):
  - "service provider" is a person who "(i) is retained under a contract to perform services for a public body, and (ii) in the course of performance of the services, uses, discloses, manages, stores or accesses personal information in the custody or under the control of a public body".
  - "associate" covers officers, affiliates, "a subcontractor, or further subcontractor", and their employees, "to or through whom access is made available to personal information".
  - "public body" means a public body under the Freedom of Information and Protection of Privacy Act, including municipalities.
  - s.2(3): PIIDPA prevails over any conflicting enactment.
  - [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- [LAW] s.3 applies the Act to every public body, its directors, officers and employees, and "all employees and associates of a service provider". — [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- [LAW] s.9(4) lets the head allow an employee "to transport personal information outside Canada temporarily" on "a computer, a cell phone or another mobile electronic device" when it is necessary for their duties. — [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- [LAW] s.6: anyone covered who receives a "foreign demand for disclosure" (a subpoena, warrant, order or request from a foreign court or authority) must immediately notify the Minister. — [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- [LAW] s.13: a service provider, employee or associate who, "contrary to subsection 5(1), stores or allows access to personal information" commits an offence. Maximum fines are $2,000 for an individual, $25,000 for an individual service provider and $500,000 for a corporation. Officers or directors who authorise or acquiesce are also guilty. Due diligence is a defence. — [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- [REG-GUIDE] NS Government FAQ:
  - PIIDPA covers "any government department, (or government-appointed agency, board, or commission), regional education centre, university, community college, district health authority, or children's aid society".
  - Service providers are subject to it if their work involves personal information.
  - Head approvals must be reported to the Minister of Justice with reasons.
  - [NS PIIDPA FAQ](https://novascotia.ca/just/iap/piidpaquest.asp)

### Inferences
- **The hybrid design does not escape PIIDPA.** London application servers reading and writing NS patient data are "access" from outside Canada. So is UK staff support access. Both breach s.5(1) unless an exception applies. Storing data at rest in ca-central-1 satisfies only the "stored" half of the test.
- In practice the route is a **s.5(2) decision by the head** of the public body (e.g. Nova Scotia Health / IWK), with conditions, reported annually. The same mechanism would be needed for all-London hosting, covering both storage and access. So legally the hybrid's only gain in NS is a narrower request (access only), which a head may find easier to approve.
- Poised would be a "service provider", and AWS a sub-contractor, i.e. an "associate". Poised is directly exposed to the s.13 offence (up to $500k) if it allows access from outside Canada without an authorisation.

### Gaps
- I read the **as-enacted 2006 text**. I did not verify against the current consolidation whether PIIDPA was amended later, e.g. the names of the health authorities, or regulations on the manner of consent. The search results mention a 59th Assembly Bill 16 (nslegislature.ca/legc/bills/59th_2nd/1st_read/b016.htm), which I did not open.
- I did not confirm from the current NS FOIPOP schedule that Nova Scotia Health Authority and IWK are "public bodies". The FAQ's "district health authority" wording predates the 2015 merger.
- I did not check how often NS heads grant s.5(2) approvals for vendor support or SaaS (the annual PIIDPA reports exist but were not fetched).
- I did not check NS PHIA (custodian-level rules) for any cross-border provision. PIIDPA would not reach private custodians such as physician offices, which are not public bodies.

## Q3. British Columbia: FIPPA s.30.1 repeal and what applies now

### Takeaway
BC's old "stored only in Canada and accessed only in Canada" rule (s.30.1) was **repealed on 25 Nov 2021**. Under s.33.1, a public body may now disclose personal information outside Canada "only if the disclosure is in accordance with the regulations". The only regulation, B.C. Reg. 294/2021, requires an assessment within the PIA where **sensitive** personal information is "disclosed to be **stored** outside of Canada". The BC OIPC says any disclosure outside Canada demands "a very high level of rigour" under s.30 security duties. One factor is "whether a reasonable alternative is available within Canada".

### Cited Findings
- [LAW] FIPPA ss.30.1–30.2 are "Repealed" by 2021-39-17. Section 33.2 is also repealed. — [BC FIPPA (bclaws)](https://www.bclaws.gov.bc.ca/civix/document/id/complete/statreg/96165_00_multi)
- [LAW] FIPPA s.33.1: "A public body may disclose personal information outside of Canada only if the disclosure is in accordance with the regulations, if any, made by the minister responsible for this Act." — [BC FIPPA](https://www.bclaws.gov.bc.ca/civix/document/id/complete/statreg/96165_00_multi)
- [LAW] FIPPA s.30: "A public body must protect personal information in its custody or under its control by making reasonable security arrangements against such risks as unauthorized collection, use, disclosure or disposal." — [BC FIPPA](https://www.bclaws.gov.bc.ca/civix/document/id/complete/statreg/96165_00_multi)
- [LAW] *Personal Information Disclosure for Storage Outside of Canada Regulation*, B.C. Reg. 294/2021 (deposited 26 Nov 2021):
  - the head of a public body must make an assessment for each programme, project or system "in which personal information that is sensitive is disclosed to be stored outside of Canada";
  - the assessment is made within the s.69 PIA;
  - it does not apply to programmes or systems already in place when it came into force;
  - the consolidation is current to 6 Oct 2026.
  - [B.C. Reg. 294/2021](https://free.bcpublications.ca/civix/document/id/complete/statreg/294_2021)
- [REG-GUIDE] BC Government guidance:
  - "Sensitive personal information is not defined in FOIPPA."
  - The assessment considers the location and method of storage, involvement of a service provider, the likelihood of unauthorised access, and the impact on individuals.
  - The head of the public body makes the final risk-based decision.
  - [BC Guidance on Disclosures Outside of Canada](https://www2.gov.bc.ca/gov/content/governments/services-for-government/information-management-technology/privacy/privacy-impact-assessments/guidance-on-disclosures-outside-of-canada); [FOIPPA Manual s.33.1](https://www2.gov.bc.ca/gov/content/governments/services-for-government/policies-procedures/foippa-manual/disclosure-outside-canada)
- [REG-GUIDE] BC OIPC guidance, *Section 30: reasonable security measures for disclosures outside of Canada*:
  - "public bodies can only disclose personal information if FIPPA authorizes it… The disclosure authority must be in FIPPA, not foreign law."
  - "Disclosure of personal information outside of Canada requires a very high level of rigour."
  - "It is unlikely that a public body would be able to meet its obligations under s. 30 when information under its control is **processed or stored** in a jurisdiction that does not respect the rule of law, has no privacy laws, or those laws are inadequate."
  - Factors include sensitivity ("personal health information is much more sensitive than contact information"), volume, foreseeability and impact of unauthorised access, "whether the personal information is stored by a service provider", and "whether a reasonable alternative is available within Canada".
  - "Disclosure outside Canada will always involve risks that no administrative, technical and contractual controls can eliminate."
  - [BC OIPC guidance (PDF)](https://oipc.bc.ca/guidance-documents/3646)
- [REG-GUIDE, history] The Minister issued separate Directions for ministry and non-ministry public bodies, widening when PIAs are required and requiring the supplementary assessment for storage outside Canada. — [search summary citing Cassels / Lawson Lundell / C. Wilson](https://www.cwilson.com/changes-to-privacy-law-fippa-data-residency-rules-and-privacy-impact-assessments/)
- [LAW, superseded] During COVID-19 (2020), BC temporarily relaxed the s.30.1 restrictions by ministerial order, including for "health care bodies" such as hospitals and regional health authorities. This was superseded by the 2021 repeal. — [Fasken, Apr 2020](https://fasken.com/en/knowledge/2020/04/9-covid-19-bc-relaxes-fippa-personal-information-disclosure-restrictions); [Cassels](https://cassels.com/insights/british-columbia-temporarily-eases-restrictions-on-the-disclosure-of-personal-information-outside-of-canada/)

### Inferences
- **Hybrid in BC:** the data is not "stored outside of Canada", so the Reg. 294/2021 supplementary assessment is arguably not triggered by the storage. Real-time processing in London and UK support access are still plausibly a "disclosure outside of Canada" under s.33.1, needing a s.33 disclosure authority (e.g. disclosure to a service provider for its duties) and a rigorous s.30 assessment. The OIPC explicitly names where data is "processed", not only stored. Transient in-memory processing is arguably not "storage", but a cautious health authority PIA team may not accept that line.
- **All-London in BC:** legally possible since Nov 2021, but it needs the supplementary assessment in the PIA and a head's risk-based decision. The OIPC's "reasonable alternative within Canada" factor works against London hosting when AWS Canadian regions exist. The UK's legal framework (rule of law, UK GDPR) should pass the OIPC's "rule of law / adequate laws" screen.
- BC health authorities are public bodies under FIPPA, so these rules apply to them. (This is inferred from the "health care bodies" references; I did not verify it from Schedule 2.)

### Gaps
- I did not retrieve the text of the current ministerial directions (e.g. for non-ministry public bodies such as health authorities) or any Ministry of Health / PHSA policy that may still require Canadian residency for health data.
- I did not check BC's *E-Health (Personal Health Information Access and Protection of Privacy) Act*, which governs designated health information banks, for any location rule.
- No source I read says explicitly whether remote access from outside Canada, without storage, triggers Reg. 294/2021.

## Q4. Quebec: Law 25 (public and private sector) and the 2023 health information Act (LRSSS)

### Takeaway
Quebec does not mandate in-Quebec storage by statute. It does regulate **processing**, though, not only storage. Under the *Act respecting health and social services information* (LRSSS, 2023 c.5, in force 1 July 2024), a health body that entrusts a person **outside Quebec** with the task to "collect, use, communicate or keep" information on its behalf must first carry out a privacy impact assessment (PIA). It may proceed only if the PIA shows "adequate protection", and it needs a written contract with mandatory clauses. Separately, a government hosting policy (effective 20 May 2026) requires "highly sensitive" public-body data to sit on **sovereign infrastructure**. That policy could be stricter than both designs.

### Cited Findings
- [LAW] LRSSS art. 77 allows a body to communicate information to a person to whom it entrusts a mandate or service contract when necessary. The mandate must be in writing and must provide, "sous peine de nullité" (on pain of nullity):
  - the applicable provisions of the Act;
  - protection measures that comply with the governance rules (art. 90) and the "règles particulières définies par le dirigeant réseau de l'information" (art. 97);
  - that the provider uses only "produits ou services technologiques autorisés par l'organisme pour recueillir le renseignement, le conserver, l'utiliser ou le communiquer lorsque le mandat est exercé ou lorsque le contrat est exécuté **à distance**" (remotely);
  - breach notification, audit rights, return of information, and destruction at the end;
  - notice of any subcontractor, who is bound by the same obligations.
  - [Loi 5 (2023, c.5), Gazette officielle 24 May 2023 (PDF)](https://www.publicationsduquebec.gouv.qc.ca/fileadmin/gazette/pdf_encrypte/lois_reglements/2023F/79764.pdf)
- [LAW] LRSSS art. 78: "Avant de confier un mandat ou de conclure un contrat de service ou d'entreprise impliquant une communication d'un renseignement à l'extérieur du Québec, l'organisme qui le détient doit s'assurer qu'une évaluation des facteurs relatifs à la vie privée a été réalisée… Le mandat ne peut être confié ou le contrat conclu que si l'évaluation démontre que le renseignement bénéficierait d'une protection adéquate, notamment au regard des principes de protection des renseignements personnels généralement reconnus… **Il en est de même lorsque l'organisme confie à une personne ou à un groupement à l'extérieur du Québec la tâche de recueillir, d'utiliser, de communiquer ou de conserver pour son compte un renseignement.**"
  - Translation: before entrusting a mandate or service contract that involves communicating information outside Québec, a PIA is required, and the mandate is allowed only if it shows adequate protection with regard to generally recognised privacy principles. *The same applies when the body entrusts a person or group outside Québec with the task of collecting, **using**, communicating or **keeping** information on its behalf.*
  - [Loi 5 (2023, c.5) PDF](https://www.publicationsduquebec.gouv.qc.ca/fileadmin/gazette/pdf_encrypte/lois_reglements/2023F/79764.pdf)
- [LAW] LRSSS art. 45 (applied to art. 78 "avec les adaptations nécessaires"): for communication outside Québec, the PIA must consider "1° la sensibilité du renseignement; 2° la finalité de son utilisation; 3° les mesures de protection, y compris celles qui sont contractuelles…; 4° le régime juridique applicable dans l'État où ce renseignement serait communiqué". — [Loi 5 PDF](https://www.publicationsduquebec.gouv.qc.ca/fileadmin/gazette/pdf_encrypte/lois_reglements/2023F/79764.pdf)
- [LAW] LRSSS art. 106: a PIA is required for any project to acquire, develop or overhaul technology products or services that involves collecting, keeping, using, communicating or destroying information. Art. 107: bodies must keep a public registry of technology products and services used. — [Loi 5 PDF](https://www.publicationsduquebec.gouv.qc.ca/fileadmin/gazette/pdf_encrypte/lois_reglements/2023F/79764.pdf)
- [LAW] Most LRSSS provisions came into force on 1 July 2024. The Act applies to "public health institutions, private health facilities, laboratories, specialized medical centres…" and the Ministry. "Before agreeing to any mandate or contract that involves communicating HSS information outside of Quebec, the Body must conduct a privacy impact assessment." — [Blakes](https://www.blakes.com/insights/quebec-s-comprehensive-health-information-privacy-law-comes-into-force/)
- [REG-GUIDE / law-firm] The Act "does not prohibit the communication of health and social services information outside of the province of Quebec", but needs a PIA showing adequate protection plus a data-processing agreement. — [Gowling WLG](https://gowlingwlg.com/en/insights-resources/articles/2024/quebec-health-information-act)
- [LAW via government guidance] Public-sector Access Act art. 70.1 (Law 25): a public body communicating personal information outside Quebec, or having a mandate performed outside Quebec, must do a PIA considering sensitivity, purpose, protection measures (including contractual ones) and the legal framework of the destination State. It may proceed if the information will receive "adequate protection", and "Cette communication ou la réalisation d'un mandat doit faire l'objet d'une entente écrite". — [Quebec.ca: communication pour la réalisation d'une tâche à l'extérieur du Québec](https://www.quebec.ca/gouvernement/travailler-gouvernement/normes-gouvernance-pratiques-internes/protection-des-renseignements-personnels/communication-de-renseignements-personnels/communication-realisation-tache-exterieur-quebec)
- [LAW, private sector, secondary] Under the private-sector Act, "Before communicating personal information outside Québec, organizations must conduct a PIA specifically assessing whether the information will receive adequate protection…" — [McCarthy Tétrault, Doing Business in Canada 2026](https://www.mccarthy.ca/en/insights/publications/privacy-laws-doing-business-in-canada-2026)
- [PROCUREMENT/POLICY] Quebec MCN, *Énoncé d'orientations en infonuagique et en hébergement*, effective **20 May 2026** (it replaces the Treasury Board cloud orientations of 8 July 2019):
  - It applies to public bodies under art. 2 of the LGGRI.
  - "Les actifs informationnels qui comportent des données hautement sensibles sont hébergés sur les infrastructures souveraines" (information assets containing highly sensitive data are hosted on sovereign infrastructure). These are the Quebec Government Cloud (NGQ), data centres owned by the Quebec or Canadian governments, or a public cloud designated by the Minister.
  - "Highly sensitive" means Protégé C, Secret or Très secret, or **Protégé B / Confidentiel data judged critical by the chief information officer**.
  - Orientation 2: "Les renseignements personnels sensibles concernant les citoyens doivent bénéficier d'une protection adéquate". Providers outside Quebec must comply with art. 70.1.
  - [Quebec.ca Énoncé d'orientations](https://www.quebec.ca/gouvernement/ministeres-organismes/cybersecurite-numerique/publications/architecture-entreprise-gouvernementale/enonce-orientations-infonuagique-hebergement)
- [PROCUREMENT/POLICY] Santé Québec, which absorbed the CISSS/CIUSSS and university health institutions, states that it is subject to the LGGRI (RLRQ c. G-1.03). — [Santé Québec security policy (PDF hosted by CHUM, 2026)](https://www.chumontreal.qc.ca/sites/default/files/2026-03/SC_Politique_de_securite_Sante_Quebec_officielle.pdf) (search-result summary; I did not read the PDF in full)

### Inferences
- **The Quebec rule explicitly reaches processing.** Art. 78 para. 3 is triggered when a body outside Quebec "uses" information on the health body's behalf, wherever the data is stored. In the hybrid design the database would sit in Canada. (AWS ca-central-1 is, to my understanding, in the Montréal area; I did not verify this in this research.) But Poised's London compute and UK staff "use" the data outside Quebec, so the art. 78 PIA, the adequacy finding and the art. 77 contract are needed **in both designs**. The hybrid buys nothing legally under the LRSSS.
- The UK (UK GDPR / DPA 2018, an independent judiciary) should be able to support an "adequate protection… generally recognised principles" finding. That is a judgment for each health body's PIA, not a certainty.
- The bigger risk is the **May 2026 sovereign-hosting orientation**. If a Santé Québec CIO classifies pre-operative patient data as critical Protégé B or Confidentiel, it must sit on NGQ or government data centres, or on a minister-designated public cloud. Then neither AWS ca-central-1 under a UK vendor nor London may qualify. Even where hosting in Canada is enough, a policy aimed at foreign-law exposure is unlikely to be met by Canadian storage plus UK compute.
- Art. 77 3°b's reference to remote performance ("à distance") shows the legislator expected remote vendor access and controls it by contract, not by prohibition.

### Gaps
- I quoted the **as-enacted** Law 5 (2023) from the Gazette. LégisQuébec blocked automated access, so I could not confirm the current consolidated article numbers or any amendments after Bill 15 / Santé Québec.
- I could not open the Fasken analysis of private-sector s.17 (HTTP 403), so the exact current s.17 wording is not quoted.
- I did not verify which "règles particulières" the health-sector "dirigeant réseau de l'information" has issued under art. 97. Contract clauses must comply with them, and they could contain hosting-location rules.
- I did not confirm whether any public cloud region (e.g. AWS ca-central-1) has been "designated by the Minister" as sovereign infrastructure, or how Santé Québec classifies clinical records.

## Q5. Other provinces: Ontario, Alberta, New Brunswick, Newfoundland, Manitoba, Saskatchewan, PEI

### Takeaway
None of the provinces checked imposes a statutory in-Canada storage or access rule on health custodians, except possibly New Brunswick (unconfirmed). Alberta's regulation expressly covers health information "stored **or used**" outside Alberta, and requires a written agreement before it, not a prohibition. Ontario treats giving information to an agent (vendor) as a "use", so its "disclosure outside Ontario" limit is not triggered.

### Cited Findings
- [LAW] **Ontario PHIPA s.6(1):** providing personal health information between a custodian and its agent is "a use by the custodian, and not a disclosure by the person providing the information or a collection by the person to whom the information is provided". **s.37(2):** "Where a custodian may use personal health information, the custodian also may provide the information to the custodian's agents, who may use the information for that purpose on behalf of the custodian." — [PHIPA, S.O. 2004 c.3 Sch. A (via vLex)](https://ca.vlex.com/vid/use-of-personal-health-869211571); official text at [ontario.ca/laws/statute/04p03](https://www.ontario.ca/laws/statute/04p03)
- [LAW] **Ontario PHIPA s.50(1)** opens: "A health information custodian may disclose personal health information about an individual collected in Ontario to a person outside Ontario only if…". It then lists consent and limited exceptions. This is a limit on **disclosure**, not storage or processing. — [ontario.ca/laws PHIPA](https://www.ontario.ca/laws/statute/04p03) (opening words confirmed by exact-phrase search; full clause list not retrieved)
- [LAW] **Alberta Health Information Regulation, AR 70/2001, s.8(4)** (consolidated to AR 119/2026, current as of 2 July 2026): "In order to ensure the privacy and confidentiality of health information that is to be **stored or used** by a person in a jurisdiction outside Alberta or that is to be disclosed to a person in a jurisdiction outside Alberta, the custodian must, prior to the storage, use or disclosure of the information, enter into a written agreement with the person that (a) provides for the custodian to retain control over the health information, (b) adequately addresses the risks associated with the storage, use or disclosure of the health information, (c) requires the person to implement and maintain adequate safeguards…, (d) allows the custodian to monitor compliance…, and (e) contains remedies to address any non-compliance…" — [Alberta HIR (King's Printer PDF)](https://kings-printer.alberta.ca/documents/Regs/2001_070.pdf)
- [LAW] Alberta HIR s.7.2: an information manager agreement under HIA s.66(2) must describe, among other things, "how health information provided to the information manager is to be protected, managed, returned or destroyed". s.8(1)–(3) require written records and periodic assessment of administrative, technical and physical safeguards. — [Alberta HIR](https://kings-printer.alberta.ca/documents/Regs/2001_070.pdf)
- [PROCUREMENT] Alberta Health Services' innovators page (©2022): "It is generally recommended that innovators consider keeping their data operations within Canada". This is a recommendation, not a mandate. — [AHS Connect Care innovators: data privacy, security, residency](https://innovators.connect-care.ca/legal/data-privacy-security-residency)
- [LAW] **PEI Health Information Act** (consolidated to 30 March 2026):
  - s.35: "A custodian may disclose personal health information relating to an individual that is collected in the province to a person outside the province without the consent of the individual only in the circumstances described in subsections 23(7), (10) and (13) or as specified in the regulations."
  - s.42(1): "A custodian may provide personal health information to an information manager for the purpose of processing, storing or destroying the personal health information or providing the custodian with information management or information technology services". s.42(2) requires a written agreement protecting against "unauthorized access to or use or disclosure".
  - s.43: the information manager must comply with the same protection, retention and destruction requirements as the custodian.
  - [PEI Health Information Act (PDF)](https://www.princeedwardisland.ca/sites/default/files/legislation/h-01-41-health_information_act.pdf)
- [LAW, partially verified] **New Brunswick**: General Regulation 2010-112 under PHIPAA has a section headed "Personal health information stored outside Canada" (s.22). Search snippets give **conflicting** paraphrases of it:
  - (a) that information managers providing a public body with IM/IT services "may store personal health information… outside Canada";
  - (b) that custodians must ensure information managers do not store PHI outside Canada unless providing maintenance and technical support.
  - s.21 requires a written custodian–information manager agreement describing services and safeguards.
  - [NB Reg 2010-112 (gnb.ca PDF)](https://www2.gnb.ca/content/dam/gnb/Departments/ag-pg/PDF/RegulationsReglements/2010/2010-112.pdf); [CanLII NB Reg 2010-112](https://www.canlii.org/en/nb/laws/regu/nb-reg-2010-112/latest/nb-reg-2010-112.html) (both blocked automated access; content from search snippets only)
- [Secondary, low confidence] Manitoba PHIA, Saskatchewan HIPA and Newfoundland & Labrador PHIA are described by secondary sources as having no explicit prohibition on storing health information outside Canada. — [search summary; e.g. LearnSignal comparison](https://www.learnsignal.com/blog/canadian-health-privacy-laws-phipa-hia-phia-by-province/)
- [Contradicted claim] Some vendor blogs claim Ontario PHIPA "effectively mandat[es] Canadian storage" and that BC still "prohibits" storage or access outside Canada. These claims conflict with the PHIPA s.6(1)/s.37(2) agent-is-a-use scheme and with the 2021 repeal of BC s.30.1. Treat them as unreliable. — [e.g. Capital Hill Group](https://capitalhillgroup.ca/data-sovereignty-in-canada-by-province/) vs [BC FIPPA](https://www.bclaws.gov.bc.ca/civix/document/id/complete/statreg/96165_00_multi)

### Inferences
- **Ontario:** a UK vendor acting as the hospital's agent or service provider can lawfully store and process PHI in London or in a hybrid. The transfer is a "use", so s.50 is not triggered. The constraints are contract and security (agent obligations, s.12 safeguards), plus whatever hospital or Ontario Health procurement demands.
- **Alberta:** both designs are lawful if a s.8(4) agreement is signed before the out-of-province storage or use. The hybrid still triggers s.8(4), because data is "used" outside Alberta in London. Note that ca-central-1 is outside Alberta too, so s.8(4) applies even to an all-Canada deployment in Montréal.
- **PEI:** no Canada-only rule found. It is ambiguous whether giving PHI to an out-of-province information manager under s.42 is a "disclosure" caught by s.35. A cautious custodian may treat it as one.
- **New Brunswick** is the most likely remaining place for a storage-location rule. It must be checked against the official text before any NB deal.

### Gaps
- I did not obtain the full text of NB Reg 2010-112 s.22 (the official sites and CanLII were blocked). This is a material gap.
- I did not verify from primary text: Manitoba PHIA, Saskatchewan HIPA, Newfoundland & Labrador PHIA, Nova Scotia PHIA, or the territories (Yukon HIPMA, NWT HIA, Nunavut). The secondary sources are low quality.
- I did not retrieve Ontario IPC guidance on cloud or out-of-country storage, nor PHIPA's electronic-service-provider rules (O. Reg. 329/04 s.6).

## Q6. Procurement and policy (stricter than law?)

### Takeaway
Procurement and policy are where Canadian residency is most often demanded. The sources found are Quebec's 2026 sovereign-hosting orientation (binding on Quebec public bodies), AHS's recommendation to keep data in Canada, the BC OIPC's "reasonable alternative within Canada" factor, and the federal Protected B residency rule, which hospitals often borrow as a benchmark. I found no published Ontario Health or AHS mandate requiring that data never be *accessed* from abroad.

### Cited Findings
- [PROCUREMENT/POLICY] Quebec: highly sensitive data "sont hébergés sur les infrastructures souveraines" (NGQ, Quebec or Canadian government data centres, or a minister-designated public cloud). This covers Protégé B / Confidentiel data that the CIO judges critical. Effective 20 May 2026. — [Quebec.ca Énoncé d'orientations](https://www.quebec.ca/gouvernement/ministeres-organismes/cybersecurite-numerique/publications/architecture-entreprise-gouvernementale/enonce-orientations-infonuagique-hebergement)
- [PROCUREMENT/POLICY] Quebec's Government Cloud (NGQ), located in Quebec, has been progressively put into service since September 2023 "for processing and storing sensitive data from public organizations". — [search summary of Quebec MCN sources](https://www.quebec.ca/gouvernement/ministere/cybersecurite-numerique/publications/strategie-gouvernementale-cybersecurite-et-numerique-2024-2028/axe-3-developper-des-infrastructures-technologiques-perennes-et-securitaires)
- [PROCUREMENT] AHS: "It is generally recommended that innovators consider keeping their data operations within Canada." — [AHS Connect Care innovators](https://innovators.connect-care.ca/legal/data-privacy-security-residency)
- [REG-GUIDE] BC OIPC factor: "whether a reasonable alternative is available within Canada". — [BC OIPC guidance](https://oipc.bc.ca/guidance-documents/3646)
- [POLICY – federal departments] Directive on Service and Digital 4.4.3.14 (Protected B/C/classified residency); departmental CIO approval is needed for storage outside Canada. — [Canada.ca (search summary)](https://www.canada.ca/en/government/system/digital-government/digital-government-innovations/cloud-services/direction-electronic-data-residency.html)

### Inferences
- In practice, Canadian hospital RFPs often ask for Canadian data residency, and sometimes for "no access from outside Canada". The second matters most for Poised. A "residency" clause read literally as storage would be met by the hybrid. A "no offshore access" clause would not be met by either design without in-Canada compute and Canada-based support (or tightly controlled, approved break-glass access).

### Gaps
- I did not find primary procurement documents from Ontario Health, Ontario hospitals (e.g. Supply Ontario / OntarioMD), BC health authorities (PHSA), Nova Scotia Health, or Santé Québec stating residency or offshore-access clauses. Their wording, storage-only vs storage-and-access, is unknown.

## Q7. Does "processing in memory abroad" or "remote access by foreign staff" count as a transfer, disclosure or access outside Canada?

### Takeaway
Yes, in most Canadian regimes that regulate the cross-border question at all. **The belief that "residency rules are only about storage, never about processing" is incorrect for Canada.** Nova Scotia regulates "access". Quebec regulates entrusting someone outside Quebec to "use" information. Alberta regulates information "stored or used" outside Alberta. PIPEDA treats foreign processing as a "transfer for processing", which needs transparency and comparable protection. BC's current regulation is the main one keyed specifically to *storage*.

### Cited Findings
- NS PIIDPA s.5(1) "stored only in Canada and accessed only in Canada". s.13 makes it an offence to store "or allow access" contrary to s.5(1). — [PIIDPA 2006 c.3](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)
- Quebec LRSSS art. 78 para. 3: "…confie à une personne ou à un groupement à l'extérieur du Québec la tâche de recueillir, d'utiliser, de communiquer ou de conserver pour son compte un renseignement". — [Loi 5 PDF](https://www.publicationsduquebec.gouv.qc.ca/fileadmin/gazette/pdf_encrypte/lois_reglements/2023F/79764.pdf)
- Alberta HIR s.8(4): "health information that is to be stored or used by a person in a jurisdiction outside Alberta". — [Alberta HIR](https://kings-printer.alberta.ca/documents/Regs/2001_070.pdf)
- OPC: individuals must be told their information "may be processed in a foreign country"; "A transfer for processing is a 'use'". — [OPC 2009 Guidelines](https://www.priv.gc.ca/en/privacy-topics/airports-and-borders/gl_dab_090127/)
- BC Reg. 294/2021 is keyed to sensitive personal information "disclosed to be stored outside of Canada". The BC OIPC nonetheless frames the s.30 risk in terms of where information is "processed or stored". — [B.C. Reg. 294/2021](https://free.bcpublications.ca/civix/document/id/complete/statreg/294_2021); [BC OIPC guidance](https://oipc.bc.ca/guidance-documents/3646)
- Ontario PHIPA s.6(1): providing PHI to an agent is a "use", not a "disclosure". There is therefore no cross-border trigger for agent processing. — [PHIPA via vLex](https://ca.vlex.com/vid/use-of-personal-health-869211571)

### Inferences
- Application servers in London that query a Canadian database pull patient data into UK memory and return it to clinicians and patients. Under NS law that is "access" outside Canada. Under Quebec law it is "use" outside Quebec by a mandated party. Under Alberta law it is "use" outside Alberta. It is not a storage-only matter.
- UK support staff querying production data, or viewing it through admin tools, is "access" (NS) or "use" (Quebec, Alberta) on the same reasoning.

### Gaps
- No regulator decision or case law was found that specifically addresses transient in-memory processing with data at rest in Canada. The conclusions above rest on statutory wording.

## Q8. Verdict: (1) hybrid (DB + S3 in Canada, compute + support in UK) vs (2) all-London

### Takeaway
The hybrid design gives **little legal advantage in Canada**. It does not satisfy Nova Scotia, where offshore access needs a head-of-public-body authorisation either way. It does not avoid Quebec's PIA/adequacy regime, which covers use outside Quebec. It is not legally required in Ontario, Alberta or PEI. Its real benefits are (a) in BC, where it arguably avoids the "stored outside Canada" supplementary assessment, and (b) satisfying storage-only residency clauses in procurement. A deployment that would clear most Canadian buyers is full in-Canada hosting (compute **and** data) with tightly controlled, contractually authorised (or Canada-based) support access.

### Cited Findings (summary table; sources as cited above)

| Jurisdiction | Binding rule | (1) Hybrid | (2) All-London |
|---|---|---|---|
| Federal (PIPEDA) | No localisation; accountability + comparable protection + notice ([OPC](https://www.priv.gc.ca/en/privacy-topics/airports-and-borders/gl_dab_090127/)) | OK with contract + notice | OK with contract + notice |
| Nova Scotia (PIIDPA s.5) | Stored **and accessed** only in Canada unless consent / head's s.5(2) approval ([PIIDPA](https://nslegislature.ca/sites/default/files/legc/PDFs/annual%20statutes/2006/c003.pdf)) | **Non-compliant by default** (London compute + UK support = access); needs s.5(2) head approval | **Non-compliant by default**; needs s.5(2) approval for storage + access |
| British Columbia (FIPPA s.33.1 + Reg 294/2021) | Disclosure outside Canada allowed per regs; supplementary PIA assessment if sensitive PI is *stored* outside Canada; s.30 "very high rigour" ([FIPPA](https://www.bclaws.gov.bc.ca/civix/document/id/complete/statreg/96165_00_multi), [Reg](https://free.bcpublications.ca/civix/document/id/complete/statreg/294_2021), [OIPC](https://oipc.bc.ca/guidance-documents/3646)) | Likely permissible; storage trigger arguably avoided; still needs PIA/s.30 assessment for UK processing/access | Permissible after PIA + supplementary assessment + head's decision; "reasonable Canadian alternative" factor weighs against |
| Quebec (LRSSS arts 77–78; Access Act 70.1) | PIA + "adequate protection" + written contract when a party outside Quebec collects/**uses**/keeps data ([Loi 5](https://www.publicationsduquebec.gouv.qc.ca/fileadmin/gazette/pdf_encrypte/lois_reglements/2023F/79764.pdf)) | Same obligations as all-London (UK use triggers art. 78) | Permissible if PIA finds adequacy + art. 77 contract |
| Quebec policy (MCN Énoncé, May 2026) | Highly sensitive data on sovereign infrastructure ([Énoncé](https://www.quebec.ca/gouvernement/ministeres-organismes/cybersecurite-numerique/publications/architecture-entreprise-gouvernementale/enonce-orientations-infonuagique-hebergement)) | Possibly non-compliant if data is classified highly sensitive | Possibly non-compliant if classified highly sensitive |
| Ontario (PHIPA) | No location rule; agent = use (s.6(1)); s.50 limits *disclosure* outside Ontario ([PHIPA](https://ca.vlex.com/vid/use-of-personal-health-869211571)) | OK (contract/security) | OK (contract/security) |
| Alberta (HIR s.8(4)) | Written agreement before storage **or use** outside Alberta ([HIR](https://kings-printer.alberta.ca/documents/Regs/2001_070.pdf)) | OK with s.8(4) agreement | OK with s.8(4) agreement |
| PEI (HIA ss.35, 42) | Out-of-province disclosure limited; IM provision allowed by agreement ([PEI HIA](https://www.princeedwardisland.ca/sites/default/files/legislation/h-01-41-health_information_act.pdf)) | Probably OK (ambiguity on s.35) | Probably OK (ambiguity on s.35) |
| New Brunswick (Reg 2010-112 s.22) | Section on PHI "stored outside Canada"; text unverified | Unknown, check | Unknown, check |
| MB / SK / NL | No location rule reported (secondary only) | Probably OK | Probably OK |

### Inferences
- **For Poised's architecture decision:**
  - If the first Canadian customer is in Nova Scotia (or is a BC or Quebec public body with strict procurement), the hybrid does not remove the need for a formal approval or PIA. A full in-Canada stack would. That means a Canadian compute tier in ca-central-1 running the same Go Lambdas, which fits the existing `store.ForOrg(orgID)` plan to run a trust's data plane in another region.
  - If the customer is in Ontario or Alberta, all-London is lawful with the right contract. The hybrid adds complexity mainly to satisfy procurement wording.
- The hybrid also has a technical and legal weakness: every request brings patient data into UK memory. "Residency" for Canadian buyers is increasingly about sovereignty, i.e. foreign-law reach and offshore access. The GC white paper and Quebec's 2026 orientation both say this. A buyer who cares about that will not see storage-only residency as an answer.
- Under NS PIIDPA, UK break-glass support access would need explicit s.5(2) authorisation even in an all-Canada deployment, unless support is given by Canada-based staff.

### Gaps
- Verdicts for New Brunswick, Manitoba, Saskatchewan, Newfoundland & Labrador, the territories, and NS PHIA (custodian level) are not verified against primary text.
- Actual RFP and contract clauses used by Canadian health systems (storage-only vs no-offshore-access) were not found.
- This is research notes, not legal advice. Before contracting, a Canadian privacy lawyer should confirm the current consolidated texts (PIIDPA, LRSSS, NB Reg 2010-112).
