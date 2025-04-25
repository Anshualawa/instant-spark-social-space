
import React, { useState } from 'react';
import { useChat } from '@/contexts/ChatContext';
import { Chat, User } from '@/types';
import { useAuth } from '@/contexts/AuthContext';
import { formatDistanceToNow } from 'date-fns';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Checkbox } from '@/components/ui/checkbox';
import { userApi } from '@/services/api';
import { useToast } from '@/components/ui/use-toast';
import { Avatar, AvatarImage, AvatarFallback } from '@/components/ui/avatar';

const ChatList: React.FC = () => {
  const { chats, activeChat, setActiveChat, createChat, createGroupChat } = useChat();
  const { user } = useAuth();
  const [isNewChatOpen, setIsNewChatOpen] = useState(false);
  const [isNewGroupOpen, setIsNewGroupOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [users, setUsers] = useState<User[]>([]);
  const [selectedUsers, setSelectedUsers] = useState<string[]>([]);
  const [groupName, setGroupName] = useState('');
  const { toast } = useToast();
  
  // Get chat name based on participants or group name
  const getChatName = (chat: Chat) => {
    if (chat.isGroup) return chat.name;
    
    const otherParticipant = chat.participants.find(p => p.id !== user?.id);
    return otherParticipant?.username || 'Unknown User';
  };
  
  // Get last message preview
  const getLastMessagePreview = (chat: Chat) => {
    if (!chat.lastMessage) return 'No messages yet';
    
    const isSender = chat.lastMessage.senderId === user?.id;
    const prefix = isSender ? 'You: ' : '';
    
    return `${prefix}${chat.lastMessage.content.slice(0, 30)}${chat.lastMessage.content.length > 30 ? '...' : ''}`;
  };
  
  // Get time since last message
  const getTimeAgo = (chat: Chat) => {
    if (!chat.lastMessage?.timestamp) return '';
    return formatDistanceToNow(new Date(chat.lastMessage.timestamp), { addSuffix: true });
  };
  
  // Search users to start a new chat
  const searchUsers = async () => {
    if (searchTerm.length < 2) {
      setUsers([]);
      return;
    }
    
    try {
      // In a real app we'd have a search endpoint
      // For now, let's get all users and filter them
      const allUsers = await userApi.getUsers();
      const filtered = allUsers.filter(u => 
        u.id !== user?.id && 
        u.username.toLowerCase().includes(searchTerm.toLowerCase())
      );
      setUsers(filtered);
    } catch (error) {
      toast({
        variant: "destructive",
        title: "Failed to search users",
        description: "Please try again later",
      });
    }
  };
  
  // Handle starting a new chat
  const handleStartChat = async (userId: string) => {
    try {
      await createChat(userId);
      setIsNewChatOpen(false);
      setSearchTerm('');
      setUsers([]);
    } catch (error) {
      // Error handled in context
    }
  };
  
  // Handle creating a new group
  const handleCreateGroup = async () => {
    if (!groupName.trim() || selectedUsers.length === 0) {
      toast({
        variant: "destructive",
        title: "Invalid group",
        description: "Please provide a group name and select at least one participant",
      });
      return;
    }
    
    try {
      await createGroupChat(groupName, selectedUsers);
      setIsNewGroupOpen(false);
      setGroupName('');
      setSelectedUsers([]);
      setUsers([]);
    } catch (error) {
      // Error handled in context
    }
  };
  
  // Toggle user selection for group chat
  const toggleUserSelection = (userId: string) => {
    if (selectedUsers.includes(userId)) {
      setSelectedUsers(prev => prev.filter(id => id !== userId));
    } else {
      setSelectedUsers(prev => [...prev, userId]);
    }
  };
  
  // Load users for group creation
  const loadUsersForGroup = async () => {
    try {
      const allUsers = await userApi.getUsers();
      setUsers(allUsers.filter(u => u.id !== user?.id));
    } catch (error) {
      toast({
        variant: "destructive",
        title: "Failed to load users",
        description: "Please try again later",
      });
    }
  };
  
  return (
    <div className="h-full flex flex-col bg-gray-50 dark:bg-gray-900 border-r">
      <div className="p-4 border-b">
        <h2 className="text-lg font-semibold mb-3">Your Chats</h2>
        <div className="flex space-x-2">
          {/* New Chat Dialog */}
          <Dialog open={isNewChatOpen} onOpenChange={setIsNewChatOpen}>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">New Chat</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Start a new conversation</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <div className="flex space-x-2">
                  <Input 
                    placeholder="Search users..." 
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                  />
                  <Button onClick={searchUsers}>Search</Button>
                </div>
                
                <ScrollArea className="h-60">
                  {users.length > 0 ? (
                    <ul className="divide-y">
                      {users.map(u => (
                        <li 
                          key={u.id} 
                          className="py-2 px-1 flex items-center justify-between hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer"
                          onClick={() => handleStartChat(u.id)}
                        >
                          <div className="flex items-center space-x-3">
                            <div className={`w-2 h-2 rounded-full ${u.isOnline ? 'bg-green-500' : 'bg-gray-400'}`} />
                            <span>{u.username}</span>
                          </div>
                          <Button size="sm" variant="ghost">Chat</Button>
                        </li>
                      ))}
                    </ul>
                  ) : (
                    <p className="text-center text-muted-foreground p-4">
                      {searchTerm.length < 2 
                        ? 'Type at least 2 characters to search'
                        : 'No users found'}
                    </p>
                  )}
                </ScrollArea>
              </div>
            </DialogContent>
          </Dialog>
          
          {/* New Group Dialog */}
          <Dialog open={isNewGroupOpen} onOpenChange={(open) => {
            setIsNewGroupOpen(open);
            if (open) loadUsersForGroup();
          }}>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">New Group</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create a new group</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                <Input 
                  placeholder="Group name" 
                  value={groupName}
                  onChange={(e) => setGroupName(e.target.value)}
                />
                
                <div>
                  <label className="text-sm font-medium mb-2 block">Select members:</label>
                  <ScrollArea className="h-60">
                    <ul className="divide-y">
                      {users.map(u => (
                        <li key={u.id} className="py-2 px-1 flex items-center justify-between">
                          <div className="flex items-center space-x-3">
                            <Checkbox 
                              checked={selectedUsers.includes(u.id)} 
                              onCheckedChange={() => toggleUserSelection(u.id)} 
                              id={`user-${u.id}`}
                            />
                            <label 
                              htmlFor={`user-${u.id}`}
                              className="flex items-center space-x-2 cursor-pointer"
                            >
                              <div className={`w-2 h-2 rounded-full ${u.isOnline ? 'bg-green-500' : 'bg-gray-400'}`} />
                              <span>{u.username}</span>
                            </label>
                          </div>
                        </li>
                      ))}
                    </ul>
                  </ScrollArea>
                </div>
                
                <Button onClick={handleCreateGroup} className="w-full">
                  Create Group
                </Button>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>
      
      <ScrollArea className="flex-1">
        <ul className="divide-y">
          {chats.length > 0 ? (
            chats.map(chat => (
              <li 
                key={chat.id}
                onClick={() => setActiveChat(chat)}
                className={`py-3 px-4 cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors ${
                  activeChat?.id === chat.id ? 'bg-gray-100 dark:bg-gray-800' : ''
                }`}
              >
                <div className="flex justify-between">
                  <div className="flex items-center space-x-3">
                    <Avatar>
                      {chat.isGroup ? (
                        <AvatarFallback className="bg-chat-accent text-white">
                          {chat.name.substring(0, 2).toUpperCase()}
                        </AvatarFallback>
                      ) : (
                        <>
                          <AvatarFallback>
                            {getChatName(chat).substring(0, 2).toUpperCase()}
                          </AvatarFallback>
                          {chat.participants.find(p => p.id !== user?.id)?.avatar && (
                            <AvatarImage src={chat.participants.find(p => p.id !== user?.id)?.avatar} />
                          )}
                        </>
                      )}
                    </Avatar>
                    <div>
                      <div className="flex items-center space-x-2">
                        <h3 className="font-medium">{getChatName(chat)}</h3>
                        {!chat.isGroup && (
                          <div className={`w-2 h-2 rounded-full ${
                            chat.participants.find(p => p.id !== user?.id)?.isOnline ? 'bg-green-500' : 'bg-gray-400'
                          }`} />
                        )}
                      </div>
                      <p className="text-sm text-muted-foreground truncate">
                        {getLastMessagePreview(chat)}
                      </p>
                    </div>
                  </div>
                  <div className="flex flex-col items-end text-xs">
                    <span className="text-muted-foreground">
                      {getTimeAgo(chat)}
                    </span>
                    {chat.unreadCount ? (
                      <span className="bg-chat-accent text-white rounded-full px-2 py-1 mt-1">
                        {chat.unreadCount}
                      </span>
                    ) : null}
                  </div>
                </div>
              </li>
            ))
          ) : (
            <li className="py-10 px-4 text-center text-muted-foreground">
              <p>No conversations yet</p>
              <p className="text-sm mt-1">Start a new chat to begin messaging</p>
            </li>
          )}
        </ul>
      </ScrollArea>
    </div>
  );
};

export default ChatList;
