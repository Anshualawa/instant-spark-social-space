
// User related types
export interface User {
  id: string;
  username: string;
  email: string;
  avatar?: string;
  isOnline: boolean;
  lastSeen?: string;
}

export interface AuthUser extends User {
  token: string;
}

// Chat related types
export interface Message {
  id: string;
  chatId: string;
  senderId: string;
  content: string;
  timestamp: string;
  isRead: boolean;
}

export interface Chat {
  id: string;
  name: string; // For group chats
  isGroup: boolean;
  participants: User[];
  lastMessage?: Message;
  unreadCount?: number;
}

// WebSocket types
export interface WebSocketMessage {
  type: 'message' | 'typing' | 'status' | 'join' | 'leave';
  payload: any;
}
