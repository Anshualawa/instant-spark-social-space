
# Real-Time Chat Application

A full-stack real-time chat application built with React, TypeScript, Go, and MySQL.

## Features

- User authentication (login/register)
- One-to-one private messaging
- Group chats
- Real-time messaging with WebSockets
- Online/offline status indicators
- Typing indicators
- Message history

## Tech Stack

### Frontend
- React with TypeScript
- Tailwind CSS for styling
- WebSocket API for real-time communication
- Context API for state management
- React Router for navigation

### Backend
- Go
- JWT for authentication
- Gorilla WebSocket for real-time communication
- Gorilla Mux for routing
- bcrypt for password hashing

### Database
- MySQL (in production)
- In-memory storage for development

## Getting Started

### Frontend

1. Install dependencies:

```bash
npm install
```

2. Start the development server:

```bash
npm run dev
```

The frontend will run on http://localhost:8080

### Backend

1. Navigate to the server directory:

```bash
cd server
```

2. Install Go dependencies:

```bash
go mod download
```

3. Run the server:

```bash
go run main.go
```

The backend will run on http://localhost:8000

## Project Structure

```
├── src/
│   ├── components/         # UI components
│   │   ├── auth/           # Authentication components
│   │   └── chat/           # Chat interface components
│   ├── contexts/           # React contexts for state management
│   ├── pages/              # Application pages
│   ├── services/           # API and WebSocket services
│   ├── types/              # TypeScript type definitions
│   └── App.tsx             # Main application component
├── server/
│   ├── main.go             # Go backend server
│   └── go.mod              # Go module dependencies
```

## Production Setup

For a production environment, you would need to:

1. Set up a MySQL database
2. Configure environment variables for database connection
3. Deploy the frontend and backend to separate servers or containers
4. Configure CORS properly for security
5. Use HTTPS for secure communication
6. Store JWT secrets securely
7. Implement logging and monitoring

## Future Improvements

- File sharing capability
- Message reactions
- Read receipts
- Message search functionality
- User profiles
- Push notifications
