import { Pipe, PipeTransform } from '@angular/core';
import { marked } from 'marked';

@Pipe({
  name: 'assistantMarkdown',
})
export class AssistantMarkdownPipe implements PipeTransform {
  transform(value: string | null | undefined): string {
    if (!value) {
      return '';
    }

    // Do not allow model output to inject raw HTML. Angular applies another
    // sanitization pass when this string is bound through [innerHTML].
    const withoutRawHtml = value.replaceAll('&', '&amp;').replaceAll('<', '&lt;');
    return marked.parse(withoutRawHtml, { async: false, breaks: true, gfm: true });
  }
}
