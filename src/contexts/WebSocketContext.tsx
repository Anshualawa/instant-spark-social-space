
import React, { createContext, useContext, useEffect } from 'react';
import { WebSocketMessage } from '../types';
import webSocketService from '../services/websocket';
import { useAuth } from './AuthContext';
import { useMessages } from './MessageContext';
import { useToast } from '@/components/ui/use-toast';

interface WebSocketContextType {
  wsStatus: 'connected' | 'disconnected' | 'connecting';
  sendMessage: (message: WebSocketMessage) => void;
}

const WebSocketContext = createContext<WebSocketContextType>({
  wsStatus: 'disconnected',
  sendMessage: () => {},
});

export const useWebSocket = () => useContext(WebSocketContext);

export const WebSocketProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [wsStatus, setWsStatus] = React.useState<'connected' | 'disconnected' | 'connecting'>('disconnected');
  const { isAuthenticated } = useAuth();
  const { handleNewMessage } = useMessages();
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

  const sendMessage = (message: WebSocketMessage) => {
    webSocketService.sendMessage(message);
  };

  return (
    <WebSocketContext.Provider
      value={{
        wsStatus,
        sendMessage,
      }}
    >
      {children}
    </WebSocketContext.Provider>
  );
};
