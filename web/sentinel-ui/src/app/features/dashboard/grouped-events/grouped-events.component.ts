import { Component, Input, Output, EventEmitter, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatCardModule } from '@angular/material/card';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatExpansionModule } from '@angular/material/expansion';
import { MatTooltipModule } from '@angular/material/tooltip';
import { RouterModule } from '@angular/router';
import { FileEvent, EventType, GroupedEvents } from '../../../core/models/event.model';
import { FormatDatePipe } from '../../../shared/pipes/format-date.pipe';
import { FileSizePipe } from '../../../shared/pipes/file-size.pipe';
import { TranslatePipe } from '../../../shared/pipes/translate.pipe';
import { TranslateService } from '../../../core/services/translate.service';
import { WebhookStatusIconComponent } from '../../../shared/components/webhook-status-icon/webhook-status-icon.component';
import { EmptyStateComponent } from '../../../shared/components/empty-state/empty-state.component';
import { slideIn, highlight } from '../../../shared/animations/list.animations';

@Component({
  selector: 'app-grouped-events',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatExpansionModule,
    MatTooltipModule,
    RouterModule,
    FormatDatePipe,
    FileSizePipe,
    TranslatePipe,
    WebhookStatusIconComponent,
    EmptyStateComponent
  ],
  animations: [slideIn, highlight],
  templateUrl: './grouped-events.component.html',
  styleUrl: './grouped-events.component.scss'
})
export class GroupedEventsComponent {
  private translateService = inject(TranslateService);

  @Input() events: FileEvent[] = [];
  @Output() eventClick = new EventEmitter<FileEvent>();

  expandedEvents = new Set<number>();

  get groupedEvents(): GroupedEvents[] {
    const groups = new Map<string, GroupedEvents>();

    this.events.forEach(event => {
      const key = `${event.watcher_id}-${event.file_path.substring(0, event.file_path.lastIndexOf('/'))}`;

      if (!groups.has(key)) {
        const dirPath = event.file_path.substring(0, event.file_path.lastIndexOf('/')) || event.file_path;
        groups.set(key, {
          path: dirPath,
          watcher_id: event.watcher_id,
          watcher_name: event.watcher_name || '',
          events: []
        });
      }

      groups.get(key)!.events.push(event);
    });

    return Array.from(groups.values());
  }

  toggleEventDetails(eventId: number): void {
    if (this.expandedEvents.has(eventId)) {
      this.expandedEvents.delete(eventId);
    } else {
      this.expandedEvents.add(eventId);
    }
  }

  isEventExpanded(eventId: number): boolean {
    return this.expandedEvents.has(eventId);
  }

  hasWebhookInfo(event: FileEvent): boolean {
    return event.webhook_status !== undefined;
  }

  getEventIcon(type: EventType): string {
    switch (type) {
      case EventType.CREATED:
        return 'add_circle';
      case EventType.MODIFIED:
        return 'edit';
      case EventType.DELETED:
        return 'delete';
      case EventType.RENAMED:
        return 'drive_file_rename_outline';
      default:
        return 'description';
    }
  }

  getEventLabel(type: EventType): string {
    switch (type) {
      case EventType.CREATED:
        return this.translateService.get('events.created');
      case EventType.MODIFIED:
        return this.translateService.get('events.modified');
      case EventType.DELETED:
        return this.translateService.get('events.deleted');
      case EventType.RENAMED:
        return this.translateService.get('events.renamed');
      default:
        return 'Unknown';
    }
  }

  getEventIconColor(type: EventType): string {
    switch (type) {
      case EventType.CREATED:
        return 'success';
      case EventType.MODIFIED:
        return 'primary';
      case EventType.DELETED:
        return 'warn';
      case EventType.RENAMED:
        return 'accent';
      default:
        return 'default';
    }
  }

  trackByPath(index: number, group: GroupedEvents): string {
    return `${group.watcher_name}-${group.path}`;
  }

  trackByEventId(index: number, event: FileEvent): number {
    return event.id;
  }
}
