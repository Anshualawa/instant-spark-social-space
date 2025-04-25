
import React, { createContext, useContext, useEffect, useState } from 'react';
import { Chat, Message, User, WebSocketMessage } from '../types';
import { chatApi } from '../services/api';
import webSocketService from '../services/websocket';
import { useAuth } from './AuthContext';
import { useToast } from '@/components/ui/use-toast';

interface ChatContextType {
  chats: Chat[];
  activeChat: Chat | null;
  messages: Message[];
  isLoading: boolean;
  sendMessage: (content: string) => Promise<void>;
  setActiveChat: (chat: Chat | null) => void;
  createChat: (participantId: string) => Promise<void>;
  createGroupChat: (name: string, participantIds: string[]) => Promise<void>;
  wsStatus: 'connected' | 'disconnected' | 'connecting';
  typingUsers: Record<string, boolean>;
  setTyping: (isTyping: boolean) => void;
}

const ChatContext = createContext<ChatContextType>({
  chats: [],
  activeChat: null,
  messages: [],
  isLoading: false,
  sendMessage: async () => {},
  setActiveChat: () => {},
  createChat: async () => {},
  createGroupChat: async () => {},
  wsStatus: 'disconnected',
  typingUsers: {},
  setTyping: () => {},
});

export const useChat = () => useContext(ChatContext);

export const ChatProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [chats, setChats] = useState<Chat[]>([]);
  const [activeChat, setActiveChat] = useState<Chat | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [wsStatus, setWsStatus] = useState<'connected' | 'disconnected' | 'connecting'>('disconnected');
  const [typingUsers, setTypingUsers] = useState<Record<string, boolean>>({});
  
  const { user, isAuthenticated } = useAuth();
  const { toast } = useToast();

  // Initialize WebSocket connection when authenticated
  useEffect(() => {
    if (isAuthenticated) {
      const token = localStorage.getItem('token');
      if (token) {
        webSocketService.connect(token);
        
        // Set up WebSocket status listener
        const statusListener = (status: 'connected' | 'disconnected' | 'connecting') => {
          setWsStatus(status);
        };
        
        webSocketService.addStatusListener(statusListener);
        
        // Clean up
        return () => {
          webSocketService.removeStatusListener(statusListener);
          webSocketService.disconnect();
        };
      }
    }
  }, [isAuthenticated]);

  // Fetch chats when authenticated
  useEffect(() => {
    if (isAuthenticated) {
      loadChats();
    }
  }, [isAuthenticated]);

  // Listen for WebSocket messages
  useEffect(() => {
    if (isAuthenticated) {
      const messageListener = (wsMessage: WebSocketMessage) => {
        switch (wsMessage.type) {
          case 'message':
            handleNewMessage(wsMessage.payload);
            break;
          case 'typing':
            handleTyping(wsMessage.payload);
            break;
          case 'status':
            handleStatusChange(wsMessage.payload);
            break;
          default:
            console.log('Unknown message type:', wsMessage.type);
        }
      };
      
      webSocketService.addMessageListener(messageListener);
      
      return () => {
        webSocketService.removeMessageListener(messageListener);
      };
    }
  }, [isAuthenticated, activeChat, chats]);

  // Fetch messages when active chat changes
  useEffect(() => {
    if (activeChat) {
      loadMessages(activeChat.id);
    } else {
      setMessages([]);
    }
  }, [activeChat]);

  // Load all chats
  const loadChats = async () => {
    setIsLoading(true);
    try {
      const chatData = await chatApi.getChats();
      setChats(chatData);
    } catch (error) {
      console.error('Error loading chats:', error);
      toast({
        variant: "destructive",
        title: "Failed to load chats",
        description: "Please check your connection and try again",
      });
    } finally {
      setIsLoading(false);
    }
  };

  // Load messages for a specific chat
  const loadMessages = async (chatId: string) => {
    setIsLoading(true);
    try {
      const messageData = await chatApi.getMessages(chatId);
      setMessages(messageData);
    } catch (error) {
      console.error(`Error loading messages for chat ${chatId}:`, error);
      toast({
        variant: "destructive",
        title: "Failed to load messages",
        description: "Please check your connection and try again",
      });
    } finally {
      setIsLoading(false);
    }
  };

  // Send a message
  const sendMessage = async (content: string) => {
    if (!activeChat || !content.trim() || !user) return;
    
    try {
      // Optimistically add message to UI
      const tempId = `temp-${Date.now()}`;
      const tempMessage: Message = {
        id: tempId,
        chatId: activeChat.id,
        senderId: user.id,
        content,
        timestamp: new Date().toISOString(),
        isRead: false,
      };
      
      setMessages(prev => [...prev, tempMessage]);
      
      // Send through API
      const sentMessage = await chatApi.sendMessage(activeChat.id, content);
      
      // Update with real message from server
      setMessages(prev => 
        prev.map(msg => msg.id === tempId ? sentMessage : msg)
      );
      
      // Update chat list with latest message
      updateChatWithLatestMessage(activeChat.id, sentMessage);
      
      // Clear typing indicator
      setTyping(false);
      
    } catch (error) {
      console.error('Error sending message:', error);
      toast({
        variant: "destructive",
        title: "Failed to send message",
        description: "Please check your connection and try again",
      });
      
      // Remove temp message on error
      setMessages(prev => prev.filter(msg => !msg.id.startsWith('temp-')));
    }
  };

  // Create a new private chat
  const createChat = async (participantId: string) => {
    setIsLoading(true);
    try {
      const newChat = await chatApi.createChat([participantId]);
      setChats(prev => [newChat, ...prev]);
      setActiveChat(newChat);
      toast({
        title: "Chat created",
        description: `New conversation started`,
      });
    } catch (error) {
      console.error('Error creating chat:', error);
      toast({
        variant: "destructive",
        title: "Failed to create chat",
        description: "Please try again later",
      });
    } finally {
      setIsLoading(false);
    }
  };

  // Create a new group chat
  const createGroupChat = async (name: string, participantIds: string[]) => {
    setIsLoading(true);
    try {
      const newChat = await chatApi.createGroupChat(name, participantIds);
      setChats(prev => [newChat, ...prev]);
      setActiveChat(newChat);
      toast({
        title: "Group created",
        description: `Group "${name}" has been created`,
      });
    } catch (error) {
      console.error('Error creating group chat:', error);
      toast({
        variant: "destructive",
        title: "Failed to create group",
        description: "Please try again later",
      });
    } finally {
      setIsLoading(false);
    }
  };

  // Handle incoming new message
  const handleNewMessage = (message: Message) => {
    // Add message to current chat if it's active
    if (activeChat && activeChat.id === message.chatId) {
      setMessages(prev => [...prev, message]);
    }
    
    // Update chat list with latest message
    updateChatWithLatestMessage(message.chatId, message);
    
    // Notify if not in active chat
    if (!activeChat || activeChat.id !== message.chatId) {
      const chat = chats.find(c => c.id === message.chatId);
      if (chat) {
        const sender = chat.participants.find(p => p.id === message.senderId);
        toast({
          title: chat.isGroup ? chat.name : sender?.username || "New message",
          description: message.content,
        });
      }
    }
  };

  // Update a chat with the latest message
  const updateChatWithLatestMessage = (chatId: string, message: Message) => {
    setChats(prev => prev.map(chat => {
      if (chat.id === chatId) {
        return {
          ...chat,
          lastMessage: message,
          unreadCount: activeChat && activeChat.id === chatId ? 0 : (chat.unreadCount || 0) + 1
        };
      }
      return chat;
    }));
  };

  // Handle typing indicator
  const handleTyping = (data: { chatId: string, userId: string, isTyping: boolean }) => {
    if (activeChat && activeChat.id === data.chatId && user?.id !== data.userId) {
      setTypingUsers(prev => ({
        ...prev,
        [data.userId]: data.isTyping
      }));
    }
  };

  // Handle user status changes
  const handleStatusChange = (data: { userId: string, isOnline: boolean }) => {
    // Update user status in chats
    setChats(prev => prev.map(chat => ({
      ...chat,
      participants: chat.participants.map(p => 
        p.id === data.userId ? { ...p, isOnline: data.isOnline } : p
      )
    })));
  };

  // Send typing indicator
  const setTyping = (isTyping: boolean) => {
    if (activeChat && user) {
      webSocketService.sendMessage({
        type: 'typing',
        payload: {
          chatId: activeChat.id,
          userId: user.id,
          isTyping
        }
      });
    }
  };

  return (
    <ChatContext.Provider
      value={{
        chats,
        activeChat,
        messages,
        isLoading,
        sendMessage,
        setActiveChat,
        createChat,
        createGroupChat,
        wsStatus,
        typingUsers,
        setTyping,
      }}
    >
      {children}
    </ChatContext.Provider>
  );
};
