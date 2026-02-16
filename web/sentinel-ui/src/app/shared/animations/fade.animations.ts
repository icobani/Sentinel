import { trigger, transition, style, animate } from '@angular/animations';

export const fadeIn = trigger('fadeIn', [
  transition(':enter', [
    style({ opacity: 0 }),
    animate('300ms ease-in', style({ opacity: 1 }))
  ])
]);

export const fadeOut = trigger('fadeOut', [
  transition(':leave', [
    animate('200ms ease-out', style({ opacity: 0 }))
  ])
]);

export const fadeInOut = trigger('fadeInOut', [
  transition(':enter', [
    style({ opacity: 0 }),
    animate('300ms ease-in', style({ opacity: 1 }))
  ]),
  transition(':leave', [
    animate('200ms ease-out', style({ opacity: 0 }))
  ])
]);
