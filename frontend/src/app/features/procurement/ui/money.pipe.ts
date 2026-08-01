import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'dxMoney',
})
export class MoneyPipe implements PipeTransform {
  transform(value: string | null | undefined, currency = ''): string {
    if (!value || !/^-?\d+(\.\d+)?$/.test(value)) {
      return '—';
    }

    const negative = value.startsWith('-');
    const unsigned = negative ? value.slice(1) : value;
    const [integerPart, fractionalPart = ''] = unsigned.split('.');
    const groupedInteger = integerPart.replace(/\B(?=(\d{3})+(?!\d))/g, '.');
    const trimmedFraction = fractionalPart.replace(/0+$/, '');
    const formatted = `${negative ? '-' : ''}${groupedInteger}${
      trimmedFraction ? `,${trimmedFraction}` : ''
    }`;

    return currency ? `${formatted} ${currency}` : formatted;
  }
}
