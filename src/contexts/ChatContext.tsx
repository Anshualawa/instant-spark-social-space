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

  useEffect(() => {
    if (isAuthenticated) {
      const token = localStorage.getItem('token');
      if (token) {
        webSocketService.connect(token);
        
        const statusListener = (status: 'connected' | 'disconnected' | 'connecting') => {
          setWsStatus(status);
        };
        
        webSocketService.addStatusListener(statusListener);
        
        return () => {
          webSocketService.removeStatusListener(statusListener);
          webSocketService.disconnect();
        };
      }
    }
  }, [isAuthenticated]);

  useEffect(() => {
    if (isAuthenticated) {
      loadChats();
    }
  }, [isAuthenticated]);

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

  useEffect(() => {
    if (activeChat) {
      loadMessages(activeChat.id);
    } else {
      setMessages([]);
    }
  }, [activeChat]);

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

  const sendMessage = async (content: string) => {
    if (!activeChat || !content.trim() || !user) return;
    
    try {
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
      
      const sentMessage = await chatApi.sendMessage(activeChat.id, content);
      
      setMessages(prev => 
        prev.map(msg => msg.id === tempId ? sentMessage : msg)
      );
      
      updateChatWithLatestMessage(activeChat.id, sentMessage);
      
      setTyping(false);
      
    } catch (error) {
      console.error('Error sending message:', error);
      toast({
        variant: "destructive",
        title: "Failed to send message",
        description: "Please check your connection and try again",
      });
      
      setMessages(prev => prev.filter(msg => !msg.id.startsWith('temp-')));
    }
  };

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

  const handleNewMessage = (message: Message) => {
    if (activeChat && activeChat.id === message.chatId) {
      setMessages(prev => [...prev, message]);
    }
    
    updateChatWithLatestMessage(message.chatId, message);
    
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

  const handleTyping = (data: { chatId: string, userId: string, isTyping: boolean }) => {
    if (activeChat && activeChat.id === data.chatId && user?.id !== data.userId) {
      setTypingUsers(prev => ({
        ...prev,
        [data.userId]: data.isTyping
      }));
    }
  };

  const handleStatusChange = (data: { userId: string, isOnline: boolean }) => {
    setChats(prev => prev.map(chat => ({
      ...chat,
      participants: chat.participants.map(p => 
        p.id === data.userId ? { ...p, isOnline: data.isOnline } : p
      )
    })));
  };

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
        messages: messages || [],
        isLoading,
        sendMessage,
        setActiveChat,
        createChat,
        createGroupChat,
        wsStatus,
        typingUsers: typingUsers || {},
        setTyping,
      }}
    >
      {children}
    </ChatContext.Provider>
  );
};
