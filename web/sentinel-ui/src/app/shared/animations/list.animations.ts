import { trigger, transition, style, animate, query, stagger } from '@angular/animations';

export const listAnimations = trigger('listAnimation', [
  transition('* => *', [
    query(':enter', [
      style({ opacity: 0, transform: 'translateY(-20px)' }),
      stagger(50, [
        animate('300ms ease-out', style({ opacity: 1, transform: 'translateY(0)' }))
      ])
    ], { optional: true })
  ])
]);

export const slideIn = trigger('slideIn', [
  transition(':enter', [
    style({ opacity: 0, transform: 'translateY(-20px)' }),
    animate('300ms ease-out', style({ opacity: 1, transform: 'translateY(0)' }))
  ])
]);

export const highlight = trigger('highlight', [
  transition(':enter', [
    style({ backgroundColor: 'rgba(76, 175, 80, 0.2)' }),
    animate('2000ms ease-out', style({ backgroundColor: 'transparent' }))
  ])
]);
