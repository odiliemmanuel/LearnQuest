# Email Verification Architecture

LearnQuest uses RabbitMQ and a standalone notification service for transactional email delivery.

## Services

- `backend`: owns users, passwords, OTP hashes, verification attempts, and activation decisions.
- `notification-service`: consumes email events and delivers them via alwaysdata's SMTP servers.
- RabbitMQ: durable transport between the services.

The backend writes a verification and an `outbox_events` row in one database transaction. The outbox dispatcher publishes only committed events to the `learnquest.events` topic exchange. This prevents account creation from losing its email event during a restart.

The notification service consumes `identity.email_verification_requested` from a durable queue. It retries failures after 30 seconds up to five times, then places the message on a dead-letter queue. Temporary SMTP failures (450-499) are retried; permanent rejections (535, 550, etc.) go straight to the dead-letter queue.

## Local Development

Start RabbitMQ:

```bash
docker compose up -d rabbitmq
```

RabbitMQ management is available at `http://localhost:15672` with the development credentials from `docker-compose.yml`.

Set these values in `backend/.env` locally:

```dotenv
OTP_SECRET=replace-with-a-long-random-secret
RABBITMQ_URL=amqp://learnquest:learnquest-dev-only@localhost:5672/
```

Set these values for the notification service shell:

```dotenv
RABBITMQ_URL=amqp://learnquest:learnquest-dev-only@localhost:5672/
SMTP_HOST=smtp-<account>.alwaysdata.net
SMTP_PORT=465
SMTP_USERNAME=no-reply@your-domain
SMTP_PASSWORD=<mailbox-password>
EMAIL_FROM=LearnQuest <no-reply@your-domain>
```

Create the sending mailbox in the alwaysdata administration interface (E-mails > Addresses) before running the service.

Run the services independently:

```bash
go run ./cmd/server
```

```bash
go run .
```

Run the second command from `notification-service`.

## Production

1. Deploy RabbitMQ as a managed service or a persistent private instance. Do not expose its management port publicly.
   Use an `amqps://` URL with TLS for any broker outside the private deployment network.
2. Store `RABBITMQ_URL`, `OTP_SECRET`, `SMTP_HOST`, `SMTP_USERNAME`, `SMTP_PASSWORD`, and `EMAIL_FROM` in the deployment secret manager.
3. Point the sending domain's MX records at `mx1.alwaysdata.com` and `mx2.alwaysdata.com`, and publish its SPF and DKIM DNS records. Add a DMARC policy after validation.
4. Use a non-personal sender such as `no-reply@your-domain`.
5. Run the main backend and notification service as separate deployments.

The notification service submits email over the alwaysdata SMTP relay (`smtp-<account>.alwaysdata.net`) using SSL/TLS on port 465 and authenticates with the mailbox credentials. It does not use Gmail app passwords.
