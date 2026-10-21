// The New episode page (plan C2, S13; decisions.md 11 Oct). It replaced the drawer and its
// Existing/New patient tabs:
//   1. search this hospital's patients (the server searches as staff type, at most 20 results);
//   2. pick one, or type a new patient's details from the referral;
//   3. the episode's details, then Create.
// For typed details the server links silently to the trust's existing record when the NHS number
// (or the name, date of birth and a contact) agree, and shows nothing about it. It answers 409
// for an NHS number whose date of birth or surname differ ("mismatch"), and for a possible
// duplicate, which staff can override with "Create anyway". Made-up patients only (NFR-03).
// The procedure comes from the trust's list (S19): picking one chooses its default HQ (or the
// hospital's), which staff can still change before Create. The page opens with the hospital's.
import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import {
  Alert, Anchor, Button, Card, Group, Select, SimpleGrid, Stack, Table, Text, TextInput, Title,
} from "@mantine/core";
import { IconArrowLeft, IconSearch } from "@tabler/icons-react";
import { api } from "@/lib/api";
import { useHospital } from "@/lib/auth";
import { hqForProcedure, initialHQ, procedureLabel, type OfferedProcedure, type PublishedHQ } from "@/features/procedures/procedures";
import { nationalIdLabel } from "@/lib/config";
import { formatDate } from "@/lib/format";
import { partlyTypedDate } from "./dates";
import { conflictOf, hospitalNumberOnChoose, latestHospitalNumber, searchQuery, type Conflict } from "./newEpisode";
import { patientName, SEXES, type FoundPatient, type PatientRecord } from "./types";

type SearchResult = { patients: FoundPatient[]; more: boolean };

const EMPTY_PATIENT = { firstName: "", lastName: "", dateOfBirth: "", sex: "", nationalId: "", mobile: "", email: "" };
const ANAESTHETICS = ["General", "Regional", "Local", "Sedation"];

export function NewEpisodePage({ base, patientId }: { base: string; patientId: string | null }) {
  const navigate = useNavigate();
  const [, setParams] = useSearchParams();
  const [chosen, setChosen] = useState<FoundPatient | null>(null);
  const [patient, setPatient] = useState(EMPTY_PATIENT);
  const { trust, hospital } = useHospital();
  const [hqs, setHqs] = useState<PublishedHQ[]>([]);
  const [procedures, setProcedures] = useState<OfferedProcedure[]>([]);
  const [episode, setEpisode] = useState({
    hospitalNumber: "", versionId: "", procedureId: "", plannedDate: "", poaDate: "", anaesthetic: "", consultant: "",
  });
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState<Conflict | null>(null);
  const [saving, setSaving] = useState(false);
  const plannedRef = useRef<HTMLInputElement>(null);
  const poaRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    api<PublishedHQ[]>(`${base}/published-hqs`)
      .then((list) => {
        setHqs(list);
        setEpisode((e) => (e.versionId ? e : { ...e, versionId: initialHQ(list) }));
      })
      .catch((e: Error) => setError(e.message));
    // What a new episode here can be given: the trust's procedures and this hospital's own, not retired.
    api<OfferedProcedure[]>(`/o/${trust.id}/procedures?hospital=${hospital.id}`).then(setProcedures).catch((e: Error) => setError(e.message));
  }, [base, trust.id, hospital.id]);
  const pickProcedure = (id: string | null) => {
    const p = procedures.find((x) => x.id === id);
    setEpisode({ ...episode, procedureId: id ?? "", versionId: hqForProcedure(episode.versionId, p, hqs) });
  };

  // From a patient's page: that patient is already chosen.
  useEffect(() => {
    if (!patientId) return;
    api<{ patient: PatientRecord; episodes: { hospitalNumber: string }[] }>(`${base}/patients/${patientId}`)
      .then(({ patient: p, episodes }) => {
        const number = latestHospitalNumber(episodes);
        setChosen({ ...p, hospitalNumber: number });
        setEpisode((e) => ({ ...e, hospitalNumber: hospitalNumberOnChoose(e.hospitalNumber, number) }));
      })
      .catch((e: Error) => setError(e.message));
  }, [base, patientId]);

  const create = async (createAnyway = false) => {
    setError("");
    setConflict(null);
    const partly = partlyTypedDate([
      { label: "planned procedure date", input: plannedRef.current },
      { label: "POA appointment date", input: poaRef.current },
    ]);
    if (partly) {
      setError(partly); // a half-typed date reads as none: never drop it silently (S19 F1)
      return;
    }
    setSaving(true);
    const body = chosen ? { patientId: chosen.id, ...episode } : { patient, createAnyway, ...episode };
    try {
      const res = await api<{ id: string }>(`${base}/episodes`, { method: "POST", body: JSON.stringify(body) });
      navigate(`${base}/episodes/${res.id}`);
    } catch (e) {
      const c = conflictOf(e);
      if (c) setConflict(c);
      else setError((e as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const typed = (key: keyof typeof EMPTY_PATIENT) => ({
    value: patient[key],
    onChange: (e: React.ChangeEvent<HTMLInputElement>) => {
      setPatient({ ...patient, [key]: e.currentTarget.value });
      setConflict(null); // the warning was about the details as they were
    },
  });
  const choose = (p: FoundPatient) => {
    setChosen(p);
    setConflict(null);
    setError("");
    setEpisode({ ...episode, hospitalNumber: hospitalNumberOnChoose(episode.hospitalNumber, p.hospitalNumber) });
  };
  // The hospital number was that patient's: never carry it over to the next one.
  // …and forget them in the address (?patient=), so a reload or Back/Forward doesn't choose them again.
  const changePatient = () => {
    setChosen(null);
    setEpisode({ ...episode, hospitalNumber: "" });
    if (patientId) setParams({}, { replace: true });
  };

  return (
    <Stack maw={960} mx="auto">
      <Anchor component={Link} to={`${base}/episodes`} size="sm">
        <Group gap={4}><IconArrowLeft size={14} /> Episodes</Group>
      </Anchor>
      <Title order={3}>New episode</Title>

      {chosen ? (
        <Card withBorder>
          <Group justify="space-between" align="flex-start">
            <div>
              <Text size="sm" c="dimmed">Patient</Text>
              <Text fw={700}>{patientName(chosen)}</Text>
              <Text size="sm">
                Born {formatDate(chosen.dateOfBirth)} · {nationalIdLabel()} {chosen.nationalId || "—"}
                {chosen.hospitalNumber && ` · Hospital number ${chosen.hospitalNumber}`}
              </Text>
            </div>
            <Button variant="default" onClick={changePatient}>Change patient</Button>
          </Group>
        </Card>
      ) : (
        <>
          <PatientSearch base={base} onChoose={choose} />
          <Card withBorder>
            <Title order={4}>New patient</Title>
            <Text size="sm" c="dimmed" mb="sm">
              Not found? Enter their details from the referral. If the trust already has a record for them, the episode is linked to it.
              Made-up details only: never enter a real patient.
            </Text>
            <Stack gap="sm">
              <SimpleGrid cols={{ base: 1, sm: 2 }}>
                <TextInput label="First name" required {...typed("firstName")} />
                <TextInput label="Surname" required {...typed("lastName")} />
                <TextInput label="Date of birth" type="date" required {...typed("dateOfBirth")} />
                <Select
                  label="Sex"
                  required
                  data={SEXES}
                  value={patient.sex || null}
                  onChange={(v) => {
                    setPatient({ ...patient, sex: v ?? "" });
                    setConflict(null);
                  }}
                />
                <TextInput label={nationalIdLabel()} description="If they have one" {...typed("nationalId")} />
                <TextInput label="Mobile" type="tel" {...typed("mobile")} />
                <TextInput label="Email" type="email" {...typed("email")} />
              </SimpleGrid>
            </Stack>
          </Card>
        </>
      )}

      <Card withBorder>
        <Title order={4} mb="sm">Episode</Title>
        <SimpleGrid cols={{ base: 1, sm: 2 }}>
          <TextInput
            label="Hospital number"
            description="This hospital's number for the patient, if any"
            value={episode.hospitalNumber}
            onChange={(e) => setEpisode({ ...episode, hospitalNumber: e.currentTarget.value })}
          />
          <Select
            label="Procedure"
            description="From the trust's list. Picking one chooses its default HQ."
            data={procedures.map((p) => ({ value: p.id, label: procedureLabel(p) }))}
            value={episode.procedureId || null}
            onChange={pickProcedure}
            searchable
            clearable
            nothingFoundMessage="No procedure of that name"
          />
          <Select
            label="Health questionnaire"
            description="Only published versions can be given to patients."
            required
            data={hqs.map((h) => ({ value: h.versionId, label: `${h.name} v${h.versionNo}` }))}
            value={episode.versionId || null}
            onChange={(v) => setEpisode({ ...episode, versionId: v ?? "" })}
            nothingFoundMessage="Nothing published yet"
          />
          <TextInput label="Planned procedure date" type="date" ref={plannedRef} value={episode.plannedDate}
            onChange={(e) => setEpisode({ ...episode, plannedDate: e.currentTarget.value })} />
          <TextInput label="POA appointment date" type="date" ref={poaRef} value={episode.poaDate}
            onChange={(e) => setEpisode({ ...episode, poaDate: e.currentTarget.value })} />
          <Select
            label="Anaesthetic"
            data={ANAESTHETICS}
            clearable
            value={episode.anaesthetic || null}
            onChange={(v) => setEpisode({ ...episode, anaesthetic: v ?? "" })}
          />
          <TextInput label="Consultant" value={episode.consultant} onChange={(e) => setEpisode({ ...episode, consultant: e.currentTarget.value })} />
        </SimpleGrid>
      </Card>

      {error && <Alert color="red">{error}</Alert>}
      {conflict?.code === "mismatch" && <Alert color="red" title="The details don't match">{conflict.message}</Alert>}
      {conflict?.code === "possible_duplicate" && (
        <Alert color="orange" title="Possible duplicate">
          <Stack gap="xs">
            <Text size="sm">{conflict.message}</Text>
            {conflict.matches.map((m) => (
              <Group key={m.id} justify="space-between">
                <Text size="sm">
                  <b>{patientName(m)}</b>, born {formatDate(m.dateOfBirth)} · {nationalIdLabel()} {m.nationalId || "—"}
                  {m.hospitalNumber && ` · Hospital number ${m.hospitalNumber}`}
                </Text>
                <Button size="xs" variant="default" onClick={() => choose(m)}>Use this record</Button>
              </Group>
            ))}
            <Group>
              <Button size="xs" color="orange" onClick={() => create(true)} loading={saving}>Create anyway</Button>
            </Group>
          </Stack>
        </Alert>
      )}

      <Group justify="flex-end">
        <Button variant="default" component={Link} to={`${base}/episodes`}>Cancel</Button>
        <Button onClick={() => create()} loading={saving}>Create episode</Button>
      </Group>
    </Stack>
  );
}

/** Searches this hospital's patients as staff type; never lists anyone without a precise term. */
function PatientSearch({ base, onChoose }: { base: string; onChoose: (p: FoundPatient) => void }) {
  const [term, setTerm] = useState("");
  const [dob, setDob] = useState("");
  const [result, setResult] = useState<SearchResult | null>(null);
  const [error, setError] = useState("");
  const query = searchQuery(term, dob);

  useEffect(() => {
    setError("");
    if (!query) {
      setResult(null);
      return;
    }
    let stale = false; // a newer search replaced this one while it was on its way
    const timer = setTimeout(() => {
      api<SearchResult>(`${base}/patients?${query}`)
        .then((r) => !stale && setResult(r))
        .catch((e: Error) => !stale && setError(e.message));
    }, 300);
    return () => {
      stale = true;
      clearTimeout(timer);
    };
  }, [base, query]);

  return (
    <Card withBorder>
      <Title order={4}>Find the patient</Title>
      <Text size="sm" c="dimmed" mb="sm">
        Search this hospital&apos;s patients by {nationalIdLabel()}, hospital number or surname (at least 3 letters). Add the date of
        birth to narrow a common surname.
      </Text>
      <SimpleGrid cols={{ base: 1, sm: 2 }}>
        <TextInput
          label="Search"
          placeholder={`${nationalIdLabel()}, hospital number or surname`}
          leftSection={<IconSearch size={16} />}
          value={term}
          onChange={(e) => setTerm(e.currentTarget.value)}
        />
        <TextInput label="Date of birth (optional)" type="date" value={dob} onChange={(e) => setDob(e.currentTarget.value)} />
      </SimpleGrid>
      {error && <Alert color="red" mt="sm">{error}</Alert>}
      {result && result.patients.length === 0 && (
        <Text size="sm" mt="sm">No patients at this hospital match. Enter their details below.</Text>
      )}
      {result && result.patients.length > 0 && (
        <Table mt="sm" verticalSpacing="xs" aria-label="Patients found">
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Name</Table.Th>
              <Table.Th>Date of birth</Table.Th>
              <Table.Th>{nationalIdLabel()}</Table.Th>
              <Table.Th>Hospital number</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {result.patients.map((p) => (
              <Table.Tr key={p.id}>
                <Table.Td fw={600}>{patientName(p)}</Table.Td>
                <Table.Td>{formatDate(p.dateOfBirth)}</Table.Td>
                <Table.Td>{p.nationalId || "—"}</Table.Td>
                <Table.Td>{p.hospitalNumber || "—"}</Table.Td>
                <Table.Td ta="right">
                  <Button size="xs" onClick={() => onChoose(p)} aria-label={`Choose ${patientName(p)}`}>Choose</Button>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}
      {result?.more && (
        <Text size="sm" c="dimmed" mt="xs">
          Showing the first 20. {dob ? "Type more to narrow the search." : "Type more, or add the date of birth, to narrow the search."}
        </Text>
      )}
    </Card>
  );
}
