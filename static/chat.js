function sendMessage() {
    const input = document.getElementById('message-input');
    const message = input.value.trim();
    if (!message) return;

    const messages = document.getElementById('messages');
    const userDiv = document.createElement('div');
    userDiv.className = 'message p-4 rounded-2xl bg-blue-500 text-white ml-auto max-w-3xl animate-slide-in';
    userDiv.textContent = message;
    messages.appendChild(userDiv);
    messages.scrollTop = messages.scrollHeight;

    input.value = '';
}

function toggleCalendarSize() {
    const chatPanel = document.getElementById('chat-panel');
    const calendarPanel = document.getElementById('calendar-panel');

    if (chatPanel.style.width === '20%') {
        chatPanel.style.width = '50%';
        calendarPanel.style.width = '50%';
    } else {
        chatPanel.style.width = '20%';
        calendarPanel.style.width = '80%';
    }
}

document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('message-input').focus();
});

// Автообновление календаря после создания события
document.body.addEventListener('htmx:afterRequest', function (e) {
    if (e.detail.xhr.status === 200) {
        // Перезагружаем iframe через 2 сек
        setTimeout(() => {
            const iframe = document.getElementById('google-calendar');
            iframe.src = iframe.src;
        }, 2000);
    }
});
