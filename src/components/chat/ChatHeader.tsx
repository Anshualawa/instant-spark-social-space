
import React from 'react';
import { useAuth } from '@/contexts/AuthContext';
import { Chat } from '@/types';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';

interface ChatHeaderProps {
  activeChat: Chat;
}

const ChatHeader: React.FC<ChatHeaderProps> = ({ activeChat }) => {
  const { user } = useAuth();

  const getChatName = () => {
    if (activeChat.isGroup) return activeChat.name;
    const otherParticipant = activeChat.participants.find(p => p.id !== user?.id);
    return otherParticipant?.username || 'Unknown User';
  };

  return (
    <div className="sticky top-0 bg-white dark:bg-gray-950 py-2 border-b z-10">
      <div className="flex items-center space-x-2">
        <Avatar>
          {activeChat.isGroup ? (
            <AvatarFallback className="bg-chat-accent text-white">
              {activeChat.name.substring(0, 2).toUpperCase()}
            </AvatarFallback>
          ) : (
            <>
              <AvatarFallback>
                {getChatName().substring(0, 2).toUpperCase()}
              </AvatarFallback>
              {activeChat.participants.find(p => p.id !== user?.id)?.avatar && (
                <AvatarImage 
                  src={activeChat.participants.find(p => p.id !== user?.id)?.avatar} 
                />
              )}
            </>
          )}
        </Avatar>
        <div>
          <h3 className="font-semibold">{getChatName()}</h3>
          {activeChat.isGroup ? (
            <p className="text-xs text-muted-foreground">
              {activeChat.participants.length} members
            </p>
          ) : (
            <div className="flex items-center space-x-1">
              <div className={`w-2 h-2 rounded-full ${
                activeChat.participants.find(p => p.id !== user?.id)?.isOnline 
                  ? 'bg-green-500' 
                  : 'bg-gray-400'
              }`} />
              <p className="text-xs text-muted-foreground">
                {activeChat.participants.find(p => p.id !== user?.id)?.isOnline 
                  ? 'Online' 
                  : 'Offline'}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ChatHeader;
