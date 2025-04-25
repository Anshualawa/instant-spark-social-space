
import React, { useState, useEffect, useRef } from 'react';
import { useChat } from '@/contexts/ChatContext';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { SendHorizonal } from 'lucide-react';

const MessageInput: React.FC = () => {
  const [message, setMessage] = useState('');
  const { activeChat, sendMessage, setTyping } = useChat();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  
  // Focus textarea when active chat changes
  useEffect(() => {
    if (activeChat && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [activeChat]);
  
  // Handle typing indicator with debounce
  useEffect(() => {
    if (message.trim() && activeChat) {
      setTyping(true);
      
      // Clear previous timeout
      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }
      
      // Set timeout to clear typing indicator
      typingTimeoutRef.current = setTimeout(() => {
        setTyping(false);
      }, 3000);
    } else {
      setTyping(false);
    }
    
    return () => {
      if (typingTimeoutRef.current) {
        clearTimeout(typingTimeoutRef.current);
      }
    };
  }, [message, activeChat, setTyping]);
  
  // Handle send on Enter (but allow Shift+Enter for new line)
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSendMessage();
    }
  };
  
  // Handle sending message
  const handleSendMessage = async () => {
    if (!message.trim() || !activeChat) return;
    
    await sendMessage(message);
    setMessage('');
  };
  
  if (!activeChat) return null;
  
  return (
    <div className="p-4 border-t bg-white dark:bg-gray-950">
      <div className="flex items-end space-x-2">
        <Textarea
          ref={textareaRef}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Type a message..."
          className="min-h-10 resize-none"
          maxRows={5}
        />
        <Button
          onClick={handleSendMessage}
          disabled={!message.trim()}
          className="h-10 px-3 rounded-full"
          size="icon"
        >
          <SendHorizonal className="h-5 w-5" />
        </Button>
      </div>
    </div>
  );
};

export default MessageInput;
