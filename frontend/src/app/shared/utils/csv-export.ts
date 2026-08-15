export type CsvValue = string | number | boolean | null | undefined;

export function buildCsv(rows: readonly (readonly CsvValue[])[]): string {
  const body = rows.map((row) => row.map((value) => escapeCsvValue(value)).join(',')).join('\r\n');
  return `\uFEFF${body}`;
}

export function downloadCsv(fileName: string, rows: readonly (readonly CsvValue[])[]): void {
  const blob = new Blob([buildCsv(rows)], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = fileName;
  anchor.style.display = 'none';
  document.body.append(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

function escapeCsvValue(value: CsvValue): string {
  const text = value == null ? '' : String(value);
  if (!/[",\r\n]/.test(text)) {
    return text;
  }
  return `"${text.replaceAll('"', '""')}"`;
}
