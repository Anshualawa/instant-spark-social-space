
import { User, AuthUser, Chat, Message } from '../types';

// Base API URL - in production this would be an environment variable
const API_BASE_URL = 'http://localhost:8000/api';

// Helper function for making API calls
const fetchApi = async (endpoint: string, options: RequestInit = {}) => {
  const token = localStorage.getItem('token');
  
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || 'Something went wrong');
  }

  return response.json();
};

// Auth service
export const authApi = {
  login: async (email: string, password: string): Promise<AuthUser> => {
    const data = await fetchApi('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    
    // Store token for future requests
    localStorage.setItem('token', data.token);
    
    return data;
  },
  
  register: async (username: string, email: string, password: string): Promise<AuthUser> => {
    const data = await fetchApi('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    });
    
    // Store token for future requests
    localStorage.setItem('token', data.token);
    
    return data;
  },
  
  logout: () => {
    localStorage.removeItem('token');
  },
  
  getCurrentUser: async (): Promise<User> => {
    return fetchApi('/users/me');
  },
};

// Chat service
export const chatApi = {
  getChats: async (): Promise<Chat[]> => {
    return fetchApi('/chats');
  },
  
  getChatById: async (chatId: string): Promise<Chat> => {
    return fetchApi(`/chats/${chatId}`);
  },
  
  getMessages: async (chatId: string): Promise<Message[]> => {
    return fetchApi(`/chats/${chatId}/messages`);
  },
  
  createChat: async (participantIds: string[]): Promise<Chat> => {
    return fetchApi('/chats', {
      method: 'POST',
      body: JSON.stringify({ participantIds }),
    });
  },
  
  createGroupChat: async (name: string, participantIds: string[]): Promise<Chat> => {
    return fetchApi('/chats/group', {
      method: 'POST',
      body: JSON.stringify({ name, participantIds }),
    });
  },
  
  sendMessage: async (chatId: string, content: string): Promise<Message> => {
    return fetchApi(`/chats/${chatId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    });
  },
};

// User service
export const userApi = {
  getUsers: async (): Promise<User[]> => {
    return fetchApi('/users');
  },
  
  getUserById: async (userId: string): Promise<User> => {
    return fetchApi(`/users/${userId}`);
  },
};
