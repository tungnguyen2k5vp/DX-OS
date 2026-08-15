export interface NotificationItem {
  id: string;
  eventType: string;
  resourceType: string;
  resourceId: string;
  title: string;
  body: string;
  createdAt: string;
  readAt: string | null;
}

export interface NotificationList {
  items: NotificationItem[];
  page: number;
  pageSize: number;
  total: number;
  pages: number;
  unreadCount: number;
}
