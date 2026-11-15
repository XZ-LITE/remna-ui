import React from "react";

export interface CountryOption {
  code: string;
  name: string;
}

export const COUNTRIES: CountryOption[] = [
  { code: "DE", name: "Германия (Frankfurt)" },
  { code: "NL", name: "Нидерланды (Amsterdam)" },
  { code: "FI", name: "Финляндия (Helsinki)" },
  { code: "US", name: "США (New York)" },
  { code: "TR", name: "Турция (Istanbul)" },
  { code: "SE", name: "Швеция (Stockholm)" },
  { code: "GB", name: "Великобритания (London)" },
  { code: "FR", name: "Франция (Paris)" },
  { code: "PL", name: "Польша (Warsaw)" },
  { code: "KZ", name: "Казахстан (Almaty)" },
  { code: "ES", name: "Испания (Madrid)" },
  { code: "CH", name: "Швейцария (Zurich)" },
  { code: "AT", name: "Австрия (Vienna)" },
  { code: "SG", name: "Сингапур" },
  { code: "JP", name: "Япония (Tokyo)" },
];

interface CountrySelectorProps {
  value: string;
  onChange: (code: string) => void;
  style?: React.CSSProperties;
}

export const CountrySelector: React.FC<CountrySelectorProps> = ({
  value,
  onChange,
  style,
}) => {
  return (
    <select
      className="win-select"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      style={{ minWidth: "220px", ...style }}
    >
      {COUNTRIES.map((c) => (
        <option key={c.code} value={c.code}>
          {c.code} — {c.name}
        </option>
      ))}
    </select>
  );
};
