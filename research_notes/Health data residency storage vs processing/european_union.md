# EU health data residency: storage vs processing/access (GDPR, EHDS, and national rules of the main EU countries) — as of 8 October 2026

Scope note: Ireland is covered separately. These notes cover EU-level law (GDPR, the EU–UK adequacy decision, EHDS) and national rules in France, Germany, Italy, Spain, the Netherlands, the Nordics and Austria, plus procurement and sovereignty trends. Each rule is tagged as **[STORAGE]**, **[PROCESSING]**, **[ACCESS]** or **[TRANSFER]** (what it regulates), and as **binding law**, **regulator guidance** or **procurement/contract practice**.

Source-access caveat: EUR-Lex, Légifrance and esante.gouv.fr blocked automated fetching in this session. Where the official page could not be read, the notes say so, and the wording comes from a mirror of the official text (streamlex.eu for EHDS), from the official French notification to the Commission (TRIS), or from law-firm summaries. The GDPR article wording quoted below is the official text. It is cited to EUR-Lex, but the EUR-Lex page itself was not reachable in this session.

---

## 1. GDPR Chapter V (Arts 44–50): does it require EU storage, or does it regulate transfers? Is remote access from a third country a "transfer"?

### Takeaway
The GDPR has **no storage-location requirement**. Chapter V regulates **transfers**, and the EDPB treats **remote access from a third country, even just viewing data on screen for support, as a transfer**. Its own Example 11 matches Poised's model almost exactly: a third-country processor remotely accessing data "stored in the EU", for example for support, "results in transfers… under Chapter V". So under the GDPR, "store in the EU, process and support from the UK" is still a transfer. It is lawful because of the UK adequacy decision, not because the data sits in Frankfurt or Paris. Separately, **GDPR Art 9(4) explicitly lets Member States add further conditions or limitations on processing health data**. This is the legal hook for the national localisation rules below.

### Cited Findings
- **[TRANSFER] Binding law, GDPR Art 44:** "Any transfer of personal data which are undergoing processing or are intended for processing after transfer to a third country… shall take place only if… the conditions laid down in this Chapter are complied with by the controller and processor…". The article regulates *transfers*, not where data is stored. — [GDPR, Regulation (EU) 2016/679, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2016/679/oj) (page not reachable in this session; wording from the official text)
- **[TRANSFER] Binding law, GDPR Art 45(1):** "A transfer of personal data to a third country… may take place where the Commission has decided that the third country… ensures an adequate level of protection. Such a transfer shall not require any specific authorisation." — [GDPR, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
- **[PROCESSING] Binding law, GDPR Art 9(4):** "Member States may maintain or introduce further conditions, including limitations, with regard to the processing of genetic data, biometric data or data concerning health." This is the basis on which national health-data localisation and processing-location rules can exist alongside the GDPR. — [GDPR, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2016/679/oj)
- **Regulator guidance, EDPB Guidelines 05/2021 (final version adopted 24 February 2023), three cumulative criteria for a "transfer":** "1) A controller or a processor ('exporter') is subject to the GDPR for the given processing. 2) The exporter discloses by transmission or otherwise makes personal data, subject to this processing, available to another controller, joint controller or processor ('importer'). 3) The importer is in a third country, irrespective of whether or not this importer is subject to the GDPR for the given processing in accordance with Article 3, or is an international organisation." — [EDPB Guidelines 05/2021 v2.0 (PDF)](https://www.edpb.europa.eu/system/files/documents/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf); [EDPB landing page](https://www.edpb.europa.eu/our-work-tools/our-documents/guidelines/guidelines-052021-interplay-between-application-article-3_en)
- **[ACCESS → TRANSFER] Regulator guidance, EDPB 05/2021 §2.2:** "Some examples of how personal data could be 'made available' are by creating an account, granting access rights to an existing account, 'confirming'/'accepting' an effective request for remote access… It should be kept in mind that **remote access from a third country (even if it takes place only by means of displaying personal data on a screen, for example in support situations, troubleshooting or for administration purposes)** and/or storage in a cloud situated outside the EEA offered by a service provider, **is also considered to be a transfer**, provided that the three criteria… are met." — [EDPB Guidelines 05/2021 (PDF)](https://www.edpb.europa.eu/system/files/documents/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **[ACCESS → TRANSFER] Regulator guidance, EDPB 05/2021 Example 11, "Remote access to data in the EU by a third country processor acting on behalf of EU controllers":** "A company in a third country (Company Z), with no establishment in the EU, offers services as a processor to companies in the EU. Company Z, acting as processor on behalf of the EU controllers, is remotely accessing, e.g. for support purposes, **the data which is stored in the EU**. Since Company Z is located in a third country, **such remote access results in transfers of data from the EU controllers to their processor (Company Z) in a third country under Chapter V**." — [EDPB Guidelines 05/2021 (PDF)](https://www.edpb.europa.eu/system/files/documents/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **Regulator guidance, contrast case, EDPB Example 8 (employee travelling):** An employee of a Polish controller who remotely accesses the company's databases from a third country on a business trip is **not** a transfer, because the data is not made available to *another* controller or processor. However, "such processing can still be associated with increased risks because it takes place outside the EU", which must be addressed under Arts 5, 24, 32, 33, 35 and 48. A controller "may very well conclude that extensive security measures are needed – or even that it would not be lawful – to conduct or proceed with a specific processing operation in a third country although there is no transfer situation." — [EDPB Guidelines 05/2021 (PDF)](https://www.edpb.europa.eu/system/files/documents/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **Regulator guidance, EDPB 05/2021 §4, transparency:** "when a controller intends to process personal data outside the EU (although no transfer takes place), this information should as a rule be provided to individuals as part of the controller's transparency obligations". — [EDPB Guidelines 05/2021 (PDF)](https://www.edpb.europa.eu/system/files/documents/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **Regulator guidance, EDPB Example 12 (EU processor subject to third-country law):** When an EU-established subsidiary of a third-country parent processes data only in the EU, that is not a transfer. If it complies with a third-country authority's access request, that would be a transfer, and the controller must assess this under Art 28 before engaging the processor. — [EDPB Guidelines 05/2021 (PDF)](https://www.edpb.europa.eu/system/files/documents/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)

### Inferences
- The user's premise ("residency rules are only about storage, never about processing") is **wrong as a description of the GDPR**. The GDPR does not regulate *where data is stored* at all. It regulates *making data available* to a third-country importer, which explicitly includes remote access and on-screen viewing. In Poised's hybrid design, the London application reads and writes the EU database, and UK staff provide support. That is the EDPB's Example 11 situation: a transfer from the EU hospital (controller) to Poised UK (processor in a third country).
- So moving only the database and S3 to Frankfurt or Paris does **not** avoid Chapter V. Both the hybrid design and all-London hosting involve transfers to the UK. What makes both lawful under the GDPR is the Art 45 UK adequacy decision (section 2), not where the data is stored.
- Art 9(4) means national law can go further than the GDPR on health data, and several Member States have done so (sections 4–6).

### Gaps
- No CJEU judgment was found that rules specifically on whether *remote access* (as opposed to transmission) is a "transfer". The EDPB guidelines are the authoritative interpretation but are not binding law. *Schrems II* (C-311/18) is cited by the EDPB for its risk logic.
- EDPB Recommendations 01/2020 (supplementary measures, including use cases on remote access in clear) were not re-read. They matter only if the UK adequacy decision were to lapse, because adequacy removes the need for SCCs and a transfer impact assessment.

---

## 2. The EU adequacy decision for the UK: status as of October 2026; does it cover health data and processors?

### Takeaway
**Renewed.** On **19 December 2025** the Commission adopted Implementing Decision **C(2025) 8771**, amending Decision (EU) 2021/1772. The UK GDPR adequacy decision now **expires on 27 December 2031**, unless extended. The renewal also **removed the 2021 exclusion for UK immigration-control data**. The decision covers transfers to **controllers and processors** in the UK. It has no carve-out for health data, so health data flows to a UK processor under Art 45 without SCCs. It is subject to ongoing monitoring, a review at least every four years, and possible challenge before the CJEU.

### Cited Findings
- **Binding law:** "COMMISSION IMPLEMENTING DECISION of 19.12.2025 amending Commission Implementing Decision (EU) 2021/1772 of 28 June 2021 pursuant to Regulation (EU) 2016/679… on the adequate protection of personal data by the United Kingdom", C(2025) 8771 final. — [Commission, C(2025) 8771 final (PDF)](https://commission.europa.eu/document/download/b36eedab-43b8-46eb-ac30-a8d1e6e69f51_en?filename=Renewal+UK+adequacy+GDPR.pdf)
- **Expiry:** Article 2 replaces Article 4 of 2021/1772 with: "This Decision shall expire on 27 December 2031, unless extended in accordance with the procedure referred to in Article 93(2) of Regulation (EU) 2016/679." — [C(2025) 8771 final (PDF)](https://commission.europa.eu/document/download/b36eedab-43b8-46eb-ac30-a8d1e6e69f51_en?filename=Renewal+UK+adequacy+GDPR.pdf)
- **Immigration exclusion removed:** Article 1: "Article 1(2) of Implementing Decision (EU) 2021/1772 is repealed." eucrim reports that the renewed decision now covers transfers for UK immigration control purposes, which the 2021 decision excluded. — [C(2025) 8771 final (PDF)](https://commission.europa.eu/document/download/b36eedab-43b8-46eb-ac30-a8d1e6e69f51_en?filename=Renewal+UK+adequacy+GDPR.pdf); [eucrim](https://eucrim.eu/news/commission-renewed-adequacy-decisions-for-data-transfers-to-the-uk/)
- **Controllers and processors covered:** Recital 118: "during the period of application of Implementing Decision (EU) 2021/1772, as amended by this Decision, **transfers from a controller or processor in the European Union to controllers or processors in the United Kingdom may take place without the need to obtain any further authorisation**." Recital 1 states the scope as "personal data transferred from the European Union to the United Kingdom within the scope of that Regulation". — [C(2025) 8771 final (PDF)](https://commission.europa.eu/document/download/b36eedab-43b8-46eb-ac30-a8d1e6e69f51_en?filename=Renewal+UK+adequacy+GDPR.pdf)
- **Health records** are discussed in the assessment (for example "certain health records" among UK GDPR exemption categories). No sectoral exclusion of health data was found in the operative articles. — [C(2025) 8771 final (PDF)](https://commission.europa.eu/document/download/b36eedab-43b8-46eb-ac30-a8d1e6e69f51_en?filename=Renewal+UK+adequacy+GDPR.pdf)
- **Process and history:** The 2021 decisions were adopted on 28 June 2021. A six-month technical extension ran to 27 December 2025. The EDPB adopted opinions on the drafts in October 2025 (Opinion 26/2025 for the GDPR decision). The Commission adopted the renewals on 19 December 2025, each for six years, following the UK Data (Use and Access) Act 2025. — [Hunton](https://www.hunton.com/privacy-and-cybersecurity-law-blog/european-commission-renews-uk-data-adequacy-decisions); [eucrim](https://eucrim.eu/news/commission-renewed-adequacy-decisions-for-data-transfers-to-the-uk/); [EDPB Opinion 26/2025 (PDF), referenced in the decision's footnote 181](https://www.edpb.europa.eu/system/files/2025-10/edpb_opinion_202526_united_kingdom_adequacy_gdpr_en.pdf); [Commission press release IP/25/3059](https://ec.europa.eu/commission/presscorner/detail/en/ip_25_3059)
- **Review and monitoring:** The Commission must monitor UK developments on an ongoing basis, and a periodic review is required at least four years after adoption. — [eucrim](https://eucrim.eu/news/commission-renewed-adequacy-decisions-for-data-transfers-to-the-uk/)
- **Challengeability:** Recital 119 recalls that national DPAs must have a remedy to bring doubts about an adequacy decision before a national court, which may refer to the CJEU (*Schrems I*). — [C(2025) 8771 final (PDF)](https://commission.europa.eu/document/download/b36eedab-43b8-46eb-ac30-a8d1e6e69f51_en?filename=Renewal+UK+adequacy+GDPR.pdf)

### Inferences
- Until at least 27 December 2031, an EU hospital can lawfully let Poised UK process and access its patients' data in the UK under the GDPR without SCCs or a transfer impact assessment. This holds for both the hybrid design and all-London hosting.
- **Concentration risk:** several national rules (Germany §393 SGB V, Berlin LKG, the EHDS Art 87 exception, Spain Art 46 bis) allow third-country processing *only* where an adequacy decision exists. They have no SCC fallback. If UK adequacy were ever suspended or invalidated, UK processing for those customers would become prohibited outright, not merely subject to more paperwork. The hybrid design would then need its compute moved into the EU anyway.

### Gaps
- The exact wording of IP/25/3059 was not fetched (only located via search).
- Incorporation of the renewed decision for the EEA-EFTA states (Norway, Iceland, Liechtenstein) under the EEA Agreement was not checked.

---

## 3. European Health Data Space Regulation (EU) 2025/327: storage/processing location provisions, primary vs secondary use, dates

### Takeaway
The EHDS has two location provisions:
- **Art 86 (primary use, i.e. healthcare):** the EHDS "shall not preclude" a **national** requirement that healthcare providers' **storage** of personal electronic health data be "located within the Union". It is **optional for Member States, covers storage only, applies to the Art 14 priority data categories, and does not itself impose localisation**.
- **Art 87 (secondary use):** health data access bodies, trusted data holders, the Union service, and anyone acting on their behalf **must store and process** the data in the Union, **except** in a third country covered by a GDPR Art 45 adequacy decision (which includes the UK).

General application starts **26 March 2027**. The secondary-use chapter starts **26 March 2029**.

### Cited Findings
- **[STORAGE] Binding law (enabling clause), EHDS Art 86, "Storage of personal electronic health data for primary use":** "…Member States shall ensure that a particularly high level of protection and security is in place when processing personal electronic health data for primary use… In this respect, **this Regulation shall not preclude a requirement under national law, taking into account the national context, that, in cases where personal electronic health data are processed by healthcare providers for the provision of healthcare** or by the national contact points for digital health connected to MyHealth@EU, **the storage of personal electronic health data referred to in Article 14 of this Regulation for the purpose of primary use be located within the Union**, in compliance with Union law and international commitments." — [EHDS Art 86 (streamlex mirror of OJ text)](https://streamlex.eu/articles/ehds-en-art-86/); official text: [Regulation (EU) 2025/327, EUR-Lex](https://eur-lex.europa.eu/eli/reg/2025/327/oj) (not reachable in this session)
- **[STORAGE + PROCESSING] Binding law, EHDS Art 87(1), "Storage of personal electronic health data by health data access bodies and secure processing environments":** "Health data access bodies, trusted health data holders and the Union health data access service **shall store and process personal electronic health data in the Union** when performing pseudonymisation, anonymisation and any other personal data processing operations referred to in Articles 67 to 72, through secure processing environments… That requirement shall apply to any entity performing those tasks on behalf of such bodies, holders or service." — [EHDS Art 87 (streamlex)](https://streamlex.eu/articles/ehds-en-art-87/)
- **[STORAGE + PROCESSING exception] Binding law, EHDS Art 87(2):** "By way of exception from paragraph 1… the data… **may be stored and processed in a third country**, or a territory or one or more specified sectors within that third country, **where such country, territory or sector is covered by an adequacy decision adopted pursuant to Article 45 of Regulation (EU) 2016/679**." — [EHDS Art 87 (streamlex)](https://streamlex.eu/articles/ehds-en-art-87/)
- **Dates, Art 105:** "This Regulation shall apply from 26 March 2027." Deferred dates: 26 March 2029 (patient summaries, ePrescription and eDispensation, plus Chapter IV on secondary use); 26 March 2031 (other priority categories and Chapter III for EHR systems); 26 March 2035 (Art 75(5)). — [EHDS Art 105 (streamlex)](https://streamlex.eu/articles/ehds-en-art-105/); [Noerr overview](https://www.noerr.com/en/insights/the-european-health-data-space-is-on-its-way-an-overview)
- **Entry into force:** sources say the Regulation entered into force in late March 2025 (Art 105: the twentieth day after OJ publication). One search source states 25 March 2025; others say 26 March 2025. — [EHDS Art 105 (streamlex)](https://streamlex.eu/articles/ehds-en-art-105/); [Ringholm Art 105](https://www.ringholm.com/ehds/article-105.htm)
- **Conflicting secondary description:** Orrick (January 2026) summarises the EHDS as allowing Member States to "require that health data be stored and processed exclusively within the EU, unless a GDPR adequacy decision exists". This conflates Art 86 (storage only, Member State option) with Art 87 (store and process, secondary use only, adequacy exception). The Art 86 text itself refers only to **storage**. — [Orrick](https://www.orrick.com/en/Insights/2026/01/Data-Localization-and-the-Sovereign-Cloud-EU-Cloud-Regulations-Explained)

### Inferences
- **Art 86 is the clearest EU-level confirmation that a "storage-only" localisation model exists.** It lets Member States mandate EU *storage* for healthcare providers, not EU processing or access. A hybrid design (database and S3 in an EU region) would satisfy any national rule adopted under Art 86. All-London hosting would not.
- Art 86 is not itself a localisation mandate. Whether it bites depends on national implementing law; none was identified in this research.
- Art 86 refers to "the storage of personal electronic health data referred to in Article 14", i.e. the priority categories (patient summaries, ePrescriptions/eDispensations, imaging, lab results, discharge reports, as understood from Art 105's references to categories (a)–(f)). A pre-operative questionnaire may not fall squarely in those categories unless it feeds a patient summary or discharge report. This should be verified against the Art 14 text.
- Art 86 is not in Art 105's list of deferred provisions (per the streamlex summary). So it presumably applies from 26 March 2027. Art 87 is tied to the secondary-use machinery (Chapter IV), which applies from 26 March 2029.
- Art 87 applies to secondary-use bodies (HDABs, SPEs) and their contractors, not to a primary-use SaaS like Poised. If Poised ever acted on behalf of such a body, its adequacy exception would allow processing in the UK.

### Gaps
- No Member State implementing measure under Art 86 was found (the deadline is effectively 2027). Not researched country by country.
- The EUR-Lex official text could not be fetched. Art 86, 87 and 105 wording comes from a mirror (streamlex.eu) and should be checked against the OJ.
- The Art 14 category list was not fetched directly.

---

## 4. France: HDS certification (CSP L.1111-8) and the HDS v2 referential (2024); SREN/SecNumCloud

### Takeaway
French law requires anyone who **hosts** health data on behalf of a hospital to hold an **HDS certificate**. HDS v2 (arrêté of 26 April 2024) adds a **storage-location rule**: physical hosting of health data must be **exclusively in the EEA**. On **remote access or transfer from outside the EEA**, it imposes contractual **transparency and safeguard** requirements (adequacy or Art 46 tools), not a ban. For a UK (adequate) country, even the specific disclosure duty on non-adequate remote access does not seem to apply.

So for France the user's "storage-only" premise is **largely correct**:
- the hybrid design can comply;
- all-London hosting cannot;
- **Poised itself would very likely need HDS certification**, at least for activity 5, "administration and operation".

SREN Art 31 and SecNumCloud (EU processing and administration, immunity from non-EU law) currently bind State administrations and named public bodies, **not ordinary hospitals**.

### Cited Findings
- **Binding law, CSP L.1111-8 (scope):** the HDS regime applies to any person who hosts personal health data collected during prevention, diagnosis, care or social/medico-social follow-up, on behalf of the persons who produced or collected the data or of the patient. Hosts on digital media must hold a certificate of conformity (L.1111-8, II), issued by a body accredited by COFRAC or an equivalent EU accreditation body. — [Lefebvre Dalloz, fiche "Héberger des données de santé"](https://open.lefebvre-dalloz.fr/droit-affaires/protection-donnees-personnelles/heberger-donnees-sante_a93491) (secondary; the Légifrance page for L.1111-8 returned 404/blocked in this session)
- **Binding law, official French notification to the Commission (TRIS 2023/0682/FR, received 5 December 2023):** "Conformément aux dispositions des articles L.1111-8 et R.1111-10 du code de la santé publique, tout hébergeur de données de santé à caractère personnel (HDS), doit être titulaire d'un certificat de conformité délivré par un organisme de certification sur le fondement d'un référentiel de certification, approuvé par arrêté du ministre chargé de la santé." — [TRIS notification 2023/0682/FR](https://technical-regulation-information-system.ec.europa.eu/fr/notification/25159/print)
- **[STORAGE] Binding law (approved referential), HDS v2 sovereignty requirements (from the TRIS notification):** the revised referential adds four requirements: "**Restreindre le stockage des données de santé sur le territoire d'un Etat partie de l'Espace économique européen**". — [TRIS 2023/0682/FR](https://technical-regulation-information-system.ec.europa.eu/fr/notification/25159/print). CMS's description of the final text: "l'hébergement physique des données de santé devra désormais être réalisé exclusivement dans l'Espace économique européen". — [CMS Francis Lefebvre](https://cms.law/en/fra/news-information/nouveau-referentiel-de-certification-des-hebergeurs-de-donnees-de-sante-hds)
- **[ACCESS/TRANSFER, transparency only] Binding law (referential), from the TRIS notification:** the host must "L'informer [le client] de tout transfert ou **accès distant** aux données du client depuis un territoire situé hors de l'Espace économique européen (EEE) **qui n'assure pas un niveau de protection des données adéquat au sens de l'article 45 du RGPD**, et des mesures organisationnelles et techniques mises en oeuvre pour encadrer ce transfert". It must also inform the client of any "sujétion à une réglementation extra-communautaire" that could create a risk of access by a body in a non-adequate country, and of the mitigations. It must "rendre public et tenir à jour des informations détaillées sur les éventuels transferts de données qu'il héberge vers un pays n'appartenant pas à l'EEE". — [TRIS 2023/0682/FR](https://technical-regulation-information-system.ec.europa.eu/fr/notification/25159/print)
- **Requirement numbering (secondary, from search summaries of ANS webinar material):**
  - Exigence 29: remote access from outside the EU/EEA must rest on an adequacy decision or, failing that, Art 46 GDPR safeguards documented in the contract.
  - Exigence 30: the contract lists the extra-European regulations, the mitigations and the residual risks.
  - Exigence 31: a public URL on third-country access risk; a SecNumCloud-qualified host may state "Aucun risque d'accès imposé par la législation d'un pays tiers en violation du droit de l'Union".
  - [ANS webinar HDS, 15 Oct 2025 (PDF; blocked, content via search summary)](https://industriels.esante.gouv.fr/sites/default/files/media/document/Webinaire-HDS_20251015_vf.pdf); [ANS press release (blocked)](https://esante.gouv.fr/espace-presse/publication-au-journal-officiel-du-referentiel-de-certification-hds-souverainete-des-donnees-et-ameliorations-du-referentiel)
- **Dates:** arrêté of 26 April 2024, published in the JO on 16 May 2024. New candidates have been assessed under v2 since 16 November 2024. Already-certified hosts had 24 months, to **16 May 2026**. — [CMS](https://cms.law/en/fra/news-information/nouveau-referentiel-de-certification-des-hebergeurs-de-donnees-de-sante-hds)
- **The six certified activities:** (1) physical sites; (2) hardware infrastructure; (3) virtual infrastructure; (4) application hosting platform; (5) administration and operation of the information system containing health data; (6) backup. — [CMS](https://cms.law/en/fra/news-information/nouveau-referentiel-de-certification-des-hebergeurs-de-donnees-de-sante-hds)
- CMS's characterisation: remote access from third countries is permitted subject to transparency, which is "less restrictive than SecNumCloud". — [CMS](https://cms.law/en/fra/news-information/nouveau-referentiel-de-certification-des-hebergeurs-de-donnees-de-sante-hds)
- **How the location rule is applied in practice (vendor evidence):** AWS states that its HDS v2 certification covers activities 1, 2 and 6 (physical infrastructure, backup) **only in its EEA regions** (Frankfurt, Ireland, Milan, Paris, Stockholm, Spain). Activities 3–5 (virtual infrastructure, platform, administration/operation) are certified across 26 regions, **including non-EEA regions such as London and Zurich**. — [AWS HDS compliance page](https://aws.amazon.com/compliance/hds/)
- **Does a SaaS vendor need its own HDS certificate? (secondary):** Qovery (2026) states that hyperscalers sell activities 1–4 and 6, and that "activity 5, actually running the system that touches patient data, is usually still yours" as the SaaS operator. — [Qovery](https://www.qovery.com/blog/hds-certified-hosting-providers-compliant-cloud-migration). *Conflict:* AWS's own page says its certification includes activity 5 ("Administration and operation of the information system containing health data") for its scope. — [AWS HDS](https://aws.amazon.com/compliance/hds/)
- **SREN law Art 31 / Decree n° 2026-272 of 14 April 2026 (binding, limited scope):**
  - It applies to cloud services used by State administrations, certain operators and six named GIPs (including the Agence du numérique en santé). Public hospitals and GHTs are **not explicitly listed**.
  - "Données d'une sensibilité particulière" means data covered by legally protected secrets (including medical secrecy) or needed for essential State missions (including health protection), *and* whose compromise risks a concrete harm.
  - Covered entities must use SecNumCloud (or equivalent EUCS) services protected against extraterritorial laws.
  - Derogation of up to 18 months.
  - [Banque des Territoires](https://www.banquedesterritoires.fr/secnumcloud-6-gip-assujettis-les-donnees-sensibles-enfin-definies)
- **SecNumCloud (ANSSI qualification), as summarised by Orrick:** the provider must be "immune to requests from public authorities of third countries", must "store and process client data within the EU", "service administration and supervision operations must be conducted within the EU", and its registered office must be in the EU. This is a **storage + processing + administration** rule, but only where a buyer requires SecNumCloud. — [Orrick](https://www.orrick.com/en/Insights/2026/01/Data-Localization-and-the-Sovereign-Cloud-EU-Cloud-Regulations-Explained)

### Inferences
- **The HDS location rule is a storage (physical hosting and backup) rule.** Remote access and transfers out of the EEA are governed by GDPR-style safeguards plus contractual transparency. The UK is adequate, so access from the UK appears permitted without the extra disclosure triggered for non-adequate countries. The general transfer-mapping and extra-EU-law-exposure disclosures would still apply in substance.
- **The hybrid design (RDS and S3 in eu-west-3 Paris or another EEA region, compute in London) appears compatible with HDS v2.** AWS's certification scope (activities 3–5 allowed in London) supports the reading that only physical storage and backup must be in the EEA.
  - Open point: in the hybrid design, health data is held transiently in memory on London hardware (Lambda/EC2). Whether ANS or certifiers treat that as "physical hosting" outside the EEA was not confirmed. Confirm with an HDS certification body.
  - **Backups (activity 6) must also stay in the EEA.** That means no cross-region snapshot or backup copies to eu-west-2.
- **All-London hosting fails HDS v2** for French customers, because physical hosting would be in a non-EEA country.
- **Poised will very likely need its own HDS certificate** (at least activity 5, probably also 4 depending on how the AWS scope is used). It contracts with the hospital to host and operate the health-data system. No source found says a UK-established company cannot be certified (AWS, a non-French group, is certified), but this was not confirmed.
- SecNumCloud is not legally required for an ordinary hospital buying a SaaS. Some French public buyers may still ask for it as a procurement preference. If they do, the hybrid design fails (processing and administration must be in the EU, and the provider EU-headquartered).

### Gaps
- The verbatim final text of HDS v2 requirements 28–31 (as approved by the arrêté of 26 April 2024) could not be retrieved, because esante.gouv.fr and Légifrance blocked automated access. The wording above comes from the December 2023 TRIS notification draft, which may differ slightly from the final text, and from summaries.
- ANS FAQ guidance on whether a SaaS publisher using a certified IaaS must itself be certified, and for which activities, was not retrievable.
- Whether a non-EEA (UK) legal entity can hold an HDS certificate, or must contract through an EEA entity, was not confirmed.
- No CNIL guidance specific to "storage in France/EEA, processing in the UK" was found.

---

## 5. Germany: §393 SGB V (Digital-Gesetz 2024), C5, and the Länder hospital laws

### Takeaway
Germany **does regulate processing location, not just storage.** §393(2) SGB V says the "**Verarbeitung**" (processing) of social and health data by cloud computing "darf nur" take place:
- in Germany;
- in an EU Member State;
- in an EEA/Swiss-equivalent state;
- or in a third country **with a GDPR Art 45 adequacy decision** (so the UK currently qualifies).

It also requires that the processing entity have a **German establishment (Niederlassung im Inland)** and a current **BSI C5 Type 2 attestation** (since 1 July 2025). Berlin's hospital law similarly limits processor processing of health data to the EU/EEA/Switzerland or an adequacy country. It adds requirements on criminal-law-backed secrecy and on notifying the Senate administration in advance. Bavaria's law no longer has an explicit location limit.

### Cited Findings
- **[PROCESSING] Binding law, §393(1) SGB V:** "Leistungserbringer im Sinne des Vierten Kapitels und Kranken- und Pflegekassen sowie ihre jeweiligen Auftragsdatenverarbeiter dürfen Sozialdaten und Gesundheitsdaten auch im Wege des Cloud-Computing-Dienstes verarbeiten, sofern die Voraussetzungen der Absätze 2 bis 4 erfüllt sind." — [§393 SGB V, gesetze-im-internet.de](https://www.gesetze-im-internet.de/sgb_5/__393.html)
- **[PROCESSING location + establishment] Binding law, §393(2) SGB V:** "Die **Verarbeitung** von Sozial- und Gesundheitsdaten im Wege des Cloud-Computing-Dienstes **darf nur** 1. im Inland, 2. in einem Mitgliedstaat der Europäischen Union oder 3. in einem diesem nach § 35 Absatz 7 des Ersten Buches gleichgestellten Staat oder, **sofern ein Angemessenheitsbeschluss gemäß Artikel 45 der Verordnung (EU) 2016/679 vorliegt, in einem Drittstaat erfolgen** und **sofern die datenverarbeitende Stelle über eine Niederlassung im Inland verfügt**." — [§393 SGB V](https://www.gesetze-im-internet.de/sgb_5/__393.html)
- **[SECURITY] Binding law, §393(3)–(4) SGB V:** processing is lawful only if there are state-of-the-art TOMs, "ein aktuelles C5-Testat der datenverarbeitenden Stelle im Hinblick auf die C5-Basiskriterien für die im Rahmen des Cloud-Computing-Dienstes eingesetzten Cloud-Systeme und die eingesetzte Technik", and implementation of the customer criteria in the audit report. Timing:
  - "Bis zum 30. Juni 2025 gilt… ein C5-Typ1-Testat. Ab dem 1. Juli 2025 gilt… ein aktuelles C5-Typ2-Testat."
  - Systems first placed on the market after 30 June 2025 may use Type 1 for 18 months.
  - An equivalent or higher standard is accepted (to be designated by BMG ordinance with BSI).
  - For hospitals, TOMs are deemed adequate if §391 (hospital IT security) is met (§393(5) Nr. 2).
  - [§393 SGB V](https://www.gesetze-im-internet.de/sgb_5/__393.html)
- **Scope includes hospitals:** §393 applies to "alle Leistungserbringer im Rahmen der gesetzlichen Krankenversicherung – d.h. Vertragsärzte, zugelassene Krankenhäuser…". — [Rödl & Partner, 17 May 2024](https://www.roedl.com/insights/it-auslagerungen-cloud-neue-huerden-neue-moeglichkeiten/)
- **SaaS likely in scope / interpretive uncertainty (law-firm commentary):**
  - Noerr notes the cloud definition is "digitale Dienste, die auf Abruf die Verwaltung und den umfassenden Fernzugang zu einem skalierbaren und elastischen Pool gemeinsam nutzbarer Rechenressourcen ermöglicht" (aligned with NIS2). In Noerr's view this appears to encompass SaaS.
  - Noerr on the C5 duty: "Das Gesetz lässt im Übrigen nicht klar erkennen, ob die betreffend das Testat verpflichtete 'datenverarbeitende Stelle' den Leistungserbringer/die Kasse, deren Auftragsverarbeiter oder beide meint."
  - Noerr questions whether the German-establishment requirement is compatible with EU law.
  - Noerr notes §203 StGB (professional secrecy) remains applicable alongside §393, as do Länder restrictions such as Mecklenburg-Vorpommern's §38 LKHG.
  - [Noerr](https://www.noerr.com/de/insights/neue-anforderungen-an-cloud-computing-dienste-im-gesundheitswesen)
- **[PROCESSING] Binding law, Berlin §24(7) Landeskrankenhausgesetz, as amended by the Third Amending Act (bill Drs. 19/0529 of 21 September 2022; in force 26 October 2022 per the bill):** "Die Erteilung eines Auftrages im Sinne des Artikels 28 der Verordnung (EU) 2016/679 zum Zwecke der Verarbeitung von genetischen Daten und Gesundheitsdaten… ist nur zulässig, wenn 1. der Auftragsverarbeiter sicherstellt, dass die **Verarbeitung** der genetischen Daten oder Gesundheitsdaten… **in einem Mitgliedstaat der Europäischen Union, des Europäischen Wirtschaftsraums, der Schweiz oder, sofern ein Angemessenheitsbeschluss gemäß Artikel 45 der Verordnung (EU) 2016/679 vorliegt, in einem Drittstaat erfolgt** und die Daten darüber hinaus nicht in Drittstaaten offengelegt werden". Further conditions (the PDF's columns are garbled, so this is a reconstruction):
  - the processing must be carried out "ausschließlich durch Personen erfolgt, die nach dem jeweils anwendbaren Recht in Bezug auf den Schutz der Geheimnisse einer strafbewährten Verschwiegenheitspflicht und einem Zeugnisverweigerungsrecht, das dem Schutz im Inland vergleichbar ist, unterliegen";
  - the controller must notify the Senate health administration in writing before commissioning the processor (the processor, its TOMs, the type and volume of data, and the purpose).
  - The explanatory memorandum says the previous law kept processing "nur im Krankenhaus oder in kooperierenden Krankenhäusern".
  - [Abgeordnetenhaus Berlin, Drs. 19/0529](https://www.parlament-berlin.de/ados/19/IIIPlen/vorgang/d19-0529.pdf)
- **Bavaria, Art 27 BayKrG (version valid from 1 May 2025):** the hospital may engage external parties to process patient data, subject to the protective measures in Art 27(6) and "solange keine Anhaltspunkte dafür bestehen, dass… schutzwürdige Belange von Patienten beeinträchtigt werden". No explicit restriction to German or EU territory was found. — [gesetze-bayern.de, Art 27 BayKrG](https://www.gesetze-bayern.de/Content/Document/BayKrG-27). A search summary of Noerr's commentary says the earlier rule limiting processing to hospitals (Art 27(4) sentence 6) was repealed. — [Noerr, "Änderung des bayerischen Krankenhausrechts"](https://www.noerr.com/de/insights/anderung-des-bayerischen-krankenhausrechts)
- **Trend:** Rödl reports that, after the outsourcing bans were removed from the last Landeskrankenhausgesetze, hospitals gained cloud options. — [Rödl & Partner](https://www.roedl.com/insights/it-auslagerungen-cloud-neue-huerden-neue-moeglichkeiten/)

### Inferences
- **§393 SGB V and Berlin LKG §24(7) contradict the "storage-only" premise.** Both regulate where processing takes place, not merely where data rests. Both, however, **accept the UK** while UK adequacy stands. So for Germany the hybrid design gives no legal advantage over all-London hosting on the location question. Both are permitted while adequacy lasts.
- The German blockers are elsewhere:
  - **Niederlassung im Inland:** Poised UK has no German establishment. Whether the requirement attaches to AWS (which has German establishments) or to Poised as the SaaS "datenverarbeitende Stelle" is unclear (Noerr). The conservative reading is that Poised needs a German establishment.
  - **C5 Type 2:** AWS holds C5, but Noerr says it is unclear whether the SaaS processor itself needs a C5 attestation. The conservative reading is that Poised needs a C5 Type 2 for its service (or Type 1 for its first 18 months on the market).
  - **Berlin's "comparable criminal-law secrecy and testimonial privilege" condition** for the persons processing data. UK support staff are not covered by a §203 StGB-equivalent criminal confidentiality duty with testimonial privilege, at least not obviously. This needs German counsel.
  - **No SCC fallback:** if UK adequacy fell away, §393 and Berlin would forbid UK processing outright.

### Gaps
- The other 14 Länder hospital laws (e.g. Hamburg, Bremen, Saxony, Saarland, Mecklenburg-Vorpommern §38 LKHG, NRW GDSG) and church hospital data protection rules (KDG/DSG-EKD) were not individually checked.
- The enacted text of Berlin LKG §24(7) was taken from the bill. It should be confirmed against the consolidated law at gesetze.berlin.de.
- No authoritative BMG or BSI guidance was found on (a) whether SaaS vendors need their own C5, or (b) who must have the German Niederlassung.
- §203 StGB cross-border disclosure rules (the 2017 reform on "mitwirkende Personen" abroad) were not researched.

---

## 6. Netherlands, Italy, Spain, the Nordics, Austria: any health-data localisation or processing-location rules for hospitals?

### Takeaway
Italy and Spain have binding **public-sector** rules that require **EU location of the systems that *process* the data**, not just storage:
- **Italy** (ACN cloud regulation): public administration data must be "**trattati** mediante infrastrutture localizzate sul territorio dell'Unione europea". No adequacy exception was found.
- **Spain** (Art 46 bis Ley 40/2015): systems for the "recogida, almacenamiento, **procesamiento** y gestión" of national health system user data "deberán ubicarse y prestarse dentro del territorio de la Unión Europea". Transfers to adequacy countries are permitted.

No statutory localisation rule was found for Dutch, Swedish or Finnish hospitals. Denmark, Norway and Austria were not resolved.

### Cited Findings
**Italy**
- **[PROCESSING] Binding regulation for public administrations:** ACN "Regolamento per le infrastrutture digitali e per i servizi cloud per la pubblica amministrazione" (Decreto direttoriale n. 21007/24 of 27 June 2024, adopted under Art 33-septies(4) DL 179/2012; PAs had to comply from 1 August 2024).
  - Annex 3 (characteristics of cloud services for PA, which covers SaaS, defined in the regulation as "applicativi, c.d. Software-as-a-Service (SaaS)"), **baseline level for "ordinary" data**, PR.DS-01 1_O: "I dati dell'amministrazione, ivi inclusi quelli deputati alla sicurezza… **sono trattati mediante infrastrutture localizzate sul territorio dell'Unione europea**. Salvo motivate e documentate ragioni di natura normativa o tecnica, nelle citate infrastrutture sono ricomprese quelle deputate alle funzioni di: a. Business Continuity e Disaster Recovery…; b. Content Delivery Network…".
  - "Dati dell'amministrazione" is defined as data "forniti, conservati, inviati, ricevuti, **trattati** da o per conto dell'amministrazione dal soggetto tramite il servizio cloud".
  - [ACN Regolamento 27 giugno 2024 (PDF)](https://dirittobancario.it/wp-content/uploads/2024/07/Regolamento-ACN-27-giugno-2024.pdf); [Cybersecurity360](https://www.cybersecurity360.it/legal/servizi-cloud-pa-entra-in-vigore-il-nuovo-regolamento-ecco-come-recepirlo/) (search summary for the 1 August 2024 date and the decree number)
- **[ACCESS] Binding (higher tiers):** for strategic data (Annex 3, 15_S), "Con riferimento all'accesso ai dati da parte di entità extra-UE, il soggetto: a. segnala all'Agenzia per la cybersicurezza nazionale (ACN) e all'amministrazione ogni richiesta di accesso a dati o metadati da parte di entità extra-UE; b. fornisce accesso a dati dell'Amministrazione o metadati ad entità extra-UE solo a valle di un'autorizzazione esplicita da parte dell'amministrazione." Also: "Esiste un documento aggiornato che descrive da quali sedi e infrastrutture è erogato il servizio cloud." — [ACN Regolamento (PDF)](https://dirittobancario.it/wp-content/uploads/2024/07/Regolamento-ACN-27-giugno-2024.pdf)
- **Classification:** data is "critici" if its compromise "può determinare un pregiudizio al mantenimento di funzioni rilevanti per la società, **la salute**, la sicurezza pubblica e il benessere economico e sociale del Paese". — [ACN Regolamento (PDF)](https://dirittobancario.it/wp-content/uploads/2024/07/Regolamento-ACN-27-giugno-2024.pdf)
- A text search of the ACN regulation found **no reference to GDPR adequacy decisions** as an exception to the EU-infrastructure requirement (only "inadeguatezza" of infrastructure in the revocation context). — [ACN Regolamento (PDF)](https://dirittobancario.it/wp-content/uploads/2024/07/Regolamento-ACN-27-giugno-2024.pdf)

**Spain**
- **[STORAGE + PROCESSING + TRANSFER] Binding law, Art 46 bis Ley 40/2015 (Régimen Jurídico del Sector Público), inserted by Real Decreto-ley 14/2019:** "Los sistemas de información y comunicaciones para la **recogida, almacenamiento, procesamiento y gestión** del censo electoral, los padrones municipales de habitantes y otros registros de población, datos fiscales relacionados con tributos propios o cedidos y **datos de los usuarios del sistema nacional de salud**, así como los correspondientes tratamientos de datos personales, **deberán ubicarse y prestarse dentro del territorio de la Unión Europea**. Los datos a que se refiere el apartado anterior no podrán ser objeto de transferencia a un tercer país u organización internacional, **con excepción de los que hayan sido objeto de una decisión de adecuación de la Comisión Europea** o cuando así lo exija el cumplimiento de las obligaciones internacionales asumidas por el Reino de España." — [Vademecum Legal, Art 46 bis Ley 40/2015](https://www.vademecumlegal.es/administrativo/marginales/donde-deber-ubicarse-los-sistemas-informacion-y-comunicaciones-registro-datos-personales-5743)

**Netherlands**
- **Binding law, security only:** the Besluit elektronische gegevensverwerking door zorgaanbieders (Art 3) makes NEN 7510 and NEN 7512 compliance mandatory for healthcare providers' electronic data processing. The decree as reviewed contains **no provision on where data must be stored or processed**. — [Staatsblad 2017, 446](https://zoek.officielebekendmakingen.nl/stb-2017-446.html)
- The Dutch DPA (AP) uses NEN 7510 to assess whether healthcare providers meet their security obligations (secondary). — [Kiteworks summary (vendor content; low reliability)](https://www.kiteworks.com/de/dsgvo-compliance/niederlandisches-gesundheitswesen-dsgvo-patientendaten/)

**Sweden**
- **Binding law, enabling:** a new secrecy exception, OSL (2009:400) Chapter 10 §2a, in force 1 July 2023. Secrecy does not prevent an authority from disclosing information to a private or public body engaged "only to technically process or technically store" it, unless disclosure is inappropriate in the circumstances. The bill acknowledges that third-country outsourcing raises GDPR transfer difficulties. It does not impose localisation. — [Prop. 2022/23:97 (lagen.nu)](https://lagen.nu/prop/2022/23:97); [Regeringen, "Sekretessgenombrott vid teknisk bearbetning eller lagring av uppgifter"](https://regeringen.se/contentassets/946818572f3b4b488ccc3180eef5fb19/sekretessgenombrott-vid-teknisk-bearbetning-eller-lagring-av-uppgifter.pdf) (search summary)

**Finland**
- **Binding law, no location rule found:** the Client Data Act (703/2023) harmonises processing of client data. It requires public providers to store patient data in the national Kanta archive, and requires social and healthcare information systems to be registered with Valvira before commercial use. No EU/EEA location mandate was found in summaries. — [STM (Ministry of Social Affairs and Health)](https://stm.fi/en/-/new-act-to-regulate-the-processing-of-client-data-in-healthcare-and-social-welfare-1); [DLA Piper Finland](https://finland.dlapiper.com/en/news/unique-requirements-social-and-healthcare-information-systems-finland) (search summary)

### Inferences
- **Italy (public hospitals: ASL, Aziende Ospedaliere, IRCCS pubblici)** fall within "pubbliche amministrazioni". This is an inference; the CAD/165/2001 definitions were not fetched. For them:
  - a SaaS like Poised must be ACN-qualified;
  - it must **process** the administration's data on EU-located infrastructure, including DR;
  - the UK has no adequacy carve-out.
  - **Both the hybrid design and all-London hosting fail** for Italian public hospitals. Private hospitals are outside the ACN regulation.
- **Spain (public SNS bodies):** Art 46 bis requires the *systems* that collect, store, process and manage SNS user data to be **located and provided within the EU**. Only *transfers* get the adequacy exception.
  - In the hybrid design, the London application servers are part of the "sistema de información… para el… procesamiento". On a literal reading, this fails the first paragraph even though the UK is adequate for the second.
  - All-London hosting fails more clearly.
  - Interpretive tension: the adequacy exception for transfers could be read as allowing UK processing. Spanish counsel or AEPD guidance is needed.
  - Private Spanish hospitals are outside Ley 40/2015.
- **Netherlands, Sweden, Finland:** for hospitals buying a SaaS, no statutory storage or processing location rule was identified. The GDPR (with UK adequacy) plus sector security standards govern. Both designs look legally possible, subject to buyer preferences.

### Gaps
- **Denmark, Norway, Austria:** no reliable primary sources were found in this session. Not resolved. (Austria: the Gesundheitstelematikgesetz 2012/ELGA and public-sector rules were not checked. Denmark: no localisation rule was found, but the search was thin.)
- **Italy:** whether the ACN regulation (or Art 33-septies DL 179/2012) contains an adequacy exception elsewhere, and how "ordinary/critical" classification applies to pre-op assessment data in ACN's predefined lists. Also not checked: Italian FSE (Fascicolo Sanitario Elettronico) hosting rules, and Garante positions.
- **Spain:** the ENS (Real Decreto 311/2022), mandatory for public-sector providers, was not reviewed for location requirements. No AEPD or CCN guidance on Art 46 bis was found.
- **Netherlands:** no source found on Dutch hospital or government cloud-sovereignty procurement policy in 2025–26.

---

## 7. Public procurement and sovereignty trends affecting hospital buyers (SecNumCloud, EU Cloud Sovereignty Framework, CADA)

### Takeaway
The direction of travel is toward **EU processing and storage, not just storage**, for public buyers:
- The Commission's **Cloud Sovereignty Framework** (v1.2.1, October 2025) scores providers on eight sovereignty objectives (SEAL-0 to SEAL-4). It is currently used for EU-institution procurement.
- The **Cloud and AI Development Act (CADA) proposal of 3 June 2026** would create EU "assurance levels". Even level 1 would require infrastructure and **customer data in the EU**, and the proposal reportedly sets level 1 as the minimum for all contracting authorities buying cloud. It is **a proposal, not law**.
- France's SecNumCloud and SREN Art 31 already require **EU processing and administration** for designated public bodies.

### Cited Findings
- **Procurement practice (EU institutions), Cloud Sovereignty Framework v1.2.1 (October 2025):**
  - It was presented with the Cloud III Dynamic Purchasing System tender (about €180 million of sovereign cloud for EU institutions).
  - Eight objectives: SOV-1 strategic; SOV-2 legal & jurisdictional (exposure to non-EU laws with cross-border reach); SOV-3 data & AI; SOV-4 operational; SOV-5 supply chain; SOV-6 technology; SOV-7 security & compliance; SOV-8 environmental.
  - SEAL-0 ("service under exclusive control of non-EU third parties") to SEAL-4 ("full digital sovereignty").
  - Tenders not meeting the minimum SEAL on all objectives are rejected.
  - [Commission, Cloud Sovereignty Framework (PDF)](https://commission.europa.eu/document/download/09579818-64a6-4dd5-9577-446ab6219113_en); [Commission news, 17 April 2026](https://commission.europa.eu/news-and-media/news/commission-advances-cloud-sovereignty-through-strategic-procurement-2026-04-17_en) (search summary); [ayedo summary of SEAL levels and weights](https://ayedo.de/en/cloud-sovereignty-framework/)
- **Proposal, not law: CADA (3 June 2026, part of the Tech Sovereignty Package):**
  - Union assurance levels 1–4. Level 1 means baseline requirements "including EU establishment, data localisation within the EU, and transparency regarding subcontracting and data flows". Level 4 is for "mission-critical and highly sensitive activities, in particular in the defence and national security sectors".
  - As of mid-June 2026 it remained a Commission proposal.
  - [Hogan Lovells](https://www.hlc.com/en/publications/the-eus-cloud-and-ai-development-act-cada-towards-a-sovereigntyfocused-framework-for-cloud); [Commission IP/26/1187](https://ec.europa.eu/commission/presscorner/detail/en/ip_26_1187) (page not readable in this session)
  - Covington describes level 1 as requiring providers to "establish and maintain infrastructure, assets, and customer data in the EU", and says contracting authorities must weigh "Union added value" as a non-price criterion. — [Covington, Inside Global Tech](https://www.insideglobaltech.com/2026/06/11/the-eu-cloud-and-ai-development-act-in-depth/)
  - A Covington search snippet states: "When procuring cloud computing services for their exclusive use, all contracting authorities must, as a minimum, comply with assurance level 1 (data is processed and stored in infrastructure located in the EU)". The full Covington article as fetched did not confirm this sentence, so treat it as unverified. — [Global Policy Watch (Covington)](https://globalpolicywatch.com/2026/06/the-eu-cloud-and-ai-development-act-in-depth)
- **Binding (France, limited scope):** SREN Art 31 and Decree 2026-272 require SecNumCloud-type protection from extraterritorial law for designated State bodies' sensitive data (section 4). — [Banque des Territoires](https://www.banquedesterritoires.fr/secnumcloud-6-gip-assujettis-les-donnees-sensibles-enfin-definies)
- **Binding (EU, non-personal data):** Orrick notes that the Data Act requires cloud providers to put in place safeguards against unlawful third-country access to non-personal data stored in the EU, while "companies remain free to choose where to store non-personal data". — [Orrick](https://www.orrick.com/en/Insights/2026/01/Data-Localization-and-the-Sovereign-Cloud-EU-Cloud-Regulations-Explained)

### Inferences
- If CADA is adopted roughly as proposed, and if a SaaS pre-op tool counts as a "cloud computing service" (unclear, and Hogan Lovells notes SaaS coverage is not explicit), then public hospitals as contracting authorities would need at least EU **processing and storage**. The hybrid design (compute in London) would then not qualify, and neither would all-London hosting. This is speculative until the trilogue outcome.
- In practice, EU public hospital tenders increasingly score jurisdictional exposure and the location of operations. A UK-operated service, even an adequate one, may lose points or fail minimum thresholds where buyers adopt CSF-style criteria. This is procurement practice, not law.

### Gaps
- No evidence was found of national health-sector procurement frameworks formally adopting the CSF/SEAL criteria for hospitals.
- CADA's final scope (SaaS inclusion, critical sectors such as health, application date) is unknown as of October 2026. The trilogue status was not checked.
- EUCS (the EU cloud certification scheme under the Cybersecurity Act) status and whether sovereignty requirements were reinserted were not researched.

---

## 8. Overall assessment: do (1) the hybrid design and (2) all-London hosting comply in the EU?

### Takeaway
**The user's belief is only partly right.**
- **GDPR:** treats UK processing and support access as a transfer even when data is stored in the EU. It is lawful today only because of UK adequacy, valid to 27 December 2031.
- **France (HDS):** is essentially a storage rule. The hybrid design works; all-London does not.
- **Germany (§393 SGB V, Berlin LKG):** regulates *processing* location, but accepts adequacy countries. Both designs pass on location, but Germany also requires a German establishment and C5.
- **Italy (ACN, public sector) and Spain (Art 46 bis, public SNS):** regulate *processing* location in the EU. **Both designs likely fail** for public-hospital customers there.
- **EHDS Art 86:** lets Member States mandate EU *storage* only (from 2027). The hybrid design would satisfy it.

### Cited Findings
- **Transfer even when data is stored in the EU:** EDPB Example 11 (third-country processor remotely accessing "data which is stored in the EU" for support = transfer). — [EDPB Guidelines 05/2021](https://www.edpb.europa.eu/system/files/documents/2023-02/edpb_guidelines_05-2021_interplay_between_the_application_of_art3-chapter_v_of_the_gdpr_v2_en_0.pdf)
- **UK adequacy** covers transfers to UK controllers and processors without further authorisation, until 27 December 2031. — [C(2025) 8771](https://commission.europa.eu/document/download/b36eedab-43b8-46eb-ac30-a8d1e6e69f51_en?filename=Renewal+UK+adequacy+GDPR.pdf)
- **France:** "Restreindre le stockage des données de santé sur le territoire d'un Etat partie de l'EEE"; remote-access transparency is triggered for non-adequate territories. — [TRIS 2023/0682/FR](https://technical-regulation-information-system.ec.europa.eu/fr/notification/25159/print). AWS London is eligible for HDS activities 3–5 only. — [AWS HDS](https://aws.amazon.com/compliance/hds/)
- **Germany:** processing allowed in the EU/EEA or an adequacy third country, plus a Niederlassung im Inland and C5 Type 2. — [§393 SGB V](https://www.gesetze-im-internet.de/sgb_5/__393.html). Berlin: processing in the EU/EEA/CH or an adequacy country, plus secrecy-comparable personnel and notification to the Senate. — [Drs. 19/0529](https://www.parlament-berlin.de/ados/19/IIIPlen/vorgang/d19-0529.pdf)
- **Italy:** data "trattati mediante infrastrutture localizzate sul territorio dell'Unione europea". — [ACN Regolamento](https://dirittobancario.it/wp-content/uploads/2024/07/Regolamento-ACN-27-giugno-2024.pdf)
- **Spain:** systems for "recogida, almacenamiento, procesamiento y gestión" of SNS user data "deberán ubicarse y prestarse dentro del territorio de la Unión Europea". — [Art 46 bis Ley 40/2015](https://www.vademecumlegal.es/administrativo/marginales/donde-deber-ubicarse-los-sistemas-informacion-y-comunicaciones-registro-datos-personales-5743)
- **EHDS Art 86:** a national requirement that "the storage… be located within the Union" is permitted. — [EHDS Art 86](https://streamlex.eu/articles/ehds-en-art-86/)

### Inferences
Compliance matrix (for an EU hospital customer; ✔ = appears compliant on location, ✖ = appears non-compliant, ? = unresolved):

| Regime (type) | What it regulates | (1) Hybrid: DB + S3 in EU, compute + support in UK | (2) All in London |
|---|---|---|---|
| GDPR Ch. V (binding) | Transfers, incl. remote access | ✔ via UK adequacy (still a transfer) | ✔ via UK adequacy |
| EHDS Art 86 (enabling, from 2027) | Storage only, if a Member State opts in | ✔ | ✖ where a Member State adopts it |
| EHDS Art 87 (binding, 2029; secondary use only) | Storage + processing; adequacy exception | n/a (✔ via adequacy) | n/a (✔ via adequacy) |
| France HDS v2 (binding) | Physical storage + backup in EEA; transparency on access | ✔ on location (backups must stay in EEA; confirm London in-memory processing with a certifier). Poised needs its own HDS certificate (likely activity 5) | ✖ |
| France SecNumCloud / SREN Art 31 (binding for designated State bodies; otherwise procurement preference) | Storage + processing + administration in EU; immunity | ✖ if required | ✖ |
| Germany §393 SGB V (binding) | Processing location (EU/EEA/adequacy) + German establishment + C5 | ✔ on location; ? Niederlassung; ? C5 for Poised | ✔ on location; same ? |
| Berlin LKG §24(7) (binding) | Processing location (EU/EEA/CH/adequacy) + secrecy + notification | ✔ on location; ? UK staff secrecy condition | ✔ on location; same ? |
| Bavaria Art 27 BayKrG (binding) | No explicit location rule found | ✔ | ✔ |
| Italy ACN (binding, public administrations) | Processing on EU infrastructure; extra-EU access controls | ✖ (London compute) | ✖ |
| Spain Art 46 bis Ley 40/2015 (binding, public sector) | Location of collection/storage/processing systems in EU; transfers only to adequacy countries | ✖ on a literal reading (? if the adequacy exception is read broadly) | ✖ |
| NL, SE, FI (binding: GDPR + sector security) | No location rule found | ✔ | ✔ |
| CADA (proposal) | Level 1 = infrastructure + customer data in EU | ✖ if adopted and applied to SaaS | ✖ |

- **Bottom line:**
  - The hybrid design is a sound minimum for France (HDS) and future EHDS Art 86 national rules.
  - It adds nothing over London for Germany. Germany allows the UK but needs a German establishment and C5.
  - It is insufficient for Italian and Spanish public hospitals, and for any buyer demanding SecNumCloud or CADA-style sovereignty. For those, the compute (the application Lambdas) would also have to run in an EU region.
  - Everything that relies on UK processing is exposed to the UK adequacy decision (sunset 27 December 2031; challengeable). The German and Spanish rules have no SCC fallback.
- The cheapest design that covers the most EU buyers is likely **full regional deployment** (compute + database + storage in the customer's EU region), with UK support access controlled and logged. GDPR allows UK support access via adequacy, and HDS allows it with transparency. Italy's ACN rules require notification to ACN and explicit authorisation for extra-EU access at the strategic level.

### Gaps
- None of the above is legal advice. The key open points to put to local counsel or regulators:
  - **France:** whether in-memory processing on London hardware counts as "hébergement physique" outside the EEA; Poised's HDS scope.
  - **Germany:** who needs the Niederlassung and the C5.
  - **Berlin:** the UK staff secrecy condition.
  - **Spain:** reading of Art 46 bis.
  - **Italy:** whether there is any adequacy exception.
