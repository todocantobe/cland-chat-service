const { io } = require('socket.io-client');

// Simple test client
const socket = io('http://localhost:8081', {
  query: {
    'cland-cid': '1'
  },
  reconnection: false
});

socket.on('connect', () => {
  console.log('✅ Connected successfully! Socket ID:', socket.id);
  
  // Send a test message
  const testMessage = {
    msgType: 1,
    sessionId: "test-session",
    msgId: "msg-001",
    src: "U:user_001",
    dst: "room:test-room",
    content: "Hello from simple test",
    contentType: 1,
    ts: Date.now().toString(),
    status: 1
  };
  
  console.log('📤 Sending test message:', testMessage);
  socket.emit('message', testMessage);
});

socket.on('connect_error', (err) => {
  console.log('❌ Connection failed:', err.message);
});

socket.on('disconnect', (reason) => {
  console.log('🔌 Disconnected:', reason);
});

socket.onAny((event, ...args) => {
  console.log('📥 Received event:', event, args);
});

// Auto disconnect after 5 seconds
setTimeout(() => {
  console.log('⏰ Auto-disconnecting...');
  socket.disconnect();
  process.exit(0);
}, 5000);
