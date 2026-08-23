import { AssistantMarkdownPipe } from './assistant-markdown.pipe';

describe('AssistantMarkdownPipe', () => {
  const pipe = new AssistantMarkdownPipe();

  it('renders common assistant markdown', () => {
    const html = pipe.transform('**Quan trọng**\n\n- Mục một\n- Mục hai\n\n> Lưu ý');

    expect(html).toContain('<strong>Quan trọng</strong>');
    expect(html).toContain('<ul>');
    expect(html).toContain('<blockquote>');
  });

  it('does not pass raw HTML through to the browser', () => {
    const html = pipe.transform('<img src=x onerror=alert(1)>');

    expect(html).not.toContain('<img');
    expect(html).toContain('&lt;img');
  });
});
