import React, { useMemo } from "react";
import { WinSelect, WinSelectOption } from "./WinSelect";

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
  const options: WinSelectOption[] = useMemo(
    () =>
      COUNTRIES.map((c) => ({
        value: c.code,
        label: `${c.code} — ${c.name}`,
      })),
    []
  );

  return (
    <WinSelect
      value={value}
      onChange={onChange}
      options={options}
      style={{ minWidth: "220px", ...style }}
    />
  );
};
