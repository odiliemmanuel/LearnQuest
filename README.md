# LearnQuest

An adaptive learning platform for students. LearnQuest combines a Go API backend, a React SPA, and an event-driven notification service to deliver a personalised learning experience: browse curriculum subjects and topics, study interactive lessons, take quizzes with instant feedback, and track progress, badges and challenges — with AI-assisted explanations and mistake analysis along the way.

## Features

- **Curriculum engine** — education levels, classes, terms, subjects and topics, seeded from the backend.
- **Lessons & quizzes** — topic-based lessons, quiz sessions with question-specific feedback, graded results and recovery recommendations.
- **Knowledge & progress tracking** — per-topic mastery, knowledge map and adaptive recovery suggestions.
- **Gamification** — points, badges and challenges to keep learners engaged.
- **AI assistance** — mistake analysis, working-out review, concept explanations and generated practice questions (OpenAI, with a mock mode for development).
- **Library / textbook** — original, condensed study notes for every curriculum topic, written for LearnQuest and read entirely in-app (`scripts/textbook_fetcher` generates them with Gemini and posts them via a secret-key ingest endpoint).
- **Email verification** — OTP signup flow with a transactional outbox; RabbitMQ transports events to a dedicated notification service that delivers them over SMTP.
- **Session management** — JWT-based auth persisted across refreshes; redirection to sign-in only when a session expires or a new device is used.
- **Admin dashboard** — owner-only overview, per-student stats and search, and soft-delete for accounts (data and history are preserved in the database).
- **Light & dark mode** — persisted theme with dark-mode-aware components.

## Tech Stack

### Backend (`backend/`)
- **Language:** Go
- **HTTP framework:** Gin
- **ORM:** GORM with PostgreSQL
- **Auth:** JWT (`golang-jwt/jwt/v5`) + bcrypt (`golang.org/x/crypto`)
- **Messaging:** RabbitMQ (`amqp091-go`) with a transactional outbox pattern
- **AI:** `sashabaranov/go-openai`
- **Config:** `joho/godotenv`

### Frontend (`frontend/`)
- **Framework:** React 19 + TypeScript
- **Routing:** React Router 7
- **Styling:** Tailwind CSS v4 (via `@tailwindcss/vite`)
- **Build tooling:** Vite 8, oxlint

### Notification service (`notification-service/`)
- **Language:** Go
- **Messaging:** RabbitMQ (`amqp091-go`)
- **Email:** SMTP over TLS/STARTTLS (alwaysdata) with retry and dead-letter queues
- **Config:** `joho/godotenv`

### Infrastructure
- **RabbitMQ** via Docker Compose (`rabbitmq:4-management`)
- **PostgreSQL** (local or managed)

## Architecture

```
Frontend (React/Vite)
        │  HTTP/JSON
        ▼
Backend (Gin) ───> PostgreSQL (GORM)
        │
        │  transactional outbox → publish
        ▼
   RabbitMQ (learnquest.events)
        │  consume
        ▼
Notification service ──SMTP──> alwaysdata email
```

Signups write the user and an outbox event in one database transaction. The outbox dispatcher publishes committed events to a durable RabbitMQ queue, and the notification service delivers verification emails with retry and dead-letter handling. See `docs/email-verification.md` for details.

## Getting Started

### Prerequisites
- Go 1.26+, Node.js 20+, PostgreSQL
- Docker (for RabbitMQ)

### 1. Infrastructure

```bash
docker compose up -d rabbitmq
```

RabbitMQ management is available at `http://localhost:15672`.

### 2. Backend

```bash
cd backend
cp .env.example .env     # fill in DATABASE_URL, JWT_SECRET, OTP_SECRET, RABBITMQ_URL
go run ./cmd/server
```

The backend listens on `:8080` and auto-migrates the database schema.

### 3. Notification service

```bash
cd notification-service
cp .env.example .env     # fill in SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD, EMAIL_FROM
go run .
```

### 4. Frontend

```bash
cd frontend
npm install
npm run dev
```

Open `http://localhost:5173`, register, verify your email using the OTP, and land on the dashboard.

## Repository Layout

```
backend/               Go API server (auth, curriculum, quiz, AI, gamification, library, admin, ...)
frontend/              React + TypeScript single-page app
notification-service/  Go worker that delivers emails over SMTP
scripts/textbook_fetcher/  Python script that generates library study notes via Gemini
docs/                  Architecture notes
docker-compose.yml     RabbitMQ for local development
```

## License

Add your preferred license here.