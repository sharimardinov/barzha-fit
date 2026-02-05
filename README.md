# BarzhaFit

Фитнес-приложение для Telegram Mini App с AI-поддержкой для отслеживания питания, тренировок и прогресса.

## 🚀 Возможности

- 📊 **Отслеживание питания** — AI-анализ блюд и автоматический подсчет КБЖУ
- ⏱️ **Таймер тренировок** — отслеживание подходов, отдыха и прогресса
- 🚶 **Шаги** — синхронизация с iOS HealthKit
- 🎯 **Цели** — настройка дневных целей по калориям, БЖУ и шагам
- 👤 **Профиль** — расчет коэффициента активности и целей на основе антропометрии
- 📝 **Программы тренировок** — ручное редактирование тренировочных программ

## 🏗️ Архитектура

### Backend (Go)
- **API сервер** — REST API для Telegram Mini App
- **Telegram Bot** — бот для отправки ссылок на приложение
- **PostgreSQL** — основная база данных
- **OpenAI API** — AI для анализа питания

### Frontend (Vanilla JS)
- Telegram Mini App с современным UI
- Адаптивный дизайн с эффектом матового стекла
- Модульная архитектура с разделением по вкладкам

### iOS App
- Синхронизация шагов через HealthKit
- Синхронизация пульса во время тренировок
- Нативная авторизация через Google

## 📋 Требования

- Go 1.24+
- PostgreSQL 14+
- OpenAI API ключ
- Telegram Bot Token
- Google OAuth credentials (для авторизации)


## 📁 Структура проекта

```
barzhafit/
├── backend/              # Backend логика
│   ├── config/          # Конфигурация
│   ├── domain/          # Доменные модели
│   ├── migrations/      # SQL миграции
│   ├── service/         # Бизнес-логика
│   ├── storage/         # Репозитории
│   └── util/            # Утилиты
├── cmd/
│   └── api/             # Точка входа приложения
├── tgapp/               # Telegram Mini App
│   ├── assets/          # Frontend файлы
│   │   ├── index.html   # Главная страница
│   │   ├── app.js       # Основной JS
│   │   ├── app.css      # Стили
│   │   └── tabs/        # Модули вкладок
│   ├── api.go           # API handlers
│   └── auth*.go         # Авторизация
├── ios/                 # iOS приложение
└── go.mod               # Go зависимости
```

## 🔑 API Endpoints

### Авторизация
- `POST /auth/verify` — проверка токена
- `GET /auth/google` — Google OAuth
- `POST /auth/logout` — выход

### Профиль
- `GET /api/profile/get` — получение профиля
- `POST /api/profile/set` — сохранение профиля
- `GET /api/training-profile/get` — тренировочный профиль
- `POST /api/training-profile/set` — сохранение тренировочного профиля

### Питание
- `GET /api/nutrition/today` — питание за сегодня
- `POST /api/nutrition/add` — добавление приема пищи
- `GET /api/nutrition/week` — питание за неделю

### Тренировки
- `GET /api/workout/plan/get` — план тренировки на сегодня
- `POST /api/workout/session/start` — начало тренировки
- `POST /api/workout/session/set/finish` — завершение подхода

### Программы
- `GET /api/plan/get` — получение программы
- `POST /api/plan/set` — сохранение программы
- `POST /api/activity/estimate` — пересчет активности

### Цели
- `GET /api/targets/get` — получение целей
- `POST /api/targets/set` — установка целей
- `POST /api/targets/refresh` — пересчет из профиля

## 🔒 Безопасность

- Все API endpoints требуют авторизации через Telegram
- Пароли и токены хранятся в переменных окружения
- Используется HTTPS для production
- SQL injection защита через параметризованные запросы

## 📄 Лицензия

-

## 👥 Авторы

-

## 🙏 Благодарности

- OpenAI за API для AI-функций
- Telegram за платформу Mini Apps
- PostgreSQL за надежную БД
