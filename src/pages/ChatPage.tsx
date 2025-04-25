
import React from 'react';
import ChatList from '@/components/chat/ChatList';
import MessageView from '@/components/chat/MessageView';
import MessageInput from '@/components/chat/MessageInput';
import { useAuth } from '@/contexts/AuthContext';
import { Button } from '@/components/ui/button';
import { LogOut } from 'lucide-react';

const ChatPage: React.FC = () => {
  const { logout } = useAuth();
  
  return (
    <div className="flex flex-col h-screen">
      <header className="bg-chat-dark text-white p-4 flex justify-between items-center">
        <h1 className="text-xl font-bold">Chat App</h1>
        <Button 
          variant="ghost" 
          onClick={logout}
          className="text-white hover:text-white hover:bg-opacity-20"
          size="sm"
        >
          <LogOut className="h-4 w-4 mr-1" />
          Logout
        </Button>
      </header>
      
      <div className="flex flex-1 overflow-hidden">
        <div className="w-80 h-full hidden md:block">
          <ChatList />
        </div>
        
        <div className="flex flex-col flex-1">
          <MessageView />
          <MessageInput />
        </div>
      </div>
    </div>
  );
};

export default ChatPage;
