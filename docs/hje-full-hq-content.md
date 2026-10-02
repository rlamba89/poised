# HJE Full HQ: content transcribed from screenshots

This is the source for rebuilding the HQ in Poised. It was transcribed on 1 Oct 2026 from Rahul's screenshots of the Training Author tool.

**Key:**
- `[P]` patient, `[C]` clinician.
- `opt` = optional (not required).
- `🔗` = the option or question has disclosures. The screenshots show their count (almost always 2) but not the codes or notes themselves.
- `when X is Y` = display logic.
- Select Many lists the "None" option last, after `or:`.
- Every page has **Clinical summary: on**.

Sets 5–11 haven't been captured yet: Lungs, Organs, Digestive health, Musculoskeletal, Neurological & mental health, Infection risks, and Support, lifestyle & discharge planning.

---

## 1. About you (icon: User01)

### 1.1 Patient Information
1. Profile [P]
2. Text short [P] "Preferred name:" 🔗
3. Text long [P] "What is your home address?"
   - desc: "Home number or name, street, town, city and postcode"

### 1.2 Emergency contacts
1. Text long [P] "Primary emergency contact:" 🔗
   - desc: "Including full name, relation to you and contact number"
2. Yes/No [P] "Do we have your permission to share information about your admission with your emergency contacts?"
   - desc: "This may help your hospital team provide vital care in the event of an emergency"
   - Yes 🔗, No 🔗
3. Statement [C] "Patient consents to information being shared with emergency contacts"
   - when Q2 is Yes
4. Statement [C] "Patient does not consent to information being shared with emergency contacts"
   - when Q2 is No

### 1.3 GP details
1. Text long [P] "GP details:" 🔗
   - desc: "Including practice name, address and telephone number"
2. Yes/No [P] "Do we have permission to share information about your admission with your GP?"
   - desc: "This may include a summary discharge letter from your hospital team after your procedure"
   - Yes 🔗, No 🔗
3. Statement [C] "Patient consents to admission information being shared with GP"
   - when Q2 is Yes
4. Statement [C] "Patient does not consent to admission information being shared with GP"
   - when Q2 is No

### 1.4 Neurodevelopment & learning disability
1. Select Many [P] "Do any of the following apply to your own neurodevelopment?"
   - Attention deficit hyperactivity disorder (ADHD) 🔗
   - Autism 🔗
   - Learning disability 🔗
   - I identify as neurodivergent, but I do not have a formal diagnosis 🔗
   - Other neurodevelopmental condition 🔗
   - or: No, I do not have any neurodevelopmental conditions 🔗
2. Text long [P] opt "Are there any adjustments or additional support that would help you feel comfortable, communicate with the team or receive care during your hospital visit?"
   - when Q1 is not "No, I do not have any neurodevelopmental conditions"
3. Statement [C] "Learning disability disclosed – review communication needs, Health and Care Passport, reasonable adjustments, decision-making support and peri-operative plan before attendance."
   - when Q1 is Learning disability
4. Statement [C] "Autism disclosed – review communication needs, Health and Care Passport, reasonable adjustments, decision-making support and peri-operative plan before attendance."
   - when Q1 is Autism
5. Yes/No [P] "Do you carry a Health and Care Passport or Hospital Passport?"
   - Yes 🔗, No

### 1.5 Communication, accessibility and reasonable adjustments
1. Statement [P] "We want to make sure that information and care are provided in a way that meets your needs. Please tell us about any communication, accessibility or support requirements."
2. Select Many [P] "Do you have any communication, hearing, sight, sensory or accessibility needs that may affect your hospital care?"
   - Difficulty seeing, even when using glasses or contact lenses 🔗
   - Difficulty hearing, even when using hearing aids 🔗
   - I use hearing aids 🔗
   - I use British Sign Language or another form of signed communication 🔗
   - I need information in an accessible format 🔗
   - I have sensory needs that may affect my hospital experience 🔗
   - or: No to all 🔗
3. Text long [C] opt "Reasonable adjustments and support plan:" 🔗
   - desc: "Record the patient's communication, sensory and accessibility needs; agreed reasonable adjustments; accessible information requirements; carer, advocate or specialist-team involvement; referrals made; and arrangements for admission."
   - when Q2 is not "No to all"

### 1.6 Advanced care planning
1. Select Many [P] "Have you made any formal advance care statements or decisions?"
   - desc: "Advance care is a part of end of life planning that sets down your preferences, wishes, beliefs and values regarding your future care."
   - DNAR - Do Not Attempt Resuscitation 🔗
   - Advanced directive 🔗
   - Healthcare power of attorney 🔗
   - or: No, I have not made any advance care statements or decisions 🔗
2. Yes/No [C] "Are there any safeguarding concerns for the patient?"
   - Yes, No 🔗
3. Text long [C] "Please provide more details on the safeguarding concerns:" 🔗
   - desc: "Including any action you have or plan to make"
   - when Q2 is Yes

### 1.7 Language, religion & cultural needs
1. Text short [P] opt "Religion:"
   - desc: "If none, please type 'None'"
2. Text short [P] "Ethnicity:" 🔗
3. Text short [P] "Nationality:" 🔗
4. Text short [P] "Preferred language" 🔗
5. Select One [P] "Do you need translation of written materials?"
   - Yes 🔗, No 🔗 (stacked, so Select One rather than Yes/No)
6. Statement [C] "Translation of written materials required"
   - when Q5 is Yes
7. Select One [P] "Do you need an interpreter to assist verbal communication?"
   - Yes 🔗, No 🔗
8. Statement [C] "Interpreter required for verbal communication"
   - when Q7 is Yes

### 1.8 BMI
1. BMI [P] (the only element on the page)

---

## 2. Medication and Allergies (icon: Medication)

### 2.1 Prescribed medication
1. Yes/No [P] "Are you taking any prescribed medications?"
   - desc: "This includes medications prescribed by your GP, consultant or healthcare professional"
2. Medication (prescribed) [P] opt "Please list any medication that you are taking that has been prescribed by a healthcare professional:"
   - desc: "You will be asked about non-prescribed/ over the counter medication separately"
   - when Q1 is Yes
3. Text long [C] opt "Please document any advice given to the patient regarding their prescribed medications:" 🔗
   - when Q1 is Yes

### 2.2 Non-prescribed medication and Recreational drug use
1. Yes/No [P] "Are you taking any non-prescribed medications?"
   - desc: "This may include herbal remedies, over the counter medications, health supplements or elective weight loss injections"
2. Medication (non-prescribed) [P] "Please list any over the counter (non-prescribed) medications and herbal remedies that you are taking:"
   - desc: "We ask this as some over the counter and herbal remedies can have additional blood thinning properties which may need to be stopped prior to your procedure"
   - when Q1 is Yes
3. Text long [C] opt "Please document any advice given to the patient regarding their non-prescribed medications:" 🔗
   - when Q1 is Yes
4. Yes/No [P] "Do you use any recreational drugs?"
   - desc: "It is important to know this as recreational drug use can affect the way anaesthetics work and how individuals react to pain after their operation, and can also lead to withdrawal problems after an anaesthetic"
   - Yes 🔗, No 🔗
5. Text long [P] "Please list the recreational drugs you take including the drug name and frequency of use:"
   - when Q4 is Yes
6. Text long [C] opt "Please provide a clinical overview of the patients drug history and any advice given to the patient regarding their recreational drug use:" 🔗
   - when Q4 is Yes

### 2.3 Steroid use
1. Yes/No [P] "Have you taken steroids within the last 3 months?"
   - Yes 🔗, No 🔗
2. Text long [P] "Please provide more information on your history of steroid use:"
   - desc: "Including when and why you required steroids"
   - when Q1 is Yes
3. Yes/No [P] "Do you have a steroid emergency card?"
   - Yes 🔗, No 🔗
4. Statement [C] "Record the patient's current steroid treatment and any required perioperative plan in the Clinical Summary below. Include the medication, dose, route, indication and duration. This information will appear on the POA Summary."
   - when Q1 is Yes

### 2.4 Allergies
1. Select Many [P] "Do you have any allergies or sensitivities?"
   - Medication 🔗
   - Latex 🔗
   - Adhesives or dressings 🔗
   - Food 🔗
   - Other allergy/s 🔗
   - I am not sure 🔗
   - or: No known allergies 🔗
2. Text long [P] "Please list each allergy or sensitivity and describe what happens when you are exposed to it:"
   - desc: "For example: Pencillin - facial swelling and difficulty breathing; latex - itchy rash."
   - when Q1 is not "No known allergies"
3. Statement [C] "Review the patient's linked allergy disclosures and document a concise clinical summary below.\n\nDo not copy the full patient response unless clinically necessary. This entry will appear on the POA Summary."
   - when Q1 is not "No known allergies"
4. Text long [C] opt "Include the allergen, nature and severity of the reaction, and any required perioperative action or escalation." 🔗
   - desc: "For example: Pencillin - facial swelling and difficulty breathing; latex - itchy rash."
   - when Q1 is not "No known allergies"
5. Yes/No [P] "Have you ever experienced an anaphylactic reaction to any substance?"
   - desc: "Anaphylaxis is a life-threatening allergic reaction that happens very quickly. It can be caused by food, medicine or insect stings and causes symptoms such as swelling of your throat and tongue and difficulty breathing."
   - Yes 🔗, No 🔗
6. Text long [P] "Please provide more details on your history of anaphylaxis:"
   - desc: "Include dates and if you were hospitalised"
   - when Q5 is Yes
7. Yes/No [P] "Do you carry an Epipen?"
   - Yes 🔗, No 🔗
   - when Q5 is Yes
8. Yes/No [C] "Have you advised the patient to bring their epipen on admission"
   - Yes 🔗, No
   - when Q7 is Yes
9. Statement [C] "Review the patient's responses regarding anaphylaxis and adrenaline auto-injector use.  This entry will appear on the POA Summary."
10. Text long [C] opt "Add a concise clinical narrative, including known triggers, previous reactions, frequency or recency, current auto-injector use and any required perioperative actions or escalation." 🔗

---

## 3. Medical and Anaesthetic history

### 3.1 Admission history
1. Yes/No [P] "Have you been admitted to hospital before?"
   - desc: "This can include an overnight stay, or where you were admitted for treatment in hospital without an overnight stay. This does not include visits to A&E where no treatment was carried out in hospital."
2. Admissions [P] opt "Please list any previous hospital admissions for either an operation or other health reasons with the approximate year of admission:"
   - desc: "This includes day-case procedures"
   - when Q1 is Yes

### 3.2 Anaesthetic history
1. Select Many [P] "Have you experienced any of the following during or after a previous anaesthetic or operation?"
   - desc: "Select all that apply."
   - Severe nausea or vomiting 🔗
   - Difficulty placing a breathing tube or managing my airway 🔗
   - Difficulty breathing or unexpected admission to intensive care 🔗
   - Taking longer than expected to wake up 🔗
   - Severe confusion or agitation after the operation 🔗
   - Difficulty passing urine 🔗
   - Severe or poorly controlled pain 🔗
   - Other complication/s 🔗
   - I have never had an anaesthetic or operation 🔗
   - or: I have had no previous complications 🔗
2. Text long [P] opt "Please provide more details on your anaesthetic/surgical complications:"
   - desc: "Include the operation or procedure, approximately when it happened, what you were told about the complication and whether you required any additional treatment or intensive care."
   - when Q1 is not "I have had no previous complications"
3. Text long [C] opt "Document relevant airway assessment findings, including mouth opening, dentition, neck movement and Mallampati score. Record any required anaesthetic action or escalation." 🔗
   - desc: "This entry will appear on the POA Summary."
4. Select Many [P] "Do you have any of the following dental, mouth or neck considerations?"
   - desc: "Select all that apply."
   - Loose teeth, false teeth, caps or crowns 🔗
   - Dental treatment in the last 6 months 🔗
   - Difficulty opening your mouth wide (such as taking a bite from an apple) 🔗
   - Stiff neck 🔗
   - Difficulty tilting your head back 🔗
   - Previous neck surgery 🔗
   - or: None of the above 🔗
5. Select Many [P] "Do any of the following apply to your back, spine or ability to remain still?"
   - desc: "Select all that apply."
   - I have had previous spinal surgery 🔗
   - I have metalwork in my spine 🔗
   - I have had a previous spinal injury 🔗
   - I am unable to remain still for periods of time 🔗
   - or: None of the above
6. Statement **[P]** "Review the patient's anaesthetic history disclosures and document a concise clinical summary below if further context is required.  This entry will appear on the POA Summary."
   - The wording is for clinicians but the badge says PATIENT. This looks like a data error in Lifebox.

### 3.3 Transfusion history
1. Yes/No [P] "In the event of an emergency, would you consent to receiving a blood transfusion and/or blood products?"
   - Yes 🔗, No 🔗
2. Text long [C] "Review the patient's response and document their specific wishes regarding blood and blood products, any advance decision or supporting documentation, and the discussions, escalation or perioperative planning required." 🔗
   - desc: "This entry will appear on the POA Summary."
   - when Q1 is No
3. Yes/No [P] "Have you ever had a blood transfusion?"
   - Yes 🔗, No
4. Text long [P] "Please provide more details on your previous blood transfusion/s:"
   - desc: "Including the date and reason you needed a blood transfusion"
   - when Q3 is Yes
5. Yes/No [P] "Did you have any reaction to the blood transfusion?"
   - Yes 🔗, No 🔗
   - when Q3 is Yes
6. Text long [P] "Please provide more details on your reaction to a previous blood transfusion:"
   - desc: "Include what your reaction was and how it was treated (if known)"
   - when Q5 is Yes
7. Text long [C] "Review the patient's reported transfusion reaction and document the suspected type and severity, timing, treatment received, relevant investigation or blood-bank information, and any required escalation or precautions."
   - desc: "This entry will appear on the POA Summary."
   - when Q5 is Yes

### 3.4 Functional capacity
1. Yes/No [P] "Can you climb two flights of stairs without stopping or becoming significantly breathless?"
   - Yes 🔗, No 🔗
2. Text long [P] "What limits you or makes you stop?"
   - desc: "Shortness of breath, chest pain/tightness, leg or joint pain, poor balance, feeling faint etc"
   - when Q1 is No
3. Yes/No [P] "Can you walk for >100metres on the flat without stopping?"
   - Yes 🔗, No 🔗
4. Text long [P] "What limits you or makes you stop?"
   - desc: same as Q2
   - when Q3 is No
5. Statement [C] "Review the patient's reported functional limitation. Add clinical context in the summary box where required, including usual level of activity, limiting symptoms, recent deterioration and any investigation, optimisation or escalation required."

---

## 4. Heart and Blood

### 4.1 Cardiovascular
1. Select Many [P] "Please select if you have experienced or been diagnosed with any of the following conditions related to your heart or blood pressure:"
   - Previous heart attack 🔗
   - High blood pressure 🔗
   - Heart murmur/valve disease 🔗
   - Heart failure 🔗
   - Angina 🔗
   - Atrial fibrillation 🔗
   - Other cardiac arrhythmia/palpitations 🔗
   - Previous chest pain/tightness (no diagnosis of Angina) 🔗
   - Other heart condition 🔗
   - or: I have no diagnosed heart conditions 🔗
2. Select Many [C] "Does the patient report any features suggesting new or worsening angina?"
   - New or recently developed chest pain 🔗
   - Chest pain when resting or with minimal activity 🔗
   - Chest pain is becoming more frequent, severe or prolonged 🔗
   - Chest pain occurring with less activity than previously 🔗
   - Associated breathlessness, sweating, nausea, dizziness or collapse 🔗
   - or: None of these features identified 🔗
3. Statement [C] "New, ongoing or worsening chest pain may require urgent medical assessment. Follow the organisation's escalation pathway and do not rely solely on documentation within the POA Summary."
4. Statement [P] "Very high blood pressure may require further assessment and could result in your procedure being delayed while it is brought under better control. If you are concerned that your blood pressure may be high, please arrange a review with your GP."
   - when Q1 is High blood pressure
5. Text long [P] "Please provide more information on your heart condition/s:"
   - desc: "Include relevant dates, treatment and anything else you think the team will find useful"
6. Select One [C] "For patients prescribed GTN, is their chest discomfort relieved by rest or GTN within approximately five minutes?"
   - Yes 🔗
   - Partially 🔗
   - No 🔗
   - The patient has not needed to use GTN 🔗
   - Not known 🔗
   - when Q1 is Angina
7. Text long [C] opt "Document the pattern, frequency and duration of the patient's chest discomfort, precipitating and relieving factors, GTN use and response, recent changes, and any assessment or escalation undertaken." 🔗
   - desc: "This entry will appear on the POA Summary."
   - when Q1 is Angina
8. Statement [C] "Review the patient's cardiac history and symptoms. Record a concise clinical summary of current stability, relevant investigations or specialist follow-up, functional impact, medication considerations, and any optimisation or escalation required. Do not duplicate the full patient response. This entry will appear on the POA Summary."

### 4.2 Implantable cardiac devices
1. Select Many [P] "Please select if you have ever had any of the following Implantable cardiac devices inserted:"
   - Pacemaker 🔗
   - Pacemaker with a defibrillator component 🔗
   - Implantable cardioverter defibrillator (ICD) 🔗
   - or: None of the above 🔗

### 4.3 Pacemaker (page shown when 4.2 Q1 is Pacemaker)
1. Date [P] "When was your pacemaker inserted?" 🔗
   - desc: "(Month/Year)"
2. Statement [P] "To ensure your safety during surgery you will need to attend a pacemaker clinic review within the last year. If available, please upload your most recent pacemaker report to LifeBox."
3. Date [P] opt "Please enter the date of your last pacemaker review?" 🔗
   - desc: "(Day/Month/Year)"
4. Select Many [C] "Does the patient require a pacemaker review before their procedure?"
   - Pacemaker review required before procedure 🔗
   - Last pacemaker review within date 🔗

### 4.4 Pacemaker with a defibrillator component (page shown when 4.2 Q1 is Pacemaker with a defibrillator component)
1. Date [P] "When was your pacemaker with a defibrillator component inserted?" 🔗
   - desc: "(Month/Year)"
2. Statement [P]: same text as 4.3 Q2
3. Date [P] opt "Please enter the date of your last pacemaker with defibrillator component review?" 🔗
   - desc: "(Day/Month/Year)"
4. Select Many [C] "Does the patient require a pacemaker with defibrillator component review before their procedure?"
   - Pacemaker review required 🔗
   - Pacemaker review within date 🔗

### 4.5 Implantable cardioverter defibrillator (ICD) (page shown when 4.2 Q1 is Implantable cardioverter defibrillator (ICD))
1. Date [P] "When was your Implantable cardioverter defibrillator (ICD) inserted?" 🔗
   - desc: "(Month/Year)"
2. Statement [P] "To ensure your safety during surgery you will need to attend a review of your Implantable cardioverter defibrillator (ICD) within the last year of your procedure. If available, please upload your most recent report to LifeBox."
3. Date [P] opt "Please enter the date of your last ICD review?" 🔗
   - desc: "(Day/Month/Year)"
4. Select Many [C] "Does the patient require an ICD review before their procedure?"
   - ICD review required before procedure 🔗
   - ICD review within date 🔗

### 4.6 Blood, bleeding & circulation
1. Select Many [P] "Have you ever been diagnosed with, or experienced, any of the following blood, bleeding or circulation conditions?"
   - Anaemia 🔗
   - Sickle cell disease 🔗
   - Sickle cell trait 🔗
   - Thalaessmia 🔗
   - Thalassaemia trait 🔗
   - Platelet Disorder 🔗
   - Haemophilia, von Willebrand disease or another bleeding disorder 🔗
   - Thrombophilia or another clotting disorder 🔗
   - Peripheral vascular disease 🔗
   - Aneurysm/s 🔗
   - Unexplained or excessive bruising or bleeding 🔗
   - Other blood or circulation condition/s 🔗
   - or: None of these apply 🔗
2. Text long [P] "Please tell us more about your blood, bleeding or circulation condition(s):"
   - desc: "Include the diagnosis, approximate date, current treatment, specialist involvement and any recent bleeding, blood clots, anaemia treatment or sickle cell crises."
   - when Q1 is not "None of these apply"
3. Statement [C] "Review the patient's disclosures and record a concise clinical summary."
   - when Q1 is not "None of these apply"
4. Text long [C] "Include the confirmed diagnosis and current status, relevant recent results or specialist plans, medication considerations, previous bleeding or clotting events, and any perioperative action or escalation required." 🔗
   - desc: "This entry will appear on the POA Summary."
   - when Q1 is not "None of these apply"

### 4.7 VTE history
1. Yes/No [P] "Have you or a first degree relative (parent, child, sibling) ever had a blood clot in your legs (Deep vein thrombosis) or Lungs (Pulmonary embolism)?"
   - Yes, No 🔗
2. Select Many [P] "Please select from the statements below regarding your history of blood clots:"
   - I have had a Blood clot in my leg/s 🔗
   - I have had a Blood clot in my lung/s 🔗
   - I have had a blood clot somewhere else 🔗
   - A first degree relative of mine has had a blood clot 🔗
   - or: No
   - when Q1 is Yes
3. Text long [P] "Please provide any further details on your history of blood clots:"
   - desc: "Include date of occurrence and brief statement on how this was managed"
   - when Q1 is Yes
4. Text long [C] opt "Review the patient's personal and family VTE history. Document the type, site and date of any previous clot and the required preoperative VTE prophylaxis or anticoagulation plan." 🔗
   - desc: "This entry will appear on the POA Summary."
   - when Q1 is Yes
