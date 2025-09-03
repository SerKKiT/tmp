<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# 🎬 StreamApp - Live Video Streaming Platform

Полнофункциональная платформа для live видео стриминга с поддержкой SRT входящих потоков и HLS доставки контента. Построена на Go с микросервисной архитектурой, Docker контейнеризацией и автоматическим CI/CD развертыванием.

## 📋 Содержание

- [Особенности](#-%D0%BE%D1%81%D0%BE%D0%B1%D0%B5%D0%BD%D0%BD%D0%BE%D1%81%D1%82%D0%B8)
- [Архитектура](#-%D0%B0%D1%80%D1%85%D0%B8%D1%82%D0%B5%D0%BA%D1%82%D1%83%D1%80%D0%B0)
- [Требования](#-%D1%82%D1%80%D0%B5%D0%B1%D0%BE%D0%B2%D0%B0%D0%BD%D0%B8%D1%8F)
- [Быстрый старт](#-%D0%B1%D1%8B%D1%81%D1%82%D1%80%D1%8B%D0%B9-%D1%81%D1%82%D0%B0%D1%80%D1%82)
- [API Документация](#-api-%D0%B4%D0%BE%D0%BA%D1%83%D0%BC%D0%B5%D0%BD%D1%82%D0%B0%D1%86%D0%B8%D1%8F)
- [Конфигурация](#-%D0%BA%D0%BE%D0%BD%D1%84%D0%B8%D0%B3%D1%83%D1%80%D0%B0%D1%86%D0%B8%D1%8F)
- [Развертывание](#-%D1%80%D0%B0%D0%B7%D0%B2%D0%B5%D1%80%D1%82%D1%8B%D0%B2%D0%B0%D0%BD%D0%B8%D0%B5)
- [Безопасность](#-%D0%B1%D0%B5%D0%B7%D0%BE%D0%BF%D0%B0%D1%81%D0%BD%D0%BE%D1%81%D1%82%D1%8C)
- [Мониторинг](#-%D0%BC%D0%BE%D0%BD%D0%B8%D1%82%D0%BE%D1%80%D0%B8%D0%BD%D0%B3)
- [Troubleshooting](#-troubleshooting)


## 🚀 Особенности

### **Streaming возможности:**

- 📡 **SRT прием** - высококачественный входящий протокол для стримов
- 🎥 **HLS доставка** - потоковая передача для всех устройств
- 🔄 **Real-time перепаковка** - без перекодирования видео (сохранение качества)
- 📊 **Live мониторинг** - автоматическое отслеживание статуса потоков
- ⚡ **Автоперезапуск** - устойчивость к сбоям сети


### **API и интеграция:**

- 🌐 **RESTful API** - полное управление потоками
- 🎯 **HLS Metadata API** - специализированный endpoint для видео плееров
- 🔐 **Token аутентификация** - безопасный доступ к контенту
- 🌍 **CORS поддержка** - интеграция с веб-приложениями
- 📱 **Responsive UI** - веб-интерфейс для управления


### **Инфраструктура:**

- 🐳 **Docker контейнеры** - легкое развертывание
- 🔒 **HTTPS по умолчанию** - безопасность всех соединений
- 🔄 **CI/CD автоматизация** - GitLab pipelines
- 📈 **Горизонтальное масштабирование** - готовность к нагрузке
- 💾 **PostgreSQL** - надежное хранение данных


## 🏗️ Архитектура

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client App    │    │   Web Browser   │    │  Mobile App     │
│                 │    │                 │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │ HTTPS/443
                    ┌────────────▼────────────┐
                    │       nginx             │
                    │   (Reverse Proxy)       │
                    │   SSL Termination       │
                    └────┬─────────────┬──────┘
                         │             │
                ┌────────▼──────┐   ┌──▼──────────────┐
                │   go-app      │   │ streaming-      │
                │   :8080       │   │ service :8081   │
                │               │   │                 │
                │ • Tasks API   │   │ • Stream API    │
                │ • Web UI      │   │ • HLS API       │
                │ • Health      │   │ • FFmpeg Mgmt   │
                └───────┬───────┘   └─────────┬───────┘
                        │                     │
                        └──────────┬──────────┘
                                   │
                        ┌──────────▼──────────┐
                        │    PostgreSQL       │
                        │     :5432           │
                        │                     │
                        │ • Tasks            │
                        │ • Streams          │
                        │ • Metadata         │
                        └─────────────────────┘

              ┌─────────────────────────────────────┐
              │         External Components          │
              │                                     │
              │  SRT Input  →  FFmpeg  →  HLS Files │
              │  :10000+        Process      /hls/  │
              └─────────────────────────────────────┘
```


### **Компоненты системы:**

| Компонент | Порт | Назначение |
| :-- | :-- | :-- |
| **nginx** | 80, 443 | Reverse proxy, SSL termination, статические файлы |
| **go-app** | 8080 | Основное приложение, веб-интерфейс, управление задачами |
| **streaming-service** | 8081 | Управление потоками, FFmpeg, HLS API |
| **PostgreSQL** | 5432 | База данных |
| **FFmpeg** | 10000+ | SRT прием и HLS генерация |

## 🛠️ Требования

### **Системные требования:**

- **OS:** Linux (Ubuntu 20.04+, CentOS 8+, Debian 11+)
- **CPU:** 2+ cores, x64 архитектура
- **RAM:** 4GB+ (рекомендуется 8GB для production)
- **Storage:** 20GB+ свободного места
- **Network:** Стабильное интернет соединение


### **Программное обеспечение:**

- **Docker:** 24.0+
- **Docker Compose:** 2.0+
- **Git:** Для клонирования репозитория
- **OpenSSL:** Для генерации SSL сертификатов


### **Сетевые порты:**

- **80** - HTTP (редирект на HTTPS)
- **443** - HTTPS (основной доступ)
- **8080** - go-app (внутренний)
- **8081** - streaming-service (внутренний)
- **5432** - PostgreSQL (внутренний)
- **10000-10100** - SRT входящие потоки


## 🚀 Быстрый старт

### **1. Клонирование репозитория:**

```bash
git clone https://github.com/your-org/streamapp.git
cd streamapp
```


### **2. Настройка окружения:**

```bash
# Создайте конфигурационный файл
cp deployments/.env.production.template deployments/.env

# Отредактируйте переменные окружения
nano deployments/.env
```

**Пример `.env` файла:**

```env
# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_secure_password
DB_NAME=streamapp
DB_SSLMODE=disable

# Application Configuration  
STREAMING_SERVICE_URL=http://streaming-service:8081
SERVER_IP=192.168.1.100
ENVIRONMENT=production

# Security
STREAM_TOKEN_SECRET=your_secret_key_change_in_production
CDN_DOMAIN=your-domain.com
```


### **3. SSL сертификаты:**

**Для разработки (самоподписанный):**

```bash
mkdir -p nginx/ssl
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout nginx/ssl/server.key \
  -out nginx/ssl/server.crt \
  -days 365 \
  -subj "/C=RU/ST=State/L=City/O=StreamApp/CN=your-domain.com" \
  -addext "subjectAltName=IP:192.168.1.100,DNS:your-domain.com"

chmod 600 nginx/ssl/server.key
chmod 644 nginx/ssl/server.crt
```

**Для production (Let's Encrypt):**

```bash
# Установите certbot
sudo apt install certbot

# Получите сертификат
sudo certbot certonly --standalone -d your-domain.com

# Скопируйте сертификаты
sudo cp /etc/letsencrypt/live/your-domain.com/fullchain.pem nginx/ssl/server.crt
sudo cp /etc/letsencrypt/live/your-domain.com/privkey.pem nginx/ssl/server.key
```


### **4. Запуск приложения:**

```bash
cd deployments

# Сборка и запуск всех сервисов
docker-compose up -d

# Проверка статуса
docker-compose ps

# Просмотр логов
docker-compose logs -f
```


### **5. Проверка работоспособности:**

```bash
# Проверка API
curl -k https://your-domain.com/api/tasks

# Проверка веб-интерфейса
# Откройте https://your-domain.com в браузере
```


## 📚 API Документация

### **Основные endpoints:**

#### **📋 Управление задачами:**

```http
# Получить список задач
GET /api/tasks
Accept: application/json

# Создать новую задачу
POST /api/tasks
Content-Type: application/json
{
  "name": "Live Stream 1",
  "stream_id": "unique-stream-id"
}
```


#### **🎥 Управление потоками:**

```http
# Список активных потоков
GET /api/streams
Accept: application/json

# Информация о потоке
GET /api/streams/{stream_id}
Accept: application/json

# Запуск потока
POST /api/streams
Content-Type: application/json
{
  "stream_id": "unique-stream-id",
  "action": "start"
}

# Остановка потока  
POST /api/streams
Content-Type: application/json
{
  "stream_id": "unique-stream-id",
  "action": "stop"
}
```


#### **📺 HLS Metadata API:**

```http
# Получить HLS метаданные
GET /api/hls/{stream_id}
Accept: application/json

# Пример ответа:
{
  "message": "HLS stream metadata",
  "stream_id": "unique-stream-id",
  "status": "running",
  "data": {
    "hls_url": "https://your-domain.com/hls/unique-stream-id/playlist.m3u8",
    "stream_url": "https://your-domain.com/hls/unique-stream-id/playlist.m3u8?token=abc123",
    "access_token": "abc123def456",
    "cdn_domain": "your-domain.com",
    "is_live": true,
    "start_time": "2025-09-03T00:00:00Z"
  }
}
```


#### **🔍 Мониторинг:**

```http
# Health check основного приложения
GET /api/health

# Health check streaming сервиса  
GET /streaming/api/health

# Debug информация о плейлисте
GET /api/debug/{stream_id}
```


### **Статусы потоков:**

- **`starting`** - поток инициализируется, ожидает SRT подключение
- **`running`** - активная трансляция, генерируются HLS сегменты
- **`stopped`** - поток остановлен
- **`error`** - ошибка в работе потока


## ⚙️ Конфигурация

### **nginx конфигурация:**

Основные location блоки в `nginx/nginx.conf`:

```nginx
# Основное приложение (веб-интерфейс, задачи)
location / {
    proxy_pass http://go_app;
}

# Управление потоками  
location /api/streams {
    proxy_pass http://streaming_service;
}

# HLS метаданные для плееров
location /api/hls/ {
    proxy_pass http://streaming_service;
}

# HLS статические файлы
location /hls/ {
    alias /opt/streamapp/hls_output/;
}
```


### **Docker конфигурация:**

`deployments/docker-compose.yml` содержит:

- Multi-stage сборку Go приложений
- Оптимизированные образы на Alpine Linux
- Правильные зависимости между сервисами
- Health checks для всех компонентов
- Persistent volumes для данных


### **Переменные окружения:**

| Переменная | Описание | По умолчанию |
| :-- | :-- | :-- |
| `DB_HOST` | Хост PostgreSQL | `postgres` |
| `DB_PASSWORD` | Пароль БД | `mypassword123` |
| `SERVER_IP` | IP сервера | `localhost` |
| `CDN_DOMAIN` | Домен для HLS URLs | `SERVER_IP` |
| `STREAM_TOKEN_SECRET` | Ключ для генерации токенов | `default-secret` |

## 🚀 Развертывание

### **Production развертывание:**

1. **Подготовка сервера:**
```bash
# Обновление системы
sudo apt update && sudo apt upgrade -y

# Установка Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Добавление пользователя в группу docker
sudo usermod -aG docker $USER
```

2. **Настройка firewall:**
```bash
# UFW
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp  
sudo ufw allow 443/tcp
sudo ufw allow 10000:10100/udp
sudo ufw --force enable

# Или iptables
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT  
sudo iptables -A INPUT -p udp --dport 10000:10100 -j ACCEPT
```

3. **SSL сертификаты для production:**
```bash
# Let's Encrypt автоматическое обновление
echo "0 12 * * * /usr/bin/certbot renew --quiet" | sudo crontab -
```


### **GitLab CI/CD:**

Автоматическое развертывание настроено через `.gitlab-ci.yml`:

**Этапы pipeline:**

1. **Build** - сборка Docker образов
2. **Deploy:staging** - автоматический deploy на staging
3. **Test** - автоматическое тестирование
4. **Deploy:production** - ручной deploy на production

**Настройка переменных в GitLab:**

```
Settings → CI/CD → Variables:
- SSH_PRIVATE_KEY: SSH ключ для доступа к серверам
- STAGING_SERVER: IP staging сервера  
- STAGING_USER: Пользователь на staging сервере
- PRODUCTION_SERVER: IP production сервера
- PRODUCTION_USER: Пользователь на production сервере
```


## 🔒 Безопасность

### **SSL/TLS конфигурация:**

- Принудительное перенаправление HTTP → HTTPS
- TLS 1.2+ протоколы
- Современные cipher suites
- HSTS headers для браузеров


### **Аутентификация API:**

- Token-based доступ к HLS контенту
- SHA256 подписи для потоковых URL
- Временные токены с истечением


### **Сетевая безопасность:**

- Firewall правила для минимального доступа
- Internal Docker networks
- Nginx rate limiting (опционально)


### **Рекомендации:**

```bash
# Смените пароли по умолчанию
DB_PASSWORD=secure_random_password_123

# Используйте сильный secret key
STREAM_TOKEN_SECRET=cryptographically_secure_random_key

# Обновляйте SSL сертификаты
# Let's Encrypt автоматически каждые 90 дней
```


## 📊 Мониторинг

### **Health Checks:**

```bash
# Проверка всех сервисов
curl -k https://your-domain.com/api/health

# Streaming service health  
curl -k https://your-domain.com/streaming/api/health

# Docker контейнеры
docker-compose ps
```


### **Логи:**

```bash
# Все логи приложения
docker-compose logs -f

# Конкретный сервис
docker-compose logs -f go-app
docker-compose logs -f streaming-service
docker-compose logs -f nginx

# FFmpeg логи потоков
tail -f /opt/streamapp/logs/stream-id.log
```


### **Метрики:**

Автоматически отслеживается:

- Количество активных потоков
- Статусы потоков (starting/running/stopped)
- HLS активность (новые сегменты)
- Сетевые соединения
- Ошибки FFmpeg процессов


## 🔧 Troubleshooting

### **Частые проблемы:**

**1. Nginx не запускается (SSL ошибки):**

```bash
# Проверьте сертификаты
openssl x509 -in nginx/ssl/server.crt -text -noout
openssl rsa -in nginx/ssl/server.key -check -noout

# Права доступа  
chmod 600 nginx/ssl/server.key
chmod 644 nginx/ssl/server.crt

# Пересоздайте сертификат
rm nginx/ssl/server.*
# Следуйте инструкциям из раздела "Быстрый старт"
```

**2. 404 на API endpoints:**

```bash
# Проверьте nginx конфигурацию
docker exec nginx-proxy nginx -t

# Проверьте маршрутизацию
curl -k http://localhost:8080/api/health  # Прямой доступ
curl -k http://localhost:8081/api/health  # Прямой доступ
```

**3. FFmpeg не запускается:**

```bash
# Проверьте логи
docker logs streaming-service --tail 50

# Проверьте порты SRT
netstat -tlnp | grep :10000

# Права на директории
ls -la /opt/streamapp/hls_output/
```

**4. База данных не подключается:**

```bash
# Проверьте статус PostgreSQL
docker-compose logs postgres

# Тест подключения
docker exec -it postgres-db psql -U postgres -c "SELECT 1;"

# Проверьте переменные окружения
docker-compose config
```


### **Диагностические команды:**

```bash
# Системная информация
docker system df
docker system info

# Сетевое подключение
docker network ls
docker network inspect deployments_app-network

# Процессы и ресурсы
docker stats

# Полная диагностика
docker-compose ps
docker-compose top
docker-compose logs --tail 100
```


### **Восстановление после сбоев:**

```bash
# Полный рестарт системы
docker-compose down
docker system prune -f
docker-compose up -d

# Сброс данных (ОСТОРОЖНО!)
docker-compose down -v
docker system prune -af --volumes
# Восстановите из бекапа или пересоздайте
```


