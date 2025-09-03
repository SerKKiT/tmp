class StreamManager {
    constructor() {
        this.streams = new Map();
        this.updateInterval = null;
        this.isUpdating = false;
        this.isCreatingStream = false;
        
        this.expandedStreams = new Set();
        
        this.updateStreams = this.updateStreams.bind(this);
        this.handleCreateStream = this.handleCreateStream.bind(this);
        
        document.addEventListener('DOMContentLoaded', () => {
            this.init();
        });
    }

    async init() {
        console.log('🚀 Инициализация Stream Manager...');
        
        this.bindEvents();
        await this.loadStreams();
        this.startAutoUpdate();
        
        console.log('✅ Stream Manager готов к работе');
    }

    bindEvents() {
        const createForm = document.getElementById('createStreamForm');
        if (createForm) {
            createForm.removeEventListener('submit', this.handleCreateStream);
            createForm.addEventListener('submit', this.handleCreateStream);
        }

        const refreshBtn = document.getElementById('refreshBtn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => this.loadStreams());
        }
    }

    async loadStreams() {
        try {
            this.showLoading(true);
            
            const response = await fetch('/api/tasks');
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const data = await response.json();
            const newStreams = data.data || [];
            
            this.updateStreamsInPlace(newStreams);
            this.updateLastUpdate();
            
            console.log(`📊 Загружено ${newStreams.length} стримов`);
            
        } catch (error) {
            console.error('❌ Ошибка загрузки стримов:', error);
            this.showError(`Ошибка загрузки стримов: ${error.message}`);
        } finally {
            this.showLoading(false);
        }
    }

    updateSpecificFields(card, newStream, cachedStream) {
        let hasVisualChanges = false;

        if (newStream.stream_status !== cachedStream.stream_status) {
            this.updateStatusElement(card, newStream.stream_status);
            this.updateButtonsState(card, newStream);
            hasVisualChanges = true;
        }

        if (newStream.updated_at !== cachedStream.updated_at) {
            this.updateTimeElement(card, newStream.updated_at);
        }

        if (newStream.name !== cachedStream.name) {
            this.updateNameElement(card, newStream.name);
        }

        if (hasVisualChanges) {
            this.highlightCardChange(card);
        }
    }

    updateStatusElement(card, newStatus) {
        const statusElement = card.querySelector('.stream-status');
        if (!statusElement) return;

        const newStatusClass = this.getStatusClass(newStatus);
        const newStatusIcon = this.getStatusIcon(newStatus);
        
        const newContent = `${newStatusIcon} ${newStatus}`;
        if (statusElement.innerHTML !== newContent) {
            statusElement.className = `stream-status ${newStatusClass}`;
            statusElement.innerHTML = newContent;
            
            statusElement.style.transform = 'scale(1.05)';
            statusElement.style.transition = 'transform 0.15s ease';
            setTimeout(() => {
                statusElement.style.transform = 'scale(1)';
            }, 150);
        }
    }

    updateButtonsState(card, stream) {
        const actionsContainer = card.querySelector('.stream-actions');
        if (!actionsContainer) return;

        const canStart = ['stopped', 'error'].includes(stream.stream_status);
        const canStop = ['starting', 'running'].includes(stream.stream_status);

        const startBtn = actionsContainer.querySelector('.btn-primary');
        const stopBtn = actionsContainer.querySelector('.btn-danger');

        if (startBtn) {
            startBtn.disabled = !canStart;
            startBtn.style.opacity = canStart ? '1' : '0.6';
        }

        if (stopBtn) {
            stopBtn.disabled = !canStop;
            stopBtn.style.opacity = canStop ? '1' : '0.6';
        }
    }

    resetAllStreamButtons() {
        console.log('🔄 Сброс всех кнопок стримов');
        
        document.querySelectorAll('.stream-actions .btn-primary').forEach(btn => {
            btn.disabled = false;
            btn.innerHTML = '▶️ Запустить';
        });
        
        document.querySelectorAll('.stream-actions .btn-danger').forEach(btn => {
            btn.disabled = false;
            btn.innerHTML = '⏹️ Остановить';
        });
        
        this.showSuccess('🔄 Все кнопки сброшены');
    }

    updateTimeElement(card, newTime) {
        const timeElement = card.querySelector('.updated-time');
        if (timeElement && newTime) {
            const formattedTime = this.formatDate(newTime);
            if (timeElement.textContent !== formattedTime) {
                timeElement.textContent = formattedTime;
            }
        }
    }

    updateNameElement(card, newName) {
        const nameElement = card.querySelector('.stream-name');
        if (nameElement && nameElement.textContent !== newName) {
            nameElement.textContent = newName;
        }
    }
    
    highlightCardChange(card) {
        card.style.boxShadow = '0 0 20px rgba(102, 126, 234, 0.5)';
        card.style.transition = 'box-shadow 0.3s ease';
        
        setTimeout(() => {
            card.style.boxShadow = '';
        }, 800);
    }

    addStreamCardSmoothly(container, stream) {
        const cardHTML = this.renderStreamCard(stream);
        
        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = cardHTML;
        const newCard = tempDiv.firstElementChild;
        
        newCard.style.opacity = '0';
        newCard.style.transform = 'translateY(-10px) scale(0.95)';
        newCard.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
        
        container.appendChild(newCard);
        
        requestAnimationFrame(() => {
            newCard.style.opacity = '1';
            newCard.style.transform = 'translateY(0) scale(1)';
        });
    }

    removeObsoleteStreams(container, newStreamIds) {
        const cardsToRemove = [];
        
        this.streams.forEach((cachedStream, streamId) => {
            if (!newStreamIds.has(streamId)) {
                const card = container.querySelector(`[data-stream-id="${streamId}"]`);
                if (card) {
                    cardsToRemove.push({ card, streamId });
                }
            }
        });
        
        cardsToRemove.forEach(({ card, streamId }) => {
            card.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
            card.style.opacity = '0';
            card.style.transform = 'translateY(-10px) scale(0.95)';
            
            setTimeout(() => {
                if (card.parentNode) {
                    card.remove();
                }
                this.streams.delete(streamId);
                this.expandedStreams.delete(streamId);
            }, 300);
        });
    }
    
    updateStreamsInPlace(newStreams) {
        const container = document.getElementById('streamsContainer');
        if (!container) return;

        if (newStreams.length === 0) {
            if (this.streams.size > 0) {
                container.innerHTML = `
                    <div class="no-streams">
                        <p>📭 Нет созданных стримов</p>
                        <p>Создайте новый стрим, чтобы начать работу</p>
                    </div>
                `;
                this.streams.clear();
            }
            return;
        }

        const noStreamsMsg = container.querySelector('.no-streams');
        if (noStreamsMsg) {
            noStreamsMsg.remove();
        }

        const newStreamIds = new Set(newStreams.map(s => s.stream_id));
        
        newStreams.forEach(newStream => {
            const cachedStream = this.streams.get(newStream.stream_id);
            const card = container.querySelector(`[data-stream-id="${newStream.stream_id}"]`);

            if (card && cachedStream) {
                this.updateSpecificFields(card, newStream, cachedStream);
            } else if (!card) {
                this.addStreamCardSmoothly(container, newStream);
            }

            this.streams.set(newStream.stream_id, { ...newStream });
        });

        this.removeObsoleteStreams(container, newStreamIds);
    }

    renderStreamCard(stream) {
        const statusClass = this.getStatusClass(stream.stream_status);
        const statusIcon = this.getStatusIcon(stream.stream_status);
        
        return `
            <div class="stream-card" data-stream-id="${stream.stream_id}">
                <div class="stream-header">
                    <h3 class="stream-name">${stream.name}</h3>
                    <span class="stream-status ${statusClass}">
                        ${statusIcon} ${stream.stream_status}
                    </span>
                </div>
                
                <div class="stream-info">
                    <div class="stream-detail">
                        <strong>ID:</strong> <code>${stream.stream_id}</code>
                    </div>
                    <div class="stream-detail">
                        <strong>Создан:</strong> ${this.formatDate(stream.created_at)}
                    </div>
                    ${stream.updated_at ? `
                    <div class="stream-detail">
                        <strong>Обновлен:</strong> <span class="updated-time">${this.formatDate(stream.updated_at)}</span>
                    </div>
                    ` : ''}
                </div>

                <div class="stream-actions">
                    ${this.renderStreamButtons(stream)}
                </div>

                <div class="stream-urls" id="urls-${stream.stream_id}" style="display: none;">
                    <!-- URLs будут загружены при необходимости -->
                </div>
            </div>
        `;
    }

    renderStreamButtons(stream) {
        const canStart = ['stopped', 'error'].includes(stream.stream_status);
        const canStop = ['starting', 'running'].includes(stream.stream_status);
        
        return `
            <button class="btn btn-primary" 
                    onclick="streamManager.startStream('${stream.stream_id}')"
                    ${!canStart ? 'disabled' : ''}>
                ▶️ Запустить
            </button>
            
            <button class="btn btn-danger" 
                    onclick="streamManager.stopStream('${stream.stream_id}')"
                    ${!canStop ? 'disabled' : ''}>
                ⏹️ Остановить
            </button>
            
            <button class="btn btn-info" 
                    onclick="streamManager.showStreamInfo('${stream.stream_id}')">
                ℹ️ Подробнее
            </button>
            
            <button class="btn btn-secondary" 
                    onclick="streamManager.deleteStream(${stream.id})">
                🗑️ Удалить
            </button>
        `;
    }

    // ✅ ИСПРАВЛЕННАЯ функция startStream (единственная версия)
    async startStream(streamId) {
        const card = document.querySelector(`[data-stream-id="${streamId}"]`);
        const startBtn = card ? card.querySelector('.btn-primary') : null;
        
        if (!startBtn) {
            console.error('❌ Кнопка запуска не найдена для', streamId);
            return;
        }

        if (startBtn.disabled) {
            console.log('⚠️ Кнопка уже заблокирована, пропускаем запрос');
            return;
        }

        const originalText = startBtn.innerHTML;
        startBtn.disabled = true;
        startBtn.innerHTML = '⏳ Запуск...';

        try {
            console.log(`🚀 Запуск стрима: ${streamId}`);

            // ✅ ПРАВИЛЬНЫЙ API endpoint: POST /api/streams (без ID в URL)
            const response = await fetch('/api/streams', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ 
                    stream_id: streamId, 
                    action: 'start' 
                })
            });

            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}: ${response.statusText}`);
            }

            this.showSuccess(`✅ Стрим ${streamId} запущен`);
            console.log('📡 Ответ сервера:', data);
            
            await this.loadStreams();
            
        } catch (error) {
            console.error('❌ Ошибка запуска стрима:', error);
            this.showError(`Ошибка запуска стрима: ${error.message}`);
        } finally {
            if (startBtn) {
                startBtn.disabled = false;
                startBtn.innerHTML = originalText;
            }
            console.log(`🔓 Кнопка запуска для ${streamId} разблокирована`);
        }
    }

    // ✅ ИСПРАВЛЕННАЯ функция stopStream
    async stopStream(streamId) {
        const card = document.querySelector(`[data-stream-id="${streamId}"]`);
        const stopBtn = card ? card.querySelector('.btn-danger') : null;
        
        if (!stopBtn) {
            console.error('❌ Кнопка остановки не найдена для', streamId);
            return;
        }

        const originalText = stopBtn.innerHTML;
        stopBtn.disabled = true;
        stopBtn.innerHTML = '⏳ Остановка...';

        try {
            console.log(`⏹️ Остановка стрима: ${streamId}`);
            
            // ✅ ПРАВИЛЬНЫЙ API endpoint: POST /api/streams (без ID в URL)
            const response = await fetch('/api/streams', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ 
                    stream_id: streamId, 
                    action: 'stop' 
                })
            });

            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}`);
            }

            this.showSuccess(`⏹️ Стрим ${streamId} остановлен`);
            console.log('📡 Ответ сервера:', data);
            
            await this.loadStreams();
            
        } catch (error) {
            console.error('❌ Ошибка остановки стрима:', error);
            this.showError(`Ошибка остановки стрима: ${error.message}`);
        } finally {
            if (stopBtn) {
                stopBtn.disabled = false;
                stopBtn.innerHTML = originalText;
            }
            console.log(`🔓 Кнопка остановки для ${streamId} разблокирована`);
        }
    }

    // ✅ ИСПРАВЛЕННАЯ функция showStreamInfo
    async showStreamInfo(streamId) {
        try {
            const urlsContainer = document.getElementById(`urls-${streamId}`);
            
            if (urlsContainer.style.display === 'none') {
                this.expandedStreams.add(streamId);
                
                // ✅ ИСПРАВЛЕНО: Используем правильный endpoint для получения информации о потоке
                const response = await fetch(`/api/streams/${streamId}`);
                
                if (response.ok) {
                    const data = await response.json();
                    const streamData = data.data;
                    
                    urlsContainer.innerHTML = `
                        <div class="stream-urls-content">
                            <h4>🔗 Параметры подключения:</h4>
                            <div class="url-item">
                                <strong>SRT URL:</strong>
                                <code>${streamData.srt_url || 'Недоступно'}</code>
                                ${streamData.srt_url ? `<button onclick="streamManager.copyToClipboard('${streamData.srt_url}')" class="copy-btn">📋</button>` : ''}
                            </div>
                            <div class="url-item">
                                <strong>HLS Playlist:</strong>
                                <code>${streamData.hls_url || 'Недоступно'}</code>
                                ${streamData.hls_url ? `<button onclick="streamManager.copyToClipboard('${streamData.hls_url}')" class="copy-btn">📋</button>` : ''}
                            </div>
                            <div class="url-item">
                                <strong>SRT Port:</strong> ${streamData.srt_port || 'Недоступно'}
                            </div>
                            <div class="url-item">
                                <strong>Server IP:</strong> ${streamData.server_ip || 'Недоступно'}
                            </div>
                            ${streamData.stream_start ? `
                            <div class="url-item">
                                <strong>Время начала потока:</strong> ${this.formatDate(streamData.stream_start)}
                            </div>
                            ` : ''}
                        </div>
                    `;
                } else {
                    urlsContainer.innerHTML = `
                        <div class="error">
                            ❌ Не удалось загрузить информацию о стриме
                        </div>
                    `;
                }
                
                urlsContainer.style.display = 'block';
            } else {
                this.expandedStreams.delete(streamId);
                urlsContainer.style.display = 'none';
            }
            
        } catch (error) {
            console.error('❌ Ошибка получения информации о стриме:', error);
            this.showError(`Ошибка получения информации: ${error.message}`);
        }
    }

    async copyToClipboard(text) {
        try {
            if (navigator.clipboard?.writeText) {
                await navigator.clipboard.writeText(text);
                this.showSuccess('📋 Скопировано в буфер обмена');
                return;
            }
            
            const textArea = document.createElement('textarea');
            textArea.value = text;
            textArea.style.position = 'fixed';
            textArea.style.left = '-999999px';
            document.body.appendChild(textArea);
            textArea.focus();
            textArea.select();
            
            const successful = document.execCommand('copy');
            document.body.removeChild(textArea);
            
            if (successful) {
                this.showSuccess('📋 Скопировано в буфер обмена');
            } else {
                throw new Error('Copy command failed');
            }
            
        } catch (error) {
            console.error('❌ Ошибка копирования:', error);
            
            const userWantsCopy = confirm(`Не удалось автоматически скопировать:\n\n${text}\n\nПоказать для ручного копирования?`);
            
            if (userWantsCopy) {
                prompt('Скопируйте этот текст:', text);
            }
            
            this.showError('Ошибка копирования в буфер обмена');
        }
    }

    async handleCreateStream(event) {
        event.preventDefault();
        
        if (this.isCreatingStream) {
            console.log('⚠️ Создание стрима уже в процессе, запрос проигнорирован');
            return;
        }

        const nameInput = document.getElementById('streamName');
        const createBtn = document.getElementById('createStreamBtn');
        const name = nameInput ? nameInput.value.trim() : '';

        if (!name) {
            this.showError('Введите название стрима');
            return;
        }

        try {
            this.isCreatingStream = true;
            
            if (createBtn) {
                createBtn.disabled = true;
                createBtn.textContent = '⏳ Создание...';
            }
            
            if (nameInput) {
                nameInput.disabled = true;
            }

            console.log(`➕ Создание стрима: ${name}`);
            
            const response = await fetch('/api/tasks', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ name: name })
            });

            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}`);
            }

            this.showSuccess(`✅ Стрим "${name}" создан`);
            console.log('📡 Новый стрим:', data.data);
            
            if (nameInput) nameInput.value = '';
            
            await this.loadStreams();
            
        } catch (error) {
            console.error('❌ Ошибка создания стрима:', error);
            this.showError(`Ошибка создания стрима: ${error.message}`);
        } finally {
            this.isCreatingStream = false;
            
            if (createBtn) {
                createBtn.disabled = false;
                createBtn.textContent = '➕ Создать стрим';
            }
            
            if (nameInput) {
                nameInput.disabled = false;
                nameInput.focus();
            }
        }
    }

    async deleteStream(streamDbId) {
        if (!confirm('Вы уверены, что хотите удалить этот стрим?')) {
            return;
        }

        try {
            console.log(`🗑️ Удаление стрима с ID: ${streamDbId}`);
            
            const response = await fetch(`/api/tasks/${streamDbId}`, {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json',
                }
            });

            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.error || `HTTP ${response.status}: ${response.statusText}`);
            }

            this.showSuccess('🗑️ Стрим успешно удален');
            console.log('📡 Стрим удален:', data);
            
            await this.loadStreams();
            
        } catch (error) {
            console.error('❌ Ошибка удаления стрима:', error);
            this.showError(`Ошибка удаления стрима: ${error.message}`);
        }
    }

    startAutoUpdate() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
        }

        this.updateInterval = setInterval(this.updateStreams, 4000);
        console.log('🔄 Автообновление запущено (каждые 4 секунды)');
    }

    async updateStreams() {
        if (this.isUpdating) return;
        
        this.isUpdating = true;
        try {
            const response = await fetch('/api/tasks');
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const data = await response.json();
            this.updateStreamsInPlace(data.data || []);
            this.updateLastUpdate();
            
        } catch (error) {
            console.error('❌ Ошибка автообновления:', error);
        } finally {
            this.isUpdating = false;
        }
    }

    stopAutoUpdate() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
            this.updateInterval = null;
            console.log('⏸️ Автообновление остановлено');
        }
    }

    // Утилиты
    getStatusClass(status) {
        const statusClasses = {
            'stopped': 'status-stopped',
            'starting': 'status-starting',
            'running': 'status-running',
            'error': 'status-error'
        };
        return statusClasses[status] || 'status-unknown';
    }

    getStatusIcon(status) {
        const statusIcons = {
            'stopped': '⏹️',
            'starting': '⏳',
            'running': '✅',
            'error': '❌'
        };
        return statusIcons[status] || '❓';
    }

    formatDate(dateString) {
        if (!dateString) return 'Неизвестно';
        
        try {
            const date = new Date(dateString);
            return date.toLocaleString('ru-RU', {
                year: 'numeric',
                month: '2-digit',
                day: '2-digit',
                hour: '2-digit',
                minute: '2-digit',
                second: '2-digit'
            });
        } catch (error) {
            return dateString;
        }
    }

    updateLastUpdate() {
        const lastUpdateElement = document.getElementById('lastUpdate');
        if (lastUpdateElement) {
            lastUpdateElement.textContent = this.formatDate(new Date());
        }
    }

    showLoading(show) {
        const loader = document.getElementById('loadingIndicator');
        if (loader) {
            loader.style.display = show ? 'block' : 'none';
        }
    }

    showError(message) {
        this.showNotification(message, 'error');
    }

    showSuccess(message) {
        this.showNotification(message, 'success');
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;

        let container = document.getElementById('notificationsContainer');
        if (!container) {
            container = document.createElement('div');
            container.id = 'notificationsContainer';
            container.className = 'notifications-container';
            document.body.appendChild(container);
        }

        container.appendChild(notification);

        setTimeout(() => {
            if (notification.parentNode) {
                notification.parentNode.removeChild(notification);
            }
        }, 5000);

        console.log(`📢 ${type.toUpperCase()}: ${message}`);
    }
}

// Создаем глобальный экземпляр
const streamManager = new StreamManager();
window.streamManager = streamManager;
