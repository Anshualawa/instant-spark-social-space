
import { WebSocketMessage } from '../types';

class WebSocketService {
  private socket: WebSocket | null = null;
  private reconnectTimer: NodeJS.Timeout | null = null;
  private messageListeners: ((message: WebSocketMessage) => void)[] = [];
  private statusListeners: ((status: 'connected' | 'disconnected' | 'connecting') => void)[] = [];

  // Initialize WebSocket connection
  connect(token: string): void {
    if (this.socket) {
      this.socket.close();
    }

    this.notifyStatusListeners('connecting');
    
    // WebSocket connection with authentication token
    const wsUrl = `ws://localhost:8000/ws?token=${token}`;
    console.log(`Connecting to WebSocket at ${wsUrl}`);
    this.socket = new WebSocket(wsUrl);
    
    // Connection opened
    this.socket.onopen = () => {
      console.log('WebSocket connected');
      this.notifyStatusListeners('connected');
      
      // Clear any reconnect timer
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }
    };
    
    // Listen for messages
    this.socket.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        console.log('WebSocket message received:', message);
        this.notifyMessageListeners(message);
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };
    
    // Connection closed
    this.socket.onclose = () => {
      console.log('WebSocket disconnected');
      this.notifyStatusListeners('disconnected');
      this.socket = null;
      
      // Attempt to reconnect
      this.reconnectTimer = setTimeout(() => {
        console.log('Attempting to reconnect WebSocket...');
        this.connect(token);
      }, 5000);
    };
    
    // Error handling
    this.socket.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }
  
  // Disconnect WebSocket
  disconnect(): void {
    if (this.socket) {
      this.socket.close();
      this.socket = null;
    }
    
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }
  
  // Send a message
  sendMessage(message: WebSocketMessage): void {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      const messageString = JSON.stringify(message);
      console.log('Sending WebSocket message:', message);
      this.socket.send(messageString);
    } else {
      console.error('WebSocket is not connected. Message not sent:', message);
      // Store message for later sending or notify user
    }
  }
  
  // Add a message listener
  addMessageListener(listener: (message: WebSocketMessage) => void): void {
    this.messageListeners.push(listener);
  }
  
  // Remove a message listener
  removeMessageListener(listener: (message: WebSocketMessage) => void): void {
    this.messageListeners = this.messageListeners.filter(l => l !== listener);
  }
  
  // Add a status listener
  addStatusListener(listener: (status: 'connected' | 'disconnected' | 'connecting') => void): void {
    this.statusListeners.push(listener);
  }
  
  // Remove a status listener
  removeStatusListener(listener: (status: 'connected' | 'disconnected' | 'connecting') => void): void {
    this.statusListeners = this.statusListeners.filter(l => l !== listener);
  }
  
  // Notify all message listeners
  private notifyMessageListeners(message: WebSocketMessage): void {
    this.messageListeners.forEach(listener => {
      try {
        listener(message);
      } catch (error) {
        console.error('Error in message listener:', error);
      }
    });
  }
  
  // Notify all status listeners
  private notifyStatusListeners(status: 'connected' | 'disconnected' | 'connecting'): void {
    this.statusListeners.forEach(listener => {
      try {
        listener(status);
      } catch (error) {
        console.error('Error in status listener:', error);
      }
    });
  }
}

// Create a singleton instance
const webSocketService = new WebSocketService();

export default webSocketService;
