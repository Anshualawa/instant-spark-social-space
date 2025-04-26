
import React, { useEffect, useRef } from 'react';
import { useChat } from '@/contexts/ChatContext';
import { ScrollArea } from '@/components/ui/scroll-area';
import ChatHeader from './ChatHeader';
import MessageList from './MessageList';
import TypingIndicator from './TypingIndicator';

const MessageView: React.FC = () => {
  const { activeChat, messages = [], typingUsers = {} } = useChat();
  const scrollRef = useRef<HTMLDivElement>(null);
  
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);
  
  useEffect(() => {
    console.log('MessageView - current messages:', messages);
  }, [messages]);
  
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
        <ChatHeader activeChat={activeChat} />
        <MessageList messages={messages} activeChat={activeChat} />
        <TypingIndicator typingUsers={typingUsers} activeChat={activeChat} />
        <div ref={scrollRef} />
      </div>
    </ScrollArea>
  );
};

export default MessageView;
