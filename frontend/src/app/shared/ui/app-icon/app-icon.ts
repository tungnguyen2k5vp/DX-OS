import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { NavigationIcon } from '../../../core/navigation/navigation.model';

@Component({
  selector: 'app-icon',
  templateUrl: './app-icon.html',
  styleUrl: './app-icon.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppIcon {
  readonly name = input.required<NavigationIcon>();
  readonly size = input<'sm' | 'md' | 'lg'>('md');
}
