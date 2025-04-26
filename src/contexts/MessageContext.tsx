
import React, { createContext, useContext, useState } from 'react';
import { Message } from '../types';
import { chatApi } from '../services/api';
import { useToast } from '@/components/ui/use-toast';

interface MessageContextType {
  messages: Message[];
  isLoading: boolean;
  sendMessage: (chatId: string, content: string, userId: string) => Promise<void>;
  loadMessages: (chatId: string) => Promise<void>;
  handleNewMessage: (message: Message) => void;
}

const MessageContext = createContext<MessageContextType>({
  messages: [],
  isLoading: false,
  sendMessage: async () => {},
  loadMessages: async () => {},
  handleNewMessage: () => {},
});

export const useMessages = () => useContext(MessageContext);

export const MessageProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const { toast } = useToast();

  const loadMessages = async (chatId: string) => {
    setIsLoading(true);
    try {
      const messageData = await chatApi.getMessages(chatId);
      setMessages(messageData || []);
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

  const sendMessage = async (chatId: string, content: string, userId: string) => {
    if (!content.trim() || !userId) return;
    
    try {
      const tempId = `temp-${Date.now()}`;
      const tempMessage: Message = {
        id: tempId,
        chatId,
        senderId: userId,
        content,
        timestamp: new Date().toISOString(),
        isRead: false,
      };
      
      setMessages(prev => {
        if (!prev) return [tempMessage];
        return [...prev, tempMessage];
      });
      
      const sentMessage = await chatApi.sendMessage(chatId, content);
      
      setMessages(prev => {
        if (!prev) return [sentMessage];
        return prev.map(msg => msg.id === tempId ? sentMessage : msg);
      });
      
    } catch (error) {
      console.error('Error sending message:', error);
      toast({
        variant: "destructive",
        title: "Failed to send message",
        description: "Please check your connection and try again",
      });
      
      setMessages(prev => {
        if (!prev) return [];
        return prev.filter(msg => !msg.id.startsWith('temp-'));
      });
    }
  };

  const handleNewMessage = (message: Message) => {
    setMessages(prev => {
      if (!prev) return [message];
      if (!prev.some(m => m.id === message.id)) {
        return [...prev, message];
      }
      return prev;
    });
  };

  return (
    <MessageContext.Provider
      value={{
        messages,
        isLoading,
        sendMessage,
        loadMessages,
        handleNewMessage,
      }}
    >
      {children}
    </MessageContext.Provider>
  );
};
