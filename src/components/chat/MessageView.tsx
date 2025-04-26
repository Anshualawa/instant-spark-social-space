
import React, { useEffect, useRef } from 'react';
import { useChat } from '@/contexts/ChatContext';
import { useAuth } from '@/contexts/AuthContext';
import { format } from 'date-fns';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';

const MessageView: React.FC = () => {
  const { activeChat, messages = [], typingUsers = {} } = useChat();
  const { user } = useAuth();
  const scrollRef = useRef<HTMLDivElement>(null);
  
  // Auto scroll to bottom on new messages
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);
  
  // Get chat name
  const getChatName = () => {
    if (!activeChat) return '';
    
    if (activeChat.isGroup) return activeChat.name;
    
    const otherParticipant = activeChat.participants.find(p => p.id !== user?.id);
    return otherParticipant?.username || 'Unknown User';
  };
  
  // Format message time
  const formatMessageTime = (timestamp: string) => {
    return format(new Date(timestamp), 'h:mm a');
  };
  
  // Check if user is typing
  const isAnyoneTyping = () => {
    return typingUsers && Object.values(typingUsers).some(isTyping => isTyping);
  };
  
  // Get typing indicator text
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
    } else {
      return 'Typing...';
    }
  };
  
  // Render message sender avatar and name
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
  
  if (!activeChat) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center p-6 bg-white dark:bg-gray-950">
        <div className="text-center max-w-md">
          <h3 className="text-2xl font-bold mb-2">Welcome to Chat</h3>
          <p className="text-muted-foreground">
            Select a conversation or start a new one to begin messaging
          </p>
        </div>
      </div>
    );
  }
  
  return (
    <ScrollArea className="flex-1 px-4 pt-4 pb-16">
      <div className="space-y-4">
        {/* Chat Header */}
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
        
        {/* Messages */}
        {!messages || messages.length === 0 ? (
          <div className="py-8 text-center">
            <p className="text-muted-foreground">No messages yet</p>
            <p className="text-sm">Send a message to start the conversation</p>
          </div>
        ) : (
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
        )}
        
        {/* Typing indicator */}
        {isAnyoneTyping() && (
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
        )}
        
        {/* Scroll anchor */}
        <div ref={scrollRef} />
      </div>
    </ScrollArea>
  );
};

export default MessageView;
