import { MoneyPipe } from './money.pipe';

describe('MoneyPipe', () => {
  const pipe = new MoneyPipe();

  it('formats decimal strings without converting through floating point', () => {
    expect(pipe.transform('123456789012345.6700', 'VND')).toBe('123.456.789.012.345,67 VND');
  });

  it('removes an all-zero fractional part', () => {
    expect(pipe.transform('50000000.0000', 'VND')).toBe('50.000.000 VND');
  });

  it('returns a placeholder for invalid input', () => {
    expect(pipe.transform('not-a-number', 'VND')).toBe('—');
  });
});
