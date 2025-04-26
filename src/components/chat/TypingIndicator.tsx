
import React from 'react';
import { Chat } from '@/types';

interface TypingIndicatorProps {
  typingUsers: Record<string, boolean>;
  activeChat: Chat;
}

const TypingIndicator: React.FC<TypingIndicatorProps> = ({ typingUsers, activeChat }) => {
  const isAnyoneTyping = () => {
    return typingUsers && Object.values(typingUsers).some(isTyping => isTyping);
  };

  const getTypingText = () => {
    if (!activeChat) return '';
    
    const typingUserIds = Object.entries(typingUsers || {})
      .filter(([_, isTyping]) => isTyping)
      .map(([userId]) => userId);
    
    if (typingUserIds.length === 0) return '';
    
    if (activeChat.isGroup) {
      const typingUsernames = typingUserIds
        .map(id => activeChat.participants.find(p => p.id === id)?.username || 'Someone')
        .join(', ');
      
      return `${typingUsernames} ${typingUserIds.length === 1 ? 'is' : 'are'} typing...`;
    }
    
    return 'Typing...';
  };

  if (!isAnyoneTyping()) return null;

  return (
    <div className="text-sm text-muted-foreground flex items-center space-x-2 animate-pulse-once">
      <div className="flex space-x-1">
        <div className="w-2 h-2 rounded-full bg-chat-accent animate-bounce" 
          style={{ animationDelay: '0ms' }} />
        <div className="w-2 h-2 rounded-full bg-chat-accent animate-bounce" 
          style={{ animationDelay: '200ms' }} />
        <div className="w-2 h-2 rounded-full bg-chat-accent animate-bounce" 
          style={{ animationDelay: '400ms' }} />
      </div>
      <span>{getTypingText()}</span>
    </div>
  );
};

export default TypingIndicator;
