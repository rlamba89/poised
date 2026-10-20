// The Preopsis logo: the mark (an eye with a tick: the patient seen and checked before the
// operation) and the name, with "op" in the brand colour. It takes its size from the text around
// it, so put it inside a Title or Text. docs/brand/ has the same logo as files.

const BRAND = "var(--mantine-color-brand-7)";

/** The mark alone. `inverse` is for a brand-coloured background: a white tile, a teal eye. */
export function Mark({ size = "1.3em", inverse = false }: { size?: string; inverse?: boolean }) {
  const tile = inverse ? "#fff" : BRAND;
  const eye = inverse ? BRAND : "#fff";
  return (
    <svg viewBox="0 0 32 32" width={size} height={size} aria-hidden="true" style={{ flexShrink: 0 }}>
      <rect width="32" height="32" rx="8" fill={tile} />
      <path d="M4.5 16C8.5 10.2 12 8.5 16 8.5S23.5 10.2 27.5 16C23.5 21.8 20 23.5 16 23.5S8.5 21.8 4.5 16Z" fill={eye} />
      <path d="M11.2 16.2l3.3 3.3 6.4-6.6" fill="none" stroke={tile} strokeWidth="2.6" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

/** The mark and the name. Its text is "Preopsis", so a heading holding it is named Preopsis. */
export function Logo({ inverse = false }: { inverse?: boolean }) {
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: "0.4em", verticalAlign: "middle" }}>
      <Mark inverse={inverse} />
      <span style={{ fontWeight: 700, letterSpacing: "-0.01em" }}>
        Pre<span style={{ color: inverse ? undefined : BRAND }}>op</span>sis
      </span>
    </span>
  );
}
