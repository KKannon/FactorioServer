import EventEmitter from "events";

const bus = new EventEmitter();
const wsScheme = window.location.protocol === "https:" ? "wss" : "ws";
const subscriptions = new Set();
let socket = null;
let reconnectTimer = null;

const sendControl = (type, value) => {
    if (!socket || socket.readyState !== WebSocket.OPEN) return false;
    socket.send(JSON.stringify({room_name: "", controls: {type, value}}));
    return true;
};

bus.on('log subscribe', () => {
    subscriptions.add('gamelog');
    sendControl('subscribe', 'gamelog');
});
bus.on('log unsubscribe', () => {
    subscriptions.delete('gamelog');
    sendControl('unsubscribe', 'gamelog');
});
bus.on('server status subscribe', () => {
    subscriptions.add('server_status');
    sendControl('subscribe', 'server_status');
});
bus.on('server status unsubscribe', () => {
    subscriptions.delete('server_status');
    sendControl('unsubscribe', 'server_status');
});
bus.on('metrics subscribe', () => {
    subscriptions.add('system_metrics');
    sendControl('subscribe', 'system_metrics');
});
bus.on('metrics unsubscribe', () => {
    subscriptions.delete('system_metrics');
    sendControl('unsubscribe', 'system_metrics');
});
bus.on('command send', command => {
    if (!sendControl('command', command)) window.flash?.('Console disconnected. Try again.', 'red');
});

function connect() {
    clearTimeout(reconnectTimer);
    socket = new WebSocket(`${wsScheme}://${window.location.host}/ws`);

    socket.onmessage = event => {
        try {
            const {room_name: roomName, message} = JSON.parse(event.data);
            bus.emit(roomName, message);
        } catch (error) {
            console.error('Invalid websocket message', error);
        }
    };
    socket.onopen = () => {
        subscriptions.forEach(room => sendControl('subscribe', room));
        bus.emit('connection', true);
    };
    socket.onerror = () => socket.close();
    socket.onclose = () => {
        socket = null;
        bus.emit('connection', false);
        reconnectTimer = setTimeout(connect, 5000);
    };
}

connect();

export default bus;
