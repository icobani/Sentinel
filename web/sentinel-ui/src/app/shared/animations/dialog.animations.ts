import { trigger, transition, style, animate } from '@angular/animations';

/**
 * Dialog scale animation - smooth scale and fade effect for dialogs
 * Usage: Add to dialog component's animations array
 */
export const dialogScale = trigger('dialogScale', [
  transition(':enter', [
    style({
      opacity: 0,
      transform: 'scale(0.9)'
    }),
    animate('200ms cubic-bezier(0.4, 0.0, 0.2, 1)', style({
      opacity: 1,
      transform: 'scale(1)'
    }))
  ]),
  transition(':leave', [
    animate('150ms cubic-bezier(0.4, 0.0, 1, 1)', style({
      opacity: 0,
      transform: 'scale(0.9)'
    }))
  ])
]);

/**
 * Backdrop fade animation for modal overlays
 */
export const backdropFade = trigger('backdropFade', [
  transition(':enter', [
    style({ opacity: 0 }),
    animate('200ms ease-in', style({ opacity: 1 }))
  ]),
  transition(':leave', [
    animate('150ms ease-out', style({ opacity: 0 }))
  ])
]);
