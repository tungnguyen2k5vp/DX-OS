import { TestBed } from '@angular/core/testing';
import { AppIcon } from './app-icon';

describe('AppIcon', () => {
  it('renders a decorative SVG for a supported navigation icon', async () => {
    await TestBed.configureTestingModule({ imports: [AppIcon] }).compileComponents();
    const fixture = TestBed.createComponent(AppIcon);
    fixture.componentRef.setInput('name', 'budget');
    fixture.detectChanges();

    const icon = fixture.nativeElement.querySelector('svg') as SVGElement;
    expect(icon.getAttribute('aria-hidden')).toBe('true');
    expect(icon.querySelector('path')).toBeTruthy();
  });
});
