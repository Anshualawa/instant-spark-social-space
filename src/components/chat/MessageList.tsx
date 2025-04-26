
import React from 'react';
import { Message, Chat } from '@/types';
import { useAuth } from '@/contexts/AuthContext';
import { format } from 'date-fns';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';

interface MessageListProps {
  messages: Message[];
  activeChat: Chat;
}

const MessageList: React.FC<MessageListProps> = ({ messages, activeChat }) => {
  const { user } = useAuth();

  const formatMessageTime = (timestamp: string) => {
    return format(new Date(timestamp), 'h:mm a');
  };

  const renderMessageSender = (senderId: string) => {
    if (!activeChat?.isGroup || senderId === user?.id) return null;
    
    const sender = activeChat.participants.find(p => p.id === senderId);
    if (!sender) return null;
    
    return (
      <div className="flex items-center space-x-2 mb-1">
        <Avatar className="w-6 h-6">
          <AvatarFallback className="text-xs">
            {sender.username.substring(0, 2).toUpperCase()}
          </AvatarFallback>
          {sender.avatar && <AvatarImage src={sender.avatar} />}
        </Avatar>
        <span className="text-xs font-medium">{sender.username}</span>
      </div>
    );
  };

  if (!messages || messages.length === 0) {
    return (
      <div className="py-8 text-center">
        <p className="text-muted-foreground">No messages yet</p>
        <p className="text-sm">Send a message to start the conversation</p>
      </div>
    );
  }

  return (
    <div className="space-y-4 pb-4">
      {messages.map(message => {
        const isOwnMessage = message.senderId === user?.id;
        
        return (
          <div 
            key={message.id}
            className={`flex ${isOwnMessage ? 'justify-end' : 'justify-start'}`}
          >
            <div className={`max-w-[80%] ${isOwnMessage ? 'order-2' : 'order-1'}`}>
              {renderMessageSender(message.senderId)}
              <div 
                className={`rounded-lg px-4 py-2 message-appear ${
                  isOwnMessage
                    ? 'bg-chat-accent text-white'
                    : 'bg-gray-100 dark:bg-gray-800'
                }`}
              >
                <p className="whitespace-pre-wrap break-words">{message.content}</p>
                <span className="text-xs block text-right mt-1 opacity-70">
                  {formatMessageTime(message.timestamp)}
                </span>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
};

export default MessageList;
