
import React from 'react';
import { useAuth } from '@/contexts/AuthContext';
import { ChatProvider } from '@/contexts/ChatContext';
import AuthPage from './AuthPage';
import ChatPage from './ChatPage';

const Index: React.FC = () => {
  const { isAuthenticated, isLoading } = useAuth();
  
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-chat-dark">
        <div className="text-white text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-4 border-t-chat-accent border-r-transparent border-b-transparent border-l-transparent mx-auto mb-4"></div>
          <p>Loading...</p>
        </div>
      </div>
    );
  }
  
  if (!isAuthenticated) {
    return <AuthPage />;
  }
  
  return (
    <ChatProvider>
      <ChatPage />
    </ChatProvider>
  );
};

export default Index;
