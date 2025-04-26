
import React, { useState, useRef, KeyboardEvent, ChangeEvent } from 'react';
import { useChat } from '@/contexts/ChatContext';
import { Send } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { useDebounce } from '@/hooks/use-debounce';

const MessageInput: React.FC = () => {
  const [message, setMessage] = useState<string>('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const { activeChat, sendMessage, setTyping } = useChat();

  // Debounce typing indicator to avoid sending too many events
  const debouncedTyping = useDebounce((isTyping: boolean) => {
    setTyping(isTyping);
  }, 500);

  // Handle input change
  const handleChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    setMessage(e.target.value);
    
    // Send typing indicator when user starts typing
    if (e.target.value.length > 0) {
      debouncedTyping(true);
    } else {
      debouncedTyping(false);
    }
  };

  // Handle message send
  const handleSend = async () => {
    if (!message.trim() || !activeChat) return;
    
    try {
      await sendMessage(message.trim());
      setMessage('');
      
      // Clear typing indicator
      debouncedTyping(false);
      
      // Focus back on textarea
      if (textareaRef.current) {
        textareaRef.current.focus();
      }
    } catch (error) {
      console.error('Error sending message:', error);
    }
  };

  // Handle enter key press
  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    // Send on Enter (but not with Shift for new line)
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (!activeChat) return null;

  return (
    <div className="p-4 border-t bg-background flex items-end gap-2">
      <Textarea
        ref={textareaRef}
        value={message}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder="Type a message..."
        className="min-h-[60px] resize-none"
      />
      <Button 
        onClick={handleSend} 
        disabled={!message.trim()}
        className="rounded-full h-10 w-10 p-0 flex-shrink-0"
      >
        <Send className="h-5 w-5" />
      </Button>
    </div>
  );
};

export default MessageInput;
