# syntax=docker/dockerfile:1
# Build: docker build -t learnquest-frontend ./frontend
# Run:   docker run -p 80:80 -e BACKEND_URL=http://backend:8080 learnquest-frontend

FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
# The official nginx image runs envsubst over /etc/nginx/templates/*.template
# at startup, so ${BACKEND_URL} in this file is filled in from the environment.
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
# Defaults for local/compose use; a PaaS overrides both.
ENV BACKEND_URL=http://backend:8080
ENV PORT=80
EXPOSE 80
