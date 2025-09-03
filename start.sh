#!/bin/bash

# Определяем IP адрес сервера автоматически
SERVER_IP=$(hostname -I | awk '{print $1}')

# Альтернативный способ для разных ОС
if [ -z "$SERVER_IP" ]; then
    SERVER_IP=$(ip route get 8.8.8.8 | awk -F"src " 'NR==1{split($2,a," ");print a[1]}')
fi

# Если все еще не удалось определить, используем localhost
if [ -z "$SERVER_IP" ]; then
    SERVER_IP="localhost"
fi

echo "🌐 Запуск с IP адресом: $SERVER_IP"

# Экспортируем переменную окружения
export SERVER_IP=$SERVER_IP

# Переходим в папку deployments
cd deployments/

# Останавливаем и удаляем существующие контейнеры
echo "🛑 Останавливаем существующие контейнеры..."
docker-compose down

# Собираем и запускаем контейнеры
echo "🔨 Сборка и запуск контейнеров..."
docker-compose up --build -d

# Ждем запуска сервисов
echo "⏳ Ожидание запуска сервисов..."
sleep 15

# Проверяем статус контейнеров
echo "📊 Статус контейнеров:"
docker-compose ps

# Проверяем здоровье приложения
echo "🔍 Проверка здоровья приложения..."
curl -f http://localhost:8080/api/health 2>/dev/null && echo "✅ Основное приложение работает" || echo "❌ Основное приложение не отвечает"
curl -f http://localhost:8081/api/health 2>/dev/null && echo "✅ Streaming сервис работает" || echo "❌ Streaming сервис не отвечает"

echo ""
echo "✅ Система запущена!"
echo "🔗 Основной интерфейс: http://$SERVER_IP"
echo "📺 Просмотр стримов: http://$SERVER_IP/viewer.html"
echo "🔧 API health check: http://$SERVER_IP/api/health"
echo "🎥 Streaming service: http://$SERVER_IP:8081/api/health"
echo ""
echo "📜 Для просмотра логов используйте:"
echo "   cd deployments && docker-compose logs -f"
echo ""
echo "🛑 Для остановки используйте:"
echo "   cd deployments && docker-compose down"
