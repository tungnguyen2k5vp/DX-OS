import { buildCsv } from './csv-export';

describe('buildCsv', () => {
  it('adds an Excel-compatible UTF-8 marker and escapes special values', () => {
    const csv = buildCsv([
      ['Tên', 'Ghi chú', 'Giá trị'],
      ['Phòng mua sắm', 'Có dấu phẩy, và "ngoặc kép"', 1250000],
      ['Dòng mới', 'Một\nHai', null],
    ]);

    expect(csv.startsWith('\uFEFF')).toBe(true);
    expect(csv).toContain('"Có dấu phẩy, và ""ngoặc kép"""');
    expect(csv).toContain('"Một\nHai"');
    expect(csv).toContain('1250000');
  });
});
