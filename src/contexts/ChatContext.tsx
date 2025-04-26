
import React, { createContext, useContext, useState, useEffect } from 'react';
import { Chat, Message, User } from '../types';
import { chatApi } from '../services/api';
import { useAuth } from './AuthContext';
import { useMessages } from './MessageContext';
import { useWebSocket } from './WebSocketContext';
import { useToast } from '@/components/ui/use-toast';

interface ChatContextType {
  chats: Chat[];
  activeChat: Chat | null;
  isLoading: boolean;
  typingUsers: Record<string, boolean>;
  setActiveChat: (chat: Chat | null) => void;
  createChat: (participantId: string) => Promise<void>;
  createGroupChat: (name: string, participantIds: string[]) => Promise<void>;
  setTyping: (isTyping: boolean) => void;
  updateChatWithLatestMessage: (chatId: string, message: Message) => void;
  handleTyping: (data: { chatId: string; userId: string; isTyping: boolean }) => void;
  handleStatusChange: (data: { userId: string; isOnline: boolean }) => void;
}

const ChatContext = createContext<ChatContextType>({
  chats: [],
  activeChat: null,
  isLoading: false,
  typingUsers: {},
  setActiveChat: () => {},
  createChat: async () => {},
  createGroupChat: async () => {},
  setTyping: () => {},
  updateChatWithLatestMessage: () => {},
  handleTyping: () => {},
  handleStatusChange: () => {},
});

export const useChat = () => useContext(ChatContext);

export const ChatProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [chats, setChats] = useState<Chat[]>([]);
  const [activeChat, setActiveChat] = useState<Chat | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [typingUsers, setTypingUsers] = useState<Record<string, boolean>>({});
  
  const { user, isAuthenticated } = useAuth();
  const { toast } = useToast();
  const { loadMessages, handleNewMessage } = useMessages();
  const { wsStatus, sendMessage: sendWsMessage } = useWebSocket();

  useEffect(() => {
    if (isAuthenticated) {
      loadChats();
    }
  }, [isAuthenticated]);

  useEffect(() => {
    if (activeChat) {
      loadMessages(activeChat.id);
    }
  }, [activeChat, loadMessages]);

  const loadChats = async () => {
    setIsLoading(true);
    try {
      const chatData = await chatApi.getChats();
      setChats(chatData || []);
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

  const createChat = async (participantId: string) => {
    setIsLoading(true);
    try {
      const newChat = await chatApi.createChat([participantId]);
      if (newChat) {
        setChats(prev => [newChat, ...(prev || [])]);
        setActiveChat(newChat);
        toast({
          title: "Chat created",
          description: `New conversation started`,
        });
      }
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
      if (newChat) {
        setChats(prev => [newChat, ...(prev || [])]);
        setActiveChat(newChat);
        toast({
          title: "Group created",
          description: `Group "${name}" has been created`,
        });
      }
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

  const setTyping = (isTyping: boolean) => {
    if (activeChat && user) {
      sendWsMessage({
        type: 'typing',
        payload: {
          chatId: activeChat.id,
          userId: user.id,
          isTyping
        }
      });
    }
  };

  const updateChatWithLatestMessage = (chatId: string, message: Message) => {
    setChats(prev => {
      if (!prev) return [];
      return prev.map(chat => {
        if (chat.id === chatId) {
          return {
            ...chat,
            lastMessage: message,
            unreadCount: activeChat && activeChat.id === chatId ? 0 : (chat.unreadCount || 0) + 1
          };
        }
        return chat;
      });
    });
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
    setChats(prev => {
      if (!prev) return [];
      return prev.map(chat => ({
        ...chat,
        participants: chat.participants.map(p => 
          p.id === data.userId ? { ...p, isOnline: data.isOnline } : p
        )
      }));
    });
  };

  return (
    <ChatContext.Provider
      value={{
        chats,
        activeChat,
        isLoading,
        typingUsers,
        setActiveChat,
        createChat,
        createGroupChat,
        setTyping,
        updateChatWithLatestMessage,
        handleTyping,
        handleStatusChange,
      }}
    >
      {children}
    </ChatContext.Provider>
  );
};
